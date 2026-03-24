"""Tests for AsyncCuttlegateClient."""

from __future__ import annotations

import json

import httpx
import pytest

from cuttlegate import AsyncCuttlegateClient
from cuttlegate.errors import AuthError, ConfigError, FlagNotFoundError, SDKError, ServerError
from cuttlegate.types import CuttlegateConfig, EvalContext

_CONFIG = CuttlegateConfig(
    api_key="test-key",
    server_url="https://flags.example.com",
    project="my-project",
    environment="production",
)

_CTX = EvalContext(user_id="u1")

_BULK_RESPONSE = {
    "flags": [
        {
            "key": "dark-mode",
            "enabled": True,
            "value": None,
            "value_key": "true",
            "reason": "rule_match",
            "type": "bool",
        },
        {
            "key": "theme",
            "enabled": True,
            "value": "ocean",
            "value_key": "ocean",
            "reason": "default",
            "type": "string",
        },
    ],
    "evaluated_at": "2026-03-24T10:00:00Z",
}


def _mock_transport(status: int = 200, body: dict | None = None) -> httpx.MockTransport:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            status_code=status,
            json=body if body is not None else _BULK_RESPONSE,
        )
    return httpx.MockTransport(handler)


@pytest.fixture
def config_with_transport():
    def _make(status: int = 200, body: dict | None = None) -> CuttlegateConfig:
        # Monkey-patch: we'll create the client and replace its _http
        return _CONFIG
    return _make


@pytest.mark.anyio
async def test_evaluate_all_returns_all_flags():
    client = AsyncCuttlegateClient(_CONFIG)
    client._http = httpx.AsyncClient(transport=_mock_transport())
    results = await client.evaluate_all(_CTX)
    assert set(results.keys()) == {"dark-mode", "theme"}
    assert results["dark-mode"].enabled is True
    assert results["dark-mode"].variant == "true"
    assert results["theme"].variant == "ocean"
    await client.aclose()


@pytest.mark.anyio
async def test_evaluate_single_flag():
    client = AsyncCuttlegateClient(_CONFIG)
    client._http = httpx.AsyncClient(transport=_mock_transport())
    result = await client.evaluate("dark-mode", _CTX)
    assert result.key == "dark-mode"
    assert result.enabled is True
    await client.aclose()


@pytest.mark.anyio
async def test_evaluate_unknown_key_raises():
    client = AsyncCuttlegateClient(_CONFIG)
    client._http = httpx.AsyncClient(transport=_mock_transport())
    with pytest.raises(FlagNotFoundError):
        await client.evaluate("nonexistent", _CTX)
    await client.aclose()


@pytest.mark.anyio
async def test_bool_returns_true():
    client = AsyncCuttlegateClient(_CONFIG)
    client._http = httpx.AsyncClient(transport=_mock_transport())
    assert await client.bool("dark-mode", _CTX) is True
    await client.aclose()


@pytest.mark.anyio
async def test_string_returns_variant():
    client = AsyncCuttlegateClient(_CONFIG)
    client._http = httpx.AsyncClient(transport=_mock_transport())
    assert await client.string("theme", _CTX) == "ocean"
    await client.aclose()


@pytest.mark.anyio
async def test_auth_error_on_401():
    client = AsyncCuttlegateClient(_CONFIG)
    client._http = httpx.AsyncClient(transport=_mock_transport(status=401, body={}))
    with pytest.raises(AuthError) as exc_info:
        await client.evaluate_all(_CTX)
    assert exc_info.value.status_code == 401
    await client.aclose()


@pytest.mark.anyio
async def test_server_error_on_500():
    client = AsyncCuttlegateClient(_CONFIG)
    client._http = httpx.AsyncClient(transport=_mock_transport(status=500, body={}))
    with pytest.raises(ServerError):
        await client.evaluate_all(_CTX)
    await client.aclose()


@pytest.mark.anyio
async def test_context_manager():
    async with AsyncCuttlegateClient(_CONFIG) as client:
        client._http = httpx.AsyncClient(transport=_mock_transport())
        result = await client.evaluate("dark-mode", _CTX)
        assert result.enabled is True


@pytest.mark.anyio
async def test_none_context_raises():
    client = AsyncCuttlegateClient(_CONFIG)
    with pytest.raises(ValueError):
        await client.evaluate_all(None)
    await client.aclose()


def test_invalid_config_raises_sync():
    with pytest.raises(ConfigError):
        AsyncCuttlegateClient(CuttlegateConfig(
            api_key="", server_url="https://x.com", project="p", environment="e",
        ))
