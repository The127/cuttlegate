"""Tests for CachedClient — in-memory flag cache backed by a single SSE connection.

Covers all 7 BDD scenarios from grooming:
  @happy      — bootstrap seeds cache from evaluate_all
  @happy      — SSE flag.state_changed event updates cache (enabled only)
  @happy      — bool/string return cached values without HTTP call
  @edge       — unknown flag key falls back to live HTTP, not cached
  @edge       — concurrent reads are consistent under a threading.Lock
  @edge       — close() stops the SSE thread cleanly within 2 seconds
  @error-path — server unavailable on bootstrap raises SDKError, no partial state
"""

from __future__ import annotations

import threading
import time
from typing import Any, Dict, List, Optional
from unittest.mock import MagicMock, patch

import pytest

from cuttlegate.cached import CachedClient
from cuttlegate.errors import AuthError, FlagNotFoundError, SDKError
from cuttlegate.streaming import FlagChangeEvent, StreamConnection
from cuttlegate.types import CuttlegateConfig, EvalContext, EvalResult


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


def _eval_result(
    key: str,
    enabled: bool = True,
    variant: str = "true",
    reason: str = "rule_match",
    evaluated_at: str = "2026-03-23T10:00:00Z",
) -> EvalResult:
    return EvalResult(
        key=key,
        enabled=enabled,
        variant=variant,
        reason=reason,
        evaluated_at=evaluated_at,
    )


def _make_mock_stream_connection() -> StreamConnection:
    """Return a StreamConnection backed by a no-op thread that stops immediately."""
    stop_event = threading.Event()
    thread = threading.Thread(target=lambda: stop_event.wait(), daemon=True)
    thread.start()
    conn = StreamConnection(thread=thread, stop_event=stop_event)
    return conn


class _FakeInnerClient:
    """Minimal fake that replaces CuttlegateClient inside CachedClient tests."""

    def __init__(
        self,
        flags: Dict[str, EvalResult],
        raise_on_evaluate_all: Optional[Exception] = None,
    ) -> None:
        self._flags = flags
        self._raise_on_evaluate_all = raise_on_evaluate_all
        self.evaluate_all_call_count = 0
        self.evaluate_calls: List[str] = []
        self.bool_calls: List[str] = []
        self.string_calls: List[str] = []

    def evaluate_all(self, ctx: EvalContext) -> Dict[str, EvalResult]:
        self.evaluate_all_call_count += 1
        if self._raise_on_evaluate_all is not None:
            raise self._raise_on_evaluate_all
        return dict(self._flags)

    def evaluate(self, key: str, ctx: EvalContext) -> EvalResult:
        self.evaluate_calls.append(key)
        if key not in self._flags:
            raise FlagNotFoundError(key)
        return self._flags[key]

    def bool(self, key: str, ctx: EvalContext) -> bool:
        self.bool_calls.append(key)
        result = self.evaluate(key, ctx)
        return result.enabled

    def string(self, key: str, ctx: EvalContext) -> str:
        self.string_calls.append(key)
        result = self.evaluate(key, ctx)
        return result.variant


def _build_cached_client(
    flags: Dict[str, EvalResult],
    raise_on_evaluate_all: Optional[Exception] = None,
) -> tuple[CachedClient, _FakeInnerClient, StreamConnection]:
    """Construct a CachedClient with mocked inner client and SSE connection.

    Returns (client, fake_inner, mock_stream_connection).
    """
    fake_inner = _FakeInnerClient(flags, raise_on_evaluate_all=raise_on_evaluate_all)
    mock_conn = _make_mock_stream_connection()

    with patch("cuttlegate.cached.CuttlegateClient", return_value=fake_inner), \
         patch("cuttlegate.cached.connect_stream", return_value=mock_conn):
        client = CachedClient(_config())

    return client, fake_inner, mock_conn


# ---------------------------------------------------------------------------
# @happy — bootstrap seeds cache from evaluate_all
# ---------------------------------------------------------------------------

def test_happy_bootstrap_seeds_cache():
    """@happy: evaluate_all() is called once on init; cache populated from result."""
    flags = {
        "dark-mode": _eval_result("dark-mode", enabled=True, variant="true"),
        "theme": _eval_result("theme", enabled=True, variant="blue"),
    }
    client, fake_inner, conn = _build_cached_client(flags)
    try:
        # evaluate_all called exactly once during bootstrap
        assert fake_inner.evaluate_all_call_count == 1
        # Cache is populated — bool() returns from cache without further HTTP calls
        ctx = EvalContext(user_id="u1")
        assert client.bool("dark-mode", ctx) is True
        assert client.string("theme", ctx) == "blue"
        # No live fallback calls were made
        assert fake_inner.bool_calls == []
        assert fake_inner.string_calls == []
    finally:
        client.close()


# ---------------------------------------------------------------------------
# @happy — SSE flag.state_changed event updates cache
# ---------------------------------------------------------------------------

def test_happy_sse_update_applies_to_cache():
    """@happy: FlagChangeEvent updates enabled in cache; other fields unchanged."""
    flags = {
        "dark-mode": _eval_result(
            "dark-mode",
            enabled=True,
            variant="true",
            reason="rule_match",
            evaluated_at="2026-03-23T10:00:00Z",
        ),
    }
    client, fake_inner, conn = _build_cached_client(flags)
    ctx = EvalContext(user_id="u1")

    try:
        # Before SSE event: enabled=True
        assert client.bool("dark-mode", ctx) is True

        # Simulate SSE event arriving
        event = FlagChangeEvent(
            type="flag.state_changed",
            project="test-project",
            environment="production",
            flag_key="dark-mode",
            enabled=False,
            occurred_at="2026-03-23T11:00:00Z",
        )
        client._apply_event(event)

        # After SSE event: enabled=False
        assert client.bool("dark-mode", ctx) is False

        # Other fields are unchanged from bootstrap value
        result = client.evaluate("dark-mode", ctx)
        assert result.variant == "true"
        assert result.reason == "rule_match"
        assert result.evaluated_at == "2026-03-23T10:00:00Z"
    finally:
        client.close()


# ---------------------------------------------------------------------------
# @happy — bool/string return cached values without HTTP call
# ---------------------------------------------------------------------------

def test_happy_bool_string_return_cached_no_http():
    """@happy: bool() and string() read from cache; no HTTP call for known keys."""
    flags = {
        "feature-x": _eval_result("feature-x", enabled=True, variant="true"),
        "theme": _eval_result("theme", enabled=True, variant="blue"),
    }
    client, fake_inner, conn = _build_cached_client(flags)
    ctx = EvalContext(user_id="u1")

    try:
        assert client.bool("feature-x", ctx) is True
        assert client.string("theme", ctx) == "blue"

        # No live HTTP calls made for cached keys
        assert fake_inner.bool_calls == []
        assert fake_inner.string_calls == []
        assert fake_inner.evaluate_calls == []
        # evaluate_all called exactly once (bootstrap)
        assert fake_inner.evaluate_all_call_count == 1
    finally:
        client.close()


# ---------------------------------------------------------------------------
# @edge — unknown flag key falls back to live HTTP, not cached
# ---------------------------------------------------------------------------

def test_edge_unknown_key_falls_back_to_live_not_cached():
    """@edge: unknown key calls inner client once; result is not stored in cache."""
    flags = {
        "feature-x": _eval_result("feature-x", enabled=True, variant="true"),
    }
    # Add "new-flag" as known to the fake inner, but NOT in bootstrap flags
    fake_inner = _FakeInnerClient({
        "feature-x": _eval_result("feature-x", enabled=True, variant="true"),
        "new-flag": _eval_result("new-flag", enabled=False, variant="false"),
    })
    mock_conn = _make_mock_stream_connection()

    # Bootstrap only sees "feature-x"
    fake_inner_bootstrap_flags = {"feature-x": _eval_result("feature-x")}

    class _BootstrapOnlyFake(_FakeInnerClient):
        def evaluate_all(self, ctx: EvalContext) -> Dict[str, EvalResult]:
            self.evaluate_all_call_count += 1
            return dict(fake_inner_bootstrap_flags)

    fake_inner_b = _BootstrapOnlyFake({
        "feature-x": _eval_result("feature-x"),
        "new-flag": _eval_result("new-flag", enabled=False, variant="false"),
    })

    with patch("cuttlegate.cached.CuttlegateClient", return_value=fake_inner_b), \
         patch("cuttlegate.cached.connect_stream", return_value=mock_conn):
        client = CachedClient(_config())

    ctx = EvalContext(user_id="u1")
    try:
        # Call bool() for an unknown key
        result = client.bool("new-flag", ctx)
        assert result is False

        # The fallback went to the inner client
        assert "new-flag" in fake_inner_b.bool_calls

        # "new-flag" was NOT added to cache
        cache_snapshot = client.evaluate_all(ctx)
        assert "new-flag" not in cache_snapshot
    finally:
        client.close()


# ---------------------------------------------------------------------------
# @edge — concurrent reads are consistent
# ---------------------------------------------------------------------------

def test_edge_concurrent_reads_consistent():
    """@edge: concurrent reads and a background SSE update produce no torn reads."""
    n_flags = 10
    flags = {
        f"flag-{i}": _eval_result(f"flag-{i}", enabled=True, variant="true")
        for i in range(n_flags)
    }
    client, fake_inner, conn = _build_cached_client(flags)
    ctx = EvalContext(user_id="u1")

    errors: List[Exception] = []
    barrier = threading.Barrier(21)  # 20 readers + 1 writer

    def reader() -> None:
        try:
            barrier.wait()
            for i in range(n_flags):
                val = client.bool(f"flag-{i}", ctx)
                assert isinstance(val, bool), f"flag-{i} returned non-bool: {val!r}"
        except Exception as exc:
            errors.append(exc)

    def writer() -> None:
        barrier.wait()
        # Fire a rapid sequence of SSE events
        for i in range(n_flags):
            event = FlagChangeEvent(
                type="flag.state_changed",
                project="test-project",
                environment="production",
                flag_key=f"flag-{i}",
                enabled=False,
                occurred_at="2026-03-23T11:00:00Z",
            )
            client._apply_event(event)

    threads = [threading.Thread(target=reader) for _ in range(20)]
    threads.append(threading.Thread(target=writer))
    for t in threads:
        t.start()
    for t in threads:
        t.join(timeout=10)

    try:
        assert not errors, f"Thread errors: {errors}"
        # After all SSE events applied, all flags should be enabled=False
        for i in range(n_flags):
            assert client.bool(f"flag-{i}", ctx) is False, f"flag-{i} not updated"
    finally:
        client.close()


# ---------------------------------------------------------------------------
# @edge — close() stops SSE thread cleanly
# ---------------------------------------------------------------------------

def test_edge_close_stops_thread():
    """@edge: close() stops the SSE thread within 2 seconds."""
    flags = {
        "feature-x": _eval_result("feature-x"),
    }

    # Use a real threading setup so is_alive() is meaningful
    stop_event = threading.Event()
    thread = threading.Thread(target=lambda: stop_event.wait(), daemon=True)
    thread.start()
    real_conn = StreamConnection(thread=thread, stop_event=stop_event)

    fake_inner = _FakeInnerClient(flags)

    with patch("cuttlegate.cached.CuttlegateClient", return_value=fake_inner), \
         patch("cuttlegate.cached.connect_stream", return_value=real_conn):
        client = CachedClient(_config())

    assert client.is_alive() is True

    client.close()

    # Thread should stop within 2 seconds
    thread.join(timeout=2.0)
    assert not thread.is_alive(), "SSE thread did not stop within 2 seconds"
    assert client.is_alive() is False


# ---------------------------------------------------------------------------
# @error-path — server unavailable on bootstrap raises SDKError, no partial state
# ---------------------------------------------------------------------------

def test_error_path_bootstrap_failure_raises_sdk_error():
    """@error-path: SDKError on bootstrap propagates; no cache stored, no thread started."""
    network_error = SDKError("cuttlegate: request failed: ConnectError")
    fake_inner = _FakeInnerClient({}, raise_on_evaluate_all=network_error)
    mock_conn = _make_mock_stream_connection()

    stream_started = False

    def mock_connect_stream(*args, **kwargs):
        nonlocal stream_started
        stream_started = True
        return mock_conn

    with patch("cuttlegate.cached.CuttlegateClient", return_value=fake_inner), \
         patch("cuttlegate.cached.connect_stream", side_effect=mock_connect_stream):
        with pytest.raises(SDKError):
            CachedClient(_config())

    # No SSE thread should have been started
    assert not stream_started, "SSE thread was started despite bootstrap failure"


def test_error_path_bootstrap_auth_failure_raises_auth_error():
    """@error-path: AuthError on bootstrap propagates as AuthError (SDKError subclass)."""
    auth_error = AuthError(401)
    fake_inner = _FakeInnerClient({}, raise_on_evaluate_all=auth_error)
    stream_started = False

    def mock_connect_stream(*args, **kwargs):
        nonlocal stream_started
        stream_started = True
        return _make_mock_stream_connection()

    with patch("cuttlegate.cached.CuttlegateClient", return_value=fake_inner), \
         patch("cuttlegate.cached.connect_stream", side_effect=mock_connect_stream):
        with pytest.raises(AuthError) as exc_info:
            CachedClient(_config())

    assert exc_info.value.status_code == 401
    assert "authentication failed: HTTP 401" in str(exc_info.value)
    assert not stream_started, "SSE thread was started despite auth failure"


# ---------------------------------------------------------------------------
# Structural: CachedClient satisfies CuttlegateClientProtocol
# ---------------------------------------------------------------------------

def test_structural_protocol_satisfied():
    """CachedClient structurally satisfies CuttlegateClientProtocol (duck-type check).

    CuttlegateClientProtocol is not @runtime_checkable, so we verify the four
    required methods are present and callable rather than using isinstance().
    """
    flags = {"k": _eval_result("k")}
    client, _, _ = _build_cached_client(flags)
    try:
        for method_name in ("bool", "string", "evaluate", "evaluate_all"):
            assert callable(getattr(client, method_name, None)), (
                f"CachedClient is missing required protocol method: {method_name}"
            )
    finally:
        client.close()


# ---------------------------------------------------------------------------
# FlagStore integration
# ---------------------------------------------------------------------------

def test_store_save_called_after_bootstrap():
    """Store.save() is called after successful bootstrap."""
    class RecordingStore:
        def __init__(self):
            self.saved = []
        def save(self, flags):
            self.saved.append(dict(flags))
        def load(self):
            return {}

    store = RecordingStore()
    flags = {"dark-mode": _eval_result("dark-mode", enabled=True)}
    fake_inner = _FakeInnerClient(flags)
    mock_conn = _make_mock_stream_connection()

    with patch("cuttlegate.cached.CuttlegateClient", return_value=fake_inner), \
         patch("cuttlegate.cached.connect_stream", return_value=mock_conn):
        client = CachedClient(_config(), store=store)

    try:
        assert len(store.saved) == 1
        assert "dark-mode" in store.saved[0]
    finally:
        client.close()


def test_store_load_fallback_on_bootstrap_failure():
    """When bootstrap fails, Store.load() is used as fallback."""
    class FallbackStore:
        def save(self, flags):
            pass
        def load(self):
            return {"dark-mode": _eval_result("dark-mode", enabled=True, variant="true")}

    store = FallbackStore()
    fake_inner = _FakeInnerClient({}, raise_on_evaluate_all=SDKError("connection refused"))
    mock_conn = _make_mock_stream_connection()

    with patch("cuttlegate.cached.CuttlegateClient", return_value=fake_inner), \
         patch("cuttlegate.cached.connect_stream", return_value=mock_conn):
        client = CachedClient(_config(), store=store)

    ctx = EvalContext(user_id="u1")
    try:
        assert client.bool("dark-mode", ctx) is True
    finally:
        client.close()


def test_store_load_empty_propagates_original_error():
    """When Store.load() returns empty, original bootstrap error propagates."""
    class EmptyStore:
        def save(self, flags):
            pass
        def load(self):
            return {}

    store = EmptyStore()
    fake_inner = _FakeInnerClient({}, raise_on_evaluate_all=SDKError("connection refused"))

    with patch("cuttlegate.cached.CuttlegateClient", return_value=fake_inner), \
         patch("cuttlegate.cached.connect_stream"):
        with pytest.raises(SDKError):
            CachedClient(_config(), store=store)


def test_noop_flag_store():
    """NoopFlagStore.save is a no-op and load returns empty dict."""
    from cuttlegate.types import NoopFlagStore
    store = NoopFlagStore()
    store.save({"k": _eval_result("k")})
    result = store.load()
    assert result == {}


def test_store_save_called_on_sse_event():
    """Store.save() is called after an SSE event updates the cache."""
    class RecordingStore:
        def __init__(self):
            self.save_count = 0
        def save(self, flags):
            self.save_count += 1
        def load(self):
            return {}

    store = RecordingStore()
    flags = {"dark-mode": _eval_result("dark-mode", enabled=True)}
    fake_inner = _FakeInnerClient(flags)
    mock_conn = _make_mock_stream_connection()

    with patch("cuttlegate.cached.CuttlegateClient", return_value=fake_inner), \
         patch("cuttlegate.cached.connect_stream", return_value=mock_conn):
        client = CachedClient(_config(), store=store)

    try:
        saves_before = store.save_count
        event = FlagChangeEvent(
            type="flag.state_changed",
            project="test-project",
            environment="production",
            flag_key="dark-mode",
            enabled=False,
            occurred_at="2026-03-23T11:00:00Z",
        )
        client._apply_event(event)
        assert store.save_count > saves_before
    finally:
        client.close()
