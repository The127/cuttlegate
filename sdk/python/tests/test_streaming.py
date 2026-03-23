"""Tests for the SSE streaming client.

Covers all 9 BDD scenarios from grooming:
  @happy  — valid event delivered, FlagChangeEvent fields correct
  @edge   — network drop reconnects, unknown event type ignored, malformed JSON non-fatal
  @auth-bypass — empty token raises ConfigError synchronously
  @error-path  — 401 terminal (no reconnect), 403 terminal, 500 reconnects
"""

from __future__ import annotations

import json
import threading
import time
from typing import List
from unittest.mock import patch

import httpx
import pytest

from cuttlegate.errors import AuthError, ConfigError, InvalidResponseError, ServerError
from cuttlegate.streaming import FlagChangeEvent, StreamConnection, connect_stream
from cuttlegate.types import CuttlegateConfig


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _config(
    api_key: str = "cg_test_key",
    server_url: str = "http://localhost:8080",
    project: str = "test-project",
    environment: str = "production",
) -> CuttlegateConfig:
    return CuttlegateConfig(
        api_key=api_key,
        server_url=server_url,
        project=project,
        environment=environment,
    )


def _sse_event(**fields) -> str:
    """Build a raw SSE data line from field kwargs."""
    return f"data: {json.dumps(fields)}\n\n"


def _flag_state_changed(
    flag_key: str = "dark-mode",
    enabled: bool = True,
    project: str = "test-project",
    environment: str = "production",
    occurred_at: str = "2026-03-23T10:00:00Z",
) -> str:
    return _sse_event(
        type="flag.state_changed",
        project=project,
        environment=environment,
        flag_key=flag_key,
        enabled=enabled,
        occurred_at=occurred_at,
    )


class _SSETransport(httpx.MockTransport):
    """A mock httpx transport that returns SSE lines then closes.

    Pass ``responses`` as a list of (status_code, body_text) tuples.
    On each ``handle_request`` call, returns the next response in order.
    Once exhausted, raises httpx.RemoteProtocolError to simulate server close.
    """

    def __init__(self, responses: List[tuple]) -> None:
        self._responses = list(responses)
        self._index = 0

    def handle_request(self, request: httpx.Request) -> httpx.Response:
        if self._index >= len(self._responses):
            raise httpx.RemoteProtocolError("server closed connection", request=request)
        status, body = self._responses[self._index]
        self._index += 1
        stream = httpx.ByteStream(body.encode())
        return httpx.Response(status, stream=stream, request=request)


# ---------------------------------------------------------------------------
# @happy — valid event delivered
# ---------------------------------------------------------------------------

def test_happy_event_delivered():
    """@happy: on_change called with correct FlagChangeEvent fields."""
    body = _flag_state_changed(
        flag_key="dark-mode",
        enabled=True,
        project="test-project",
        environment="production",
        occurred_at="2026-03-23T10:00:00Z",
    )
    received: List[FlagChangeEvent] = []
    done = threading.Event()
    transport = _SSETransport([(200, body)])

    def on_change(event: FlagChangeEvent) -> None:
        received.append(event)
        done.set()

    with patch("httpx.stream") as mock_stream:
        mock_stream.side_effect = lambda method, url, **kwargs: _patched_stream(
            transport, method, url, **kwargs
        )
        conn = connect_stream(_config(), on_change)
        done.wait(timeout=5)
        conn.close()

    assert len(received) >= 1
    evt = received[0]
    assert evt.type == "flag.state_changed"
    assert evt.project == "test-project"
    assert evt.environment == "production"
    assert evt.flag_key == "dark-mode"
    assert evt.enabled is True
    assert evt.occurred_at == "2026-03-23T10:00:00Z"


def test_happy_flag_change_event_all_fields():
    """@happy: FlagChangeEvent has all six locked fields from the wire format."""
    body = _flag_state_changed(
        flag_key="feature-x",
        enabled=False,
        project="acme",
        environment="staging",
        occurred_at="2026-01-01T00:00:00Z",
    )
    received: List[FlagChangeEvent] = []
    done = threading.Event()
    transport = _SSETransport([(200, body)])

    def on_change(event: FlagChangeEvent) -> None:
        received.append(event)
        done.set()

    with patch("httpx.stream") as mock_stream:
        mock_stream.side_effect = lambda method, url, **kwargs: _patched_stream(
            transport, method, url, **kwargs
        )
        conn = connect_stream(_config(project="acme", environment="staging"), on_change)
        done.wait(timeout=5)
        conn.close()

    assert len(received) >= 1
    evt = received[0]
    assert evt.type == "flag.state_changed"
    assert evt.project == "acme"
    assert evt.environment == "staging"
    assert evt.flag_key == "feature-x"
    assert evt.enabled is False
    assert evt.occurred_at == "2026-01-01T00:00:00Z"


# ---------------------------------------------------------------------------
# @edge — unknown event type silently ignored
# ---------------------------------------------------------------------------

def test_edge_unknown_event_type_silently_ignored():
    """@edge: unknown event type does not call on_change or on_error."""
    unknown = _sse_event(
        type="flag.something_new", project="p", environment="e",
        flag_key="k", enabled=True, occurred_at="2026-01-01T00:00:00Z",
    )
    known = _flag_state_changed(flag_key="real-flag")
    body = unknown + known

    received: List[FlagChangeEvent] = []
    errors: List[Exception] = []
    done = threading.Event()
    transport = _SSETransport([(200, body)])

    def on_change(event: FlagChangeEvent) -> None:
        received.append(event)
        done.set()

    def on_error(exc: Exception) -> None:
        errors.append(exc)

    with patch("httpx.stream") as mock_stream:
        mock_stream.side_effect = lambda method, url, **kwargs: _patched_stream(
            transport, method, url, **kwargs
        )
        conn = connect_stream(_config(), on_change, on_error)
        done.wait(timeout=5)
        conn.close()

    # Only the known event triggers on_change; unknown is silent
    assert len(received) >= 1
    assert received[0].flag_key == "real-flag"
    # No error for the unknown type
    assert not errors


# ---------------------------------------------------------------------------
# @edge — malformed JSON is non-fatal
# ---------------------------------------------------------------------------

def test_edge_malformed_json_nonfatal():
    """@edge: malformed JSON calls on_error with InvalidResponseError, stream continues."""
    bad_line = "data: {not-json}\n\n"
    good_line = _flag_state_changed(flag_key="after-bad")
    body = bad_line + good_line

    received: List[FlagChangeEvent] = []
    errors: List[Exception] = []
    done = threading.Event()
    transport = _SSETransport([(200, body)])

    def on_change(event: FlagChangeEvent) -> None:
        received.append(event)
        done.set()

    def on_error(exc: Exception) -> None:
        errors.append(exc)

    with patch("httpx.stream") as mock_stream:
        mock_stream.side_effect = lambda method, url, **kwargs: _patched_stream(
            transport, method, url, **kwargs
        )
        conn = connect_stream(_config(), on_change, on_error)
        done.wait(timeout=5)
        conn.close()

    assert any(isinstance(e, InvalidResponseError) for e in errors)
    assert len(received) >= 1
    assert received[0].flag_key == "after-bad"


# ---------------------------------------------------------------------------
# @edge — network drop reconnects
# ---------------------------------------------------------------------------

def test_edge_network_drop_reconnects():
    """@edge: network error triggers reconnect; on_change fires after reconnect."""
    good_body = _flag_state_changed(flag_key="post-reconnect")
    received: List[FlagChangeEvent] = []
    done = threading.Event()
    call_count = 0

    def mock_stream_fn(method, url, **kwargs):
        nonlocal call_count
        call_count += 1
        if call_count == 1:
            raise httpx.ConnectError("connection refused")
        # Second+ call: return real SSE data
        transport = _SSETransport([(200, good_body)])
        return _patched_stream(transport, method, url, **kwargs)

    def on_change(event: FlagChangeEvent) -> None:
        received.append(event)
        done.set()

    with patch("httpx.stream", side_effect=mock_stream_fn):
        with patch("cuttlegate.streaming._backoff_delay", return_value=0.01):
            conn = connect_stream(_config(), on_change)
            done.wait(timeout=5)
            conn.close()

    assert call_count >= 2
    assert len(received) >= 1
    assert received[0].flag_key == "post-reconnect"


# ---------------------------------------------------------------------------
# @auth-bypass — empty token raises ConfigError synchronously
# ---------------------------------------------------------------------------

def test_auth_bypass_empty_token_raises_config_error():
    """@auth-bypass: empty token raises ConfigError before any network call."""
    cfg = _config(api_key="")
    with pytest.raises(ConfigError):
        connect_stream(cfg, lambda e: None)


# ---------------------------------------------------------------------------
# @error-path — 401 is terminal, no reconnect
# ---------------------------------------------------------------------------

def test_error_path_401_terminal_no_reconnect():
    """@error-path: 401 calls on_error with AuthError(401), no reconnect."""
    errors: List[Exception] = []
    call_count = 0
    done = threading.Event()

    def mock_stream_fn(method, url, **kwargs):
        nonlocal call_count
        call_count += 1
        transport = _SSETransport([(401, '{"error":"unauthorized"}')])
        return _patched_stream(transport, method, url, **kwargs)

    def on_error(exc: Exception) -> None:
        errors.append(exc)
        done.set()

    with patch("httpx.stream", side_effect=mock_stream_fn):
        conn = connect_stream(_config(), lambda e: None, on_error)
        done.wait(timeout=5)
        # Give thread time to exit the loop cleanly before asserting call_count
        time.sleep(0.1)
        conn.close()

    assert call_count == 1, f"Expected exactly 1 connection attempt, got {call_count}"
    assert len(errors) == 1
    auth_err = errors[0]
    assert isinstance(auth_err, AuthError)
    assert auth_err.status_code == 401
    # API key must not appear in error message
    assert "cg_test_key" not in str(auth_err)


# ---------------------------------------------------------------------------
# @error-path — 403 is terminal, no reconnect
# ---------------------------------------------------------------------------

def test_error_path_403_terminal_no_reconnect():
    """@error-path: 403 calls on_error with AuthError(403), no reconnect."""
    errors: List[Exception] = []
    call_count = 0
    done = threading.Event()

    def mock_stream_fn(method, url, **kwargs):
        nonlocal call_count
        call_count += 1
        transport = _SSETransport([(403, '{"error":"forbidden"}')])
        return _patched_stream(transport, method, url, **kwargs)

    def on_error(exc: Exception) -> None:
        errors.append(exc)
        done.set()

    with patch("httpx.stream", side_effect=mock_stream_fn):
        conn = connect_stream(_config(), lambda e: None, on_error)
        done.wait(timeout=5)
        time.sleep(0.1)
        conn.close()

    assert call_count == 1, f"Expected exactly 1 connection attempt, got {call_count}"
    assert len(errors) == 1
    assert isinstance(errors[0], AuthError)
    assert errors[0].status_code == 403


# ---------------------------------------------------------------------------
# @error-path — 500 triggers reconnect with backoff
# ---------------------------------------------------------------------------

def test_error_path_500_reconnects():
    """@error-path: 500 calls on_error with ServerError, then reconnects."""
    errors: List[Exception] = []
    received: List[FlagChangeEvent] = []
    done = threading.Event()
    call_count = 0

    def mock_stream_fn(method, url, **kwargs):
        nonlocal call_count
        call_count += 1
        if call_count == 1:
            transport = _SSETransport([(500, '{"error":"internal server error"}')])
        else:
            body = _flag_state_changed(flag_key="after-500")
            transport = _SSETransport([(200, body)])
        return _patched_stream(transport, method, url, **kwargs)

    def on_change(event: FlagChangeEvent) -> None:
        received.append(event)
        done.set()

    def on_error(exc: Exception) -> None:
        errors.append(exc)

    with patch("httpx.stream", side_effect=mock_stream_fn):
        with patch("cuttlegate.streaming._backoff_delay", return_value=0.01):
            conn = connect_stream(_config(), on_change, on_error)
            done.wait(timeout=5)
            conn.close()

    assert call_count >= 2
    assert any(isinstance(e, ServerError) and e.status_code == 500 for e in errors)
    assert len(received) >= 1
    assert received[0].flag_key == "after-500"


# ---------------------------------------------------------------------------
# Thread model
# ---------------------------------------------------------------------------

def test_thread_is_daemon():
    """Background thread must be daemon=True so process can exit without close()."""
    transport = _SSETransport([])

    with patch("httpx.stream") as mock_stream:
        mock_stream.side_effect = lambda method, url, **kwargs: _patched_stream(
            transport, method, url, **kwargs
        )
        conn = connect_stream(_config(), lambda e: None)
        assert conn._thread.daemon is True
        conn.close()


# ---------------------------------------------------------------------------
# Keep-alive lines skipped
# ---------------------------------------------------------------------------

def test_keepalive_line_skipped():
    """Keep-alive SSE comment lines are skipped without error or callback."""
    keepalive = ": keep-alive\n\n"
    good = _flag_state_changed(flag_key="after-keepalive")
    body = keepalive + good

    received: List[FlagChangeEvent] = []
    errors: List[Exception] = []
    done = threading.Event()
    transport = _SSETransport([(200, body)])

    def on_change(event: FlagChangeEvent) -> None:
        received.append(event)
        done.set()

    def on_error(exc: Exception) -> None:
        errors.append(exc)

    with patch("httpx.stream") as mock_stream:
        mock_stream.side_effect = lambda method, url, **kwargs: _patched_stream(
            transport, method, url, **kwargs
        )
        conn = connect_stream(_config(), on_change, on_error)
        done.wait(timeout=5)
        conn.close()

    assert not errors
    assert len(received) >= 1
    assert received[0].flag_key == "after-keepalive"


# ---------------------------------------------------------------------------
# Patching helper — adapts _SSETransport to httpx.stream() context manager
# ---------------------------------------------------------------------------

class _StreamContextManager:
    """Wraps a mock transport's response as an httpx.stream() context manager."""

    def __init__(self, transport: _SSETransport, request: httpx.Request) -> None:
        self._transport = transport
        self._request = request

    def __enter__(self) -> httpx.Response:
        return self._transport.handle_request(self._request)

    def __exit__(self, *args) -> None:
        pass


def _patched_stream(
    transport: _SSETransport,
    method: str,
    url: str,
    **kwargs,
) -> _StreamContextManager:
    request = httpx.Request(method, url, headers=kwargs.get("headers", {}))
    return _StreamContextManager(transport, request)
