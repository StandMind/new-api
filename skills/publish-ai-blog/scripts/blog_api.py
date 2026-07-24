#!/usr/bin/env python3
"""Validate and safely publish multilingual posts through the new-api blog API."""

from __future__ import annotations

import argparse
import copy
import json
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any


SUPPORTED_LOCALES = ("zh", "en", "es", "fr", "ru", "ja", "vi")
ALLOWED_FIELDS = {
    "id",
    "slug",
    "status",
    "tags",
    "cover_image",
    "author",
    "published_time",
    "translations",
}
SLUG_RE = re.compile(r"^[a-z0-9][a-z0-9._-]{0,127}$")
H1_RE = re.compile(r"(?m)^#\s+")
H2_RE = re.compile(r"(?m)^##\s+")
RAW_HTML_RE = re.compile(r"</?[A-Za-z][^>]*>")
MARKDOWN_IMAGE_RE = re.compile(r"!\[[^\]]*\]\([^)]+\)")
MARKDOWN_LINK_RE = re.compile(
    r"\[[^\]]+\]\(([^)\s]+)(?:\s+['\"][^'\"]*['\"])?\)"
)
EXTERNAL_URL_RE = re.compile(r"https?://[^\s<>()]+", re.IGNORECASE)
PLACEHOLDER_RE = re.compile(
    r"__replace[^\s]*__|\bTODO\b|\bTBD\b|待填写|待补充",
    re.IGNORECASE,
)
SECRET_PATTERNS = (
    re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----"),
    re.compile(r"\bsk-[A-Za-z0-9_-]{20,}\b"),
    re.compile(r"\bBearer\s+[A-Za-z0-9._~+/=-]{20,}\b", re.IGNORECASE),
)


class BlogApiError(RuntimeError):
    pass


class NoAdminRedirectHandler(urllib.request.HTTPRedirectHandler):
    def redirect_request(
        self,
        req: urllib.request.Request,
        fp: Any,
        code: int,
        msg: str,
        headers: Any,
        newurl: str,
    ) -> urllib.request.Request | None:
        return None


def eprint(message: str) -> None:
    print(message, file=sys.stderr)


def normalize_slug(value: str) -> str:
    normalized = value.strip().lower().strip("/").replace(" ", "-")
    while "--" in normalized:
        normalized = normalized.replace("--", "-")
    return normalized


def load_payload(path: str | Path) -> dict[str, Any]:
    payload_path = Path(path)
    try:
        raw = payload_path.read_text(encoding="utf-8")
    except OSError as exc:
        raise BlogApiError(f"cannot read {payload_path}: {exc}") from exc
    try:
        payload = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise BlogApiError(
            f"invalid JSON in {payload_path} at line {exc.lineno}, column {exc.colno}: {exc.msg}"
        ) from exc
    if not isinstance(payload, dict):
        raise BlogApiError("the payload root must be a JSON object")
    return payload


def canonical_payload(payload: dict[str, Any]) -> dict[str, Any]:
    result = {key: copy.deepcopy(payload[key]) for key in ALLOWED_FIELDS if key in payload}
    result.setdefault("id", 0)
    result.setdefault("slug", "")
    result.setdefault("status", "draft")
    result.setdefault("tags", [])
    result.setdefault("cover_image", "")
    result.setdefault("author", "")
    result.setdefault("published_time", 0)
    result.setdefault("translations", {})
    return result


def external_urls(text: str) -> set[str]:
    urls: set[str] = set()
    for match in EXTERNAL_URL_RE.findall(text):
        urls.add(match.rstrip(".,;:!?)]}\"'，。；：！？）》」"))
    return urls


def validate_link_target(
    target: str, locale: str, errors: list[str], warnings: list[str]
) -> None:
    target = target.strip().strip("<>")
    if target.startswith(("#", "/", "./", "../")):
        return
    parsed = urllib.parse.urlparse(target)
    scheme = parsed.scheme.lower()
    if scheme not in {"http", "https", "mailto"}:
        errors.append(f"translations.{locale}.content has an unsafe link: {target}")
    elif scheme == "http":
        warnings.append(f"translations.{locale}.content uses a non-HTTPS link: {target}")


def validate_payload(
    payload: dict[str, Any], require_all_locales: bool = True
) -> tuple[list[str], list[str]]:
    errors: list[str] = []
    warnings: list[str] = []

    unknown_fields = sorted(set(payload) - ALLOWED_FIELDS)
    if unknown_fields:
        warnings.append("unknown top-level fields will not be sent: " + ", ".join(unknown_fields))

    post_id = payload.get("id", 0)
    if isinstance(post_id, bool) or not isinstance(post_id, int) or post_id < 0:
        errors.append("id must be a non-negative integer")

    slug = payload.get("slug")
    if not isinstance(slug, str):
        errors.append("slug must be a string")
    else:
        normalized = normalize_slug(slug)
        if normalized != slug:
            errors.append(f"slug must already be normalized; suggested value: {normalized!r}")
        if not SLUG_RE.fullmatch(slug):
            errors.append(
                "slug must be 1-128 characters and contain only lowercase ASCII letters, "
                "numbers, dots, underscores, and hyphens"
            )

    status = payload.get("status")
    if status not in {"draft", "published"}:
        errors.append("status must be either 'draft' or 'published'")

    tags = payload.get("tags")
    if not isinstance(tags, list) or any(not isinstance(tag, str) for tag in tags):
        errors.append("tags must be an array of strings")
    else:
        if len(tags) > 20:
            errors.append("tags may contain at most 20 items")
        seen: set[str] = set()
        for index, tag in enumerate(tags):
            cleaned = tag.strip().lower().strip("#")
            if not cleaned:
                errors.append(f"tags[{index}] is empty")
                continue
            if cleaned != tag:
                errors.append(f"tags[{index}] must already be normalized as {cleaned!r}")
            if len(cleaned) > 32:
                errors.append(f"tags[{index}] exceeds 32 characters")
            if cleaned in seen:
                errors.append(f"tags[{index}] duplicates {cleaned!r}")
            seen.add(cleaned)
        if not tags:
            warnings.append("tags is empty")

    cover_image = payload.get("cover_image")
    if not isinstance(cover_image, str):
        errors.append("cover_image must be a string")
    elif cover_image:
        parsed_cover = urllib.parse.urlparse(cover_image)
        if parsed_cover.scheme != "https" or not parsed_cover.netloc:
            errors.append("cover_image must be empty or an absolute HTTPS URL")
    elif status == "published":
        warnings.append("published payload has no cover_image")

    author = payload.get("author")
    if not isinstance(author, str):
        errors.append("author must be a string")
    elif len(author) > 128:
        errors.append("author exceeds 128 characters")
    elif not author.strip():
        warnings.append("author is empty")

    published_time = payload.get("published_time")
    if (
        isinstance(published_time, bool)
        or not isinstance(published_time, int)
        or published_time < 0
    ):
        errors.append("published_time must be a non-negative Unix timestamp in seconds")

    translations = payload.get("translations")
    if not isinstance(translations, dict):
        errors.append("translations must be an object keyed by locale")
        return errors, warnings

    unsupported = sorted(set(translations) - set(SUPPORTED_LOCALES))
    if unsupported:
        errors.append("unsupported translation locales: " + ", ".join(unsupported))

    if require_all_locales:
        missing = [locale for locale in SUPPORTED_LOCALES if locale not in translations]
        if missing:
            errors.append("missing required translation locales: " + ", ".join(missing))

    contents_by_locale: dict[str, str] = {}
    urls_by_locale: dict[str, set[str]] = {}
    for locale in SUPPORTED_LOCALES:
        if locale not in translations:
            continue
        translation = translations[locale]
        if not isinstance(translation, dict):
            errors.append(f"translations.{locale} must be an object")
            continue

        values: dict[str, str] = {}
        for field in ("title", "summary", "content"):
            value = translation.get(field)
            if not isinstance(value, str):
                errors.append(f"translations.{locale}.{field} must be a string")
                continue
            if value != value.strip():
                errors.append(f"translations.{locale}.{field} has leading or trailing whitespace")
            if not value.strip():
                errors.append(f"translations.{locale}.{field} must not be empty")
            if PLACEHOLDER_RE.search(value):
                errors.append(f"translations.{locale}.{field} still contains a placeholder")
            for secret_pattern in SECRET_PATTERNS:
                if secret_pattern.search(value):
                    errors.append(f"translations.{locale}.{field} may contain a secret")
                    break
            values[field] = value

        title = values.get("title", "")
        summary = values.get("summary", "")
        content = values.get("content", "")
        if title and len(title) < 8:
            warnings.append(f"translations.{locale}.title is very short")
        if title and len(title) > 120:
            warnings.append(f"translations.{locale}.title exceeds 120 characters")
        if summary and len(summary) < 50:
            warnings.append(f"translations.{locale}.summary is very short")
        if summary and len(summary) > 320:
            warnings.append(f"translations.{locale}.summary exceeds 320 characters")
        if content:
            if H1_RE.search(content):
                errors.append(
                    f"translations.{locale}.content contains an H1; the title field is the page H1"
                )
            if not H2_RE.search(content):
                warnings.append(f"translations.{locale}.content has no H2 section")
            if RAW_HTML_RE.search(content):
                warnings.append(
                    f"translations.{locale}.content contains raw HTML that may differ in server rendering"
                )
            if MARKDOWN_IMAGE_RE.search(content):
                warnings.append(
                    f"translations.{locale}.content contains a Markdown image; use cover_image instead"
                )
            if len(content) < 600:
                warnings.append(f"translations.{locale}.content is shorter than 600 characters")
            for target in MARKDOWN_LINK_RE.findall(content):
                validate_link_target(target, locale, errors, warnings)
            urls = external_urls(content)
            urls_by_locale[locale] = urls
            if len(urls) < 2:
                warnings.append(
                    f"translations.{locale}.content contains fewer than two external source URLs"
                )
            contents_by_locale[locale] = content

    base_locale = "en" if "en" in urls_by_locale else "zh"
    base_urls = urls_by_locale.get(base_locale, set())
    if base_urls:
        for locale, urls in urls_by_locale.items():
            missing_urls = sorted(base_urls - urls)
            if missing_urls:
                warnings.append(
                    f"translations.{locale}.content is missing {len(missing_urls)} URL(s) used by {base_locale}"
                )

    locales = list(contents_by_locale)
    for index, locale in enumerate(locales):
        for other in locales[index + 1 :]:
            if (
                len(contents_by_locale[locale]) >= 200
                and contents_by_locale[locale] == contents_by_locale[other]
            ):
                warnings.append(
                    f"translations.{locale}.content and translations.{other}.content are identical"
                )

    serialized = json.dumps(payload, ensure_ascii=False)
    if PLACEHOLDER_RE.search(serialized):
        errors.append("payload still contains one or more placeholders")
    for secret_pattern in SECRET_PATTERNS:
        if secret_pattern.search(serialized):
            errors.append("payload may contain a secret")
            break

    return sorted(set(errors)), sorted(set(warnings))


def print_validation(errors: list[str], warnings: list[str]) -> None:
    for error in errors:
        eprint(f"ERROR: {error}")
    for warning in warnings:
        eprint(f"WARNING: {warning}")
    if not errors:
        print(f"Validation passed with {len(warnings)} warning(s).")


def require_valid(payload: dict[str, Any]) -> None:
    errors, warnings = validate_payload(payload, require_all_locales=True)
    print_validation(errors, warnings)
    if errors:
        raise BlogApiError("payload validation failed")


def resolve_base_url(args: argparse.Namespace) -> str:
    value = (args.base_url or os.environ.get("NEW_API_BASE_URL", "")).strip().rstrip("/")
    if not value:
        raise BlogApiError("set NEW_API_BASE_URL or pass --base-url")
    parsed = urllib.parse.urlparse(value)
    if parsed.scheme not in {"http", "https"} or not parsed.netloc:
        raise BlogApiError("base URL must be an absolute HTTP(S) origin")
    if parsed.scheme != "https" and parsed.hostname not in {"localhost", "127.0.0.1", "::1"}:
        raise BlogApiError("refusing to send credentials over non-HTTPS transport")
    return value


def admin_headers(args: argparse.Namespace) -> dict[str, str]:
    token = os.environ.get("NEW_API_ADMIN_ACCESS_TOKEN", "").strip()
    user_id = (args.user_id or os.environ.get("NEW_API_ADMIN_USER_ID", "")).strip()
    if not token:
        raise BlogApiError("set NEW_API_ADMIN_ACCESS_TOKEN in the environment")
    if not user_id.isdigit() or int(user_id) <= 0:
        raise BlogApiError("set NEW_API_ADMIN_USER_ID to a positive integer")
    if token.lower().startswith("bearer "):
        token = token.split(None, 1)[1].strip()
    return {
        "Authorization": f"Bearer {token}",
        "New-Api-User": user_id,
    }


def api_request(
    args: argparse.Namespace,
    method: str,
    path: str,
    payload: dict[str, Any] | None = None,
    admin: bool = False,
) -> Any:
    base_url = resolve_base_url(args)
    headers = {
        "Accept": "application/json",
        "Accept-Language": "en",
        "User-Agent": "publish-ai-blog/1.0",
    }
    if admin:
        headers.update(admin_headers(args))
    data = None
    if payload is not None:
        headers["Content-Type"] = "application/json"
        data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    request = urllib.request.Request(
        base_url + path,
        data=data,
        headers=headers,
        method=method,
    )
    opener = (
        urllib.request.build_opener(NoAdminRedirectHandler())
        if admin
        else urllib.request.build_opener()
    )
    try:
        with opener.open(request, timeout=args.timeout) as response:
            raw = response.read().decode("utf-8", errors="replace")
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", errors="replace")
        raise BlogApiError(f"HTTP {exc.code} for {method} {path}: {raw[:1000]}") from exc
    except urllib.error.URLError as exc:
        raise BlogApiError(
            f"network error for {method} {path}: {exc.reason}; "
            "if this was a write, read the admin post state before retrying"
        ) from exc
    except TimeoutError as exc:
        raise BlogApiError(
            f"timeout for {method} {path}; if this was a write, read the admin post state before retrying"
        ) from exc

    if not raw:
        return None
    try:
        result = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise BlogApiError(f"server returned non-JSON content for {method} {path}") from exc
    if isinstance(result, dict) and result.get("success") is False:
        raise BlogApiError(
            f"API rejected {method} {path}: {result.get('message') or 'unknown business error'}"
        )
    return result


def response_data(response: Any) -> Any:
    if isinstance(response, dict) and "data" in response:
        return response["data"]
    return response


def print_json(value: Any) -> None:
    print(json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True))


def write_plan(args: argparse.Namespace, method: str, path: str, payload: dict[str, Any]) -> None:
    base_url = (args.base_url or os.environ.get("NEW_API_BASE_URL", "<NEW_API_BASE_URL>"))
    summary = {
        "id": payload.get("id"),
        "slug": payload.get("slug"),
        "status": payload.get("status"),
        "published_time": payload.get("published_time"),
        "tags": payload.get("tags"),
        "locales": sorted((payload.get("translations") or {}).keys()),
    }
    print("DRY RUN: no remote write was performed")
    print(f"{method} {base_url.rstrip('/')}{path}")
    print_json(summary)


def admin_list_response(
    args: argparse.Namespace,
    keyword: str = "",
    status: str = "",
    page: int = 1,
    page_size: int = 20,
) -> Any:
    query = urllib.parse.urlencode(
        {
            "p": page,
            "page_size": page_size,
            "keyword": keyword,
            "status": status,
        }
    )
    return api_request(args, "GET", f"/api/blog/admin/posts?{query}", admin=True)


def find_exact_slug(args: argparse.Namespace, slug: str) -> dict[str, Any] | None:
    page = 1
    page_size = 100
    while True:
        response = admin_list_response(
            args, keyword=slug, page=page, page_size=page_size
        )
        data = response_data(response)
        items = data.get("items", []) if isinstance(data, dict) else []
        for item in items:
            if isinstance(item, dict) and item.get("slug") == slug:
                return item
        total = data.get("total", 0) if isinstance(data, dict) else 0
        if not items or page * page_size >= total:
            break
        page += 1
    return None


def cmd_validate(args: argparse.Namespace) -> int:
    payload = load_payload(args.payload)
    errors, warnings = validate_payload(
        payload, require_all_locales=not args.allow_missing_locales
    )
    print_validation(errors, warnings)
    return 1 if errors else 0


def cmd_preflight(args: argparse.Namespace) -> int:
    api_request(args, "GET", "/api/status", admin=False)
    admin_list_response(args, page_size=1)
    print(f"Public API and administrator authentication are available at {resolve_base_url(args)}")
    return 0


def cmd_admin_list(args: argparse.Namespace) -> int:
    response = admin_list_response(
        args,
        keyword=args.keyword,
        status=args.status,
        page=args.page,
        page_size=args.page_size,
    )
    print_json(response_data(response))
    return 0


def cmd_get_admin(args: argparse.Namespace) -> int:
    response = api_request(
        args, "GET", f"/api/blog/admin/posts/{args.post_id}", admin=True
    )
    print_json(response_data(response))
    return 0


def cmd_create_draft(args: argparse.Namespace) -> int:
    payload = canonical_payload(load_payload(args.payload))
    payload["id"] = 0
    payload["status"] = "draft"
    payload["published_time"] = 0
    require_valid(payload)
    if not args.execute:
        write_plan(args, "POST", "/api/blog/admin/posts", payload)
        return 0

    existing = find_exact_slug(args, payload["slug"])
    if existing:
        raise BlogApiError(
            f"slug {payload['slug']!r} already exists as post ID {existing.get('id')}; "
            "use update-draft or publish instead of creating another post"
        )
    response = api_request(
        args, "POST", "/api/blog/admin/posts", payload=payload, admin=True
    )
    print_json(response_data(response))
    return 0


def cmd_update_draft(args: argparse.Namespace) -> int:
    payload = canonical_payload(load_payload(args.payload))
    payload["id"] = args.post_id
    payload["status"] = "draft"
    require_valid(payload)
    path = f"/api/blog/admin/posts/{args.post_id}"
    if not args.execute:
        write_plan(args, "PUT", path, payload)
        return 0

    current_response = api_request(args, "GET", path, admin=True)
    current = response_data(current_response)
    if (
        isinstance(current, dict)
        and current.get("status") == "published"
        and not args.confirm_unpublish
    ):
        raise BlogApiError(
            "the current post is published; add --confirm-unpublish to move it back to draft"
        )
    response = api_request(args, "PUT", path, payload=payload, admin=True)
    print_json(response_data(response))
    return 0


def cmd_publish(args: argparse.Namespace) -> int:
    payload = canonical_payload(load_payload(args.payload))
    payload["id"] = args.post_id
    payload["status"] = "published"
    require_valid(payload)
    path = f"/api/blog/admin/posts/{args.post_id}"
    if not args.execute:
        write_plan(args, "PUT", path, payload)
        print("Execution additionally requires --execute --confirm-publish")
        return 0
    if not args.confirm_publish:
        raise BlogApiError("refusing to publish without --confirm-publish")

    api_request(args, "GET", path, admin=True)
    response = api_request(args, "PUT", path, payload=payload, admin=True)
    print_json(response_data(response))
    return 0


def cmd_verify(args: argparse.Namespace) -> int:
    expected_payload: dict[str, Any] | None = None
    if args.payload:
        expected_payload = canonical_payload(load_payload(args.payload))
        require_valid(expected_payload)

    base_url = resolve_base_url(args)
    failures: list[str] = []
    encoded_slug = urllib.parse.quote(args.slug, safe="")
    for locale in args.locales:
        query = urllib.parse.urlencode({"lang": locale})
        path = f"/api/blog/posts/{encoded_slug}?{query}"
        try:
            response = api_request(args, "GET", path, admin=False)
            actual = response_data(response)
            if not isinstance(actual, dict):
                raise BlogApiError("public API returned an unexpected payload")
            if expected_payload:
                expected = expected_payload["translations"].get(locale, {})
                for field in ("title", "summary", "content"):
                    actual_value = str(actual.get(field, "")).strip()
                    expected_value = str(expected.get(field, "")).strip()
                    if actual_value != expected_value:
                        failures.append(f"{locale}: public {field} does not match the payload")
            page_url = f"{base_url}/blog/{encoded_slug}?{query}"
            print(f"OK {locale}: {actual.get('title', '')} | {page_url}")
        except BlogApiError as exc:
            failures.append(f"{locale}: {exc}")

    for failure in failures:
        eprint(f"ERROR: {failure}")
    if failures:
        return 1
    print(f"Verified {len(args.locales)} locale(s).")
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Validate and safely publish multilingual new-api blog posts."
    )
    parser.add_argument(
        "--base-url",
        help="site origin; defaults to NEW_API_BASE_URL",
    )
    parser.add_argument(
        "--user-id",
        help="administrator user ID; defaults to NEW_API_ADMIN_USER_ID",
    )
    parser.add_argument(
        "--timeout",
        type=float,
        default=30.0,
        help="HTTP timeout in seconds (default: 30)",
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    validate_parser = subparsers.add_parser("validate", help="validate a local payload")
    validate_parser.add_argument("payload")
    validate_parser.add_argument(
        "--allow-missing-locales",
        action="store_true",
        help="allow an incomplete work-in-progress draft",
    )
    validate_parser.set_defaults(func=cmd_validate)

    preflight_parser = subparsers.add_parser(
        "preflight", help="check the public API and administrator authentication"
    )
    preflight_parser.set_defaults(func=cmd_preflight)

    list_parser = subparsers.add_parser("admin-list", help="list administrator posts")
    list_parser.add_argument("--keyword", default="")
    list_parser.add_argument("--status", choices=("", "draft", "published"), default="")
    list_parser.add_argument("--page", type=int, default=1)
    list_parser.add_argument("--page-size", type=int, default=20)
    list_parser.set_defaults(func=cmd_admin_list)

    get_parser = subparsers.add_parser("get-admin", help="get one administrator post")
    get_parser.add_argument("post_id", type=int)
    get_parser.set_defaults(func=cmd_get_admin)

    create_parser = subparsers.add_parser(
        "create-draft", help="create a draft; dry-run unless --execute is supplied"
    )
    create_parser.add_argument("payload")
    create_parser.add_argument("--execute", action="store_true")
    create_parser.set_defaults(func=cmd_create_draft)

    update_parser = subparsers.add_parser(
        "update-draft", help="replace a draft; dry-run unless --execute is supplied"
    )
    update_parser.add_argument("post_id", type=int)
    update_parser.add_argument("payload")
    update_parser.add_argument("--execute", action="store_true")
    update_parser.add_argument("--confirm-unpublish", action="store_true")
    update_parser.set_defaults(func=cmd_update_draft)

    publish_parser = subparsers.add_parser(
        "publish", help="publish or schedule a post; defaults to dry-run"
    )
    publish_parser.add_argument("post_id", type=int)
    publish_parser.add_argument("payload")
    publish_parser.add_argument("--execute", action="store_true")
    publish_parser.add_argument("--confirm-publish", action="store_true")
    publish_parser.set_defaults(func=cmd_publish)

    verify_parser = subparsers.add_parser(
        "verify", help="verify every public locale and optionally compare a payload"
    )
    verify_parser.add_argument("slug")
    verify_parser.add_argument("--payload")
    verify_parser.add_argument(
        "--locales",
        nargs="+",
        choices=SUPPORTED_LOCALES,
        default=list(SUPPORTED_LOCALES),
    )
    verify_parser.set_defaults(func=cmd_verify)

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    try:
        return int(args.func(args))
    except BlogApiError as exc:
        eprint(f"ERROR: {exc}")
        return 1
    except KeyboardInterrupt:
        eprint("ERROR: interrupted")
        return 130


if __name__ == "__main__":
    raise SystemExit(main())
