#!/usr/bin/env python3
import argparse
import json
import re
from datetime import datetime
from decimal import Decimal, InvalidOperation
from pathlib import Path
from urllib.parse import urlsplit


SCHEMA = "aivrae-model-sync/v1"
TOP_LEVEL_FIELDS = {
    "schema",
    "generated_at",
    "channel_id",
    "channel_name",
    "expected_original_models",
    "remove_models",
    "add_models",
    "source",
    "source_pricing",
    "managed_option_keys",
    "option_patch",
    "model_metadata",
    "route_tests",
}
LANGUAGES = {"en", "zh", "es", "fr", "ru", "ja", "vi"}
OPTION_KEYS = (
    "ModelRatio",
    "CompletionRatio",
    "CacheRatio",
    "CreateCacheRatio",
    "ImageRatio",
    "AudioRatio",
    "AudioCompletionRatio",
    "ModelPrice",
    "billing_setting.billing_mode",
    "billing_setting.billing_expr",
)
NUMERIC_OPTION_KEYS = set(OPTION_KEYS[:8])
ENDPOINT_TYPES = {
    "openai",
    "openai-response",
    "openai-response-compact",
    "anthropic",
    "gemini",
    "jina-rerank",
    "image-generation",
    "image-edit",
    "embeddings",
    "openai-video",
}
ALLOWED_FAILURE_CLASSIFICATIONS = {"upstream_capacity"}
MODEL_FIELDS = {
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
SECRET_FIELD_NAMES = {
    "access_token",
    "api_key",
    "apikey",
    "authorization",
    "channel_key",
    "client_secret",
    "cookie",
    "database_url",
    "password",
    "private_key",
    "redis_url",
    "root_token",
    "secret",
    "ssh_key",
}
SECRET_VALUE_PATTERNS = (
    re.compile(r"(?i)\bbearer\s+[a-z0-9._~+/=-]{12,}"),
    re.compile(r"\bsk-(?:ant-)?[A-Za-z0-9_-]{12,}"),
    re.compile(r"\bAIza[0-9A-Za-z_-]{20,}"),
    re.compile(r"\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}"),
    re.compile(r"-----BEGIN [A-Z ]*PRIVATE KEY-----"),
    re.compile(
        r"(?i)(?:api[_-]?key|access[_-]?token|token|password|secret|key)="
        r"[^&\s]{8,}"
    ),
    re.compile(
        r"(?i)[\"']?(?:api[_-]?key|access[_-]?token|token|password|secret|key)"
        r"[\"']?\s*:\s*[\"'][^\"']{8,}"
    ),
    re.compile(r"(?i)\b(?:postgres(?:ql)?|redis)://[^/\s:@]+:[^@\s/]+@"),
)
SHA256_PATTERN = re.compile(r"^[0-9a-fA-F]{64}$")
NUMBER_TOLERANCE = Decimal("1e-10")


def reject_duplicate_keys(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("JSON object contains a duplicate key")
        result[key] = value
    return result


def path_label(path):
    return path or "$"


def append_error(errors, path, message):
    errors.append(f"{path_label(path)}: {message}")


def decimal_value(value, path, errors, positive=False):
    if isinstance(value, bool) or value is None or value == "":
        append_error(errors, path, "must be a number")
        return None
    try:
        number = Decimal(str(value))
    except (InvalidOperation, ValueError):
        append_error(errors, path, "must be a number")
        return None
    if not number.is_finite():
        append_error(errors, path, "must be finite")
        return None
    if positive and number <= 0:
        append_error(errors, path, "must be positive")
        return None
    if not positive and number < 0:
        append_error(errors, path, "must be non-negative")
        return None
    return number


def required_decimal(mapping, key, path, errors, positive=False):
    if key not in mapping:
        append_error(errors, f"{path}.{key}", "is required")
        return None
    return decimal_value(mapping[key], f"{path}.{key}", errors, positive)


def numbers_close(actual, expected):
    difference = abs(actual - expected)
    scale = max(Decimal(1), abs(expected))
    return difference <= NUMBER_TOLERANCE * scale


def check_derived(mapping, key, expected, path, errors):
    actual = required_decimal(mapping, key, path, errors)
    if actual is not None and not numbers_close(actual, expected):
        append_error(
            errors,
            f"{path}.{key}",
            f"does not match the recomputed value {expected}",
        )


def option_map(patch, key):
    value = patch.get(key, {})
    return value if isinstance(value, dict) else {}


def validate_timestamp(value, path, errors):
    if not isinstance(value, str) or not value.strip():
        append_error(errors, path, "must be a non-empty ISO 8601 timestamp")
        return
    normalized = value.strip()
    if normalized.endswith("Z"):
        normalized = normalized[:-1] + "+00:00"
    try:
        parsed = datetime.fromisoformat(normalized)
    except ValueError:
        append_error(errors, path, "must be a valid ISO 8601 timestamp")
        return
    if parsed.tzinfo is None or parsed.utcoffset() is None:
        append_error(errors, path, "must include a timezone")


def validate_string_list(value, path, errors, allow_empty=False):
    if not isinstance(value, list):
        append_error(errors, path, "must be an array")
        return None
    if not allow_empty and not value:
        append_error(errors, path, "must not be empty")
    normalized = []
    for index, item in enumerate(value):
        if not isinstance(item, str) or not item.strip():
            append_error(errors, f"{path}[{index}]", "must be a non-empty string")
            continue
        normalized.append(item.strip())
    if len(normalized) != len(set(normalized)):
        append_error(errors, path, "must not contain duplicates")
    return normalized


def scan_secrets(value, path, errors):
    if isinstance(value, dict):
        for key, child in value.items():
            normalized_key = re.sub(r"[^a-z0-9]+", "_", str(key).lower()).strip("_")
            child_path = f"{path}.{key}" if path else str(key)
            if normalized_key in SECRET_FIELD_NAMES:
                append_error(errors, child_path, "secret-bearing fields are forbidden")
            scan_secrets(child, child_path, errors)
        return
    if isinstance(value, list):
        for index, child in enumerate(value):
            scan_secrets(child, f"{path}[{index}]", errors)
        return
    if not isinstance(value, str):
        return
    for pattern in SECRET_VALUE_PATTERNS:
        if pattern.search(value):
            append_error(errors, path, "value resembles a credential")
            return


def validate_source(source, errors, require_groups):
    if not isinstance(source, dict):
        append_error(errors, "source", "must be an object")
        return None

    provider = source.get("provider")
    if not isinstance(provider, str) or not provider.strip():
        append_error(errors, "source.provider", "must be a non-empty string")

    source_url = source.get("url")
    if not isinstance(source_url, str) or not source_url.strip():
        append_error(errors, "source.url", "must be a non-empty URL")
    else:
        parsed = urlsplit(source_url)
        if parsed.scheme not in {"http", "https"} or not parsed.netloc:
            append_error(errors, "source.url", "must be an HTTP(S) URL")
        if parsed.username is not None or parsed.password is not None:
            append_error(errors, "source.url", "must not contain credentials")

    validate_timestamp(source.get("fetched_at"), "source.fetched_at", errors)
    sha256 = source.get("sha256")
    if not isinstance(sha256, str) or not SHA256_PATTERN.fullmatch(sha256):
        append_error(errors, "source.sha256", "must be a 64-character SHA256")

    groups = validate_string_list(
        source.get("token_groups"),
        "source.token_groups",
        errors,
        allow_empty=not require_groups,
    )
    if source.get("selection_rule") != "highest_effective_ratio":
        append_error(
            errors,
            "source.selection_rule",
            "must be highest_effective_ratio",
        )

    settings = {
        "points_per_cny": required_decimal(
            source,
            "points_per_cny",
            "source",
            errors,
            positive=True,
        ),
        "cny_per_usd": required_decimal(
            source,
            "cny_per_usd",
            "source",
            errors,
            positive=True,
        ),
        "markup_multiplier": required_decimal(
            source,
            "markup_multiplier",
            "source",
            errors,
            positive=True,
        ),
    }
    formula = source.get("formula")
    if not isinstance(formula, str) or not formula.strip():
        append_error(errors, "source.formula", "must be a non-empty string")
    settings["groups"] = set(groups or [])
    if any(settings[key] is None for key in (
        "points_per_cny",
        "cny_per_usd",
        "markup_multiplier",
    )):
        return None
    return settings


def validate_option_patch(manifest, add_models, errors):
    managed = validate_string_list(
        manifest.get("managed_option_keys"),
        "managed_option_keys",
        errors,
    )
    if managed is not None and set(managed) != set(OPTION_KEYS):
        append_error(
            errors,
            "managed_option_keys",
            "must contain the complete supported option-key set",
        )

    patch = manifest.get("option_patch")
    if not isinstance(patch, dict):
        append_error(errors, "option_patch", "must be an object")
        return {}
    unknown_keys = set(patch) - set(OPTION_KEYS)
    if unknown_keys:
        append_error(
            errors,
            "option_patch",
            f"contains unsupported keys: {sorted(unknown_keys)}",
        )

    additions = set(add_models)
    for key, values in patch.items():
        key_path = f"option_patch.{key}"
        if not isinstance(values, dict) or not values:
            append_error(errors, key_path, "must be a non-empty object")
            continue
        unknown_models = set(values) - additions
        if unknown_models:
            append_error(
                errors,
                key_path,
                f"contains models outside add_models: {sorted(unknown_models)}",
            )
        for model_name, value in values.items():
            value_path = f"{key_path}.{model_name}"
            if key in NUMERIC_OPTION_KEYS:
                decimal_value(value, value_path, errors, positive=True)
            elif key == "billing_setting.billing_mode":
                if value != "tiered_expr":
                    append_error(errors, value_path, "must be tiered_expr")
            elif not isinstance(value, str) or not value.strip():
                append_error(errors, value_path, "must be a non-empty expression")

    modes = patch.get("billing_setting.billing_mode", {})
    expressions = patch.get("billing_setting.billing_expr", {})
    if isinstance(modes, dict) and isinstance(expressions, dict):
        if set(modes) != set(expressions):
            append_error(
                errors,
                "option_patch",
                "billing mode and expression model sets must match",
            )

    model_ratios = option_map(patch, "ModelRatio")
    model_prices = option_map(patch, "ModelPrice")
    for model_name in add_models:
        has_ratio = isinstance(model_ratios, dict) and model_name in model_ratios
        has_price = isinstance(model_prices, dict) and model_name in model_prices
        has_tiered = isinstance(modes, dict) and model_name in modes
        if has_ratio and has_price:
            append_error(
                errors,
                f"option_patch.{model_name}",
                "cannot use both ModelRatio and ModelPrice",
            )
        if not has_ratio and not has_price and not has_tiered:
            append_error(
                errors,
                f"option_patch.{model_name}",
                "has no base billing configuration",
            )
    return patch


def validate_token_pricing(
    model_name,
    pricing,
    patch,
    settings,
    path,
    errors,
):
    group_ratio = required_decimal(
        pricing,
        "selected_group_ratio",
        path,
        errors,
        positive=True,
    )
    model_ratio = required_decimal(
        pricing,
        "model_ratio",
        path,
        errors,
        positive=True,
    )
    completion_ratio = required_decimal(
        pricing,
        "completion_ratio",
        path,
        errors,
        positive=True,
    )
    if group_ratio is None or model_ratio is None or completion_ratio is None:
        return

    input_points = Decimal(2) * model_ratio * group_ratio
    output_points = input_points * completion_ratio
    input_cost_cny = input_points / settings["points_per_cny"]
    output_cost_cny = output_points / settings["points_per_cny"]
    input_cost_usd = input_cost_cny / settings["cny_per_usd"]
    output_cost_usd = output_cost_cny / settings["cny_per_usd"]
    input_sale_usd = input_cost_usd * settings["markup_multiplier"]
    output_sale_usd = output_cost_usd * settings["markup_multiplier"]

    checks = {
        "input_points_per_1m": input_points,
        "output_points_per_1m": output_points,
        "input_cost_cny_per_1m": input_cost_cny,
        "output_cost_cny_per_1m": output_cost_cny,
        "input_cost_usd_per_1m": input_cost_usd,
        "output_cost_usd_per_1m": output_cost_usd,
        "input_sale_usd_per_1m": input_sale_usd,
        "output_sale_usd_per_1m": output_sale_usd,
    }
    for key, expected in checks.items():
        check_derived(pricing, key, expected, path, errors)

    model_ratio_patch = option_map(patch, "ModelRatio")
    if model_name not in model_ratio_patch:
        append_error(
            errors,
            f"option_patch.ModelRatio.{model_name}",
            "is required for token pricing",
        )
    else:
        actual = decimal_value(
            model_ratio_patch[model_name],
            f"option_patch.ModelRatio.{model_name}",
            errors,
            positive=True,
        )
        expected = input_sale_usd / Decimal(2)
        if actual is not None and not numbers_close(actual, expected):
            append_error(
                errors,
                f"option_patch.ModelRatio.{model_name}",
                f"does not match the recomputed value {expected}",
            )

    completion_patch = option_map(patch, "CompletionRatio")
    if model_name not in completion_patch:
        append_error(
            errors,
            f"option_patch.CompletionRatio.{model_name}",
            "is required for token pricing",
        )
    else:
        actual = decimal_value(
            completion_patch[model_name],
            f"option_patch.CompletionRatio.{model_name}",
            errors,
            positive=True,
        )
        if actual is not None and not numbers_close(actual, completion_ratio):
            append_error(
                errors,
                f"option_patch.CompletionRatio.{model_name}",
                "does not match source completion_ratio",
            )

    relative_fields = (
        ("cache_ratio", "CacheRatio"),
        ("cache_creation_ratio", "CreateCacheRatio"),
        ("image_ratio", "ImageRatio"),
    )
    for source_key, option_key in relative_fields:
        option_values = option_map(patch, option_key)
        source_present = source_key in pricing
        option_present = model_name in option_values
        if source_present != option_present:
            append_error(
                errors,
                f"option_patch.{option_key}.{model_name}",
                f"presence must match {path}.{source_key}",
            )
            continue
        if not source_present:
            continue
        source_value = decimal_value(
            pricing[source_key],
            f"{path}.{source_key}",
            errors,
            positive=True,
        )
        option_value = decimal_value(
            option_values[model_name],
            f"option_patch.{option_key}.{model_name}",
            errors,
            positive=True,
        )
        if (
            source_value is not None
            and option_value is not None
            and not numbers_close(source_value, option_value)
        ):
            append_error(
                errors,
                f"option_patch.{option_key}.{model_name}",
                f"does not match {path}.{source_key}",
            )

    audio_input_options = option_map(patch, "AudioRatio")
    has_audio_input = "audio_input_points_per_1m" in pricing
    if has_audio_input != (model_name in audio_input_options):
        append_error(
            errors,
            f"option_patch.AudioRatio.{model_name}",
            f"presence must match {path}.audio_input_points_per_1m",
        )
    elif has_audio_input:
        audio_points = required_decimal(
            pricing,
            "audio_input_points_per_1m",
            path,
            errors,
            positive=True,
        )
        audio_ratio = decimal_value(
            audio_input_options[model_name],
            f"option_patch.AudioRatio.{model_name}",
            errors,
            positive=True,
        )
        if (
            audio_points is not None
            and audio_ratio is not None
            and not numbers_close(audio_ratio, audio_points / input_points)
        ):
            append_error(
                errors,
                f"option_patch.AudioRatio.{model_name}",
                "does not match recomputed audio input ratio",
            )

    audio_output_options = option_map(patch, "AudioCompletionRatio")
    has_audio_output = "audio_output_points_per_1m" in pricing
    if has_audio_output != (model_name in audio_output_options):
        append_error(
            errors,
            f"option_patch.AudioCompletionRatio.{model_name}",
            f"presence must match {path}.audio_output_points_per_1m",
        )
    elif has_audio_output:
        if not has_audio_input:
            append_error(
                errors,
                f"{path}.audio_output_points_per_1m",
                "requires audio_input_points_per_1m",
            )
        else:
            audio_input_points = required_decimal(
                pricing,
                "audio_input_points_per_1m",
                path,
                errors,
                positive=True,
            )
            audio_output_points = required_decimal(
                pricing,
                "audio_output_points_per_1m",
                path,
                errors,
                positive=True,
            )
            audio_completion = decimal_value(
                audio_output_options[model_name],
                f"option_patch.AudioCompletionRatio.{model_name}",
                errors,
                positive=True,
            )
            if (
                audio_input_points is not None
                and audio_output_points is not None
                and audio_completion is not None
                and not numbers_close(
                    audio_completion,
                    audio_output_points / audio_input_points,
                )
            ):
                append_error(
                    errors,
                    f"option_patch.AudioCompletionRatio.{model_name}",
                    "does not match recomputed audio output ratio",
                )


def validate_fixed_pricing(
    model_name,
    pricing,
    patch,
    settings,
    path,
    errors,
):
    group_ratio = required_decimal(
        pricing,
        "selected_group_ratio",
        path,
        errors,
        positive=True,
    )
    model_price = required_decimal(
        pricing,
        "model_price",
        path,
        errors,
        positive=True,
    )
    if group_ratio is None or model_price is None:
        return

    points = model_price * group_ratio
    cost_cny = points / settings["points_per_cny"]
    cost_usd = cost_cny / settings["cny_per_usd"]
    sale_usd = cost_usd * settings["markup_multiplier"]
    checks = {
        "points_per_request": points,
        "cost_cny_per_request": cost_cny,
        "cost_usd_per_request": cost_usd,
        "sale_usd_per_request": sale_usd,
    }
    for key, expected in checks.items():
        check_derived(pricing, key, expected, path, errors)

    model_price_patch = option_map(patch, "ModelPrice")
    if model_name not in model_price_patch:
        append_error(
            errors,
            f"option_patch.ModelPrice.{model_name}",
            "is required for fixed pricing",
        )
        return
    actual = decimal_value(
        model_price_patch[model_name],
        f"option_patch.ModelPrice.{model_name}",
        errors,
        positive=True,
    )
    if actual is not None and not numbers_close(actual, sale_usd):
        append_error(
            errors,
            f"option_patch.ModelPrice.{model_name}",
            f"does not match the recomputed value {sale_usd}",
        )


def validate_source_pricing(manifest, add_models, patch, settings, errors):
    pricing_by_model = manifest.get("source_pricing")
    if not isinstance(pricing_by_model, dict):
        append_error(errors, "source_pricing", "must be an object")
        return
    if set(pricing_by_model) != set(add_models):
        append_error(
            errors,
            "source_pricing",
            "model set must exactly match add_models",
        )
    if settings is None:
        return

    for model_name in add_models:
        path = f"source_pricing.{model_name}"
        pricing = pricing_by_model.get(model_name)
        if not isinstance(pricing, dict):
            append_error(errors, path, "must be an object")
            continue
        selected_group = pricing.get("selected_group")
        if not isinstance(selected_group, str) or not selected_group.strip():
            append_error(errors, f"{path}.selected_group", "is required")
        elif selected_group not in settings["groups"]:
            append_error(
                errors,
                f"{path}.selected_group",
                "must be listed in source.token_groups",
            )
        if pricing.get("group_ratio_source") not in {
            "group_ratio",
            "model_override",
        }:
            append_error(
                errors,
                f"{path}.group_ratio_source",
                "must be group_ratio or model_override",
            )

        quota_type = pricing.get("quota_type")
        if (
            isinstance(quota_type, bool)
            or not isinstance(quota_type, int)
            or quota_type not in {0, 1}
        ):
            append_error(errors, f"{path}.quota_type", "must be 0 or 1")
            continue
        if quota_type == 0:
            validate_token_pricing(
                model_name,
                pricing,
                patch,
                settings,
                path,
                errors,
            )
        else:
            validate_fixed_pricing(
                model_name,
                pricing,
                patch,
                settings,
                path,
                errors,
            )


def validate_localized_text(value, path, errors):
    if not isinstance(value, dict):
        append_error(errors, path, "must be an object")
        return
    if set(value) != LANGUAGES:
        append_error(
            errors,
            path,
            f"must contain exactly these languages: {sorted(LANGUAGES)}",
        )
    for language in LANGUAGES:
        text = value.get(language)
        if not isinstance(text, str) or not text.strip():
            append_error(errors, f"{path}.{language}", "must be non-empty")


def validate_metadata(manifest, add_models, errors):
    metadata = manifest.get("model_metadata")
    if not isinstance(metadata, list):
        append_error(errors, "model_metadata", "must be an array")
        return

    names = []
    for index, item in enumerate(metadata):
        path = f"model_metadata[{index}]"
        if not isinstance(item, dict):
            append_error(errors, path, "must be an object")
            continue
        unknown_fields = set(item) - MODEL_FIELDS
        missing_fields = MODEL_FIELDS - set(item)
        if unknown_fields:
            append_error(
                errors,
                path,
                f"contains unsupported fields: {sorted(unknown_fields)}",
            )
        if missing_fields:
            append_error(
                errors,
                path,
                f"is missing fields: {sorted(missing_fields)}",
            )

        model_name = item.get("model_name")
        if not isinstance(model_name, str) or not model_name.strip():
            append_error(errors, f"{path}.model_name", "must be non-empty")
        else:
            names.append(model_name)
        description = item.get("description")
        if not isinstance(description, str) or not description.strip():
            append_error(errors, f"{path}.description", "must be non-empty")
        validate_localized_text(
            item.get("description_i18n"),
            f"{path}.description_i18n",
            errors,
        )
        if (
            isinstance(description, str)
            and isinstance(item.get("description_i18n"), dict)
            and item["description_i18n"].get("en") != description
        ):
            append_error(
                errors,
                f"{path}.description_i18n.en",
                "must match description",
            )

        for field in ("icon", "tags"):
            if not isinstance(item.get(field), str) or not item[field].strip():
                append_error(errors, f"{path}.{field}", "must be non-empty")
        validate_localized_text(
            item.get("tags_i18n"),
            f"{path}.tags_i18n",
            errors,
        )
        if (
            isinstance(item.get("tags"), str)
            and isinstance(item.get("tags_i18n"), dict)
            and item["tags_i18n"].get("en") != item["tags"]
        ):
            append_error(
                errors,
                f"{path}.tags_i18n.en",
                "must match tags",
            )

        vendor_id = item.get("vendor_id")
        if isinstance(vendor_id, bool) or not isinstance(vendor_id, int) or vendor_id <= 0:
            append_error(errors, f"{path}.vendor_id", "must be a positive integer")
        endpoints = item.get("endpoints")
        if not isinstance(endpoints, str):
            append_error(errors, f"{path}.endpoints", "must be a JSON string")
        elif endpoints:
            try:
                endpoint_map = json.loads(
                    endpoints,
                    object_pairs_hook=reject_duplicate_keys,
                )
            except (json.JSONDecodeError, ValueError):
                append_error(errors, f"{path}.endpoints", "must contain valid JSON")
            else:
                if not isinstance(endpoint_map, dict):
                    append_error(
                        errors,
                        f"{path}.endpoints",
                        "must encode a JSON object",
                    )
                else:
                    scan_secrets(endpoint_map, f"{path}.endpoints", errors)
                    for endpoint_type, endpoint in endpoint_map.items():
                        endpoint_path = (
                            f"{path}.endpoints.{endpoint_type}"
                        )
                        if endpoint_type not in ENDPOINT_TYPES:
                            append_error(
                                errors,
                                endpoint_path,
                                "uses an unsupported endpoint type",
                            )
                            continue
                        if (
                            not isinstance(endpoint, dict)
                            or set(endpoint) != {"path", "method"}
                        ):
                            append_error(
                                errors,
                                endpoint_path,
                                "must contain exactly path and method",
                            )
                            continue
                        if (
                            not isinstance(endpoint.get("path"), str)
                            or not endpoint["path"].startswith("/")
                        ):
                            append_error(
                                errors,
                                f"{endpoint_path}.path",
                                "must be an absolute API path",
                            )
                        if endpoint.get("method") not in {
                            "DELETE",
                            "GET",
                            "PATCH",
                            "POST",
                            "PUT",
                        }:
                            append_error(
                                errors,
                                f"{endpoint_path}.method",
                                "must be an uppercase HTTP method",
                            )
        if (
            isinstance(item.get("status"), bool)
            or not isinstance(item.get("status"), int)
            or item.get("status") != 1
        ):
            append_error(errors, f"{path}.status", "must be 1")
        if (
            isinstance(item.get("sync_official"), bool)
            or not isinstance(item.get("sync_official"), int)
            or item.get("sync_official") != 0
        ):
            append_error(errors, f"{path}.sync_official", "must be 0")
        if (
            isinstance(item.get("name_rule"), bool)
            or not isinstance(item.get("name_rule"), int)
            or item.get("name_rule") != 0
        ):
            append_error(errors, f"{path}.name_rule", "must be 0")

    if len(names) != len(set(names)):
        append_error(errors, "model_metadata", "contains duplicate model names")
    if set(names) != set(add_models):
        append_error(
            errors,
            "model_metadata",
            "model set must exactly match add_models",
        )


def validate_route_tests(manifest, add_models, errors):
    route_tests = manifest.get("route_tests")
    if not isinstance(route_tests, list):
        append_error(errors, "route_tests", "must be an array")
        return

    names = []
    allowed_fields = {
        "model",
        "endpoint_type",
        "allowed_failure_classifications",
    }
    for index, item in enumerate(route_tests):
        path = f"route_tests[{index}]"
        if not isinstance(item, dict):
            append_error(errors, path, "must be an object")
            continue
        if set(item) != allowed_fields:
            append_error(
                errors,
                path,
                f"must contain exactly these fields: {sorted(allowed_fields)}",
            )
        model_name = item.get("model")
        if not isinstance(model_name, str) or not model_name.strip():
            append_error(errors, f"{path}.model", "must be non-empty")
        else:
            names.append(model_name)
        endpoint_type = item.get("endpoint_type")
        if endpoint_type not in ENDPOINT_TYPES:
            append_error(
                errors,
                f"{path}.endpoint_type",
                f"must be one of {sorted(ENDPOINT_TYPES)}",
            )
        allowed = validate_string_list(
            item.get("allowed_failure_classifications"),
            f"{path}.allowed_failure_classifications",
            errors,
            allow_empty=True,
        )
        if allowed is not None:
            unsupported = set(allowed) - ALLOWED_FAILURE_CLASSIFICATIONS
            if unsupported:
                append_error(
                    errors,
                    f"{path}.allowed_failure_classifications",
                    f"contains unsupported values: {sorted(unsupported)}",
                )

    if len(names) != len(set(names)):
        append_error(errors, "route_tests", "contains duplicate model tests")
    if set(names) != set(add_models):
        append_error(
            errors,
            "route_tests",
            "model set must exactly match add_models",
        )


def validate_manifest(manifest):
    errors = []
    if not isinstance(manifest, dict):
        return ["$: manifest must be a JSON object"]
    scan_secrets(manifest, "", errors)
    unknown_fields = set(manifest) - TOP_LEVEL_FIELDS
    missing_fields = TOP_LEVEL_FIELDS - set(manifest)
    if unknown_fields:
        append_error(
            errors,
            "$",
            f"contains unsupported fields: {sorted(unknown_fields)}",
        )
    if missing_fields:
        append_error(
            errors,
            "$",
            f"is missing fields: {sorted(missing_fields)}",
        )

    if manifest.get("schema") != SCHEMA:
        append_error(errors, "schema", f"must be {SCHEMA}")
    validate_timestamp(manifest.get("generated_at"), "generated_at", errors)

    channel_id = manifest.get("channel_id")
    if isinstance(channel_id, bool) or not isinstance(channel_id, int) or channel_id <= 0:
        append_error(errors, "channel_id", "must be a positive integer")
    channel_name = manifest.get("channel_name")
    if not isinstance(channel_name, str) or not channel_name.strip():
        append_error(errors, "channel_name", "must be a non-empty string")

    original = validate_string_list(
        manifest.get("expected_original_models"),
        "expected_original_models",
        errors,
        allow_empty=True,
    )
    additions = validate_string_list(
        manifest.get("add_models"),
        "add_models",
        errors,
        allow_empty=True,
    )
    removals = validate_string_list(
        manifest.get("remove_models"),
        "remove_models",
        errors,
        allow_empty=True,
    )
    original = original or []
    additions = additions or []
    removals = removals or []
    if not additions and not removals:
        append_error(errors, "$", "add_models and remove_models cannot both be empty")
    if set(additions) & set(removals):
        append_error(errors, "$", "add_models and remove_models must be disjoint")
    existing_additions = set(additions) & set(original)
    if existing_additions:
        append_error(
            errors,
            "add_models",
            f"already present in expected_original_models: {sorted(existing_additions)}",
        )
    missing_removals = set(removals) - set(original)
    if missing_removals:
        append_error(
            errors,
            "remove_models",
            f"not present in expected_original_models: {sorted(missing_removals)}",
        )

    settings = validate_source(
        manifest.get("source"),
        errors,
        require_groups=bool(additions),
    )
    patch = validate_option_patch(manifest, additions, errors)
    validate_source_pricing(manifest, additions, patch, settings, errors)
    validate_metadata(manifest, additions, errors)
    validate_route_tests(manifest, additions, errors)
    return errors


def load_manifest(path):
    try:
        raw = path.read_text(encoding="utf-8")
    except OSError as error:
        raise ValueError(f"cannot read manifest: {error}") from error
    try:
        manifest = json.loads(raw, object_pairs_hook=reject_duplicate_keys)
    except json.JSONDecodeError as error:
        raise ValueError(
            f"manifest is not valid JSON at line {error.lineno}, column {error.colno}"
        ) from error
    return manifest


def main():
    parser = argparse.ArgumentParser(
        description="Validate an Aivrae model-sync manifest without changing state."
    )
    parser.add_argument("manifest", type=Path)
    args = parser.parse_args()

    try:
        manifest = load_manifest(args.manifest)
    except ValueError as error:
        print(
            json.dumps(
                {"valid": False, "errors": [str(error)]},
                ensure_ascii=False,
                indent=2,
            )
        )
        return 2

    errors = validate_manifest(manifest)
    result = {
        "valid": not errors,
        "schema": manifest.get("schema"),
        "channel_id": manifest.get("channel_id"),
        "add_count": len(manifest.get("add_models", []))
        if isinstance(manifest.get("add_models"), list)
        else None,
        "remove_count": len(manifest.get("remove_models", []))
        if isinstance(manifest.get("remove_models"), list)
        else None,
        "errors": errors,
    }
    print(json.dumps(result, ensure_ascii=False, indent=2))
    return 2 if errors else 0


if __name__ == "__main__":
    raise SystemExit(main())
