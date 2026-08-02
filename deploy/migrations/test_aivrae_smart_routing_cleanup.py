import gzip
import json
import tempfile
import unittest
from pathlib import Path

import aivrae_smart_routing_cleanup as cleanup


def ability(channel_id, group, model, enabled=True):
    return {
        "channel_id": channel_id,
        "group_name": group,
        "model": model,
        "enabled": enabled,
        "priority": 10,
        "weight": 100,
        "tag": None,
    }


def valid_state():
    pool_models = [f"pool-model-{index}" for index in range(11)]
    pool = {
        "id": 10,
        "name": cleanup.POOL_CHANNEL_NAME,
        "type": 1,
        "status": 1,
        "group_name": "default,group_1",
        "models": ",".join(pool_models),
        "priority": 10,
        "weight": 100,
        "row_fingerprint": "pool-row-fingerprint",
        "deleted_at": None,
    }
    compatibility = [
        {
            "id": 20 + index,
            "name": f"{cleanup.COMPATIBILITY_PREFIX}{index}",
            "type": 1,
            "status": 1,
            "group_name": f"upstream-{index}",
            "models": "compat-model",
            "priority": 0,
            "weight": 100,
            "row_fingerprint": f"compat-row-fingerprint-{index}",
            "deleted_at": None,
        }
        for index in range(6)
    ]
    historical = [
        {
            "id": 40 + index,
            "name": f"{cleanup.COMPATIBILITY_PREFIX}history-{index}",
            "type": 1,
            "status": 2,
            "group_name": "default",
            "models": "old-model",
            "priority": 0,
            "weight": 0,
            "row_fingerprint": f"history-row-fingerprint-{index}",
            "deleted_at": None,
        }
        for index in range(3)
    ]
    pool_abilities = [
        ability(10, group, model)
        for group in ("default", "group_1")
        for model in pool_models
    ]
    compatibility_abilities = [
        ability(20 + (index % 6), "default", f"compat-model-{index}")
        for index in range(28)
    ]
    historical_abilities = [
        ability(40 + index, "default", "old-model", enabled=False)
        for index in range(3)
    ]
    return {
        "channels": [pool, *compatibility, *historical],
        "abilities": [*pool_abilities, *compatibility_abilities, *historical_abilities],
        "default_abilities": [
            *[row for row in pool_abilities if row["group_name"] == "default"],
            *compatibility_abilities,
        ],
        "tokens": [
            {
                "id": index + 1,
                "user_id": index + 1,
                "name": f"token-{index}",
                "status": 1,
                "group_name": "default",
                "group_chain": '["default"]',
            "routing_priority": "",
            "deleted_at": None,
            "user_group": "default",
            }
            for index in range(6)
        ],
        "options": [
            {"key": "GroupRatio", "value": '{"default":1,"group_1":1.3}'},
            {"key": "GroupGroupRatio", "value": '{}'},
            {"key": "UserUsableGroups", "value": '{"default":"Default"}'},
        ],
        "enabled_groups": ["group_1"],
        "routes": [],
        "bindings": [],
    }


class CleanupTests(unittest.TestCase):
    def test_valid_state_is_actionable_and_report_hides_keys(self):
        state = valid_state()
        analysis = cleanup.analyze_state(state)
        self.assertEqual([], analysis["blockers"])
        self.assertEqual(39, analysis["enabled_default_ability_count"])
        self.assertEqual(28, analysis["compatibility_ability_count"])
        self.assertEqual(
            [{"id": index, "group": "group_1", "group_chain": '["group_1"]'} for index in range(1, 7)],
            analysis["token_routes"],
        )

        report = cleanup.public_report(
            "dry-run", cleanup.fingerprint_state(state), analysis
        )
        serialized = json.dumps(report)
        self.assertNotIn("row_fingerprint", serialized)

        sql = cleanup.build_apply_sql(state, analysis)
        self.assertIn("BEGIN;", sql)
        self.assertIn("LOCK TABLE", sql)
        self.assertIn("COMMIT;", sql)
        self.assertIn("routing_priority = 'price'", sql)
        self.assertIn("DELETE FROM channels", sql)

    def test_route_or_unexpected_default_ability_blocks_cleanup(self):
        state = valid_state()
        state["routes"] = [
            {
                "group_name": "default",
                "model": "pool-model-0",
                "tiers": '[{"priority":1,"channels":[{"channel_id":10,"weight":100}]}]',
                "updated_at": 1,
            }
        ]
        state["default_abilities"].append(ability(999, "default", "foreign"))

        blockers = cleanup.analyze_state(state)["blockers"]
        self.assertTrue(any("explicit routes" in blocker for blocker in blockers))
        self.assertTrue(any("outside the cleanup targets" in blocker for blocker in blockers))

    def test_fingerprint_changes_with_concurrent_target_edit(self):
        state = valid_state()
        original = cleanup.fingerprint_state(state)
        state["channels"][0]["weight"] = 99
        self.assertNotEqual(original, cleanup.fingerprint_state(state))

    def test_gzip_backup_validation(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "backup.sql.gz"
            with gzip.open(path, "wb") as handle:
                handle.write(b"select 1;\n")
            cleanup.validate_backup(path.resolve())


if __name__ == "__main__":
    unittest.main()
