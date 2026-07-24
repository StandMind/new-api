#!/usr/bin/env python3
import argparse
import hashlib
import json
import math
import sys
from datetime import datetime, timezone
from decimal import Decimal, InvalidOperation
from pathlib import Path


OPTION_KEYS = (
    "ModelRatio",
    "CompletionRatio",
    "CacheRatio",
    "CreateCacheRatio",
    "ImageRatio",
    "AudioRatio",
    "AudioCompletionRatio",
    "ModelPrice",
)


def reject_duplicate_keys(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            fail(f"JSON contains duplicate key: {key}")
        result[key] = value
    return result


def fail(message):
    raise ValueError(message)


def load_json(path):
    with path.open("r", encoding="utf-8") as handle:
        return json.load(
            handle,
            parse_float=Decimal,
            object_pairs_hook=reject_duplicate_keys,
        )


def decimal_value(value, field, allow_zero=True):
    if isinstance(value, bool) or value is None or value == "":
        fail(f"{field} is not a number")
    try:
        number = value if isinstance(value, Decimal) else Decimal(str(value))
    except (InvalidOperation, ValueError):
        fail(f"{field} is not a number")
    if not number.is_finite():
        fail(f"{field} must be finite")
    if number < 0 or (not allow_zero and number == 0):
        comparator = "positive" if not allow_zero else "non-negative"
        fail(f"{field} must be {comparator}")
    return number


def optional_decimal(value, field):
    if value is None or value == "":
        return None
    return decimal_value(value, field)


def optional_configured_decimal(value, field):
    number = optional_decimal(value, field)
    if number == 0:
        return None
    return number


def json_number(value):
    number = float(value)
    if not math.isfinite(number):
        fail("computed price is not finite")
    return number


def compact_options(options):
    return {
        key: values
        for key, values in options.items()
        if values
    }


def effective_group_ratio(payload, model_name, group):
    overrides = payload.get("group_model_ratio") or {}
    group_overrides = overrides.get(group) or {}
    if not isinstance(group_overrides, dict):
        fail(f"group_model_ratio.{group} must be an object")
    if model_name in group_overrides:
        return decimal_value(
            group_overrides[model_name],
            f"group_model_ratio.{group}.{model_name}",
            allow_zero=False,
        ), "model_override"
    ratios = payload.get("group_ratio") or {}
    if group not in ratios:
        return None, None
    return decimal_value(
        ratios[group],
        f"group_ratio.{group}",
        allow_zero=False,
    ), "group_ratio"


def select_group(payload, model, allowed_groups):
    enabled_groups = model.get("enable_groups")
    if not isinstance(enabled_groups, list):
        fail(f"{model['model_name']} has no enable_groups array")
    enabled = set(enabled_groups)
    candidates = []
    for order, group in enumerate(allowed_groups):
        if group not in enabled:
            continue
        ratio, source = effective_group_ratio(
            payload,
            model["model_name"],
            group,
        )
        if ratio is not None:
            candidates.append((ratio, -order, group, source))
    if not candidates:
        fail(
            f"{model['model_name']} has no priced group shared by the key and model"
        )
    ratio, _, group, source = max(candidates)
    return group, ratio, source


def formula_values(points, settings):
    cost_cny = points / settings["points_per_cny"]
    cost_usd = cost_cny / settings["cny_per_usd"]
    sale_usd = cost_usd * settings["markup_multiplier"]
    return cost_cny, cost_usd, sale_usd


def compile_token_model(model, group, group_ratio, group_ratio_source, settings):
    name = model["model_name"]
    steps = model.get("step_ratios")
    if steps is not None and not isinstance(steps, list):
        fail(f"{name}.step_ratios must be an array")
    if steps:
        fail(f"{name} has step_ratios; use tiered billing review")
    cache_1h = optional_decimal(
        model.get("cache_creation_1h_ratio"),
        f"{name}.cache_creation_1h_ratio",
    )
    if cache_1h is not None and cache_1h > 0:
        fail(f"{name} has 1h cache creation pricing; use tiered billing review")

    model_ratio = decimal_value(
        model.get("model_ratio"),
        f"{name}.model_ratio",
        allow_zero=False,
    )
    completion_ratio = decimal_value(
        model.get("completion_ratio"),
        f"{name}.completion_ratio",
        allow_zero=False,
    )
    input_points = Decimal("2") * model_ratio * group_ratio
    output_points = input_points * completion_ratio
    input_cost_cny, input_cost_usd, input_sale_usd = formula_values(
        input_points,
        settings,
    )
    output_cost_cny, output_cost_usd, output_sale_usd = formula_values(
        output_points,
        settings,
    )

    source_pricing = {
        "quota_type": 0,
        "selected_group": group,
        "selected_group_ratio": json_number(group_ratio),
        "group_ratio_source": group_ratio_source,
        "model_ratio": json_number(model_ratio),
        "completion_ratio": json_number(completion_ratio),
        "input_points_per_1m": json_number(input_points),
        "output_points_per_1m": json_number(output_points),
        "input_cost_cny_per_1m": json_number(input_cost_cny),
        "output_cost_cny_per_1m": json_number(output_cost_cny),
        "input_cost_usd_per_1m": json_number(input_cost_usd),
        "output_cost_usd_per_1m": json_number(output_cost_usd),
        "input_sale_usd_per_1m": json_number(input_sale_usd),
        "output_sale_usd_per_1m": json_number(output_sale_usd),
    }
    option_entries = {
        "ModelRatio": json_number(input_sale_usd / Decimal("2")),
        "CompletionRatio": json_number(completion_ratio),
    }

    optional_ratios = (
        ("cache_ratio", "CacheRatio"),
        ("image_ratio", "ImageRatio"),
    )
    for source_key, option_key in optional_ratios:
        value = optional_configured_decimal(
            model.get(source_key),
            f"{name}.{source_key}",
        )
        if value is not None:
            source_pricing[source_key] = json_number(value)
            option_entries[option_key] = json_number(value)

    create_cache = model.get("cache_creation_5m_ratio")
    if create_cache is None:
        create_cache = model.get("cache_creation_ratio")
    create_cache = optional_configured_decimal(
        create_cache,
        f"{name}.cache_creation_ratio",
    )
    if create_cache is not None:
        source_pricing["cache_creation_ratio"] = json_number(create_cache)
        option_entries["CreateCacheRatio"] = json_number(create_cache)

    audio_input = optional_configured_decimal(
        model.get("audio_ratio"),
        f"{name}.audio_ratio",
    )
    audio_output = optional_configured_decimal(
        model.get("audio_completion_ratio"),
        f"{name}.audio_completion_ratio",
    )
    if audio_output is not None and audio_input is None:
        fail(f"{name} has audio output pricing without audio input pricing")
    if audio_input is not None:
        audio_input_points = audio_input * group_ratio
        source_pricing["audio_input_points_per_1m"] = json_number(
            audio_input_points
        )
        option_entries["AudioRatio"] = json_number(
            audio_input_points / input_points
        )
    if audio_output is not None:
        audio_output_points = audio_output * group_ratio
        source_pricing["audio_output_points_per_1m"] = json_number(
            audio_output_points
        )
        option_entries["AudioCompletionRatio"] = json_number(
            audio_output_points / (audio_input * group_ratio)
        )

    return source_pricing, option_entries, []


def compile_fixed_model(model, group, group_ratio, group_ratio_source, settings):
    name = model["model_name"]
    steps = model.get("step_ratios")
    if steps is not None and not isinstance(steps, list):
        fail(f"{name}.step_ratios must be an array")
    if steps:
        fail(f"{name} has step_ratios; fixed request pricing is not constant")
    model_price = decimal_value(
        model.get("model_price"),
        f"{name}.model_price",
        allow_zero=False,
    )
    points = model_price * group_ratio
    cost_cny, cost_usd, sale_usd = formula_values(points, settings)
    source_pricing = {
        "quota_type": 1,
        "selected_group": group,
        "selected_group_ratio": json_number(group_ratio),
        "group_ratio_source": group_ratio_source,
        "model_price": json_number(model_price),
        "points_per_request": json_number(points),
        "cost_cny_per_request": json_number(cost_cny),
        "cost_usd_per_request": json_number(cost_usd),
        "sale_usd_per_request": json_number(sale_usd),
    }
    warning = (
        f"{name}: verify that n, resolution, duration, and request parameters "
        "do not change upstream cost before applying ModelPrice"
    )
    return source_pricing, {"ModelPrice": json_number(sale_usd)}, [warning]


def compile_candidate(source_path, selection_path):
    source_bytes = source_path.read_bytes()
    payload = json.loads(
        source_bytes,
        parse_float=Decimal,
        object_pairs_hook=reject_duplicate_keys,
    )
    selection = load_json(selection_path)

    if not isinstance(payload, dict) or payload.get("success") is not True:
        fail("Yunwu pricing response is not a successful object")
    if not isinstance(payload.get("data"), list):
        fail("Yunwu pricing response has no data array")
    if not isinstance(payload.get("group_ratio"), dict):
        fail("Yunwu pricing response has no group_ratio object")
    if not isinstance(payload.get("group_model_ratio") or {}, dict):
        fail("Yunwu group_model_ratio must be an object")
    if selection.get("schema") != "yunwu-aivrae-pricing-selection/v1":
        fail("unexpected pricing selection schema")
    if selection.get("selection_rule") != "highest_effective_ratio":
        fail("selection_rule must be highest_effective_ratio")

    allowed_groups = selection.get("allowed_groups")
    if (
        not isinstance(allowed_groups, list)
        or not allowed_groups
        or any(not isinstance(group, str) or not group.strip() for group in allowed_groups)
    ):
        fail("allowed_groups must contain non-empty group names")
    allowed_groups = list(dict.fromkeys(group.strip() for group in allowed_groups))

    selected_names = selection.get("models")
    if (
        not isinstance(selected_names, list)
        or not selected_names
        or any(not isinstance(name, str) or not name.strip() for name in selected_names)
    ):
        fail("models must contain non-empty model names")
    selected_names = [name.strip() for name in selected_names]
    if len(selected_names) != len(set(selected_names)):
        fail("models contains duplicates")

    raw_settings = selection.get("settings") or {}
    settings = {
        "points_per_cny": decimal_value(
            raw_settings.get("points_per_cny", 2),
            "settings.points_per_cny",
            allow_zero=False,
        ),
        "cny_per_usd": decimal_value(
            raw_settings.get("cny_per_usd", 6.9),
            "settings.cny_per_usd",
            allow_zero=False,
        ),
        "markup_multiplier": decimal_value(
            raw_settings.get("markup_multiplier", 1.3),
            "settings.markup_multiplier",
            allow_zero=False,
        ),
    }

    by_name = {}
    for model in payload["data"]:
        if isinstance(model, dict) and isinstance(model.get("model_name"), str):
            model_name = model["model_name"]
            if not model_name or model_name.strip() != model_name:
                fail("Yunwu pricing data contains an invalid model_name")
            if model_name in by_name:
                fail(f"Yunwu pricing data contains duplicate model: {model_name}")
            by_name[model_name] = model

    source_pricing = {}
    option_patch = {key: {} for key in OPTION_KEYS}
    selected_models = []
    warnings = []
    errors = []

    for name in selected_names:
        model = by_name.get(name)
        if model is None:
            errors.append(f"{name}: missing from Yunwu pricing data")
            continue
        try:
            group, ratio, ratio_source = select_group(
                payload,
                model,
                allowed_groups,
            )
            quota_type = model.get("quota_type")
            if (
                isinstance(quota_type, bool)
                or not isinstance(quota_type, int)
                or quota_type not in {0, 1}
            ):
                fail(f"{name} has unsupported quota_type={quota_type}")
            if quota_type == 0:
                pricing, entries, model_warnings = compile_token_model(
                    model,
                    group,
                    ratio,
                    ratio_source,
                    settings,
                )
            elif quota_type == 1:
                pricing, entries, model_warnings = compile_fixed_model(
                    model,
                    group,
                    ratio,
                    ratio_source,
                    settings,
                )
            source_pricing[name] = pricing
            selected_models.append(
                {
                    "model": name,
                    "quota_type": quota_type,
                    "selected_group": group,
                    "selected_group_ratio": json_number(ratio),
                    "group_ratio_source": ratio_source,
                }
            )
            for key, value in entries.items():
                option_patch[key][name] = value
            warnings.extend(model_warnings)
        except (TypeError, ValueError) as error:
            errors.append(f"{name}: {error}")

    result = {
        "schema": "aivrae-pricing-candidate/v1",
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "source": {
            "url": "https://yunwu.ai/api/pricing_new",
            "sha256": hashlib.sha256(source_bytes).hexdigest(),
        },
        "formula": {
            "points_per_cny": json_number(settings["points_per_cny"]),
            "cny_per_usd": json_number(settings["cny_per_usd"]),
            "markup_multiplier": json_number(settings["markup_multiplier"]),
            "text": (
                "sale_usd = upstream_points / points_per_cny / "
                "cny_per_usd * markup_multiplier"
            ),
        },
        "allowed_groups": allowed_groups,
        "selection_rule": "highest_effective_ratio",
        "selected_models": selected_models,
        "source_pricing": source_pricing,
        "option_patch": compact_options(option_patch),
        "warnings": warnings,
        "errors": errors,
    }
    return result


def main():
    parser = argparse.ArgumentParser(
        description="Compile a reviewed Yunwu pricing snapshot into an Aivrae patch."
    )
    parser.add_argument("--source", required=True, type=Path)
    parser.add_argument("--selection", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()

    output_path = args.output.resolve()
    if output_path in {
        args.source.resolve(),
        args.selection.resolve(),
    }:
        print(
            "error: output must not overwrite the source or selection file",
            file=sys.stderr,
        )
        return 1

    try:
        result = compile_candidate(args.source, args.selection)
    except (OSError, json.JSONDecodeError, ValueError) as error:
        print(f"error: {error}", file=sys.stderr)
        return 1

    try:
        args.output.write_text(
            json.dumps(result, ensure_ascii=False, indent=2) + "\n",
            encoding="utf-8",
        )
    except OSError as error:
        print(f"error: cannot write output: {error}", file=sys.stderr)
        return 1
    print(
        json.dumps(
            {
                "output": str(args.output),
                "selected": len(result["selected_models"]),
                "warnings": len(result["warnings"]),
                "errors": len(result["errors"]),
            },
            ensure_ascii=False,
        )
    )
    return 2 if result["errors"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
