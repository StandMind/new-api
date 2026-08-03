import gzip
import tempfile
import unittest
from pathlib import Path

import aivrae_access_policy_contract as contract


def valid_status():
    return {
        "legacy_columns": list(contract.LEGACY_COLUMNS),
        "legacy_option_keys": list(contract.LEGACY_OPTIONS),
        "access_policy_state": {
            "id": 1,
            "expanded_at": 100,
            "contracted_at": 0,
            "initial_route_group_codes": '["route-a","route-disabled"]',
            "initial_route_group_fingerprint": contract.fingerprint(
                ["route-a", "route-disabled"]
            ),
        },
        "invalid_references": {
            "active_users_missing_level": 0,
            "subscription_plan_upgrade": 0,
            "subscription_plan_downgrade": 0,
            "user_subscription_upgrade": 0,
            "user_subscription_downgrade": 0,
            "user_subscription_previous": 0,
            "level_route_grants": 0,
        },
    }


def valid_state():
    initial_codes = ["route-a", "route-disabled"]
    return {
        "options": [{"key": "GroupRatio", "value": '{"default":1}'}],
        "users": [
            {
                "id": 1,
                "user_level": "standard",
                "legacy_group": "default",
                "deleted_at": None,
            },
            {
                "id": 2,
                "user_level": "vip",
                "legacy_group": "vip",
                "deleted_at": None,
            },
        ],
        "subscription_plans": [
            {
                "id": 1,
                "upgrade_user_level": "vip",
                "downgrade_user_level": "standard",
                "upgrade_group": "vip",
                "downgrade_group": "default",
            }
        ],
        "user_subscriptions": [
            {
                "id": 1,
                "upgrade_user_level": "vip",
                "downgrade_user_level": "standard",
                "previous_user_level": "standard",
                "upgrade_group": "vip",
                "downgrade_group": "default",
                "prev_user_group": "default",
            }
        ],
        "user_levels": [
            {
                "code": "standard",
                "name": "Standard",
                "description": "",
                "is_default": True,
                "enabled": True,
                "topup_ratio": 1,
                "request_limit": 0,
                "success_request_limit": 0,
                "created_at": 1,
                "updated_at": 1,
            },
            {
                "code": "vip",
                "name": "VIP",
                "description": "",
                "is_default": False,
                "enabled": True,
                "topup_ratio": 1.2,
                "request_limit": 10,
                "success_request_limit": 8,
                "created_at": 1,
                "updated_at": 1,
            },
        ],
        "route_groups": [
            {
                "code": "default",
                "name": "Default",
                "description": "",
                "base_ratio": 1,
                "enabled": False,
                "created_at": 1,
                "updated_at": 1,
            },
            {
                "code": "route-a",
                "name": "Route A",
                "description": "",
                "base_ratio": 1.3,
                "enabled": True,
                "created_at": 1,
                "updated_at": 1,
            },
            {
                "code": "route-disabled",
                "name": "Disabled",
                "description": "",
                "base_ratio": 1.1,
                "enabled": False,
                "created_at": 1,
                "updated_at": 1,
            },
        ],
        "grants": [
            {
                "user_level_code": "standard",
                "route_group_code": "route-a",
                "price_ratio": 0.8,
                "created_at": 1,
                "updated_at": 1,
            },
            {
                "user_level_code": "standard",
                "route_group_code": "route-disabled",
                "price_ratio": None,
                "created_at": 1,
                "updated_at": 1,
            },
        ],
        "access_policy_state": [
            {
                "id": 1,
                "initial_route_group_codes": '["route-a","route-disabled"]',
                "initial_route_group_fingerprint": contract.fingerprint(initial_codes),
                "expanded_at": 100,
                "initial_user_level_cleanup_at": 100,
                "contracted_at": 0,
            }
        ],
        "enabled_default_abilities": [],
    }


class AccessPolicyContractTests(unittest.TestCase):
    def test_valid_state_is_contractible(self):
        status = contract.analyze_status(valid_status())
        state = contract.analyze_state(valid_state())

        self.assertEqual([], status["blockers"])
        self.assertEqual([], state["blockers"])
        self.assertEqual(
            ["route-a", "route-disabled"], state["standard_authorization"]
        )

    def test_user_and_subscription_mismatches_block(self):
        state = valid_state()
        state["users"][0]["user_level"] = "vip"
        state["subscription_plans"][0]["downgrade_user_level"] = "vip"

        blockers = contract.analyze_state(state)["blockers"]
        self.assertTrue(any("user 1 new/legacy level mismatch" in item for item in blockers))
        self.assertTrue(any("downgrade_user_level" in item for item in blockers))

    def test_missing_reference_and_default_ability_block(self):
        state = valid_state()
        state["users"][1]["user_level"] = "missing"
        state["users"][1]["legacy_group"] = "missing"
        state["enabled_default_abilities"] = [
            {"channel_id": 9, "model": "model-a", "priority": 0, "weight": 100}
        ]

        blockers = contract.analyze_state(state)["blockers"]
        self.assertTrue(any("references missing level" in item for item in blockers))
        self.assertTrue(any("enabled Ability" in item for item in blockers))

    def test_standard_authorization_must_match_frozen_snapshot(self):
        state = valid_state()
        state["grants"].pop()

        blockers = contract.analyze_state(state)["blockers"]
        self.assertIn(
            "standard authorization differs from the frozen expand snapshot",
            blockers,
        )

    def test_state_fingerprint_detects_concurrent_change(self):
        state = valid_state()
        before = contract.fingerprint(state)
        state["route_groups"][1]["base_ratio"] = 1.31
        self.assertNotEqual(before, contract.fingerprint(state))

    def test_contract_sql_locks_and_removes_only_legacy_objects(self):
        sql = contract.build_contract_sql(valid_state(), 123)

        self.assertIn("IN ACCESS EXCLUSIVE MODE", sql)
        self.assertIn('ALTER TABLE users DROP COLUMN "group"', sql)
        self.assertIn("DROP COLUMN upgrade_group", sql)
        self.assertIn("DROP COLUMN prev_user_group", sql)
        self.assertIn("DELETE FROM options", sql)
        self.assertIn("SET contracted_at = 123", sql)
        self.assertNotIn("DROP COLUMN user_level", sql)

    def test_already_contracted_is_reported_idempotently(self):
        status = valid_status()
        status["legacy_columns"] = []
        status["legacy_option_keys"] = []
        status["access_policy_state"]["contracted_at"] = 200

        analysis = contract.analyze_status(status)
        self.assertTrue(analysis["already_contracted"])
        self.assertEqual([], analysis["blockers"])

    def test_partial_contraction_is_blocking(self):
        status = valid_status()
        status["legacy_columns"].remove("users.group")

        blockers = contract.analyze_status(status)["blockers"]
        self.assertTrue(any("column set is incomplete" in item for item in blockers))

    def test_backup_validation(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "backup.sql.gz"
            with gzip.open(path, "wb") as handle:
                handle.write(b"select 1;\n")
            contract.validate_backup(path.resolve())


if __name__ == "__main__":
    unittest.main()
