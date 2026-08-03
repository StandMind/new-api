#!/usr/bin/env python3
"""Arm the guarded Aivrae user-level and route-group expand migration."""

from __future__ import annotations

import argparse
import gzip
import hashlib
import json
import math
import os
import re
import subprocess
import sys
import tempfile
import time
from pathlib import Path
from typing import Any


SCHEMA = "aivrae-access-policy-expand/v1"
DEFAULT_ROUTE_GROUP = "default"
GUARD_ID = 1


class ExpandMigrationError(RuntimeError):
    pass


# Keep this expression structurally identical to
# model.postgresLegacyAccessPolicyStateSQL. The Go migration normalizes and
# compares this complete state again after taking its own transaction locks.
STATE_EXPRESSION = r'''
jsonb_build_object(
    'options', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.key)
        FROM (
            SELECT key, value FROM options
            WHERE key IN (
                'GroupRatio', 'UserUsableGroups', 'TopupGroupRatio',
                'ModelRequestRateLimitGroup', 'GroupGroupRatio',
                'group_ratio_setting.group_special_usable_group'
            ) ORDER BY key
        ) AS item
    ), '[]'::jsonb),
    'channels', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (SELECT id, "group" AS group_name FROM channels ORDER BY id) AS item
    ), '[]'::jsonb),
    'abilities', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.channel_id, item.group_name, item.model, item.enabled, item.priority, item.weight)
        FROM (
            SELECT channel_id, "group" AS group_name, model, enabled, priority, weight
            FROM abilities ORDER BY channel_id, "group", model, enabled, priority, weight
        ) AS item
    ), '[]'::jsonb),
    'tokens', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (
            SELECT id, "group" AS group_name, group_chain
            FROM tokens WHERE deleted_at IS NULL ORDER BY id
        ) AS item
    ), '[]'::jsonb),
    'routes', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.group_name, item.model)
        FROM (
            SELECT "group" AS group_name, model, tiers, updated_at
            FROM group_model_routes ORDER BY "group", model
        ) AS item
    ), '[]'::jsonb),
    'users', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (
            SELECT id, "group" AS group_name FROM users
            WHERE deleted_at IS NULL ORDER BY id
        ) AS item
    ), '[]'::jsonb),
    'subscription_plans', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (
            SELECT id, upgrade_group, downgrade_group
            FROM subscription_plans ORDER BY id
        ) AS item
    ), '[]'::jsonb),
    'user_subscriptions', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (
            SELECT id, upgrade_group, downgrade_group, prev_user_group
            FROM user_subscriptions ORDER BY id
        ) AS item
    ), '[]'::jsonb)
)
'''.strip()

STATE_SQL = f"SELECT {STATE_EXPRESSION}::text;"


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
        return json.loads(
            raw,
            parse_constant=lambda value: (_ for _ in ()).throw(
                ValueError(f"non-finite number {value}")
            ),
        )
    except (json.JSONDecodeError, ValueError) as error:
        raise ExpandMigrationError(f"{label} is not valid JSON: {error}") from error


def decode_option_map(
    option_map: dict[str, Any], key: str
) -> dict[str, Any] | None:
    raw = option_map.get(key)
    if raw is None or (isinstance(raw, str) and not raw.strip()):
        return None
    if not isinstance(raw, str):
        raise ExpandMigrationError(f"{key} is not stored as JSON text")
    value = parse_json(raw, key)
    if value is None:
        return None
    if not isinstance(value, dict):
        raise ExpandMigrationError(f"{key} must be a JSON object")
    return value


def valid_number(value: Any) -> bool:
    return (
        isinstance(value, (int, float))
        and not isinstance(value, bool)
        and math.isfinite(float(value))
    )


def validate_code(code: Any, label: str) -> str:
    if not isinstance(code, str):
        raise ExpandMigrationError(f"{label} contains a non-string code")
    normalized = code.strip()
    if not normalized or normalized == "auto":
        raise ExpandMigrationError(f"{label} contains an empty or reserved code")
    if normalized != code:
        raise ExpandMigrationError(f"{label} code {code!r} has surrounding whitespace")
    return code


def split_route_groups(raw: Any) -> list[str]:
    if not isinstance(raw, str):
        return []
    return [
        code
        for part in raw.split(",")
        if (code := part.strip()) and code != "auto"
    ]


def validate_legacy_options(option_map: dict[str, Any]) -> dict[str, dict[str, Any]]:
    group_ratios = decode_option_map(option_map, "GroupRatio")
    if group_ratios is None:
        group_ratios = {DEFAULT_ROUTE_GROUP: 1}
    usable_groups = decode_option_map(option_map, "UserUsableGroups") or {}
    topup_ratios = decode_option_map(option_map, "TopupGroupRatio") or {}
    rate_limits = decode_option_map(option_map, "ModelRequestRateLimitGroup") or {}
    price_overrides = decode_option_map(option_map, "GroupGroupRatio") or {}
    special_groups = (
        decode_option_map(
            option_map, "group_ratio_setting.group_special_usable_group"
        )
        or {}
    )

    for code, ratio in group_ratios.items():
        validate_code(code, "GroupRatio")
        if not valid_number(ratio) or float(ratio) < 0:
            raise ExpandMigrationError(f"GroupRatio.{code} must be non-negative and finite")
    for code, name in usable_groups.items():
        validate_code(code, "UserUsableGroups")
        if not isinstance(name, str):
            raise ExpandMigrationError(f"UserUsableGroups.{code} must be a string")
    for code, ratio in topup_ratios.items():
        validate_code(code, "TopupGroupRatio")
        if not valid_number(ratio) or float(ratio) <= 0:
            raise ExpandMigrationError(f"TopupGroupRatio.{code} must be positive and finite")
    for code, limits in rate_limits.items():
        validate_code(code, "ModelRequestRateLimitGroup")
        if (
            not isinstance(limits, list)
            or len(limits) != 2
            or any(not isinstance(item, int) or isinstance(item, bool) for item in limits)
            or any(item < 0 for item in limits)
        ):
            raise ExpandMigrationError(
                f"ModelRequestRateLimitGroup.{code} must contain two non-negative integers"
            )
    for level_code, overrides in price_overrides.items():
        validate_code(level_code, "GroupGroupRatio")
        if not isinstance(overrides, dict):
            raise ExpandMigrationError(f"GroupGroupRatio.{level_code} must be an object")
        for route_code, ratio in overrides.items():
            validate_code(route_code, "GroupGroupRatio")
            if not valid_number(ratio) or float(ratio) < 0:
                raise ExpandMigrationError(
                    f"GroupGroupRatio.{level_code}.{route_code} must be non-negative and finite"
                )
    for level_code, rules in special_groups.items():
        validate_code(level_code, "group_special_usable_group")
        if not isinstance(rules, dict):
            raise ExpandMigrationError(
                f"group_special_usable_group.{level_code} must be an object"
            )
        for raw_route_code, description in rules.items():
            if not isinstance(raw_route_code, str):
                raise ExpandMigrationError("group_special_usable_group has a non-string rule")
            route_code = raw_route_code
            if route_code.startswith("+:") or route_code.startswith("-:"):
                route_code = route_code[2:]
            validate_code(route_code, "group_special_usable_group")
            if not isinstance(description, str):
                raise ExpandMigrationError(
                    f"group_special_usable_group.{level_code}.{raw_route_code} must be a string"
                )

    return {
        "group_ratios": group_ratios,
        "usable_groups": usable_groups,
        "topup_ratios": topup_ratios,
        "rate_limits": rate_limits,
        "price_overrides": price_overrides,
        "special_groups": special_groups,
    }


def analyze_state(state: dict[str, Any]) -> dict[str, Any]:
    blockers: list[str] = []
    fields = (
        "options",
        "channels",
        "abilities",
        "tokens",
        "routes",
        "users",
        "subscription_plans",
        "user_subscriptions",
    )
    for field in fields:
        if not isinstance(state.get(field), list):
            raise ExpandMigrationError(f"database state field {field} is not an array")

    option_map = {
        row.get("key"): row.get("value")
        for row in state["options"]
        if isinstance(row, dict)
    }
    try:
        options = validate_legacy_options(option_map)
    except ExpandMigrationError as error:
        blockers.append(str(error))
        options = {"group_ratios": {}, "usable_groups": {}}

    codes: set[str] = set()
    for code in options["group_ratios"]:
        if code and code != "auto":
            codes.add(code)
    for code in options["usable_groups"]:
        if code and code != "auto":
            codes.add(code)
    for overrides in options.get("price_overrides", {}).values():
        codes.update(code for code in overrides if code and code != "auto")
    for rules in options.get("special_groups", {}).values():
        for raw_code in rules:
            if raw_code.startswith("-:"):
                continue
            code = raw_code[2:] if raw_code.startswith("+:") else raw_code
            if code and code != "auto":
                codes.add(code)
    for row in state["channels"]:
        codes.update(split_route_groups(row.get("group_name")))
    for row in state["abilities"]:
        codes.update(split_route_groups(row.get("group_name")))
    for row in state["tokens"]:
        codes.update(split_route_groups(row.get("group_name")))
        raw_chain = row.get("group_chain")
        if raw_chain:
            try:
                chain = parse_json(raw_chain, f"token {row.get('id')} group_chain")
                if not isinstance(chain, list) or any(
                    not isinstance(item, str) for item in chain
                ):
                    raise ExpandMigrationError(
                        f"token {row.get('id')} group_chain must be a string array"
                    )
                for item in chain:
                    normalized = item.strip()
                    if normalized and normalized != "auto":
                        codes.add(normalized)
            except ExpandMigrationError as error:
                blockers.append(str(error))
    for row in state["routes"]:
        codes.update(split_route_groups(row.get("group_name")))

    enabled_default_abilities = sum(
        1
        for row in state["abilities"]
        if row.get("group_name") == DEFAULT_ROUTE_GROUP and row.get("enabled") is True
    )
    if enabled_default_abilities:
        blockers.append(
            "default route group still has "
            f"{enabled_default_abilities} enabled Ability rows"
        )

    all_codes = sorted(codes)
    standard_codes = [code for code in all_codes if code != DEFAULT_ROUTE_GROUP]
    return {
        "blockers": blockers,
        "all_route_group_codes": all_codes,
        "standard_route_group_codes": standard_codes,
        "route_group_fingerprint": fingerprint(standard_codes),
        "enabled_default_abilities": enabled_default_abilities,
        "counts": {field: len(state[field]) for field in fields},
    }


def build_arm_sql(
    state: dict[str, Any], analysis: dict[str, Any], armed_at: int
) -> str:
    expected_state = sql_literal(canonical_json(state))
    expected_codes = sql_literal(canonical_json(analysis["standard_route_group_codes"]))
    expected_fingerprint = sql_literal(analysis["route_group_fingerprint"])
    return f'''
BEGIN;
LOCK TABLE options, channels, abilities, tokens, users,
           subscription_plans, user_subscriptions, group_model_routes
           IN SHARE ROW EXCLUSIVE MODE;

DO $access_policy_state$
DECLARE
    current_state jsonb;
BEGIN
    SELECT {STATE_EXPRESSION} INTO current_state;
    IF current_state <> {expected_state}::jsonb THEN
        RAISE EXCEPTION 'access-policy state changed after dry-run; rerun the preview';
    END IF;
END
$access_policy_state$;

CREATE TABLE IF NOT EXISTS access_policy_migration_guards (
    id bigint PRIMARY KEY,
    expected_route_group_codes text NOT NULL,
    expected_fingerprint varchar(64) NOT NULL,
    expected_legacy_state text NOT NULL,
    armed_at bigint NOT NULL,
    applied_at bigint NOT NULL DEFAULT 0
);
LOCK TABLE access_policy_migration_guards IN EXCLUSIVE MODE;

DO $access_policy_guard$
DECLARE
    expanded_rows bigint := 0;
    affected bigint;
BEGIN
    IF to_regclass('public.access_policy_states') IS NOT NULL THEN
        EXECUTE 'SELECT COUNT(*) FROM access_policy_states WHERE expanded_at > 0'
        INTO expanded_rows;
    END IF;
    IF expanded_rows <> 0 THEN
        RAISE EXCEPTION 'access-policy expand migration has already run';
    END IF;

    INSERT INTO access_policy_migration_guards (
        id, expected_route_group_codes, expected_fingerprint,
        expected_legacy_state, armed_at, applied_at
    ) VALUES (
        {GUARD_ID}, {expected_codes}, {expected_fingerprint},
        {expected_state}, {int(armed_at)}, 0
    )
    ON CONFLICT (id) DO UPDATE SET
        expected_route_group_codes = EXCLUDED.expected_route_group_codes,
        expected_fingerprint = EXCLUDED.expected_fingerprint,
        expected_legacy_state = EXCLUDED.expected_legacy_state,
        armed_at = EXCLUDED.armed_at
    WHERE access_policy_migration_guards.applied_at = 0;
    GET DIAGNOSTICS affected = ROW_COUNT;
    IF affected <> 1 THEN
        RAISE EXCEPTION 'an applied access-policy migration guard already exists';
    END IF;
END
$access_policy_guard$;
COMMIT;
'''.strip()


POSTCHECK_SQL = f'''
SELECT json_build_object(
    'id', id,
    'expected_route_group_codes', expected_route_group_codes::json,
    'expected_fingerprint', expected_fingerprint,
    'armed', armed_at > 0,
    'applied', applied_at > 0
)::text
FROM access_policy_migration_guards
WHERE id = {GUARD_ID};
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
            timeout=120,
            check=False,
        )
        if result.returncode != 0:
            raise ExpandMigrationError(
                "PostgreSQL command failed: " + result.stderr.strip()[-2000:]
            )
        if not expect_json:
            return None
        lines = [line for line in result.stdout.splitlines() if line.strip()]
        if len(lines) != 1:
            raise ExpandMigrationError(
                "PostgreSQL query returned an unexpected number of rows"
            )
        try:
            value = json.loads(lines[0])
        except json.JSONDecodeError as error:
            raise ExpandMigrationError("PostgreSQL query returned invalid JSON") from error
        if not isinstance(value, dict):
            raise ExpandMigrationError("PostgreSQL query did not return an object")
        return value

    def read_state(self) -> dict[str, Any]:
        return self.run_sql(STATE_SQL, expect_json=True)


def validate_backup(path: Path) -> None:
    if not path.is_absolute() or not path.is_file():
        raise ExpandMigrationError(f"database backup is missing: {path}")
    if path.stat().st_size <= 0:
        raise ExpandMigrationError(f"database backup is empty: {path}")
    try:
        with gzip.open(path, "rb") as handle:
            while handle.read(1024 * 1024):
                pass
    except (OSError, EOFError) as error:
        raise ExpandMigrationError(
            f"database backup failed gzip validation: {path}"
        ) from error


def create_backup(script: Path, backup_dir: Path) -> Path:
    if not script.is_absolute() or not script.is_file() or not os.access(script, os.X_OK):
        raise ExpandMigrationError(f"backup script is not executable: {script}")
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
        raise ExpandMigrationError(
            "database backup script failed: " + result.stderr.strip()[-1000:]
        )
    candidates = sorted(
        (path.resolve() for path in backup_dir.glob("*.sql.gz")),
        key=lambda path: path.stat().st_mtime,
        reverse=True,
    )
    if not candidates:
        raise ExpandMigrationError(f"backup script created no backup in {backup_dir}")
    backup = candidates[0]
    if backup in before and backup.stat().st_mtime < started_at - 1:
        raise ExpandMigrationError("backup script did not create a new database backup")
    validate_backup(backup)
    return backup


def public_report(
    mode: str,
    state_fingerprint: str,
    analysis: dict[str, Any],
    backup: Path | None = None,
    postcheck: dict[str, Any] | None = None,
) -> dict[str, Any]:
    report: dict[str, Any] = {
        "schema": SCHEMA,
        "mode": mode,
        "state_fingerprint": state_fingerprint,
        "route_group_fingerprint": analysis["route_group_fingerprint"],
        "blockers": analysis["blockers"],
        "standard_authorization": {
            "count": len(analysis["standard_route_group_codes"]),
            "codes": analysis["standard_route_group_codes"],
        },
        "preflight": {
            "all_route_group_codes": analysis["all_route_group_codes"],
            "enabled_default_abilities": analysis["enabled_default_abilities"],
            "row_counts": analysis["counts"],
        },
    }
    if backup is not None:
        report["backup"] = {"path": str(backup), "gzip_valid": True}
    if postcheck is not None:
        report["guard"] = postcheck
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
    parser.add_argument("--expected-route-group-fingerprint")
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


def require_sha256(value: str | None, option: str) -> str:
    if value is None or re.fullmatch(r"[a-f0-9]{64}", value) is None:
        raise ExpandMigrationError(
            f"{option} must be the lowercase SHA-256 from dry-run"
        )
    return value


def main() -> int:
    args = parse_args()
    if not args.compose_file.is_absolute() or not args.compose_file.is_file():
        raise ExpandMigrationError(f"compose file is missing: {args.compose_file}")
    runner = PostgresRunner(args.compose_file, args.postgres_service)
    state = runner.read_state()
    analysis = analyze_state(state)
    state_fingerprint = fingerprint(state)

    if args.dry_run:
        report = public_report("dry-run", state_fingerprint, analysis)
        if args.report:
            write_secure_json(args.report, report)
        print(json.dumps(report, ensure_ascii=False, indent=2, sort_keys=True))
        return 2 if analysis["blockers"] else 0

    expected_state = require_sha256(
        args.expected_state_fingerprint, "--expected-state-fingerprint"
    )
    expected_routes = require_sha256(
        args.expected_route_group_fingerprint,
        "--expected-route-group-fingerprint",
    )
    if state_fingerprint != expected_state:
        raise ExpandMigrationError("database state differs from dry-run; rerun the preview")
    if analysis["route_group_fingerprint"] != expected_routes:
        raise ExpandMigrationError("route-group set differs from dry-run; rerun the preview")
    if analysis["blockers"]:
        raise ExpandMigrationError(
            "access-policy expansion is blocked: " + "; ".join(analysis["blockers"])
        )

    backup = (
        args.backup_file.resolve()
        if args.backup_file
        else create_backup(args.backup_script.resolve(), args.backup_dir.resolve())
    )
    validate_backup(backup)

    state_after_backup = runner.read_state()
    if fingerprint(state_after_backup) != expected_state:
        raise ExpandMigrationError(
            "database state changed while creating the backup; rerun dry-run"
        )
    analysis_after_backup = analyze_state(state_after_backup)
    if analysis_after_backup["route_group_fingerprint"] != expected_routes:
        raise ExpandMigrationError(
            "route-group set changed while creating the backup; rerun dry-run"
        )
    if analysis_after_backup["blockers"]:
        raise ExpandMigrationError("access-policy expansion became blocked after backup")

    runner.run_sql(
        build_arm_sql(state_after_backup, analysis_after_backup, int(time.time()))
    )
    postcheck = runner.run_sql(POSTCHECK_SQL, expect_json=True)
    expected_postcheck = {
        "id": GUARD_ID,
        "expected_route_group_codes": analysis_after_backup[
            "standard_route_group_codes"
        ],
        "expected_fingerprint": expected_routes,
        "armed": True,
        "applied": False,
    }
    if postcheck != expected_postcheck:
        raise ExpandMigrationError(
            "migration guard verification failed: "
            + canonical_json({"expected": expected_postcheck, "actual": postcheck})
        )

    report = public_report(
        "apply",
        state_fingerprint,
        analysis_after_backup,
        backup=backup,
        postcheck=postcheck,
    )
    if args.report:
        write_secure_json(args.report, report)
    print(json.dumps(report, ensure_ascii=False, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (ExpandMigrationError, subprocess.SubprocessError) as error:
        print(f"access-policy expand migration failed: {error}", file=sys.stderr)
        raise SystemExit(1)
