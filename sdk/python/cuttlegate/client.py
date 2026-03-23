"""CuttlegateClient — synchronous flag evaluation client.

Backed by httpx. No network calls are made at construction time.
Async support is deferred to a follow-on issue.
"""

from __future__ import annotations

from urllib.parse import urlparse

import httpx

from .errors import AuthError, ConfigError, NotFoundError, ServerError
from .types import CuttlegateConfig, EvalContext, EvalResult


class CuttlegateClient:
    """Synchronous client for the Cuttlegate feature flag service.

    Implements CuttlegateClientProtocol — type-hint against that interface
    for easier testing and substitution.

    Construction validates the config immediately and raises ConfigError for
    any invalid field. No network call is made until evaluate_all() is first
    called.
    """

    def __init__(self, config: CuttlegateConfig) -> None:
        _validate_config(config)
        # api_key is stored as a private attribute — never exposed on the
        # public interface, never referenced in __repr__ or error messages.
        self._api_key = config.api_key
        self._server_url = config.server_url.rstrip("/")
        self._project = config.project
        self._environment = config.environment
        self._http = httpx.Client(timeout=config.timeout_ms / 1000)

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def evaluate_all(self, ctx: EvalContext) -> dict[str, EvalResult]:
        """Evaluate all flags for ctx. One HTTP round trip."""
        url = (
            f"{self._server_url}/api/v1/projects/{self._project}"
            f"/environments/{self._environment}/evaluate"
        )
        payload = {"context": {"user_id": ctx.user_id, "attributes": ctx.attributes}}
        response = self._do_post(url, payload)
        return _parse_bulk_response(response)

    def evaluate(self, key: str, ctx: EvalContext) -> EvalResult:
        """Evaluate a single flag by key. Raises NotFoundError if absent."""
        results = self.evaluate_all(ctx)
        try:
            return results[key]
        except KeyError:
            raise NotFoundError(key) from None

    def bool(self, key: str, ctx: EvalContext) -> bool:
        """Return True if the flag variant is 'true'. Raises NotFoundError if absent."""
        result = self.evaluate(key, ctx)
        return result.value == "true"

    def string(self, key: str, ctx: EvalContext) -> str:
        """Return the flag's string variant value. Raises NotFoundError if absent."""
        result = self.evaluate(key, ctx)
        return result.value

    # ------------------------------------------------------------------
    # Internal
    # ------------------------------------------------------------------

    def _do_post(self, url: str, payload: dict) -> dict:
        """Execute a POST request and return the parsed JSON body.

        Catches all httpx exceptions and re-raises as typed SDK errors so
        that the Authorization header (which contains the api_key) never
        appears in an unhandled exception traceback.
        """
        try:
            resp = self._http.post(
                url,
                json=payload,
                headers={"Authorization": f"Bearer {self._api_key}"},
            )
        except httpx.TimeoutException as exc:
            raise TimeoutError("evaluation request timed out") from exc
        except httpx.RequestError as exc:
            raise _SDKRequestError(str(exc)) from exc

        if resp.status_code in (401, 403):
            raise AuthError(resp.status_code)
        if resp.status_code >= 500:
            raise ServerError(resp.status_code)
        if resp.status_code != 200:
            raise _SDKRequestError(f"unexpected status {resp.status_code}")

        return resp.json()


# ------------------------------------------------------------------
# Module-private helpers
# ------------------------------------------------------------------

class _SDKRequestError(Exception):
    """Internal: unexpected HTTP status or transport error not covered by typed errors."""


def _validate_config(config: CuttlegateConfig) -> None:
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


def _parse_bulk_response(body: dict) -> dict[str, EvalResult]:
    evaluated_at = body.get("evaluated_at", "")
    results: dict[str, EvalResult] = {}
    for flag in body.get("flags", []):
        key = flag["key"]
        # value_key is the canonical variant identifier; fall back to value
        # for older server responses that don't include value_key.
        value = flag.get("value_key") or flag.get("value") or ""
        results[key] = EvalResult(
            key=key,
            enabled=flag.get("enabled", False),
            value=value,
            reason=flag.get("reason", ""),
            evaluated_at=evaluated_at,
        )
    return results
