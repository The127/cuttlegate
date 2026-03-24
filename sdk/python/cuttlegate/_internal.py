"""Shared helpers used by both CuttlegateClient and AsyncCuttlegateClient.

These are internal — not part of the public API. Do not import from outside
the cuttlegate package.
"""

from __future__ import annotations

from urllib.parse import urlparse

from .errors import ConfigError, SDKError
from .types import CuttlegateConfig, EvalContext, EvalResult

# Required fields in each flag entry of the bulk evaluate response.
REQUIRED_FLAG_FIELDS = ("key", "enabled", "value_key", "reason")


def require_context(ctx: EvalContext | None) -> None:
    if ctx is None:
        raise ValueError("context must not be None")


def validate_config(config: CuttlegateConfig) -> None:
    if not config.api_key:
        raise ConfigError("api_key is required")
    if not config.server_url:
        raise ConfigError("server_url is required")
    parsed = urlparse(config.server_url)
    if parsed.scheme not in ("http", "https") or not parsed.netloc:
        raise ConfigError("server_url must be an http or https URL")
    if not config.project:
        raise ConfigError("project is required")
    if not config.environment:
        raise ConfigError("environment is required")


def parse_bulk_response(body: dict) -> dict[str, EvalResult]:
    """Parse the bulk evaluate response body into a dict of EvalResult."""
    evaluated_at = body.get("evaluated_at", "")
    results: dict[str, EvalResult] = {}
    for flag in body.get("flags", []):
        for field_name in REQUIRED_FLAG_FIELDS:
            if field_name not in flag:
                raise SDKError(
                    f"cuttlegate: malformed response: missing field {field_name!r}"
                )
        key = flag["key"]
        results[key] = EvalResult(
            key=key,
            enabled=flag["enabled"],
            variant=flag["value_key"],
            reason=flag["reason"],
            evaluated_at=evaluated_at,
            value=flag.get("value") or "",
        )
    return results


def defaults_as_results(defaults: dict) -> dict[str, EvalResult]:
    """Convert a defaults dict to EvalResult entries with reason 'default_fallback'."""
    results: dict[str, EvalResult] = {}
    for key, cfg in defaults.items():
        results[key] = EvalResult(
            key=key,
            enabled=cfg.get("enabled", False),
            variant=cfg.get("variant", ""),
            reason="default_fallback",
            evaluated_at="",
            value="",
        )
    return results
