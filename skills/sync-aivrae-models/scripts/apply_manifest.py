#!/usr/bin/env python3
import argparse
import copy
import json
import math
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

from validate_manifest import (
    load_manifest,
    reject_duplicate_keys,
    validate_manifest,
)


READ_ONLY_CHANNEL_FIELDS = {
    "created_time",
    "test_time",
    "response_time",
    "balance",
    "balance_updated_time",
    "used_quota",
    "status",
    "key",
}
MODEL_PAYLOAD_FIELDS = {
    "model_name",
    "description",
    "description_i18n",
    "icon",
    "tags",
    "tags_i18n",
    "vendor_id",
    "endpoints",
    "status",
    "sync_official",
    "name_rule",
}
PUBLIC_OPTION_FIELDS = {
    "ModelRatio": "model_ratio",
    "CompletionRatio": "completion_ratio",
    "CacheRatio": "cache_ratio",
    "CreateCacheRatio": "create_cache_ratio",
    "ImageRatio": "image_ratio",
    "AudioRatio": "audio_ratio",
    "AudioCompletionRatio": "audio_completion_ratio",
    "ModelPrice": "model_price",
    "billing_setting.billing_mode": "billing_mode",
    "billing_setting.billing_expr": "billing_expr",
}
REDACTION_PATTERNS = (
    (
        re.compile(r"(?i)\bbearer\s+[a-z0-9._~+/=-]{8,}"),
        "Bearer [REDACTED]",
    ),
    (
        re.compile(r"\bsk-(?:ant-)?[A-Za-z0-9_-]{8,}"),
        "[REDACTED_API_KEY]",
    ),
    (
        re.compile(r"\bAIza[0-9A-Za-z_-]{12,}"),
        "[REDACTED_API_KEY]",
    ),
    (
        re.compile(
            r"\beyJ[A-Za-z0-9_-]{8,}\."
            r"[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}"
        ),
        "[REDACTED_JWT]",
    ),
    (
        re.compile(
            r"(?i)(api[_-]?key|access[_-]?token|token|password|secret|key)"
            r"([\"'=:\s]+)[^,\s\"'}&]{6,}"
        ),
        r"\1\2[REDACTED]",
    ),
)


def fail(message):
    raise RuntimeError(message)


class ApplyFailure(RuntimeError):
    def __init__(self, detail):
        self.detail = detail
        super().__init__(canonical_json(detail))


def redact_text(value):
    text = str(value)
    for pattern, replacement in REDACTION_PATTERNS:
        text = pattern.sub(replacement, text)
    return text


def canonical_json(value):
    return json.dumps(
        value,
        ensure_ascii=False,
        separators=(",", ":"),
        sort_keys=True,
    )


def close_number(actual, expected):
    try:
        return math.isclose(
            float(actual),
            float(expected),
            rel_tol=1e-10,
            abs_tol=1e-12,
        )
    except (TypeError, ValueError):
        return False


def positive_int(value, label):
    if isinstance(value, bool) or not isinstance(value, int) or value <= 0:
        fail(f"{label} must be a positive integer")
    return value


def channel_payload(value):
    payload = copy.deepcopy(value)
    for field in READ_ONLY_CHANNEL_FIELDS:
        payload.pop(field, None)
    return payload


def model_payload(value):
    return {
        key: copy.deepcopy(value.get(key))
        for key in MODEL_PAYLOAD_FIELDS
    }


def split_models(value):
    return [item for item in (value or "").split(",") if item]


def validate_api_base(value):
    parsed = urllib.parse.urlsplit(value)
    if parsed.scheme not in {"http", "https"} or not parsed.netloc:
        fail("AIVRAE_API_BASE must be an HTTP(S) URL")
    if parsed.username is not None or parsed.password is not None:
        fail("AIVRAE_API_BASE must not contain credentials")
    if parsed.query or parsed.fragment:
        fail("AIVRAE_API_BASE must not contain a query or fragment")


class NoRedirectHandler(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *_args, **_kwargs):
        return None


class ApiClient:
    def __init__(self, api_base, root_id, root_token):
        self.api_base = api_base.rstrip("/")
        self.opener = urllib.request.build_opener(NoRedirectHandler())
        self.admin_headers = {
            "Authorization": f"Bearer {root_token}",
            "New-Api-User": root_id,
            "Content-Type": "application/json",
            "Accept": "application/json",
            "User-Agent": "AivraeModelSync/1.0",
        }
        self.public_headers = {
            "Accept": "application/json",
            "User-Agent": "AivraeModelSyncVerifier/1.0",
        }

    def request(self, method, path, payload=None, public=False, timeout=60):
        body = None
        if payload is not None:
            body = canonical_json(payload).encode("utf-8")
        headers = self.public_headers if public else self.admin_headers
        request = urllib.request.Request(
            f"{self.api_base}{path}",
            data=body,
            method=method,
            headers=headers,
        )
        try:
            with self.opener.open(request, timeout=timeout) as response:
                raw = response.read()
        except urllib.error.HTTPError as error:
            detail = error.read().decode("utf-8", errors="replace")[:1000]
            fail(
                f"{method} {path} returned HTTP {error.code}: "
                f"{redact_text(detail)}"
            )
        except urllib.error.URLError as error:
            fail(f"{method} {path} failed: {redact_text(error.reason)}")
        except TimeoutError:
            fail(f"{method} {path} timed out")
        try:
            result = json.loads(
                raw,
                object_pairs_hook=reject_duplicate_keys,
            )
        except (json.JSONDecodeError, ValueError):
            fail(f"{method} {path} returned invalid JSON")
        if not isinstance(result, dict):
            fail(f"{method} {path} returned a non-object response")
        return result

    def api(self, method, path, payload=None, timeout=60):
        result = self.request(method, path, payload, timeout=timeout)
        if result.get("success") is not True:
            message = redact_text(result.get("message", "unknown error"))
            fail(f"{method} {path} failed: {message}")
        return result


class ManifestApplier:
    def __init__(self, client, manifest):
        self.client = client
        self.manifest = manifest
        self.channel_id = manifest["channel_id"]
        self.add_models = list(manifest["add_models"])
        self.remove_models = list(manifest["remove_models"])
        self.managed_keys = list(manifest["managed_option_keys"])
        self.option_patch = manifest["option_patch"]

        self.original_maps = {}
        self.original_channel = None
        self.original_models = []
        self.original_metadata = {}
        self.original_metadata_ids = {}
        self.staged_maps = {}
        self.final_maps = {}
        self.target_models = []

        self.attempted_option_keys = []
        self.created_model_ids = []
        self.metadata_creation_attempted = False
        self.channel_attempted = False
        self.removed_metadata_attempts = []

    def read_options(self):
        result = self.client.api("GET", "/api/option/")
        data = result.get("data")
        if not isinstance(data, list):
            fail("option lookup returned invalid data")
        options = {}
        for item in data:
            if isinstance(item, dict) and isinstance(item.get("key"), str):
                if item["key"] in options:
                    fail(f"option lookup returned duplicate key: {item['key']}")
                options[item["key"]] = str(item.get("value", ""))
        return options

    def parse_option_map(self, options, key):
        if key not in options:
            fail(f"managed option is unavailable: {key}")
        raw = options[key] or "{}"
        try:
            value = json.loads(
                raw,
                object_pairs_hook=reject_duplicate_keys,
            )
        except (json.JSONDecodeError, ValueError):
            fail(f"option {key} is not valid JSON")
        if not isinstance(value, dict):
            fail(f"option {key} is not a JSON object")
        return value

    def update_option(self, key, value):
        self.client.api(
            "PUT",
            "/api/option/",
            {"key": key, "value": canonical_json(value)},
        )

    def option_entry_matches(self, actual, expected, model_name):
        actual_present = model_name in actual
        expected_present = model_name in expected
        if actual_present != expected_present:
            return False
        if not actual_present:
            return True
        actual_value = actual[model_name]
        expected_value = expected[model_name]
        if isinstance(expected_value, (int, float)):
            return close_number(actual_value, expected_value)
        return actual_value == expected_value

    def update_option_guarded(self, key, expected, desired):
        affected_models = self.add_models + self.remove_models
        changed_models = [
            name
            for name in affected_models
            if not self.option_entry_matches(expected, desired, name)
        ]
        if not changed_models:
            return

        current = self.parse_option_map(self.read_options(), key)
        for name in affected_models:
            if not self.option_entry_matches(current, expected, name):
                fail(
                    f"concurrent targeted option change detected in "
                    f"{key} for {name}"
                )

        target = copy.deepcopy(current)
        for name in changed_models:
            if name in desired:
                target[name] = copy.deepcopy(desired[name])
            else:
                target.pop(name, None)
        if canonical_json(target) == canonical_json(current):
            return
        if key not in self.attempted_option_keys:
            self.attempted_option_keys.append(key)
        self.update_option(key, target)

    def search_exact_model(self, name):
        query = urllib.parse.urlencode(
            {"keyword": name, "p": 1, "page_size": 100}
        )
        result = self.client.api("GET", f"/api/models/search?{query}")
        data = result.get("data")
        if not isinstance(data, dict) or not isinstance(data.get("items"), list):
            fail(f"model search returned invalid data for {name}")
        exact = [
            item
            for item in data["items"]
            if isinstance(item, dict) and item.get("model_name") == name
        ]
        if len(exact) > 1:
            fail(f"duplicate active model metadata for {name}")
        return exact[0] if exact else None

    def list_channels(self):
        channels = []
        page = 1
        while True:
            query = urllib.parse.urlencode(
                {"p": page, "page_size": 100, "id_sort": "true"}
            )
            result = self.client.api("GET", f"/api/channel/?{query}")
            data = result.get("data")
            if not isinstance(data, dict) or not isinstance(data.get("items"), list):
                fail("channel listing returned invalid data")
            items = data["items"]
            channels.extend(
                item for item in items if isinstance(item, dict)
            )
            total = data.get("total")
            if isinstance(total, bool) or not isinstance(total, int) or total < 0:
                fail("channel listing returned an invalid total")
            if len(channels) >= total:
                return channels
            if not items:
                fail("channel listing ended before reaching its reported total")
            page += 1

    def assert_removals_not_shared(self):
        if not self.remove_models:
            return
        removal_set = set(self.remove_models)
        other_bindings = {}
        for current in self.list_channels():
            if current.get("id") == self.channel_id:
                continue
            overlap = removal_set & set(split_models(current.get("models")))
            for model_name in overlap:
                other_bindings.setdefault(model_name, []).append(
                    {
                        "id": current.get("id"),
                        "name": current.get("name"),
                    }
                )
        if other_bindings:
            fail(
                "removal targets are still bound to other channels: "
                f"{canonical_json(other_bindings)}"
            )

    def read_public_pricing(self):
        result = self.client.request(
            "GET",
            "/api/pricing?lang=en",
            public=True,
            timeout=30,
        )
        if result.get("success") is not True:
            fail(
                "public pricing request failed: "
                f"{redact_text(result.get('message', 'unknown error'))}"
            )
        rows = result.get("data")
        if not isinstance(rows, list):
            fail("public pricing returned invalid data")
        by_name = {}
        for row in rows:
            if isinstance(row, dict) and isinstance(row.get("model_name"), str):
                if row["model_name"] in by_name:
                    fail(
                        "public pricing returned duplicate model: "
                        f"{row['model_name']}"
                    )
                by_name[row["model_name"]] = row
        return by_name

    def validate_vendor_ids(self):
        metadata = self.manifest["model_metadata"]
        vendor_ids = sorted({item["vendor_id"] for item in metadata})
        for vendor_id in vendor_ids:
            result = self.client.api("GET", f"/api/vendors/{vendor_id}")
            vendor = result.get("data")
            if not isinstance(vendor, dict) or vendor.get("id") != vendor_id:
                fail(f"vendor lookup did not match vendor_id={vendor_id}")

    def preflight(self):
        options = self.read_options()
        self.original_maps = {
            key: self.parse_option_map(options, key)
            for key in self.managed_keys
        }

        channel = self.client.api(
            "GET",
            f"/api/channel/{self.channel_id}",
        ).get("data")
        if not isinstance(channel, dict):
            fail("channel lookup returned invalid data")
        if channel.get("id") != self.channel_id:
            fail("channel ID mismatch")
        if channel.get("name") != self.manifest["channel_name"]:
            fail("channel name mismatch")
        if channel.get("status") != 1:
            fail("target channel is not enabled")
        self.original_channel = channel
        self.original_models = split_models(channel.get("models"))
        if self.original_models != self.manifest["expected_original_models"]:
            fail(
                "channel model list changed after review; "
                "refusing to overwrite concurrent changes"
            )

        self.assert_removals_not_shared()

        for name in self.remove_models:
            current = self.search_exact_model(name)
            if current is None:
                fail(f"removal target metadata is missing: {name}")
            self.original_metadata[name] = model_payload(current)
            self.original_metadata_ids[name] = positive_int(
                current.get("id"),
                f"metadata ID for {name}",
            )
        for name in self.add_models:
            if self.search_exact_model(name) is not None:
                fail(f"addition target metadata already exists: {name}")

        self.validate_vendor_ids()

        self.staged_maps = {
            key: copy.deepcopy(value)
            for key, value in self.original_maps.items()
        }
        for key in self.managed_keys:
            for name in self.add_models:
                self.staged_maps[key].pop(name, None)
        for key, values in self.option_patch.items():
            if key not in self.staged_maps:
                fail(f"option patch contains unmanaged key: {key}")
            self.staged_maps[key].update(values)

        self.final_maps = {
            key: copy.deepcopy(value)
            for key, value in self.staged_maps.items()
        }
        for key in self.managed_keys:
            for name in self.remove_models:
                self.final_maps[key].pop(name, None)

        removal_set = set(self.remove_models)
        self.target_models = [
            name for name in self.original_models
            if name not in removal_set
        ]
        self.target_models.extend(
            name for name in self.add_models
            if name not in self.target_models
        )

    def assert_preflight_still_current(self):
        channel = self.client.api(
            "GET",
            f"/api/channel/{self.channel_id}",
        ).get("data")
        if not isinstance(channel, dict):
            fail("channel recheck returned invalid data")
        if channel.get("id") != self.channel_id:
            fail("channel ID changed after preflight")
        if channel.get("name") != self.manifest["channel_name"]:
            fail("channel name changed after preflight")
        if channel.get("status") != 1:
            fail("target channel was disabled after preflight")
        if split_models(channel.get("models")) != self.original_models:
            fail("channel model list changed after preflight")

        options = self.read_options()
        affected_models = self.add_models + self.remove_models
        for key in self.managed_keys:
            current = self.parse_option_map(options, key)
            for name in affected_models:
                if not self.option_entry_matches(
                    current,
                    self.original_maps[key],
                    name,
                ):
                    fail(
                        f"targeted option changed after preflight in "
                        f"{key} for {name}"
                    )

        for name in self.add_models:
            if self.search_exact_model(name) is not None:
                fail(f"addition target appeared after preflight: {name}")
        for name in self.remove_models:
            current = self.search_exact_model(name)
            if current is None:
                fail(f"removal target disappeared after preflight: {name}")
            if positive_int(
                current.get("id"),
                f"metadata ID for {name}",
            ) != self.original_metadata_ids[name]:
                fail(f"removal target identity changed after preflight: {name}")
        self.assert_removals_not_shared()

    def dry_run_result(self):
        public = self.read_public_pricing()
        staged_keys = [
            key
            for key in self.managed_keys
            if canonical_json(self.staged_maps[key])
            != canonical_json(self.original_maps[key])
        ]
        final_keys = [
            key
            for key in self.managed_keys
            if canonical_json(self.final_maps[key])
            != canonical_json(self.staged_maps[key])
        ]
        return {
            "success": True,
            "mode": "dry-run",
            "channel_id": self.channel_id,
            "channel_name": self.original_channel.get("name"),
            "channel_status": self.original_channel.get("status"),
            "current_model_count": len(self.original_models),
            "target_model_count": len(self.target_models),
            "add_models": self.add_models,
            "remove_models": self.remove_models,
            "option_keys_changed_for_additions": staged_keys,
            "option_keys_changed_for_removals": final_keys,
            "public_before": {
                name: name in public
                for name in self.add_models + self.remove_models
            },
            "ready_to_execute": True,
        }

    def rollback(self):
        errors = []
        if self.channel_attempted:
            try:
                current = self.client.api(
                    "GET",
                    f"/api/channel/{self.channel_id}",
                ).get("data")
                if not isinstance(current, dict):
                    fail("channel rollback lookup returned invalid data")
                current_models = split_models(current.get("models"))
                if current_models not in (
                    self.original_models,
                    self.target_models,
                ):
                    fail(
                        "channel model list changed concurrently during rollback"
                    )
                if current_models != self.original_models:
                    restore_channel = channel_payload(current)
                    restore_channel["models"] = ",".join(
                        self.original_models
                    )
                    self.client.api(
                        "PUT",
                        "/api/channel/",
                        restore_channel,
                    )
            except Exception as error:
                errors.append(
                    f"channel rollback failed: {redact_text(error)}"
                )

        for name in self.removed_metadata_attempts:
            try:
                if self.search_exact_model(name) is None:
                    self.client.api(
                        "POST",
                        "/api/models/",
                        self.original_metadata[name],
                    )
            except Exception as error:
                errors.append(
                    f"metadata restore failed for {name}: "
                    f"{redact_text(error)}"
                )

        for model_id in reversed(self.created_model_ids):
            try:
                self.client.api("DELETE", f"/api/models/{model_id}")
            except Exception as error:
                errors.append(
                    f"metadata cleanup failed for ID {model_id}: "
                    f"{redact_text(error)}"
                )

        affected_models = self.add_models + self.remove_models
        for key in reversed(self.attempted_option_keys):
            try:
                current = self.parse_option_map(self.read_options(), key)
                for name in affected_models:
                    known_states = (
                        self.original_maps[key],
                        self.staged_maps[key],
                        self.final_maps[key],
                    )
                    if not any(
                        self.option_entry_matches(current, state, name)
                        for state in known_states
                    ):
                        fail(
                            f"concurrent targeted option change detected "
                            f"during rollback in {key} for {name}"
                        )
                restored = copy.deepcopy(current)
                for name in affected_models:
                    if name in self.original_maps[key]:
                        restored[name] = copy.deepcopy(
                            self.original_maps[key][name]
                        )
                    else:
                        restored.pop(name, None)
                if canonical_json(restored) != canonical_json(current):
                    self.update_option(key, restored)
            except Exception as error:
                errors.append(
                    f"option rollback failed for {key}: "
                    f"{redact_text(error)}"
                )
        try:
            self.verify_rollback()
        except Exception as error:
            errors.append(
                f"rollback verification failed: {redact_text(error)}"
            )
        return errors

    def verify_rollback(self):
        channel = self.client.api(
            "GET",
            f"/api/channel/{self.channel_id}",
        ).get("data")
        if not isinstance(channel, dict):
            fail("rollback channel verification returned invalid data")
        if split_models(channel.get("models")) != self.original_models:
            fail("rollback did not restore the original channel models")

        options = self.read_options()
        affected_models = self.add_models + self.remove_models
        for key in self.managed_keys:
            current = self.parse_option_map(options, key)
            for name in affected_models:
                if not self.option_entry_matches(
                    current,
                    self.original_maps[key],
                    name,
                ):
                    fail(
                        f"rollback did not restore {key} for {name}"
                    )

        for name in self.add_models:
            if self.search_exact_model(name) is not None:
                fail(f"rollback left added metadata active: {name}")
        for name in self.remove_models:
            current = self.search_exact_model(name)
            if current is None:
                fail(f"rollback did not restore metadata: {name}")
            if canonical_json(model_payload(current)) != canonical_json(
                self.original_metadata[name]
            ):
                fail(f"rollback metadata differs for {name}")

    def verify_options(self):
        options = self.read_options()
        maps = {
            key: self.parse_option_map(options, key)
            for key in self.managed_keys
        }
        for key in self.managed_keys:
            for name in self.remove_models:
                if name in maps[key]:
                    fail(f"removed model remains in {key}: {name}")
            for name in self.add_models:
                expected_present = name in self.option_patch.get(key, {})
                actual_present = name in maps[key]
                if expected_present != actual_present:
                    fail(f"{name} presence mismatch in {key}")
                if not expected_present:
                    continue
                expected = self.option_patch[key][name]
                actual = maps[key][name]
                if isinstance(expected, (int, float)):
                    if not close_number(actual, expected):
                        fail(f"{name} value mismatch in {key}")
                elif actual != expected:
                    fail(f"{name} value mismatch in {key}")

    def wait_for_public_pricing(self):
        last_error = None
        for _ in range(15):
            try:
                public = self.read_public_pricing()
                if all(name in public for name in self.add_models) and all(
                    name not in public for name in self.remove_models
                ):
                    return public
                last_error = "public pricing cache has not converged"
            except Exception as error:
                last_error = redact_text(error)
            time.sleep(1)
        fail(last_error or "public pricing verification failed")

    def verify_public_values(self, public):
        route_by_model = {
            item["model"]: item["endpoint_type"]
            for item in self.manifest["route_tests"]
        }
        summary = {}
        for name in self.add_models:
            row = public[name]
            for option_key, values in self.option_patch.items():
                if name not in values:
                    continue
                public_field = PUBLIC_OPTION_FIELDS[option_key]
                actual = row.get(public_field)
                expected = values[name]
                if isinstance(expected, (int, float)):
                    if not close_number(actual, expected):
                        fail(
                            f"public {public_field} mismatch for {name}"
                        )
                elif actual != expected:
                    fail(f"public {public_field} mismatch for {name}")

            endpoint_types = row.get("supported_endpoint_types")
            if not isinstance(endpoint_types, list):
                fail(f"public endpoint list is invalid for {name}")
            expected_endpoint = route_by_model[name]
            if expected_endpoint not in endpoint_types:
                fail(
                    f"{name} does not expose the expected "
                    f"{expected_endpoint} endpoint"
                )
            item = {
                "quota_type": row.get("quota_type"),
                "supported_endpoint_types": endpoint_types,
            }
            if name in self.option_patch.get("ModelRatio", {}):
                item["model_ratio"] = row.get("model_ratio")
                item["completion_ratio"] = row.get("completion_ratio")
            if name in self.option_patch.get("ModelPrice", {}):
                item["model_price"] = row.get("model_price")
            if name in self.option_patch.get(
                "billing_setting.billing_mode",
                {},
            ):
                item["billing_mode"] = row.get("billing_mode")
            summary[name] = item
        return summary

    def verify_after_apply(self):
        channel = self.client.api(
            "GET",
            f"/api/channel/{self.channel_id}",
        ).get("data")
        if not isinstance(channel, dict):
            fail("channel verification returned invalid data")
        models = split_models(channel.get("models"))
        if models != self.target_models:
            fail("channel model verification failed")
        if channel.get("name") != self.manifest["channel_name"]:
            fail("channel name changed during apply")
        if channel.get("status") != 1:
            fail("channel was not left enabled")

        for name in self.add_models:
            if self.search_exact_model(name) is None:
                fail(f"new model metadata is missing after apply: {name}")
        for name in self.remove_models:
            if self.search_exact_model(name) is not None:
                fail(f"removed model metadata remains active: {name}")

        self.verify_options()
        public = self.wait_for_public_pricing()
        public_summary = self.verify_public_values(public)
        return channel, models, public_summary

    def execute(self):
        try:
            self.assert_preflight_still_current()
            for key in self.managed_keys:
                self.update_option_guarded(
                    key,
                    self.original_maps[key],
                    self.staged_maps[key],
                )

            metadata_by_name = {
                item["model_name"]: item
                for item in self.manifest["model_metadata"]
            }
            for name in self.add_models:
                self.metadata_creation_attempted = True
                result = self.client.api(
                    "POST",
                    "/api/models/",
                    metadata_by_name[name],
                )
                created = result.get("data")
                created_id = (
                    created.get("id")
                    if isinstance(created, dict)
                    else None
                )
                if (
                    isinstance(created_id, bool)
                    or not isinstance(created_id, int)
                    or created_id <= 0
                ):
                    current = self.search_exact_model(name)
                    if (
                        current is not None
                        and canonical_json(model_payload(current))
                        == canonical_json(
                            model_payload(metadata_by_name[name])
                        )
                    ):
                        self.created_model_ids.append(
                            positive_int(
                                current.get("id"),
                                f"metadata ID for {name}",
                            )
                        )
                    fail(f"metadata creation returned no ID for {name}")
                self.created_model_ids.append(created_id)

            current_channel = self.client.api(
                "GET",
                f"/api/channel/{self.channel_id}",
            ).get("data")
            if not isinstance(current_channel, dict):
                fail("channel update recheck returned invalid data")
            if current_channel.get("name") != self.manifest["channel_name"]:
                fail("channel name changed before channel update")
            if current_channel.get("status") != 1:
                fail("target channel was disabled before channel update")
            if split_models(current_channel.get("models")) != self.original_models:
                fail("channel model list changed before channel update")
            target_channel = channel_payload(current_channel)
            target_channel["models"] = ",".join(self.target_models)
            self.channel_attempted = True
            self.client.api("PUT", "/api/channel/", target_channel)

            for key in self.managed_keys:
                self.update_option_guarded(
                    key,
                    self.staged_maps[key],
                    self.final_maps[key],
                )

            self.assert_removals_not_shared()
            for name in self.remove_models:
                current = self.search_exact_model(name)
                if current is None:
                    fail(f"removal target disappeared during apply: {name}")
                if positive_int(
                    current.get("id"),
                    f"metadata ID for {name}",
                ) != self.original_metadata_ids[name]:
                    fail(f"removal target identity changed during apply: {name}")
                self.removed_metadata_attempts.append(name)
                self.client.api("DELETE", f"/api/models/{current['id']}")

            channel, models, public_summary = self.verify_after_apply()
            return {
                "success": True,
                "mode": "execute",
                "channel_id": self.channel_id,
                "channel_name": channel.get("name"),
                "channel_status": channel.get("status"),
                "removed_models": self.remove_models,
                "added_models": self.add_models,
                "channel_models": models,
                "updated_option_keys": self.attempted_option_keys,
                "created_model_ids": self.created_model_ids,
                "public_pricing": public_summary,
                "rollback_performed": False,
            }
        except Exception as apply_error:
            mutation_attempted = bool(
                self.attempted_option_keys
                or self.metadata_creation_attempted
                or self.channel_attempted
                or self.removed_metadata_attempts
            )
            rollback_errors = self.rollback() if mutation_attempted else []
            detail = {
                "apply_error": redact_text(apply_error),
                "rollback_performed": mutation_attempted,
                "rollback_errors": rollback_errors,
            }
            raise ApplyFailure(detail) from apply_error


def parse_args():
    parser = argparse.ArgumentParser(
        description=(
            "Dry-run or apply a validated Aivrae model-sync manifest. "
            "Execution requires an explicit channel-ID confirmation."
        )
    )
    parser.add_argument("manifest", type=Path)
    parser.add_argument(
        "--execute",
        action="store_true",
        help="perform writes after all preflight checks pass",
    )
    parser.add_argument(
        "--confirm-channel-id",
        type=int,
        help="must exactly match manifest.channel_id when --execute is used",
    )
    return parser.parse_args()


def main():
    args = parse_args()
    try:
        manifest = load_manifest(args.manifest)
        errors = validate_manifest(manifest)
        if errors:
            fail(
                "manifest validation failed: "
                + canonical_json(errors)
            )

        if args.execute:
            if args.confirm_channel_id != manifest["channel_id"]:
                fail(
                    "--execute requires --confirm-channel-id matching "
                    "manifest.channel_id"
                )
        elif args.confirm_channel_id is not None:
            fail("--confirm-channel-id is only valid with --execute")

        api_base = os.environ.get("AIVRAE_API_BASE", "").rstrip("/")
        root_id = os.environ.get("AIVRAE_ROOT_ID", "")
        root_token = os.environ.get("AIVRAE_ROOT_TOKEN", "")
        if not api_base or not root_id or not root_token:
            fail(
                "missing AIVRAE_API_BASE, AIVRAE_ROOT_ID, or "
                "AIVRAE_ROOT_TOKEN"
            )
        validate_api_base(api_base)

        applier = ManifestApplier(
            ApiClient(api_base, root_id, root_token),
            manifest,
        )
        applier.preflight()
        result = applier.execute() if args.execute else applier.dry_run_result()
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 0
    except ApplyFailure as error:
        print(
            json.dumps(
                {
                    "success": False,
                    **error.detail,
                },
                ensure_ascii=False,
                indent=2,
            ),
            file=sys.stderr,
        )
        return 1
    except Exception as error:
        print(
            json.dumps(
                {
                    "success": False,
                    "error": redact_text(error),
                },
                ensure_ascii=False,
                indent=2,
            ),
            file=sys.stderr,
        )
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
