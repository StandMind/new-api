#!/usr/bin/env python3
"""Safely migrate Aivrae production tokens and remove enabled default routes."""

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


SCHEMA = "aivrae-smart-routing-cleanup/v1"
POOL_CHANNEL_NAME = "OpenAI-\u81ea\u5efa\u53f7\u6c60"
COMPATIBILITY_PREFIX = "OpenLux-Default/"
DEFAULT_GROUP = "default"
POOL_GROUP = "group_1"
EXPECTED_COMPATIBILITY_CHANNELS = 6
EXPECTED_POOL_MODELS = 11
EXPECTED_COMPATIBILITY_ABILITIES = 28
EXPECTED_ACTIVE_TOKENS = 6


class CleanupError(RuntimeError):
    pass


STATE_SQL = r'''
WITH target_channels AS (
    SELECT id
    FROM channels
    WHERE name = U&'OpenAI-\81EA\5EFA\53F7\6C60' OR name LIKE 'OpenLux-Default/%'
)
SELECT jsonb_build_object(
    'channels', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (
            SELECT id, name, type, status, "group" AS group_name, models,
                   priority, weight, md5(to_jsonb(channels)::text) AS row_fingerprint
            FROM channels
            WHERE name = U&'OpenAI-\81EA\5EFA\53F7\6C60' OR name LIKE 'OpenLux-Default/%'
            ORDER BY id
        ) AS item
    ), '[]'::jsonb),
    'abilities', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.channel_id, item.group_name, item.model)
        FROM (
            SELECT channel_id, "group" AS group_name, model, enabled,
                   priority, weight, tag
            FROM abilities
            WHERE channel_id IN (SELECT id FROM target_channels)
            ORDER BY channel_id, "group", model
        ) AS item
    ), '[]'::jsonb),
    'default_abilities', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.channel_id, item.model)
        FROM (
            SELECT channel_id, "group" AS group_name, model, enabled,
                   priority, weight, tag
            FROM abilities
            WHERE enabled = TRUE AND "group" = 'default'
            ORDER BY channel_id, model
        ) AS item
    ), '[]'::jsonb),
    'tokens', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (
            SELECT t.id, t.user_id, t.name, t.status, t."group" AS group_name,
                   t.group_chain, COALESCE(to_jsonb(t)->>'routing_priority', '') AS routing_priority,
                   t.deleted_at, u."group" AS user_group
            FROM tokens t
            LEFT JOIN users u ON u.id = t.user_id AND u.deleted_at IS NULL
            WHERE t.deleted_at IS NULL
            ORDER BY t.id
        ) AS item
    ), '[]'::jsonb),
    'options', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.key)
        FROM (
            SELECT key, value
            FROM options
            WHERE key IN (
                'GroupRatio', 'GroupGroupRatio', 'UserUsableGroups',
                'group_ratio_setting.group_special_usable_group'
            )
            ORDER BY key
        ) AS item
    ), '[]'::jsonb),
    'enabled_groups', COALESCE((
        SELECT jsonb_agg(item.group_name ORDER BY item.group_name)
        FROM (
            SELECT DISTINCT a."group" AS group_name
            FROM abilities a
            JOIN channels c ON c.id = a.channel_id
            WHERE a.enabled = TRUE AND c.status = 1 AND a."group" <> 'default'
              AND NOT (c.name LIKE 'OpenLux-Default/%' AND c.status = 1)
        ) AS item
    ), '[]'::jsonb),
    'routes', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.group_name, item.model)
        FROM (
            SELECT "group" AS group_name, model, tiers, updated_at
            FROM group_model_routes
            ORDER BY "group", model
        ) AS item
    ), '[]'::jsonb),
    'bindings', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.source_group, item.channel_id)
        FROM (
            SELECT id, source_group, channel_id, created_at, updated_at
            FROM openlux_price_sync_bindings
            ORDER BY source_group, channel_id
        ) AS item
    ), '[]'::jsonb)
)::text;
'''.strip()


def canonical_json(value: Any) -> str:
    return json.dumps(
        value,
        ensure_ascii=True,
        sort_keys=True,
        separators=(",", ":"),
        allow_nan=False,
    )


def fingerprint_state(state: dict[str, Any]) -> str:
    return hashlib.sha256(canonical_json(state).encode("utf-8")).hexdigest()


def parse_json_object(raw: Any, label: str) -> dict[str, Any]:
    if not isinstance(raw, str):
        raise CleanupError(f"{label} is not stored as JSON text")
    try:
        value = json.loads(raw)
    except json.JSONDecodeError as error:
        raise CleanupError(f"{label} is not valid JSON") from error
    if not isinstance(value, dict):
        raise CleanupError(f"{label} must be a JSON object")
    return value


def parse_optional_json_object(raw: Any, label: str) -> dict[str, Any]:
    if raw is None:
        return {}
    return parse_json_object(raw, label)


def split_csv(raw: Any) -> list[str]:
    if not isinstance(raw, str):
        return []
    return [item.strip() for item in raw.split(",") if item.strip()]


def route_channel_ids(raw_tiers: Any) -> set[int]:
    if isinstance(raw_tiers, str):
        try:
            raw_tiers = json.loads(raw_tiers)
        except json.JSONDecodeError as error:
            raise CleanupError("an explicit route contains invalid tier JSON") from error
    result: set[int] = set()
    if not isinstance(raw_tiers, list):
        return result
    for tier in raw_tiers:
        if not isinstance(tier, dict):
            continue
        channels = tier.get("channels")
        if not isinstance(channels, list):
            continue
        for channel in channels:
            if not isinstance(channel, dict):
                continue
            channel_id = channel.get("channel_id")
            if isinstance(channel_id, int):
                result.add(channel_id)
    return result


def analyze_state(state: dict[str, Any]) -> dict[str, Any]:
    blockers: list[str] = []
    channels = state.get("channels")
    abilities = state.get("abilities")
    default_abilities = state.get("default_abilities")
    tokens = state.get("tokens")
    options = state.get("options")
    enabled_groups = state.get("enabled_groups")
    routes = state.get("routes")
    bindings = state.get("bindings")
    for label, value in (
        ("channels", channels),
        ("abilities", abilities),
        ("default abilities", default_abilities),
        ("tokens", tokens),
        ("options", options),
        ("enabled groups", enabled_groups),
        ("routes", routes),
        ("bindings", bindings),
    ):
        if not isinstance(value, list):
            raise CleanupError(f"database state field {label} is not an array")

    pool_channels = [row for row in channels if row.get("name") == POOL_CHANNEL_NAME]
    active_compatibility = [
        row
        for row in channels
        if str(row.get("name", "")).startswith(COMPATIBILITY_PREFIX)
        and row.get("status") == 1
    ]
    historical_compatibility = [
        row
        for row in channels
        if str(row.get("name", "")).startswith(COMPATIBILITY_PREFIX)
        and row.get("status") != 1
    ]

    if len(pool_channels) != 1:
        blockers.append(f"expected one active {POOL_CHANNEL_NAME} channel, found {len(pool_channels)}")
        pool_channel: dict[str, Any] = {}
    else:
        pool_channel = pool_channels[0]
        if pool_channel.get("status") != 1:
            blockers.append("the self-hosted pool channel is not enabled")
        if set(split_csv(pool_channel.get("group_name"))) != {DEFAULT_GROUP, POOL_GROUP}:
            blockers.append("the self-hosted pool channel groups are not exactly default and group_1")
        models = split_csv(pool_channel.get("models"))
        if len(models) != EXPECTED_POOL_MODELS or len(set(models)) != EXPECTED_POOL_MODELS:
            blockers.append("the self-hosted pool channel no longer has exactly 11 unique models")

    if len(active_compatibility) != EXPECTED_COMPATIBILITY_CHANNELS:
        blockers.append(
            "expected six enabled OpenLux-Default compatibility channels, "
            f"found {len(active_compatibility)}"
        )
    pool_id = int(pool_channel.get("id", 0) or 0)
    compatibility_ids = sorted(int(row["id"]) for row in active_compatibility)
    target_ids = {pool_id, *compatibility_ids}
    target_abilities = [row for row in abilities if row.get("channel_id") in target_ids]
    pool_abilities = [row for row in target_abilities if row.get("channel_id") == pool_id]
    pool_default = [row for row in pool_abilities if row.get("group_name") == DEFAULT_GROUP]
    pool_group = [row for row in pool_abilities if row.get("group_name") == POOL_GROUP]
    compatibility_abilities = [
        row for row in target_abilities if row.get("channel_id") in compatibility_ids
    ]
    historical_id_set = {int(row["id"]) for row in historical_compatibility}
    historical_abilities = [
        row for row in abilities if row.get("channel_id") in historical_id_set
    ]
    if len(pool_abilities) != EXPECTED_POOL_MODELS * 2:
        blockers.append("the self-hosted pool no longer has exactly 22 Ability rows")
    if len(pool_default) != EXPECTED_POOL_MODELS or not all(row.get("enabled") for row in pool_default):
        blockers.append("the self-hosted pool default group does not have 11 enabled Ability rows")
    if len(pool_group) != EXPECTED_POOL_MODELS or not all(row.get("enabled") for row in pool_group):
        blockers.append("the self-hosted pool group_1 does not have 11 enabled Ability rows")
    if len(compatibility_abilities) != EXPECTED_COMPATIBILITY_ABILITIES:
        blockers.append("the compatibility channels no longer have exactly 28 Ability rows")
    if not all(row.get("enabled") for row in compatibility_abilities):
        blockers.append("one or more compatibility channel Ability rows are disabled")
    if any(row.get("enabled") for row in historical_abilities):
        blockers.append("one or more historical channel Ability rows are enabled")

    default_target_ids = {int(row.get("channel_id", 0)) for row in default_abilities}
    if len(default_abilities) != EXPECTED_POOL_MODELS + EXPECTED_COMPATIBILITY_ABILITIES:
        blockers.append("the global enabled default Ability count is not 39")
    if not default_target_ids.issubset(target_ids):
        blockers.append("default contains enabled Ability rows outside the cleanup targets")

    if len(tokens) != EXPECTED_ACTIVE_TOKENS:
        blockers.append(f"expected six undeleted tokens, found {len(tokens)}")

    option_map = {row.get("key"): row.get("value") for row in options}
    group_ratios: dict[str, Any] = {}
    try:
        group_ratios = parse_json_object(option_map.get("GroupRatio"), "GroupRatio")
        if float(group_ratios.get(POOL_GROUP, 0)) != 1.3:
            blockers.append("GroupRatio.group_1 is not 1.3")
    except (CleanupError, TypeError, ValueError) as error:
        blockers.append(str(error))
    try:
        usable_groups = parse_json_object(
            option_map.get("UserUsableGroups"), "UserUsableGroups"
        )
    except CleanupError as error:
        blockers.append(str(error))
        usable_groups = {}
    try:
        group_group_ratios = parse_optional_json_object(
            option_map.get("GroupGroupRatio"), "GroupGroupRatio"
        )
    except CleanupError as error:
        blockers.append(str(error))
        group_group_ratios = {}
    try:
        special_usable_groups = parse_optional_json_object(
            option_map.get("group_ratio_setting.group_special_usable_group"),
            "group_ratio_setting.group_special_usable_group",
        )
    except CleanupError as error:
        blockers.append(str(error))
        special_usable_groups = {}
    target_usable_groups = dict(usable_groups)
    if POOL_GROUP not in target_usable_groups:
        target_usable_groups[POOL_GROUP] = "\u81ea\u5efa\u53f7\u6c60"
    target_usable_groups_json = json.dumps(
        target_usable_groups,
        ensure_ascii=False,
        sort_keys=True,
        separators=(",", ":"),
    )

    enabled_group_set = {
        group for group in enabled_groups if isinstance(group, str) and group
    }
    token_routes: list[dict[str, Any]] = []
    for token in tokens:
        token_id = int(token.get("id", 0) or 0)
        user_group = token.get("user_group")
        if not isinstance(user_group, str) or not user_group:
            blockers.append(f"token {token_id} has no active user group")
            continue

        token_usable_groups = dict(target_usable_groups)
        special = special_usable_groups.get(user_group, {})
        if not isinstance(special, dict):
            blockers.append(f"special usable groups for {user_group} are not an object")
            continue
        for configured_group, description in special.items():
            if configured_group.startswith("-:"):
                token_usable_groups.pop(configured_group[2:], None)
            elif configured_group.startswith("+:"):
                token_usable_groups[configured_group[2:]] = description
            else:
                token_usable_groups[configured_group] = description
        token_usable_groups.setdefault(user_group, "user group")

        user_overrides = group_group_ratios.get(user_group, {})
        if not isinstance(user_overrides, dict):
            blockers.append(f"group ratios for {user_group} are not an object")
            continue
        route_groups = [
            group
            for group in token_usable_groups
            if group != DEFAULT_GROUP
            and group in enabled_group_set
            and group in group_ratios
        ]
        try:
            route_groups.sort(
                key=lambda group: (
                    float(user_overrides.get(group, group_ratios[group])),
                    group,
                )
            )
        except (TypeError, ValueError) as error:
            blockers.append(f"token {token_id} compatibility pricing is invalid: {error}")
            continue
        if not route_groups:
            blockers.append(f"token {token_id} has no non-default compatibility route")
            continue
        token_routes.append(
            {
                "id": token_id,
                "group": route_groups[0],
                "group_chain": json.dumps(
                    route_groups, ensure_ascii=False, separators=(",", ":")
                ),
            }
        )

    compatibility_id_set = set(compatibility_ids)
    route_references: list[str] = []
    for route in routes:
        referenced = route_channel_ids(route.get("tiers"))
        unsafe = referenced.intersection(compatibility_id_set)
        if pool_id in referenced and route.get("group_name") == DEFAULT_GROUP:
            unsafe.add(pool_id)
        if unsafe:
            route_references.append(
                f"{route.get('group_name')}/{route.get('model')} -> {sorted(unsafe)}"
            )
    if route_references:
        blockers.append("explicit routes reference cleanup targets: " + "; ".join(route_references))

    binding_references = [
        row for row in bindings if row.get("channel_id") in compatibility_id_set
    ]
    if binding_references:
        blockers.append("OpenLux price sync bindings reference compatibility channels")

    return {
        "blockers": blockers,
        "pool_channel_id": pool_id,
        "compatibility_channel_ids": compatibility_ids,
        "compatibility_channel_names": sorted(row["name"] for row in active_compatibility),
        "historical_channel_ids": sorted(int(row["id"]) for row in historical_compatibility),
        "historical_channel_count": len(historical_compatibility),
        "historical_ability_count": len(historical_abilities),
        "token_ids": sorted(int(row["id"]) for row in tokens),
        "token_routes": sorted(token_routes, key=lambda row: row["id"]),
        "pool_default_ability_count": len(pool_default),
        "pool_group_ability_count": len(pool_group),
        "compatibility_ability_count": len(compatibility_abilities),
        "enabled_default_ability_count": len(default_abilities),
        "target_user_usable_groups": target_usable_groups_json,
    }


def sql_literal(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def sql_int_array(values: list[int]) -> str:
    if not values:
        return "ARRAY[]::bigint[]"
    return "ARRAY[" + ",".join(str(value) for value in values) + "]::bigint[]"


def sql_token_routes(values: list[dict[str, Any]]) -> str:
    rows = []
    for value in values:
        rows.append(
            "("
            + str(int(value["id"]))
            + ","
            + sql_literal(value["group"])
            + ","
            + sql_literal(value["group_chain"])
            + ")"
        )
    return ",".join(rows)


def build_apply_sql(state: dict[str, Any], analysis: dict[str, Any]) -> str:
    expected_state = sql_literal(canonical_json(state))
    compatibility_ids = sql_int_array(analysis["compatibility_channel_ids"])
    historical_ids = sql_int_array(analysis["historical_channel_ids"])
    historical_channel_count = int(analysis["historical_channel_count"])
    pool_id = int(analysis["pool_channel_id"])
    usable_groups = sql_literal(analysis["target_user_usable_groups"])
    token_routes = sql_token_routes(analysis["token_routes"])
    historical_ability_count = int(analysis["historical_ability_count"])
    return f'''
BEGIN;
LOCK TABLE channels, abilities, tokens, options, group_model_routes,
           openlux_price_sync_bindings IN SHARE ROW EXCLUSIVE MODE;

DO $cleanup_state$
DECLARE
    current_state jsonb;
BEGIN
    {STATE_SQL.removesuffix(';').removesuffix('::text')} INTO current_state;
    IF current_state <> {expected_state}::jsonb THEN
        RAISE EXCEPTION 'cleanup state changed after dry-run; rerun the preview';
    END IF;
END
$cleanup_state$;

DO $cleanup_apply$
DECLARE
    affected bigint;
BEGIN
    UPDATE tokens AS token
    SET routing_priority = 'price', "group" = target.group_name,
        group_chain = target.group_chain
    FROM (VALUES {token_routes}) AS target(id, group_name, group_chain)
    WHERE token.id = target.id AND token.deleted_at IS NULL;
    GET DIAGNOSTICS affected = ROW_COUNT;
    IF affected <> {EXPECTED_ACTIVE_TOKENS} THEN
        RAISE EXCEPTION 'unexpected token update count: %', affected;
    END IF;

    UPDATE channels SET "group" = '{POOL_GROUP}'
    WHERE id = {pool_id};
    GET DIAGNOSTICS affected = ROW_COUNT;
    IF affected <> 1 THEN
        RAISE EXCEPTION 'self-hosted pool channel changed concurrently';
    END IF;

    DELETE FROM abilities
    WHERE channel_id = {pool_id} AND "group" = '{DEFAULT_GROUP}';
    GET DIAGNOSTICS affected = ROW_COUNT;
    IF affected <> {EXPECTED_POOL_MODELS} THEN
        RAISE EXCEPTION 'unexpected self-hosted pool default Ability count: %', affected;
    END IF;

    DELETE FROM abilities WHERE channel_id = ANY({compatibility_ids});
    GET DIAGNOSTICS affected = ROW_COUNT;
    IF affected <> {EXPECTED_COMPATIBILITY_ABILITIES} THEN
        RAISE EXCEPTION 'unexpected compatibility Ability count: %', affected;
    END IF;

    DELETE FROM channels WHERE id = ANY({compatibility_ids});
    GET DIAGNOSTICS affected = ROW_COUNT;
    IF affected <> {EXPECTED_COMPATIBILITY_CHANNELS} THEN
        RAISE EXCEPTION 'unexpected compatibility channel count: %', affected;
    END IF;

    UPDATE options SET value = {usable_groups} WHERE key = 'UserUsableGroups';
    GET DIAGNOSTICS affected = ROW_COUNT;
    IF affected <> 1 THEN
        RAISE EXCEPTION 'UserUsableGroups option is missing';
    END IF;

    IF (SELECT COUNT(*) FROM abilities WHERE enabled = TRUE AND "group" = 'default') <> 0 THEN
        RAISE EXCEPTION 'enabled default Ability rows remain after cleanup';
    END IF;
    IF (SELECT COUNT(*) FROM abilities WHERE enabled = TRUE AND "group" = 'group_1' AND channel_id = {pool_id}) <> {EXPECTED_POOL_MODELS} THEN
        RAISE EXCEPTION 'self-hosted pool group_1 Ability rows changed';
    END IF;
    IF (SELECT COUNT(*) FROM channels WHERE id = ANY({compatibility_ids})) <> 0 THEN
        RAISE EXCEPTION 'compatibility channels remain active';
    END IF;
    IF (SELECT COUNT(*) FROM abilities WHERE channel_id = ANY({compatibility_ids})) <> 0 THEN
        RAISE EXCEPTION 'compatibility Ability rows remain';
    END IF;
    IF (SELECT COUNT(*) FROM tokens WHERE deleted_at IS NULL AND routing_priority = 'price') <> {EXPECTED_ACTIVE_TOKENS} THEN
        RAISE EXCEPTION 'not every active token uses price routing';
    END IF;
    IF (SELECT COUNT(*) FROM channels WHERE id = ANY({historical_ids})) <> {historical_channel_count} THEN
        RAISE EXCEPTION 'disabled historical channels changed';
    END IF;
    IF (SELECT COUNT(*) FROM abilities WHERE channel_id = ANY({historical_ids})) <> {historical_ability_count} THEN
        RAISE EXCEPTION 'disabled historical channel Ability rows changed';
    END IF;
END
$cleanup_apply$;

COMMIT;
'''.strip()


POSTCHECK_SQL = r'''
SELECT jsonb_build_object(
    'enabled_default_abilities', (
        SELECT COUNT(*) FROM abilities WHERE enabled = TRUE AND "group" = 'default'
    ),
    'pool_group', (
        SELECT "group" FROM channels
        WHERE name = U&'OpenAI-\81EA\5EFA\53F7\6C60'
    ),
    'pool_group_abilities', (
        SELECT COUNT(*) FROM abilities a
        JOIN channels c ON c.id = a.channel_id
        WHERE c.name = U&'OpenAI-\81EA\5EFA\53F7\6C60'
          AND a.enabled = TRUE AND a."group" = 'group_1'
    ),
    'active_compatibility_channels', (
        SELECT COUNT(*) FROM channels
        WHERE name LIKE 'OpenLux-Default/%' AND status = 1
    ),
    'compatibility_abilities', (
        SELECT COUNT(*) FROM abilities a
        JOIN channels c ON c.id = a.channel_id
        WHERE c.name LIKE 'OpenLux-Default/%' AND c.status = 1
    ),
    'active_tokens', (
        SELECT COUNT(*) FROM tokens WHERE deleted_at IS NULL
    ),
    'price_tokens', (
        SELECT COUNT(*) FROM tokens WHERE deleted_at IS NULL AND routing_priority = 'price'
    ),
    'compatible_tokens', (
        SELECT COUNT(*) FROM tokens
        WHERE deleted_at IS NULL AND "group" <> 'default'
          AND group_chain::jsonb->>0 = "group"
    ),
    'group_1_is_usable', COALESCE((
        SELECT (value::jsonb ? 'group_1') FROM options WHERE key = 'UserUsableGroups'
    ), FALSE)
)::text;
'''.strip()


def validate_postcheck(result: dict[str, Any]) -> None:
    expected = {
        "enabled_default_abilities": 0,
        "pool_group": POOL_GROUP,
        "pool_group_abilities": EXPECTED_POOL_MODELS,
        "active_compatibility_channels": 0,
        "compatibility_abilities": 0,
        "active_tokens": EXPECTED_ACTIVE_TOKENS,
        "price_tokens": EXPECTED_ACTIVE_TOKENS,
        "compatible_tokens": EXPECTED_ACTIVE_TOKENS,
        "group_1_is_usable": True,
    }
    if result != expected:
        raise CleanupError(
            "post-migration verification differs: "
            + canonical_json({"expected": expected, "actual": result})
        )


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
            detail = result.stderr.strip()[-2000:]
            raise CleanupError(f"PostgreSQL command failed: {detail}")
        if not expect_json:
            return None
        lines = [line for line in result.stdout.splitlines() if line.strip()]
        if len(lines) != 1:
            raise CleanupError("PostgreSQL state query returned an unexpected number of rows")
        try:
            value = json.loads(lines[0])
        except json.JSONDecodeError as error:
            raise CleanupError("PostgreSQL state query returned invalid JSON") from error
        if not isinstance(value, dict):
            raise CleanupError("PostgreSQL state query did not return an object")
        return value

    def read_state(self) -> dict[str, Any]:
        return self.run_sql(STATE_SQL, expect_json=True)


def validate_backup(path: Path) -> None:
    if not path.is_absolute() or not path.is_file():
        raise CleanupError(f"database backup is missing: {path}")
    if path.stat().st_size <= 0:
        raise CleanupError(f"database backup is empty: {path}")
    try:
        with gzip.open(path, "rb") as handle:
            while handle.read(1024 * 1024):
                pass
    except (OSError, EOFError) as error:
        raise CleanupError(f"database backup failed gzip validation: {path}") from error


def create_backup(script: Path, backup_dir: Path) -> Path:
    if not script.is_absolute() or not script.is_file() or not os.access(script, os.X_OK):
        raise CleanupError(f"backup script is not executable: {script}")
    before = {path.resolve() for path in backup_dir.glob("*.sql.gz")} if backup_dir.is_dir() else set()
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
        raise CleanupError("database backup script failed: " + result.stderr.strip()[-1000:])
    candidates = sorted(
        (path.resolve() for path in backup_dir.glob("*.sql.gz")),
        key=lambda path: path.stat().st_mtime,
        reverse=True,
    )
    if not candidates:
        raise CleanupError(f"backup script created no backup in {backup_dir}")
    backup = candidates[0]
    if backup in before and backup.stat().st_mtime < started_at - 1:
        raise CleanupError("backup script did not create a new database backup")
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
        "fingerprint": state_fingerprint,
        "blockers": analysis["blockers"],
        "preflight": {
            "pool_channel_id": analysis["pool_channel_id"],
            "compatibility_channel_ids": analysis["compatibility_channel_ids"],
            "compatibility_channel_names": analysis["compatibility_channel_names"],
            "historical_channel_ids": analysis["historical_channel_ids"],
            "historical_channel_count": analysis["historical_channel_count"],
            "historical_ability_count": analysis["historical_ability_count"],
            "token_ids": analysis["token_ids"],
            "token_routes": analysis["token_routes"],
            "enabled_default_ability_count": analysis["enabled_default_ability_count"],
            "pool_group_ability_count": analysis["pool_group_ability_count"],
            "compatibility_ability_count": analysis["compatibility_ability_count"],
        },
        "planned_changes": {
            "token_routing_priority": "price",
            "pool_channel_group": POOL_GROUP,
            "remove_pool_default_abilities": EXPECTED_POOL_MODELS,
            "remove_compatibility_channels": EXPECTED_COMPATIBILITY_CHANNELS,
            "remove_compatibility_abilities": EXPECTED_COMPATIBILITY_ABILITIES,
            "ensure_user_usable_group": POOL_GROUP,
        },
    }
    if backup is not None:
        report["backup"] = {"path": str(backup), "gzip_valid": True}
    if postcheck is not None:
        report["postcheck"] = postcheck
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
    parser.add_argument("--expected-fingerprint")
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


def main() -> int:
    args = parse_args()
    if not args.compose_file.is_absolute() or not args.compose_file.is_file():
        raise CleanupError(f"compose file is missing: {args.compose_file}")
    runner = PostgresRunner(args.compose_file, args.postgres_service)
    state = runner.read_state()
    analysis = analyze_state(state)
    state_fingerprint = fingerprint_state(state)

    if args.dry_run:
        report = public_report("dry-run", state_fingerprint, analysis)
        if args.report:
            write_secure_json(args.report, report)
        print(json.dumps(report, ensure_ascii=False, indent=2, sort_keys=True))
        return 2 if analysis["blockers"] else 0

    if not args.expected_fingerprint or not re.fullmatch(r"[a-f0-9]{64}", args.expected_fingerprint):
        raise CleanupError("--expected-fingerprint must be the lowercase SHA-256 from dry-run")
    if state_fingerprint != args.expected_fingerprint:
        raise CleanupError("database state differs from dry-run; rerun the preview")
    if analysis["blockers"]:
        raise CleanupError("cleanup is blocked: " + "; ".join(analysis["blockers"]))

    backup = args.backup_file.resolve() if args.backup_file else create_backup(
        args.backup_script.resolve(), args.backup_dir.resolve()
    )
    validate_backup(backup)

    state_after_backup = runner.read_state()
    if fingerprint_state(state_after_backup) != args.expected_fingerprint:
        raise CleanupError("database state changed while creating the backup; rerun dry-run")
    analysis_after_backup = analyze_state(state_after_backup)
    if analysis_after_backup["blockers"]:
        raise CleanupError("cleanup became blocked after backup")

    runner.run_sql(build_apply_sql(state_after_backup, analysis_after_backup))
    postcheck = runner.run_sql(POSTCHECK_SQL, expect_json=True)
    validate_postcheck(postcheck)
    report = public_report(
        "apply", state_fingerprint, analysis_after_backup, backup=backup, postcheck=postcheck
    )
    if args.report:
        write_secure_json(args.report, report)
    print(json.dumps(report, ensure_ascii=False, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (CleanupError, subprocess.SubprocessError) as error:
        print(f"smart routing cleanup failed: {error}", file=sys.stderr)
        raise SystemExit(1)
