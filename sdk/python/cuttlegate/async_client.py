"""AsyncCuttlegateClient — async flag evaluation client backed by httpx.AsyncClient.

Usage with async context manager (recommended)::

    async with AsyncCuttlegateClient(config) as client:
        result = await client.evaluate("dark-mode", ctx)

Or manual lifecycle::

    client = AsyncCuttlegateClient(config)
    try:
        result = await client.evaluate("dark-mode", ctx)
    finally:
        await client.aclose()
"""

from __future__ import annotations

import httpx

from ._internal import parse_bulk_response, require_context, validate_config
from .errors import AuthError, FlagNotFoundError, SDKError, ServerError
from .types import CuttlegateConfig, EvalContext, EvalResult


class AsyncCuttlegateClient:
    """Async client for the Cuttlegate feature flag service.

    Backed by httpx.AsyncClient. No network calls are made at construction time.
    The caller must close the client when done — use ``async with`` or call
    ``await client.aclose()``.
    """

    def __init__(self, config: CuttlegateConfig) -> None:
        validate_config(config)
        self._api_key = config.api_key
        self._server_url = config.server_url.rstrip("/")
        self._project = config.project
        self._environment = config.environment
        self._http = httpx.AsyncClient(timeout=config.timeout_ms / 1000)

    async def __aenter__(self) -> AsyncCuttlegateClient:
        return self

    async def __aexit__(self, *exc: object) -> None:
        await self.aclose()

    async def aclose(self) -> None:
        """Close the underlying HTTP client."""
        await self._http.aclose()

    async def evaluate_all(self, ctx: EvalContext) -> dict[str, EvalResult]:
        """Evaluate all flags for ctx. One HTTP round trip."""
        require_context(ctx)
        url = (
            f"{self._server_url}/api/v1/projects/{self._project}"
            f"/environments/{self._environment}/evaluate"
        )
        payload = {"context": {"user_id": ctx.user_id, "attributes": ctx.attributes}}
        body = await self._do_post(url, payload)
        return parse_bulk_response(body)

    async def evaluate(self, key: str, ctx: EvalContext) -> EvalResult:
        """Evaluate a single flag by key. Raises FlagNotFoundError if absent."""
        results = await self.evaluate_all(ctx)
        if key not in results:
            raise FlagNotFoundError(key)
        return results[key]

    async def bool(self, key: str, ctx: EvalContext) -> bool:
        """Return True if the flag variant is 'true'."""
        return (await self.evaluate(key, ctx)).variant == "true"

    async def string(self, key: str, ctx: EvalContext) -> str:
        """Return the flag's variant string."""
        return (await self.evaluate(key, ctx)).variant

    async def _do_post(self, url: str, payload: dict) -> dict:
        try:
            resp = await self._http.post(
                url,
                json=payload,
                headers={"Authorization": f"Bearer {self._api_key}"},
            )
        except httpx.RequestError as exc:
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

