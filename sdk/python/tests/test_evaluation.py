"""Unit tests for CuttlegateClient evaluation methods — all 12 BDD scenarios from #66.

Uses httpx.MockTransport so no live server is required.
"""

from __future__ import annotations

import json

import httpx
import pytest

from cuttlegate import (
    AuthError,
    CuttlegateClient,
    CuttlegateConfig,
    EvalContext,
    EvalResult,
    FlagNotFoundError,
    SDKError,
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _make_config(
    api_key: str = "cg_test-key-abc",
    server_url: str = "https://flags.example.com",
    project: str = "acme",
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


def _flags_response(
    flags: list[dict], evaluated_at: str = "2026-03-23T10:00:00Z"
) -> httpx.Response:
    body = json.dumps({"flags": flags, "evaluated_at": evaluated_at})
    return httpx.Response(200, content=body.encode(), headers={"content-type": "application/json"})


def _client_with_transport(
    transport: httpx.MockTransport, **config_kwargs
) -> CuttlegateClient:
    """Construct a CuttlegateClient whose httpx.Client uses the given mock transport."""
    config = _make_config(**config_kwargs)
    client = CuttlegateClient(config)
    client._http = httpx.Client(transport=transport)
    return client


# ---------------------------------------------------------------------------
# @happy — bool flag evaluates to True
# ---------------------------------------------------------------------------

def test_happy_bool_flag_evaluates_to_true():
    """@happy: bool() returns True for a flag with value_key 'true'.

    Server receives POST .../evaluate with correct Authorization header and body.
    """
    api_key = "cg_test-key-abc"
    captured_requests: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        captured_requests.append(request)
        return _flags_response([
            {"key": "my-bool-flag", "enabled": True, "value": None,
             "value_key": "true", "reason": "default", "type": "boolean"},
        ])

    client = _client_with_transport(httpx.MockTransport(handler), api_key=api_key)
    result = client.bool("my-bool-flag", EvalContext(user_id="u1"))

    assert result is True
    assert len(captured_requests) == 1
    req = captured_requests[0]
    assert req.method == "POST"
    assert "/api/v1/projects/acme/environments/production/evaluate" in str(req.url)
    assert req.headers["Authorization"] == f"Bearer {api_key}"
    body = json.loads(req.content)
    assert body["context"]["user_id"] == "u1"
    assert body["context"]["attributes"] == {}


# ---------------------------------------------------------------------------
# @happy — string flag returns variant value
# ---------------------------------------------------------------------------

def test_happy_string_flag_returns_variant_value():
    """@happy: string() returns the value_key string for a string flag."""

    def handler(request: httpx.Request) -> httpx.Response:
        return _flags_response([
            {"key": "variant-flag", "enabled": True, "value": None,
             "value_key": "blue", "reason": "targeting_rule", "type": "string"},
        ])

    client = _client_with_transport(httpx.MockTransport(handler))
    result = client.string("variant-flag", EvalContext(user_id="u1"))
    assert result == "blue"


# ---------------------------------------------------------------------------
# @happy — evaluate() returns fully populated EvalResult
# ---------------------------------------------------------------------------

def test_happy_evaluate_returns_fully_populated_eval_result():
    """@happy: evaluate() returns EvalResult with all fields set including variant and evaluated_at."""

    def handler(request: httpx.Request) -> httpx.Response:
        return _flags_response(
            [{"key": "my-flag", "enabled": True, "value": None,
              "value_key": "true", "reason": "default", "type": "boolean"}],
            evaluated_at="2026-03-23T10:00:00Z",
        )

    client = _client_with_transport(httpx.MockTransport(handler))
    result = client.evaluate("my-flag", EvalContext(user_id="u1"))

    assert isinstance(result, EvalResult)
    assert result.key == "my-flag"
    assert result.enabled is True
    assert result.variant == "true"
    assert result.reason == "default"
    assert result.evaluated_at == "2026-03-23T10:00:00Z"
    assert result.value == ""   # deprecated field; JSON null → ""


# ---------------------------------------------------------------------------
# @happy — evaluate_all() returns dict keyed by flag key; one HTTP request
# ---------------------------------------------------------------------------

def test_happy_evaluate_all_returns_dict_keyed_by_flag_key():
    """@happy: evaluate_all() returns dict[str, EvalResult]; exactly one HTTP request."""
    request_count = 0

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal request_count
        request_count += 1
        return _flags_response([
            {"key": "flag-a", "enabled": True, "value": None,
             "value_key": "true", "reason": "default", "type": "boolean"},
            {"key": "flag-b", "enabled": False, "value": None,
             "value_key": "red", "reason": "targeting_rule", "type": "string"},
        ])

    client = _client_with_transport(httpx.MockTransport(handler))
    results = client.evaluate_all(EvalContext(user_id="u1"))

    assert set(results.keys()) == {"flag-a", "flag-b"}
    assert isinstance(results["flag-a"], EvalResult)
    assert results["flag-a"].variant == "true"
    assert isinstance(results["flag-b"], EvalResult)
    assert results["flag-b"].variant == "red"
    assert request_count == 1


# ---------------------------------------------------------------------------
# @error-path — flag key not found in response raises FlagNotFoundError
# ---------------------------------------------------------------------------

def test_error_flag_key_not_found_raises_flag_not_found_error():
    """@error-path: FlagNotFoundError raised when key absent from 200 response; no api_key in message."""
    api_key = "cg_secret-key-xyz"

    def handler(request: httpx.Request) -> httpx.Response:
        return _flags_response([])

    client = _client_with_transport(httpx.MockTransport(handler), api_key=api_key)

    with pytest.raises(FlagNotFoundError) as exc_info:
        client.bool("missing-flag", EvalContext(user_id="u1"))

    err = exc_info.value
    assert err.key == "missing-flag"
    assert api_key not in str(err)


# ---------------------------------------------------------------------------
# @error-path — invalid API key raises AuthError (HTTP 401)
# ---------------------------------------------------------------------------

def test_error_invalid_api_key_raises_auth_error_401():
    """@error-path: HTTP 401 raises AuthError; message does not contain api_key."""
    api_key = "cg_invalid-key-xyz"

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(401, content=b'{"error":"Unauthorized"}',
                              headers={"content-type": "application/json"})

    client = _client_with_transport(httpx.MockTransport(handler), api_key=api_key)

    with pytest.raises(AuthError) as exc_info:
        client.bool("my-flag", EvalContext(user_id="u1"))

    err = exc_info.value
    assert err.status_code == 401
    assert api_key not in str(err)
    assert api_key not in repr(err)


# ---------------------------------------------------------------------------
# @error-path — revoked API key raises AuthError (HTTP 403); indistinguishable
# ---------------------------------------------------------------------------

def test_error_revoked_api_key_raises_auth_error_403_indistinguishable():
    """@error-path: HTTP 403 raises AuthError; indistinguishable from 401 by type."""
    api_key = "cg_revoked-key-xyz"

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(403, content=b'{"error":"Forbidden"}',
                              headers={"content-type": "application/json"})

    client = _client_with_transport(httpx.MockTransport(handler), api_key=api_key)

    with pytest.raises(AuthError) as exc_info:
        client.bool("my-flag", EvalContext(user_id="u1"))

    err = exc_info.value
    assert err.status_code == 403
    # Both 401 and 403 raise AuthError — same type, not distinguishable
    assert type(err) is AuthError
    assert api_key not in str(err)


# ---------------------------------------------------------------------------
# @error-path — server 500 raises SDKError with correct message
# ---------------------------------------------------------------------------

def test_error_server_500_raises_sdk_error():
    """@error-path: HTTP 500 raises SDKError (via ServerError subclass) with message 'cuttlegate: server error 500'."""
    api_key = "cg_test-key-abc"

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(500, content=b'{"error":"internal server error"}')

    client = _client_with_transport(httpx.MockTransport(handler), api_key=api_key)

    with pytest.raises(SDKError) as exc_info:
        client.bool("my-flag", EvalContext(user_id="u1"))

    err = exc_info.value
    assert str(err) == "cuttlegate: server error 500"
    assert api_key not in str(err)


# ---------------------------------------------------------------------------
# @edge — network timeout raises SDKError without credential leak
# ---------------------------------------------------------------------------

def test_edge_network_timeout_raises_sdk_error_no_credential_leak():
    """@edge: httpx.TimeoutException raises SDKError starting 'cuttlegate: request failed:'; no api_key."""
    api_key = "cg_secret-key-xyz"

    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.TimeoutException("timed out", request=request)

    client = _client_with_transport(httpx.MockTransport(handler), api_key=api_key)

    with pytest.raises(SDKError) as exc_info:
        client.bool("my-flag", EvalContext(user_id="u1"))

    msg = str(exc_info.value)
    assert msg.startswith("cuttlegate: request failed:")
    assert api_key not in msg


# ---------------------------------------------------------------------------
# @edge — server returns 200 with malformed JSON raises SDKError
# ---------------------------------------------------------------------------

def test_edge_malformed_json_raises_sdk_error():
    """@edge: HTTP 200 with non-JSON body raises SDKError starting 'cuttlegate: malformed response:'."""

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, content=b"not json",
                              headers={"content-type": "text/plain"})

    client = _client_with_transport(httpx.MockTransport(handler))

    with pytest.raises(SDKError) as exc_info:
        client.evaluate("my-flag", EvalContext(user_id="u1"))

    assert str(exc_info.value).startswith("cuttlegate: malformed response:")


# ---------------------------------------------------------------------------
# @edge — server returns 200 with missing required field raises SDKError
# ---------------------------------------------------------------------------

def test_edge_missing_required_field_raises_sdk_error():
    """@edge: HTTP 200 with flag entry missing 'reason' raises SDKError; not KeyError."""
    flags = [
        {"key": "my-flag", "enabled": True, "value": None, "value_key": "true"}
        # deliberately missing "reason"
    ]

    def handler(request: httpx.Request) -> httpx.Response:
        return _flags_response(flags)

    client = _client_with_transport(httpx.MockTransport(handler))

    with pytest.raises(SDKError) as exc_info:
        client.evaluate("my-flag", EvalContext(user_id="u1"))

    msg = str(exc_info.value)
    assert msg == "cuttlegate: malformed response: missing field 'reason'"
    assert not isinstance(exc_info.value, KeyError)


# ---------------------------------------------------------------------------
# @edge — context=None raises ValueError before any network call
# ---------------------------------------------------------------------------

def test_edge_context_none_raises_value_error_no_network_call():
    """@edge: context=None raises ValueError immediately; no HTTP request is made."""
    request_count = 0

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal request_count
        request_count += 1
        return _flags_response([])

    client = _client_with_transport(httpx.MockTransport(handler))

    with pytest.raises(ValueError):
        client.bool("my-flag", None)  # type: ignore[arg-type]

    assert request_count == 0
