"""Unit tests for CuttlegateClient — covers all 13 BDD scenarios from grooming.

Uses httpx.MockTransport so no live server is required.
"""

from __future__ import annotations

import json

import httpx
import pytest

from cuttlegate import (
    AuthError,
    ConfigError,
    CuttlegateClient,
    CuttlegateConfig,
    EvalContext,
    FlagNotFoundError,
    NotFoundError,
    SDKError,
    ServerError,
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _make_config(
    api_key: str = "key-123",
    server_url: str = "https://flags.example.com",
    project: str = "my-project",
    environment: str = "production",
    timeout_ms: int = 10_000,
) -> CuttlegateConfig:
    return CuttlegateConfig(
        api_key=api_key,
        server_url=server_url,
        project=project,
        environment=environment,
        timeout_ms=timeout_ms,
    )


def _flags_response(flags: list[dict], evaluated_at: str = "2026-03-23T00:00:00Z") -> httpx.Response:
    body = json.dumps({"flags": flags, "evaluated_at": evaluated_at})
    return httpx.Response(200, content=body.encode(), headers={"content-type": "application/json"})


def _client_with_transport(transport: httpx.MockTransport, **config_kwargs) -> CuttlegateClient:
    """Construct a CuttlegateClient whose httpx.Client uses the given mock transport."""
    config = _make_config(**config_kwargs)
    client = CuttlegateClient(config)
    # Replace the internal http client with one using the mock transport.
    client._http = httpx.Client(transport=transport)
    return client


# ---------------------------------------------------------------------------
# @happy — create client with valid config → succeeds, no network call
# ---------------------------------------------------------------------------

def test_happy_create_client_valid_config_no_network():
    """@happy: CuttlegateClient(config) with valid config raises no exception."""
    config = _make_config()
    client = CuttlegateClient(config)
    assert client is not None


# ---------------------------------------------------------------------------
# @happy — evaluate_all returns results keyed by flag key
# ---------------------------------------------------------------------------

def test_happy_evaluate_all_returns_results_keyed_by_flag():
    """@happy: evaluate_all returns dict keyed by flag key with correct fields."""
    flags = [
        {
            "key": "dark-mode",
            "enabled": True,
            "value": None,
            "value_key": "true",
            "reason": "default",
            "type": "bool",
        }
    ]

    def handler(request: httpx.Request) -> httpx.Response:
        return _flags_response(flags)

    client = _client_with_transport(httpx.MockTransport(handler))
    ctx = EvalContext(user_id="u1")
    results = client.evaluate_all(ctx)

    assert "dark-mode" in results
    result = results["dark-mode"]
    assert result.enabled is True
    assert result.variant == "true"
    assert result.reason == "default"


# ---------------------------------------------------------------------------
# @happy — bool() returns True for an enabled bool flag
# ---------------------------------------------------------------------------

def test_happy_bool_returns_true_for_enabled_flag():
    """@happy: bool() returns True when value_key is 'true'."""
    flags = [{"key": "feature-x", "enabled": True, "value": None, "value_key": "true", "reason": "rule", "type": "bool"}]

    def handler(request: httpx.Request) -> httpx.Response:
        return _flags_response(flags)

    client = _client_with_transport(httpx.MockTransport(handler))
    result = client.bool("feature-x", EvalContext(user_id="u1"))
    assert result is True


# ---------------------------------------------------------------------------
# @happy — string() returns variant key for a string flag
# ---------------------------------------------------------------------------

def test_happy_string_returns_variant_key():
    """@happy: string() returns value_key for a string flag."""
    flags = [{"key": "color-theme", "enabled": True, "value": None, "value_key": "blue", "reason": "rule", "type": "string"}]

    def handler(request: httpx.Request) -> httpx.Response:
        return _flags_response(flags)

    client = _client_with_transport(httpx.MockTransport(handler))
    result = client.string("color-theme", EvalContext(user_id="u1"))
    assert result == "blue"


# ---------------------------------------------------------------------------
# @error-path — missing api_key → ConfigError immediately at construction
# ---------------------------------------------------------------------------

def test_error_missing_api_key_raises_config_error():
    """@error-path: empty api_key raises ConfigError at construction with correct message."""
    config = CuttlegateConfig(
        api_key="",
        server_url="https://flags.example.com",
        project="p",
        environment="prod",
    )
    with pytest.raises(ConfigError) as exc_info:
        CuttlegateClient(config)
    assert str(exc_info.value) == "api_key is required"


# ---------------------------------------------------------------------------
# @error-path — missing server_url → ConfigError at construction
# ---------------------------------------------------------------------------

def test_error_missing_server_url_raises_config_error():
    """@error-path: empty server_url raises ConfigError at construction."""
    config = CuttlegateConfig(
        api_key="key-123",
        server_url="",
        project="p",
        environment="prod",
    )
    with pytest.raises(ConfigError) as exc_info:
        CuttlegateClient(config)
    assert str(exc_info.value) == "server_url is required"


# ---------------------------------------------------------------------------
# @error-path — invalid server_url scheme → ConfigError at construction
# ---------------------------------------------------------------------------

def test_error_invalid_server_url_scheme_raises_config_error():
    """@error-path: ftp:// scheme raises ConfigError at construction."""
    config = CuttlegateConfig(
        api_key="key-123",
        server_url="ftp://flags.example.com",
        project="p",
        environment="prod",
    )
    with pytest.raises(ConfigError) as exc_info:
        CuttlegateClient(config)
    assert str(exc_info.value) == "server_url must be an http or https URL"


# ---------------------------------------------------------------------------
# @error-path — unknown flag key → NotFoundError
# ---------------------------------------------------------------------------

def test_error_unknown_flag_key_raises_not_found_error():
    """@error-path: evaluate() raises NotFoundError when key absent from response."""
    api_key = "secret-key-xyz"

    def handler(request: httpx.Request) -> httpx.Response:
        return _flags_response([])

    client = _client_with_transport(httpx.MockTransport(handler), api_key=api_key)
    ctx = EvalContext(user_id="u1")

    with pytest.raises(NotFoundError) as exc_info:
        client.evaluate("absent-flag", ctx)

    error_str = str(exc_info.value)
    assert "absent-flag" in error_str
    assert api_key not in error_str


# ---------------------------------------------------------------------------
# @error-path — server returns 401 → AuthError
# ---------------------------------------------------------------------------

def test_error_401_raises_auth_error():
    """@error-path: HTTP 401 raises AuthError; error message does not contain api_key."""
    api_key = "secret-key-xyz"

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(401, content=b'{"error":"unauthorized"}', headers={"content-type": "application/json"})

    client = _client_with_transport(httpx.MockTransport(handler), api_key=api_key)

    with pytest.raises(AuthError) as exc_info:
        client.evaluate_all(EvalContext(user_id="u1"))

    assert api_key not in str(exc_info.value)


# ---------------------------------------------------------------------------
# @error-path — server returns 500 → ServerError
# ---------------------------------------------------------------------------

def test_error_500_raises_server_error():
    """@error-path: HTTP 500 raises ServerError."""
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(500, content=b'{"error":"internal"}')

    client = _client_with_transport(httpx.MockTransport(handler))

    with pytest.raises(ServerError):
        client.evaluate_all(EvalContext(user_id="u1"))


# ---------------------------------------------------------------------------
# @edge — api_key never appears in repr(config)
# ---------------------------------------------------------------------------

def test_edge_api_key_not_in_repr():
    """@edge: repr(CuttlegateConfig) does not contain the api_key value."""
    config = CuttlegateConfig(
        api_key="super-secret-key-abc",
        server_url="https://flags.example.com",
        project="p",
        environment="prod",
    )
    assert "super-secret-key-abc" not in repr(config)


# ---------------------------------------------------------------------------
# @edge — api_key never appears in any error message
# ---------------------------------------------------------------------------

def test_edge_api_key_not_in_any_error_message():
    """@edge: api_key does not appear in any error's str() regardless of error type."""
    api_key = "super-secret-key-abc"

    # AuthError
    auth_err = AuthError(401)
    assert api_key not in str(auth_err)

    # ServerError
    server_err = ServerError(500)
    assert api_key not in str(server_err)

    # NotFoundError
    not_found_err = NotFoundError("some-flag")
    assert api_key not in str(not_found_err)

    # ConfigError raised from client construction
    config = CuttlegateConfig(api_key=api_key, server_url="", project="p", environment="prod")
    with pytest.raises(ConfigError) as exc_info:
        CuttlegateClient(config)
    assert api_key not in str(exc_info.value)


# ---------------------------------------------------------------------------
# @edge — timeout is applied to evaluation requests
# ---------------------------------------------------------------------------

def test_edge_timeout_raises_sdk_error():
    """@edge: httpx.TimeoutException from transport is re-raised as SDKError."""
    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.TimeoutException("timeout", request=request)

    client = _client_with_transport(httpx.MockTransport(handler), timeout_ms=100)

    with pytest.raises(SDKError) as exc_info:
        client.evaluate_all(EvalContext(user_id="u1"))
    assert str(exc_info.value).startswith("cuttlegate: request failed:")


# ---------------------------------------------------------------------------
# Additional: missing project/environment → ConfigError
# ---------------------------------------------------------------------------

def test_error_missing_project_raises_config_error():
    """Missing project raises ConfigError."""
    config = CuttlegateConfig(api_key="k", server_url="https://x.com", project="", environment="prod")
    with pytest.raises(ConfigError) as exc_info:
        CuttlegateClient(config)
    assert str(exc_info.value) == "project is required"


def test_error_missing_environment_raises_config_error():
    """Missing environment raises ConfigError."""
    config = CuttlegateConfig(api_key="k", server_url="https://x.com", project="p", environment="")
    with pytest.raises(ConfigError) as exc_info:
        CuttlegateClient(config)
    assert str(exc_info.value) == "environment is required"
