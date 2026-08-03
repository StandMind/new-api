import gzip
import json
import tempfile
import unittest
from pathlib import Path

import aivrae_access_policy_expand as expand


def valid_state():
    return {
        "options": [
            {
                "key": "GroupRatio",
                "value": '{"default":1,"route-a":1.3,"route-disabled":1.1}',
            },
            {
                "key": "UserUsableGroups",
                "value": '{"route-a":"Route A"}',
            },
            {
                "key": "TopupGroupRatio",
                "value": '{"default":1}',
            },
            {
                "key": "ModelRequestRateLimitGroup",
                "value": '{"default":[20,10]}',
            },
            {
                "key": "GroupGroupRatio",
                "value": '{"default":{"route-a":0.8,"price-only":0.9}}',
            },
            {
                "key": "group_ratio_setting.group_special_usable_group",
                "value": '{"vip":{"+:special-only":"Special","-:stale":"Stale"}}',
            },
        ],
        "channels": [{"id": 1, "group_name": "route-a,channel-only"}],
        "abilities": [
            {
                "channel_id": 1,
                "group_name": "route-a",
                "model": "model-a",
                "enabled": True,
                "priority": 0,
                "weight": 100,
            }
        ],
        "tokens": [
            {
                "id": 1,
                "group_name": "route-a",
                "group_chain": '["route-a","token-only"]',
            }
        ],
        "routes": [
            {
                "group_name": "explicit-only",
                "model": "model-a",
                "tiers": "[]",
                "updated_at": 1,
            }
        ],
        "users": [{"id": 1, "group_name": "default"}],
        "subscription_plans": [
            {"id": 1, "upgrade_group": "vip", "downgrade_group": "default"}
        ],
        "user_subscriptions": [
            {
                "id": 1,
                "upgrade_group": "vip",
                "downgrade_group": "default",
                "prev_user_group": "default",
            }
        ],
    }


class AccessPolicyExpandTests(unittest.TestCase):
    def test_valid_state_freezes_complete_non_default_route_set(self):
        state = valid_state()
        analysis = expand.analyze_state(state)

        self.assertEqual([], analysis["blockers"])
        self.assertEqual(
            [
                "channel-only",
                "explicit-only",
                "price-only",
                "route-a",
                "route-disabled",
                "special-only",
                "token-only",
            ],
            analysis["standard_route_group_codes"],
        )
        self.assertEqual(
            expand.fingerprint(analysis["standard_route_group_codes"]),
            analysis["route_group_fingerprint"],
        )

        report = expand.public_report(
            "dry-run", expand.fingerprint(state), analysis
        )
        serialized = json.dumps(report)
        self.assertNotIn("expected_legacy_state", serialized)
        self.assertNotIn("group_chain", serialized)

        sql = expand.build_arm_sql(state, analysis, 123)
        self.assertIn("LOCK TABLE options, channels, abilities, tokens", sql)
        self.assertIn("CREATE TABLE IF NOT EXISTS access_policy_migration_guards", sql)
        self.assertIn(analysis["route_group_fingerprint"], sql)
        self.assertIn("COMMIT;", sql)

    def test_enabled_default_ability_and_ambiguous_option_block(self):
        state = valid_state()
        state["abilities"].append(
            {
                "channel_id": 2,
                "group_name": "default",
                "model": "legacy-model",
                "enabled": True,
                "priority": 0,
                "weight": 100,
            }
        )
        state["options"][2]["value"] = '{"default":0}'

        blockers = expand.analyze_state(state)["blockers"]
        self.assertTrue(any("TopupGroupRatio" in item for item in blockers))
        self.assertTrue(any("default route group" in item for item in blockers))

    def test_state_fingerprint_detects_concurrent_changes(self):
        state = valid_state()
        before = expand.fingerprint(state)
        state["abilities"][0]["weight"] = 99
        self.assertNotEqual(before, expand.fingerprint(state))

    def test_guard_sql_uses_collation_independent_text_ordering(self):
        self.assertIn(
            'ORDER BY item.key COLLATE "C"',
            expand.STATE_EXPRESSION,
        )
        self.assertIn(
            'ORDER BY item.channel_id, item.group_name COLLATE "C", '
            'item.model COLLATE "C", item.enabled, item.priority, item.weight',
            expand.STATE_EXPRESSION,
        )
        self.assertIn(
            'ORDER BY channel_id, "group" COLLATE "C", model COLLATE "C", '
            'enabled, priority, weight',
            expand.STATE_EXPRESSION,
        )
        self.assertIn(
            'ORDER BY item.group_name COLLATE "C", item.model COLLATE "C"',
            expand.STATE_EXPRESSION,
        )

    def test_backup_validation(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "backup.sql.gz"
            with gzip.open(path, "wb") as handle:
                handle.write(b"select 1;\n")
            expand.validate_backup(path.resolve())


if __name__ == "__main__":
    unittest.main()
