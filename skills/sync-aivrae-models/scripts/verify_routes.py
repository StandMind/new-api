#!/usr/bin/env python3
import argparse
import json
import os
import sys
import time
import urllib.parse
from pathlib import Path

from apply_manifest import ApiClient, redact_text, split_models, validate_api_base
from validate_manifest import load_manifest, validate_manifest


CAPACITY_MARKERS = (
    "429",
    "resource_exhausted",
    "resource exhausted",
    "capacity",
    "overloaded",
    "负载",
    "限流",
)
AUTHENTICATION_MARKERS = (
    "401",
    "403",
    "unauthorized",
    "forbidden",
    "authentication",
    "invalid api key",
    "鉴权",
    "认证",
)
MODEL_NOT_FOUND_MARKERS = (
    "404",
    "model_not_found",
    "model not found",
    "unknown model",
    "模型不存在",
)
SERVER_ERROR_MARKERS = (
    "500",
    "502",
    "503",
    "504",
    "internal server error",
    "bad gateway",
    "service unavailable",
)
TIMEOUT_MARKERS = (
    "timed out",
    "timeout",
    "deadline exceeded",
    "超时",
)
MEDIA_TAGS = {
    "audio",
    "editing",
    "embeddings",
    "image",
    "speech",
    "video",
}
CONTENT_CHECK_ENDPOINTS = {
    "embeddings",
    "image-edit",
    "image-generation",
    "openai-video",
}


def fail(message):
    raise RuntimeError(message)


def classify_failure(result):
    message = " ".join(
        str(result.get(key, ""))
        for key in ("message", "error_code", "http_status")
    ).lower()
    if any(marker in message for marker in CAPACITY_MARKERS):
        return "upstream_capacity"
    if any(marker in message for marker in AUTHENTICATION_MARKERS):
        return "authentication"
    if any(marker in message for marker in MODEL_NOT_FOUND_MARKERS):
        return "model_not_found"
    if any(marker in message for marker in TIMEOUT_MARKERS):
        return "timeout"
    if any(marker in message for marker in SERVER_ERROR_MARKERS):
        return "upstream_server_error"
    return "unexpected"


def target_models(manifest):
    removals = set(manifest["remove_models"])
    target = [
        name
        for name in manifest["expected_original_models"]
        if name not in removals
    ]
    target.extend(
        name for name in manifest["add_models"]
        if name not in target
    )
    return target


def model_requires_content_check(metadata, endpoint_type):
    tags = {
        tag.strip().lower()
        for tag in metadata.get("tags", "").split(",")
        if tag.strip()
    }
    return (
        endpoint_type in CONTENT_CHECK_ENDPOINTS
        or bool(tags & MEDIA_TAGS)
    )


def dry_run_result(manifest):
    metadata = {
        item["model_name"]: item
        for item in manifest["model_metadata"]
    }
    tests = []
    for route_test in manifest["route_tests"]:
        tests.append(
            {
                "model": route_test["model"],
                "endpoint_type": route_test["endpoint_type"],
                "allowed_failure_classifications": route_test[
                    "allowed_failure_classifications"
                ],
                "content_check_required": model_requires_content_check(
                    metadata[route_test["model"]],
                    route_test["endpoint_type"],
                ),
            }
        )
    return {
        "success": True,
        "mode": "dry-run",
        "channel_id": manifest["channel_id"],
        "tests": tests,
        "ready_to_execute": True,
        "note": (
            "No upstream request was sent. Route tests require --execute "
            "and an exact channel-ID confirmation."
        ),
    }


def run_tests(client, manifest):
    channel_id = manifest["channel_id"]
    channel = client.api("GET", f"/api/channel/{channel_id}").get("data")
    if not isinstance(channel, dict):
        fail("channel lookup returned invalid data")
    if channel.get("id") != channel_id:
        fail("channel ID mismatch")
    if channel.get("name") != manifest["channel_name"]:
        fail("channel name mismatch")
    if channel.get("status") != 1:
        fail("target channel is not enabled")
    expected_models = target_models(manifest)
    if split_models(channel.get("models")) != expected_models:
        fail(
            "channel model list does not match the manifest target; "
            "refusing to test stale or concurrent configuration"
        )

    metadata = {
        item["model_name"]: item
        for item in manifest["model_metadata"]
    }
    results = []
    unexpected_failures = []
    allowed_failures = []
    content_checks_required = []

    for route_test in manifest["route_tests"]:
        model_name = route_test["model"]
        endpoint_type = route_test["endpoint_type"]
        query = urllib.parse.urlencode(
            {
                "model": model_name,
                "endpoint_type": endpoint_type,
                "stream": "false",
            }
        )
        started = time.monotonic()
        try:
            response = client.request(
                "GET",
                f"/api/channel/test/{channel_id}?{query}",
                timeout=180,
            )
        except Exception as error:
            response = {
                "success": False,
                "message": redact_text(error),
            }
        elapsed = round(time.monotonic() - started, 3)
        succeeded = response.get("success") is True
        content_check_required = model_requires_content_check(
            metadata[model_name],
            endpoint_type,
        )
        item = {
            "model": model_name,
            "endpoint_type": endpoint_type,
            "success": succeeded,
            "route_verified": succeeded,
            "availability_verified": (
                succeeded and not content_check_required
            ),
            "content_check_required": content_check_required,
            "content_verified": False if content_check_required else None,
            "elapsed_seconds": elapsed,
        }
        if succeeded:
            if content_check_required:
                content_checks_required.append(model_name)
        else:
            classification = classify_failure(response)
            item["classification"] = classification
            item["error_code"] = response.get("error_code")
            item["message"] = redact_text(
                str(response.get("message", "unknown error"))[:500]
            )
            allowed = classification in set(
                route_test["allowed_failure_classifications"]
            )
            item["allowed_failure"] = allowed
            if allowed:
                allowed_failures.append(model_name)
            else:
                unexpected_failures.append(model_name)
        results.append(item)

    all_routes_succeeded = all(item["success"] for item in results)
    return {
        "success": all_routes_succeeded,
        "mode": "execute",
        "channel_id": channel_id,
        "channel_name": channel.get("name"),
        "tests": results,
        "all_routes_succeeded": all_routes_succeeded,
        "policy_passed": not unexpected_failures,
        "has_known_risks": bool(allowed_failures),
        "allowed_failures": allowed_failures,
        "unexpected_failures": unexpected_failures,
        "content_checks_required": content_checks_required,
        "availability_note": (
            "Allowed upstream-capacity failures remain unverified and must "
            "not be reported as available. Models requiring content checks "
            "also remain unverified until their returned media or vectors "
            "are inspected."
        ),
    }


def parse_args():
    parser = argparse.ArgumentParser(
        description=(
            "Plan or execute manifest-defined tests against one exact "
            "Aivrae channel."
        )
    )
    parser.add_argument("manifest", type=Path)
    parser.add_argument(
        "--execute",
        action="store_true",
        help="send the manifest-defined upstream route tests",
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
                + json.dumps(errors, ensure_ascii=False)
            )

        if not args.execute:
            if args.confirm_channel_id is not None:
                fail("--confirm-channel-id is only valid with --execute")
            print(
                json.dumps(
                    dry_run_result(manifest),
                    ensure_ascii=False,
                    indent=2,
                )
            )
            return 0

        if args.confirm_channel_id != manifest["channel_id"]:
            fail(
                "--execute requires --confirm-channel-id matching "
                "manifest.channel_id"
            )

        api_base = os.environ.get("AIVRAE_API_BASE", "").rstrip("/")
        root_id = os.environ.get("AIVRAE_ROOT_ID", "")
        root_token = os.environ.get("AIVRAE_ROOT_TOKEN", "")
        if not api_base or not root_id or not root_token:
            fail(
                "missing AIVRAE_API_BASE, AIVRAE_ROOT_ID, or "
                "AIVRAE_ROOT_TOKEN"
            )
        validate_api_base(api_base)
        result = run_tests(
            ApiClient(api_base, root_id, root_token),
            manifest,
        )
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 2 if result["unexpected_failures"] else 0
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
