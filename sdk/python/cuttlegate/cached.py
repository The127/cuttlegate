"""CachedClient — in-memory flag cache backed by a single SSE connection.

Owns:
- ``CachedClient`` class — bootstraps from evaluate_all, keeps cache fresh via SSE

Does not own:
- Config validation (delegates to CuttlegateClient and connect_stream)
- HTTP transport (CuttlegateClient)
- SSE connection lifecycle (connect_stream / StreamConnection)
- Error types (errors.py)

Start here: ``CachedClient.__init__`` — calls bootstrap() synchronously, then starts
the SSE background thread. Raises SDKError (or subclass) on bootstrap failure;
no thread is started and no partial state is stored.
"""

from __future__ import annotations

import threading
from copy import copy
from typing import Optional

from .client import CuttlegateClient
from .errors import SDKError
from .streaming import FlagChangeEvent, StreamConnection, connect_stream
from .types import CuttlegateConfig, EvalContext, EvalResult


class CachedClient:
    """In-memory flag cache backed by a single background SSE connection.

    On construction, ``evaluate_all`` is called synchronously to seed the cache.
    A single daemon SSE thread is then started to apply ``flag.state_changed``
    events as they arrive.

    ``bool()`` and ``string()`` read from the cache without making an HTTP call
    for known flag keys. Unknown keys fall back to a live HTTP call via the
    inner ``CuttlegateClient``; the result is **not** stored in the cache.

    Call ``close()`` to stop the background SSE thread. ``close()`` is
    non-blocking — it signals the thread and returns immediately.

    Raises:
        SDKError (or subclass): if ``evaluate_all`` fails during construction.
            No cache is stored and no background thread is started.
    """

    def __init__(self, config: CuttlegateConfig) -> None:
        self._config = config
        self._inner = CuttlegateClient(config)
        self._lock = threading.Lock()
        # _cache is populated by bootstrap(); until then it is empty.
        self._cache: dict[str, EvalResult] = {}
        self._stream: Optional[StreamConnection] = None

        # Bootstrap is all-or-nothing: if it raises, no thread is started.
        self._bootstrap()

    # ------------------------------------------------------------------
    # Public API — matches CuttlegateClientProtocol
    # ------------------------------------------------------------------

    def bool(self, key: str, ctx: EvalContext) -> bool:
        """Return the cached enabled value for key, or fall back to live HTTP.

        Returns:
            True if the flag is enabled (cached or live). Falls back to the
            inner client for unknown keys; result is not cached.

        Raises:
            FlagNotFoundError: if the key is absent from both cache and server.
            AuthError: on HTTP 401/403 (live fallback path only).
            SDKError: on network or server error (live fallback path only).
        """
        with self._lock:
            result = self._cache.get(key)
        if result is not None:
            return result.enabled
        # Unknown key — fall back to live HTTP; do not cache the result.
        return self._inner.bool(key, ctx)

    def string(self, key: str, ctx: EvalContext) -> str:
        """Return the cached variant string for key, or fall back to live HTTP.

        Returns:
            The flag variant string (cached or live). Falls back to the inner
            client for unknown keys; result is not cached.

        Raises:
            FlagNotFoundError: if the key is absent from both cache and server.
            AuthError: on HTTP 401/403 (live fallback path only).
            SDKError: on network or server error (live fallback path only).
        """
        with self._lock:
            result = self._cache.get(key)
        if result is not None:
            return result.variant
        return self._inner.string(key, ctx)

    def evaluate(self, key: str, ctx: EvalContext) -> EvalResult:
        """Return the cached EvalResult for key, or fall back to live HTTP.

        Note: the cached EvalResult has ``enabled`` kept current via SSE events,
        but ``variant``, ``reason``, and ``evaluated_at`` are from bootstrap time.
        For a fully fresh result, call the inner ``CuttlegateClient.evaluate()``
        directly.

        Raises:
            FlagNotFoundError: if the key is absent from both cache and server.
            AuthError: on HTTP 401/403 (live fallback path only).
            SDKError: on network or server error (live fallback path only).
        """
        with self._lock:
            result = self._cache.get(key)
        if result is not None:
            # Return a copy so callers cannot mutate the cached state.
            return copy(result)
        return self._inner.evaluate(key, ctx)

    def evaluate_all(self, ctx: EvalContext) -> dict[str, EvalResult]:
        """Return a snapshot copy of the full cache. ``ctx`` is ignored.

        Thread-safe: takes the lock and copies under it.

        Returns:
            dict mapping flag key → EvalResult for all bootstrapped flags.
        """
        with self._lock:
            return {k: copy(v) for k, v in self._cache.items()}

    def close(self) -> None:
        """Signal the background SSE thread to stop.

        Non-blocking — signals the stop event and returns immediately.
        The thread is a daemon, so process exit is not blocked if close()
        is not called.
        """
        if self._stream is not None:
            self._stream.close()

    def is_alive(self) -> bool:
        """Return True if the background SSE thread is still running."""
        if self._stream is None:
            return False
        return self._stream.is_alive()

    # ------------------------------------------------------------------
    # Internal
    # ------------------------------------------------------------------

    def _bootstrap(self) -> None:
        """Seed the cache from evaluate_all and start the SSE thread.

        All-or-nothing: if evaluate_all raises, the exception propagates
        directly and no SSE thread is started.
        """
        # EvalContext is required by the protocol; the bootstrap call uses a
        # sentinel user_id — the bulk endpoint returns canonical flag state.
        seed_ctx = EvalContext(user_id="__cuttlegate_bootstrap__")
        # May raise SDKError, AuthError, etc. — propagates directly.
        results = self._inner.evaluate_all(seed_ctx)

        with self._lock:
            self._cache = results

        # Cache is seeded — now start the SSE background thread.
        self._stream = connect_stream(
            self._config,
            on_change=self._apply_event,
        )

    def _apply_event(self, event: FlagChangeEvent) -> None:
        """Update the cache entry for event.flag_key under the lock.

        Only ``enabled`` is updated from SSE events. ``variant``, ``reason``,
        and ``evaluated_at`` are preserved from the bootstrap value.
        If the flag key is not in the cache (added after bootstrap), the
        event is silently ignored — unknown-key fallback is on-demand only.
        """
        with self._lock:
            existing = self._cache.get(event.flag_key)
            if existing is not None:
                # Replace the EvalResult with an updated copy.
                self._cache[event.flag_key] = EvalResult(
                    key=existing.key,
                    enabled=event.enabled,
                    variant=existing.variant,
                    reason=existing.reason,
                    evaluated_at=existing.evaluated_at,
                    value=existing.value,
                )
