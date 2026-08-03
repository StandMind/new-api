#!/usr/bin/env python3
"""Contract the legacy access-policy columns after the split rollout is stable."""

from __future__ import annotations

import argparse
import gzip
import hashlib
import json
import os
import re
import subprocess
import sys
import tempfile
import time
from pathlib import Path
from typing import Any


SCHEMA = "aivrae-access-policy-contract/v1"
STATE_ID = 1
STANDARD_LEVEL = "standard"
DEFAULT_ROUTE_GROUP = "default"

LEGACY_COLUMNS = (
    "subscription_plans.downgrade_group",
    "subscription_plans.upgrade_group",
    "user_subscriptions.downgrade_group",
    "user_subscriptions.prev_user_group",
    "user_subscriptions.upgrade_group",
    "users.group",
)

LEGACY_OPTIONS = (
    "GroupGroupRatio",
    "GroupRatio",
    "ModelRequestRateLimitGroup",
    "TopupGroupRatio",
    "UserUsableGroups",
    "group_ratio_setting.group_special_usable_group",
)


class ContractMigrationError(RuntimeError):
    pass


def canonical_json(value: Any) -> str:
    return json.dumps(
        value,
        ensure_ascii=True,
        sort_keys=True,
        separators=(",", ":"),
        allow_nan=False,
    )


def fingerprint(value: Any) -> str:
    return hashlib.sha256(canonical_json(value).encode("utf-8")).hexdigest()


def sql_literal(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def parse_json(raw: str, label: str) -> Any:
    try:
        return json.loads(raw)
    except (TypeError, json.JSONDecodeError) as error:
        raise ContractMigrationError(f"{label} is not valid JSON: {error}") from error


STATUS_SQL = r'''
WITH expected_columns(table_name, column_name) AS (
    VALUES
        ('users', 'group'),
        ('subscription_plans', 'upgrade_group'),
        ('subscription_plans', 'downgrade_group'),
        ('user_subscriptions', 'upgrade_group'),
        ('user_subscriptions', 'downgrade_group'),
        ('user_subscriptions', 'prev_user_group')
), present_columns AS (
    SELECT expected.table_name || '.' || expected.column_name AS name
    FROM expected_columns AS expected
    JOIN information_schema.columns AS columns
      ON columns.table_schema = current_schema()
     AND columns.table_name = expected.table_name
     AND columns.column_name = expected.column_name
), legacy_options AS (
    SELECT key
    FROM options
    WHERE key IN (
        'GroupRatio', 'GroupGroupRatio', 'UserUsableGroups',
        'TopupGroupRatio', 'ModelRequestRateLimitGroup',
        'group_ratio_setting.group_special_usable_group'
    )
)
SELECT json_build_object(
    'legacy_columns', COALESCE(
        (SELECT json_agg(name ORDER BY name COLLATE "C") FROM present_columns),
        '[]'::json
    ),
    'legacy_option_keys', COALESCE(
        (SELECT json_agg(key ORDER BY key COLLATE "C") FROM legacy_options),
        '[]'::json
    ),
    'access_policy_state', (
        SELECT json_build_object(
            'id', id,
            'expanded_at', expanded_at,
            'contracted_at', contracted_at,
            'initial_route_group_codes', initial_route_group_codes,
            'initial_route_group_fingerprint', initial_route_group_fingerprint
        )
        FROM access_policy_states
        WHERE id = 1
    ),
    'invalid_references', json_build_object(
        'active_users_missing_level', (
            SELECT COUNT(*)
            FROM users AS users
            LEFT JOIN user_levels AS levels ON levels.code = users.user_level
            WHERE users.deleted_at IS NULL
              AND (NULLIF(BTRIM(users.user_level), '') IS NULL OR levels.code IS NULL)
        ),
        'subscription_plan_upgrade', (
            SELECT COUNT(*)
            FROM subscription_plans AS plans
            LEFT JOIN user_levels AS levels ON levels.code = plans.upgrade_user_level
            WHERE NULLIF(BTRIM(plans.upgrade_user_level), '') IS NOT NULL
              AND levels.code IS NULL
        ),
        'subscription_plan_downgrade', (
            SELECT COUNT(*)
            FROM subscription_plans AS plans
            LEFT JOIN user_levels AS levels ON levels.code = plans.downgrade_user_level
            WHERE NULLIF(BTRIM(plans.downgrade_user_level), '') IS NOT NULL
              AND levels.code IS NULL
        ),
        'user_subscription_upgrade', (
            SELECT COUNT(*)
            FROM user_subscriptions AS subscriptions
            LEFT JOIN user_levels AS levels ON levels.code = subscriptions.upgrade_user_level
            WHERE NULLIF(BTRIM(subscriptions.upgrade_user_level), '') IS NOT NULL
              AND levels.code IS NULL
        ),
        'user_subscription_downgrade', (
            SELECT COUNT(*)
            FROM user_subscriptions AS subscriptions
            LEFT JOIN user_levels AS levels ON levels.code = subscriptions.downgrade_user_level
            WHERE NULLIF(BTRIM(subscriptions.downgrade_user_level), '') IS NOT NULL
              AND levels.code IS NULL
        ),
        'user_subscription_previous', (
            SELECT COUNT(*)
            FROM user_subscriptions AS subscriptions
            LEFT JOIN user_levels AS levels ON levels.code = subscriptions.previous_user_level
            WHERE NULLIF(BTRIM(subscriptions.previous_user_level), '') IS NOT NULL
              AND levels.code IS NULL
        ),
        'level_route_grants', (
            SELECT COUNT(*)
            FROM user_level_route_groups AS grants
            LEFT JOIN user_levels AS levels ON levels.code = grants.user_level_code
            LEFT JOIN route_groups AS groups ON groups.code = grants.route_group_code
            WHERE levels.code IS NULL OR groups.code IS NULL
        )
    )
)::text;
'''.strip()


# This state contains no credentials. It deliberately includes every old/new
# policy value that would become impossible to compare after DROP COLUMN.
STATE_EXPRESSION = r'''
jsonb_build_object(
    'options', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.key COLLATE "C")
        FROM (
            SELECT key, value
            FROM options
            WHERE key IN (
                'GroupRatio', 'GroupGroupRatio', 'UserUsableGroups',
                'TopupGroupRatio', 'ModelRequestRateLimitGroup',
                'group_ratio_setting.group_special_usable_group'
            )
            ORDER BY key COLLATE "C"
        ) AS item
    ), '[]'::jsonb),
    'users', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (
            SELECT id, user_level, "group" AS legacy_group, deleted_at
            FROM users ORDER BY id
        ) AS item
    ), '[]'::jsonb),
    'subscription_plans', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (
            SELECT id, upgrade_user_level, downgrade_user_level,
                   upgrade_group, downgrade_group
            FROM subscription_plans ORDER BY id
        ) AS item
    ), '[]'::jsonb),
    'user_subscriptions', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (
            SELECT id, upgrade_user_level, downgrade_user_level,
                   previous_user_level, upgrade_group, downgrade_group,
                   prev_user_group
            FROM user_subscriptions ORDER BY id
        ) AS item
    ), '[]'::jsonb),
    'user_levels', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.code COLLATE "C")
        FROM (
            SELECT code, name, description, is_default, enabled, topup_ratio,
                   request_limit, success_request_limit, created_at, updated_at
            FROM user_levels ORDER BY code COLLATE "C"
        ) AS item
    ), '[]'::jsonb),
    'route_groups', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.code COLLATE "C")
        FROM (
            SELECT code, name, description, base_ratio, enabled,
                   created_at, updated_at
            FROM route_groups ORDER BY code COLLATE "C"
        ) AS item
    ), '[]'::jsonb),
    'grants', COALESCE((
        SELECT jsonb_agg(to_jsonb(item)
            ORDER BY item.user_level_code COLLATE "C", item.route_group_code COLLATE "C")
        FROM (
            SELECT user_level_code, route_group_code, price_ratio,
                   created_at, updated_at
            FROM user_level_route_groups
            ORDER BY user_level_code COLLATE "C", route_group_code COLLATE "C"
        ) AS item
    ), '[]'::jsonb),
    'access_policy_state', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (
            SELECT id, initial_route_group_codes,
                   initial_route_group_fingerprint, expanded_at,
                   initial_user_level_cleanup_at, contracted_at
            FROM access_policy_states ORDER BY id
        ) AS item
    ), '[]'::jsonb),
    'enabled_default_abilities', COALESCE((
        SELECT jsonb_agg(to_jsonb(item)
            ORDER BY item.channel_id, item.model COLLATE "C")
        FROM (
            SELECT channel_id, model, priority, weight
            FROM abilities
            WHERE "group" = 'default' AND enabled = TRUE
            ORDER BY channel_id, model COLLATE "C"
        ) AS item
    ), '[]'::jsonb)
)
'''.strip()

STATE_SQL = f"SELECT {STATE_EXPRESSION}::text;"


def legacy_user_level(raw: Any) -> str:
    code = raw.strip() if isinstance(raw, str) else ""
    if not code or code == DEFAULT_ROUTE_GROUP:
        return STANDARD_LEVEL
    return code


def legacy_subscription_level(raw: Any) -> str:
    code = raw.strip() if isinstance(raw, str) else ""
    if not code:
        return ""
    if code == DEFAULT_ROUTE_GROUP:
        return STANDARD_LEVEL
    return code


def analyze_status(status: dict[str, Any]) -> dict[str, Any]:
    if not isinstance(status, dict):
        raise ContractMigrationError("database status is not an object")
    columns = status.get("legacy_columns")
    options = status.get("legacy_option_keys")
    state = status.get("access_policy_state")
    invalid = status.get("invalid_references")
    if not isinstance(columns, list) or not isinstance(options, list):
        raise ContractMigrationError("database status has invalid legacy object lists")
    if state is not None and not isinstance(state, dict):
        raise ContractMigrationError("database status has an invalid access-policy state")
    if not isinstance(invalid, dict):
        raise ContractMigrationError("database status has invalid reference counts")

    contracted_at = int((state or {}).get("contracted_at") or 0)
    expanded_at = int((state or {}).get("expanded_at") or 0)
    already_contracted = contracted_at > 0 and not columns and not options
    blockers: list[str] = []
    if state is None or expanded_at <= 0:
        blockers.append("access-policy expand migration has not completed")
    if contracted_at > 0 and not already_contracted:
        blockers.append("access-policy state is contracted but legacy objects remain")
    if contracted_at == 0 and set(columns) != set(LEGACY_COLUMNS):
        missing = sorted(set(LEGACY_COLUMNS) - set(columns))
        blockers.append(
            "legacy column set is incomplete before contraction: " + ", ".join(missing)
        )
    for kind, count in invalid.items():
        if not isinstance(count, int) or isinstance(count, bool) or count < 0:
            raise ContractMigrationError(f"invalid reference count for {kind}")
        if count:
            blockers.append(f"invalid {kind} references: {count}")
    return {
        "blockers": blockers,
        "already_contracted": already_contracted,
        "legacy_columns": sorted(columns),
        "legacy_option_keys": sorted(options),
        "access_policy_state": state,
        "invalid_references": invalid,
    }


def analyze_state(state: dict[str, Any]) -> dict[str, Any]:
    fields = (
        "options",
        "users",
        "subscription_plans",
        "user_subscriptions",
        "user_levels",
        "route_groups",
        "grants",
        "access_policy_state",
        "enabled_default_abilities",
    )
    for field in fields:
        if not isinstance(state.get(field), list):
            raise ContractMigrationError(f"database state field {field} is not an array")

    blockers: list[str] = []
    levels = {
        row.get("code"): row
        for row in state["user_levels"]
        if isinstance(row, dict) and isinstance(row.get("code"), str)
    }
    route_groups = {
        row.get("code"): row
        for row in state["route_groups"]
        if isinstance(row, dict) and isinstance(row.get("code"), str)
    }
    if len(levels) != len(state["user_levels"]):
        blockers.append("user level rows contain an invalid or duplicate code")
    if len(route_groups) != len(state["route_groups"]):
        blockers.append("route group rows contain an invalid or duplicate code")

    standard = levels.get(STANDARD_LEVEL)
    if standard is None:
        blockers.append("standard user level is missing")
    elif not standard.get("is_default") or not standard.get("enabled"):
        blockers.append("standard user level must be enabled and default")
    defaults = [row.get("code") for row in state["user_levels"] if row.get("is_default")]
    if defaults != [STANDARD_LEVEL]:
        blockers.append("standard must be the only default user level")

    for row in state["users"]:
        if not isinstance(row, dict):
            blockers.append("users contains an invalid row")
            continue
        user_level = row.get("user_level")
        if row.get("deleted_at") is None:
            if not isinstance(user_level, str) or not user_level.strip():
                blockers.append(f"active user {row.get('id')} has no user_level")
            elif user_level not in levels:
                blockers.append(
                    f"active user {row.get('id')} references missing level {user_level}"
                )
        expected = legacy_user_level(row.get("legacy_group"))
        if user_level != expected:
            blockers.append(
                f"user {row.get('id')} new/legacy level mismatch: {user_level!r} != {expected!r}"
            )

    plan_pairs = (
        ("upgrade_user_level", "upgrade_group"),
        ("downgrade_user_level", "downgrade_group"),
    )
    subscription_pairs = (
        ("upgrade_user_level", "upgrade_group"),
        ("downgrade_user_level", "downgrade_group"),
        ("previous_user_level", "prev_user_group"),
    )
    for collection, pairs in (
        ("subscription_plans", plan_pairs),
        ("user_subscriptions", subscription_pairs),
    ):
        for row in state[collection]:
            if not isinstance(row, dict):
                blockers.append(f"{collection} contains an invalid row")
                continue
            for new_field, old_field in pairs:
                new_value = row.get(new_field)
                expected = legacy_subscription_level(row.get(old_field))
                if new_value != expected:
                    blockers.append(
                        f"{collection} {row.get('id')} {new_field}/{old_field} mismatch"
                    )
                if expected and expected not in levels:
                    blockers.append(
                        f"{collection} {row.get('id')} references missing level {expected}"
                    )

    standard_grants: list[str] = []
    for row in state["grants"]:
        if not isinstance(row, dict):
            blockers.append("grants contains an invalid row")
            continue
        level_code = row.get("user_level_code")
        route_code = row.get("route_group_code")
        if level_code not in levels or route_code not in route_groups:
            blockers.append(
                f"grant {level_code!r}/{route_code!r} has a missing reference"
            )
        if level_code == STANDARD_LEVEL and isinstance(route_code, str):
            standard_grants.append(route_code)
    standard_grants.sort()

    state_rows = state["access_policy_state"]
    initial_codes: list[str] = []
    initial_fingerprint = ""
    if len(state_rows) != 1 or state_rows[0].get("id") != STATE_ID:
        blockers.append("access_policy_states must contain singleton id 1")
    else:
        policy_state = state_rows[0]
        if int(policy_state.get("expanded_at") or 0) <= 0:
            blockers.append("access-policy expansion is incomplete")
        if int(policy_state.get("contracted_at") or 0) != 0:
            blockers.append("access policy has already been contracted")
        try:
            parsed_codes = parse_json(
                policy_state.get("initial_route_group_codes"),
                "initial_route_group_codes",
            )
            if not isinstance(parsed_codes, list) or any(
                not isinstance(code, str) for code in parsed_codes
            ):
                raise ContractMigrationError(
                    "initial_route_group_codes must be a string array"
                )
            initial_codes = sorted(parsed_codes)
        except ContractMigrationError as error:
            blockers.append(str(error))
        initial_fingerprint = str(
            policy_state.get("initial_route_group_fingerprint") or ""
        )
        if initial_codes and fingerprint(initial_codes) != initial_fingerprint:
            blockers.append("initial route-group fingerprint does not match its code list")

    if DEFAULT_ROUTE_GROUP in standard_grants:
        blockers.append("standard must not be authorized for the default route group")
    if standard_grants != initial_codes:
        blockers.append("standard authorization differs from the frozen expand snapshot")
    default_group = route_groups.get(DEFAULT_ROUTE_GROUP)
    if default_group is None:
        blockers.append("default route group is missing before contraction")
    elif default_group.get("enabled"):
        blockers.append("default route group must be disabled before contraction")
    if state["enabled_default_abilities"]:
        blockers.append("default route group still has enabled Ability rows")

    return {
        "blockers": blockers,
        "counts": {field: len(state[field]) for field in fields},
        "standard_authorization": standard_grants,
        "initial_route_group_codes": initial_codes,
        "initial_route_group_fingerprint": initial_fingerprint,
    }


def build_contract_sql(expected_state: dict[str, Any], contracted_at: int) -> str:
    expected = sql_literal(canonical_json(expected_state))
    retired_options = ", ".join(sql_literal(key) for key in LEGACY_OPTIONS)
    return f'''
BEGIN;
LOCK TABLE users, subscription_plans, user_subscriptions,
           user_levels, route_groups, user_level_route_groups,
           access_policy_states, options, abilities
           IN ACCESS EXCLUSIVE MODE;

DO $access_policy_contract_state$
DECLARE
    current_state jsonb;
BEGIN
    SELECT {STATE_EXPRESSION} INTO current_state;
    IF current_state <> {expected}::jsonb THEN
        RAISE EXCEPTION 'access-policy state changed after dry-run; rerun the preview';
    END IF;
END
$access_policy_contract_state$;

ALTER TABLE users DROP COLUMN "group";
ALTER TABLE subscription_plans
    DROP COLUMN upgrade_group,
    DROP COLUMN downgrade_group;
ALTER TABLE user_subscriptions
    DROP COLUMN upgrade_group,
    DROP COLUMN downgrade_group,
    DROP COLUMN prev_user_group;

DELETE FROM options WHERE key IN ({retired_options});

DO $access_policy_contract_marker$
DECLARE
    affected bigint;
BEGIN
    UPDATE access_policy_states
    SET contracted_at = {int(contracted_at)}
    WHERE id = {STATE_ID} AND expanded_at > 0 AND contracted_at = 0;
    GET DIAGNOSTICS affected = ROW_COUNT;
    IF affected <> 1 THEN
        RAISE EXCEPTION 'access-policy contract marker was not updated exactly once';
    END IF;
END
$access_policy_contract_marker$;
COMMIT;
'''.strip()


class PostgresRunner:
    def __init__(self, compose_file: Path, service: str) -> None:
        self.compose_file = compose_file.resolve()
        self.service = service

    def run_sql(self, sql: str, expect_json: bool = False) -> Any:
        command = [
            "docker",
            "compose",
            "-f",
            str(self.compose_file),
            "exec",
            "-T",
            self.service,
            "sh",
            "-ec",
            'export PGPASSWORD="${POSTGRES_PASSWORD}"; '
            'exec psql -X -qAt -v ON_ERROR_STOP=1 -U "${POSTGRES_USER}" -d "${POSTGRES_DB}"',
        ]
        result = subprocess.run(
            command,
            input=sql,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            timeout=180,
            check=False,
        )
        if result.returncode != 0:
            raise ContractMigrationError(
                "PostgreSQL command failed: " + result.stderr.strip()[-2000:]
            )
        if not expect_json:
            return None
        lines = [line for line in result.stdout.splitlines() if line.strip()]
        if len(lines) != 1:
            raise ContractMigrationError(
                "PostgreSQL query returned an unexpected number of rows"
            )
        try:
            value = json.loads(lines[0])
        except json.JSONDecodeError as error:
            raise ContractMigrationError(
                "PostgreSQL query returned invalid JSON"
            ) from error
        if not isinstance(value, dict):
            raise ContractMigrationError("PostgreSQL query did not return an object")
        return value

    def read_status(self) -> dict[str, Any]:
        return self.run_sql(STATUS_SQL, expect_json=True)

    def read_state(self) -> dict[str, Any]:
        return self.run_sql(STATE_SQL, expect_json=True)


def validate_backup(path: Path) -> None:
    if not path.is_absolute() or not path.is_file():
        raise ContractMigrationError(f"database backup is missing: {path}")
    if path.stat().st_size <= 0:
        raise ContractMigrationError(f"database backup is empty: {path}")
    try:
        with gzip.open(path, "rb") as handle:
            while handle.read(1024 * 1024):
                pass
    except (OSError, EOFError) as error:
        raise ContractMigrationError(
            f"database backup failed gzip validation: {path}"
        ) from error


def create_backup(script: Path, backup_dir: Path) -> Path:
    if not script.is_absolute() or not script.is_file() or not os.access(script, os.X_OK):
        raise ContractMigrationError(f"backup script is not executable: {script}")
    before = (
        {path.resolve() for path in backup_dir.glob("*.sql.gz")}
        if backup_dir.is_dir()
        else set()
    )
    started_at = time.time()
    result = subprocess.run(
        [str(script)],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        timeout=900,
        check=False,
    )
    if result.returncode != 0:
        raise ContractMigrationError(
            "database backup script failed: " + result.stderr.strip()[-1000:]
        )
    candidates = sorted(
        (path.resolve() for path in backup_dir.glob("*.sql.gz")),
        key=lambda path: path.stat().st_mtime,
        reverse=True,
    )
    if not candidates:
        raise ContractMigrationError(f"backup script created no backup in {backup_dir}")
    backup = candidates[0]
    if backup in before and backup.stat().st_mtime < started_at - 1:
        raise ContractMigrationError("backup script did not create a new database backup")
    validate_backup(backup)
    return backup


def public_report(
    mode: str,
    status_analysis: dict[str, Any],
    state_fingerprint: str | None = None,
    state_analysis: dict[str, Any] | None = None,
    backup: Path | None = None,
) -> dict[str, Any]:
    blockers = list(status_analysis["blockers"])
    if state_analysis is not None:
        blockers.extend(state_analysis["blockers"])
    report: dict[str, Any] = {
        "schema": SCHEMA,
        "mode": mode,
        "already_contracted": status_analysis["already_contracted"],
        "blockers": blockers,
        "legacy_columns": status_analysis["legacy_columns"],
        "legacy_option_keys": status_analysis["legacy_option_keys"],
        "invalid_references": status_analysis["invalid_references"],
    }
    if state_fingerprint is not None:
        report["state_fingerprint"] = state_fingerprint
    if state_analysis is not None:
        report["standard_authorization"] = {
            "count": len(state_analysis["standard_authorization"]),
            "codes": state_analysis["standard_authorization"],
            "fingerprint": state_analysis["initial_route_group_fingerprint"],
        }
        report["row_counts"] = state_analysis["counts"]
    if backup is not None:
        report["backup"] = {"path": str(backup), "gzip_valid": True}
    state = status_analysis.get("access_policy_state") or {}
    report["expanded_at"] = int(state.get("expanded_at") or 0)
    report["contracted_at"] = int(state.get("contracted_at") or 0)
    return report


def write_secure_json(path: Path, value: dict[str, Any]) -> None:
    path = path.resolve()
    path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    fd, temporary_name = tempfile.mkstemp(prefix="." + path.name + ".", dir=path.parent)
    temporary = Path(temporary_name)
    try:
        os.fchmod(fd, 0o600)
        with os.fdopen(fd, "w", encoding="utf-8") as handle:
            json.dump(value, handle, ensure_ascii=False, indent=2, sort_keys=True)
            handle.write("\n")
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, path)
        os.chmod(path, 0o600)
    finally:
        if temporary.exists():
            temporary.unlink()


def require_sha256(value: str | None) -> str:
    if value is None or re.fullmatch(r"[a-f0-9]{64}", value) is None:
        raise ContractMigrationError(
            "--expected-state-fingerprint must be the lowercase SHA-256 from dry-run"
        )
    return value


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--dry-run", action="store_true")
    mode.add_argument("--apply", action="store_true")
    parser.add_argument(
        "--compose-file",
        type=Path,
        default=Path("/opt/new-api-stack/docker-compose.yml"),
    )
    parser.add_argument("--postgres-service", default="postgres")
    parser.add_argument("--expected-state-fingerprint")
    parser.add_argument("--backup-file", type=Path)
    parser.add_argument(
        "--backup-script",
        type=Path,
        default=Path("/opt/new-api-stack/backup-db.sh"),
    )
    parser.add_argument(
        "--backup-dir",
        type=Path,
        default=Path("/opt/new-api-stack/backups"),
    )
    parser.add_argument("--report", type=Path)
    return parser.parse_args()


def emit_report(args: argparse.Namespace, report: dict[str, Any]) -> None:
    if args.report:
        write_secure_json(args.report, report)
    print(json.dumps(report, ensure_ascii=False, indent=2, sort_keys=True))


def main() -> int:
    args = parse_args()
    if not args.compose_file.is_absolute() or not args.compose_file.is_file():
        raise ContractMigrationError(f"compose file is missing: {args.compose_file}")
    runner = PostgresRunner(args.compose_file, args.postgres_service)
    status_analysis = analyze_status(runner.read_status())

    if status_analysis["already_contracted"]:
        report = public_report(
            "apply" if args.apply else "dry-run", status_analysis
        )
        emit_report(args, report)
        return 0

    state = runner.read_state()
    state_analysis = analyze_state(state)
    state_fingerprint = fingerprint(state)
    report = public_report(
        "dry-run", status_analysis, state_fingerprint, state_analysis
    )
    blockers = report["blockers"]
    if args.dry_run:
        emit_report(args, report)
        return 2 if blockers else 0

    expected = require_sha256(args.expected_state_fingerprint)
    if state_fingerprint != expected:
        raise ContractMigrationError("database state differs from dry-run; rerun the preview")
    if blockers:
        raise ContractMigrationError(
            "access-policy contraction is blocked: " + "; ".join(blockers)
        )

    backup = (
        args.backup_file.resolve()
        if args.backup_file
        else create_backup(args.backup_script.resolve(), args.backup_dir.resolve())
    )
    validate_backup(backup)

    status_after_backup = analyze_status(runner.read_status())
    state_after_backup = runner.read_state()
    analysis_after_backup = analyze_state(state_after_backup)
    if fingerprint(state_after_backup) != expected:
        raise ContractMigrationError(
            "database state changed while creating the backup; rerun dry-run"
        )
    backup_blockers = (
        status_after_backup["blockers"] + analysis_after_backup["blockers"]
    )
    if backup_blockers:
        raise ContractMigrationError(
            "access-policy contraction became blocked after backup: "
            + "; ".join(backup_blockers)
        )

    contracted_at = int(time.time())
    runner.run_sql(build_contract_sql(state_after_backup, contracted_at))
    postcheck = analyze_status(runner.read_status())
    if not postcheck["already_contracted"]:
        raise ContractMigrationError("post-migration verification did not reach contracted state")
    if postcheck["blockers"]:
        raise ContractMigrationError(
            "post-migration verification failed: " + "; ".join(postcheck["blockers"])
        )
    actual_contracted_at = int(
        (postcheck.get("access_policy_state") or {}).get("contracted_at") or 0
    )
    if actual_contracted_at != contracted_at:
        raise ContractMigrationError("contract marker differs from the applied transaction")

    final_report = public_report(
        "apply",
        postcheck,
        state_fingerprint=expected,
        state_analysis=analysis_after_backup,
        backup=backup,
    )
    emit_report(args, final_report)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (ContractMigrationError, subprocess.SubprocessError) as error:
        print(f"access-policy contract migration failed: {error}", file=sys.stderr)
        raise SystemExit(1)
