"""CuttlegateClient — synchronous flag evaluation client.

Backed by httpx. No network calls are made at construction time.
Async support is deferred to a follow-on issue.
"""

from __future__ import annotations

import httpx

from ._internal import defaults_as_results, parse_bulk_response, require_context, validate_config
from .errors import AuthError, FlagNotFoundError, SDKError, ServerError
from .types import CuttlegateConfig, EvalContext, EvalResult


class CuttlegateClient:
    """Synchronous client for the Cuttlegate feature flag service.

    Implements CuttlegateClientProtocol — type-hint against that interface
    for easier testing and substitution.

    Construction validates the config immediately and raises ConfigError for
    any invalid field. No network call is made until an evaluation method is
    first called.
    """

    def __init__(self, config: CuttlegateConfig) -> None:
        validate_config(config)
        # api_key is stored as a private attribute — never exposed on the
        # public interface, never referenced in __repr__ or error messages.
        self._api_key = config.api_key
        self._server_url = config.server_url.rstrip("/")
        self._project = config.project
        self._environment = config.environment
        self._http = httpx.Client(timeout=config.timeout_ms / 1000)
        self._defaults = config.defaults or {}

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def evaluate_all(self, ctx: EvalContext) -> dict[str, EvalResult]:
        """Evaluate all flags for ctx. One HTTP round trip.

        If ``defaults`` are configured and the server is unreachable (network
        error, timeout, or 5xx), returns the defaults with reason
        ``"default_fallback"`` instead of raising. Auth errors (401/403) still
        raise.

        Raises:
            ValueError: if ctx is None.
            AuthError: on HTTP 401 or 403.
            SDKError: on HTTP 404, 5xx, network error, or malformed response (when no defaults).
        """
        require_context(ctx)
        url = (
            f"{self._server_url}/api/v1/projects/{self._project}"
            f"/environments/{self._environment}/evaluate"
        )
        payload = {"context": {"user_id": ctx.user_id, "attributes": ctx.attributes}}
        try:
            body = self._do_post(url, payload)
            return parse_bulk_response(body)
        except (SDKError, ServerError) as exc:
            # Don't fall back on auth errors.
            if isinstance(exc, AuthError):
                raise
            if self._defaults:
                return defaults_as_results(self._defaults)
            raise

    def evaluate(self, key: str, ctx: EvalContext) -> EvalResult:
        """Evaluate a single flag by key.

        Calls the bulk endpoint and filters by key — does not call the
        per-flag endpoint. Raises FlagNotFoundError if key is absent from
        the response.

        Raises:
            ValueError: if ctx is None.
            FlagNotFoundError: if key is absent from the 200 response.
            AuthError: on HTTP 401 or 403.
            SDKError: on HTTP 404, 5xx, network error, or malformed response.
        """
        results = self.evaluate_all(ctx)
        if key not in results:
            raise FlagNotFoundError(key)
        return results[key]

    def bool(self, key: str, ctx: EvalContext) -> bool:
        """Return True if the flag variant is 'true'.

        Calls the bulk endpoint. Raises FlagNotFoundError if key is absent.

        Raises:
            ValueError: if ctx is None.
            FlagNotFoundError: if key is absent from the 200 response.
            AuthError: on HTTP 401 or 403.
            SDKError: on HTTP 404, 5xx, network error, or malformed response.
        """
        return self.evaluate(key, ctx).variant == "true"

    def string(self, key: str, ctx: EvalContext) -> str:
        """Return the flag's variant string.

        Calls the bulk endpoint. Raises FlagNotFoundError if key is absent.

        Raises:
            ValueError: if ctx is None.
            FlagNotFoundError: if key is absent from the 200 response.
            AuthError: on HTTP 401 or 403.
            SDKError: on HTTP 404, 5xx, network error, or malformed response.
        """
        return self.evaluate(key, ctx).variant

    # ------------------------------------------------------------------
    # Internal
    # ------------------------------------------------------------------

    def _do_post(self, url: str, payload: dict) -> dict:
        """Execute a POST request and return the parsed JSON body.

        Catches all httpx exceptions and re-raises as SDKError so that the
        Authorization header (which contains the api_key) never appears in
        an unhandled exception traceback.
        """
        try:
            resp = self._http.post(
                url,
                json=payload,
                headers={"Authorization": f"Bearer {self._api_key}"},
            )
        except httpx.RequestError as exc:
            # Use only the exception class name — not str(exc), which may
            # include URL details or request headers on some httpx versions.
            raise SDKError(
                f"cuttlegate: request failed: {type(exc).__name__}"
            ) from exc

        if resp.status_code in (401, 403):
            raise AuthError(resp.status_code)
        if resp.status_code == 404:
            raise SDKError("cuttlegate: project or environment not found")
        if resp.status_code >= 500:
            raise ServerError(resp.status_code)
        if resp.status_code != 200:
            raise SDKError(f"cuttlegate: unexpected status {resp.status_code}")

        try:
            return resp.json()
        except Exception as exc:
            raise SDKError(f"cuttlegate: malformed response: {exc}") from exc

