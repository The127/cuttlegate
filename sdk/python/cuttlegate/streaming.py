"""SSE streaming client for Cuttlegate flag state changes.

Owns:
- ``FlagChangeEvent`` dataclass — the six locked wire fields
- ``StreamConnection`` handle — close() and is_alive()
- ``connect_stream()`` module-level function — starts a background daemon thread

Does not own:
- Config validation (delegates to _validate_stream_config below)
- Error types (lives in errors.py)
- Flag evaluation (lives in client.py)

Start here: ``connect_stream()`` — returns immediately with a StreamConnection.
"""

from __future__ import annotations

import json
import logging
import random
import threading
from dataclasses import dataclass
from typing import Callable, Optional, Tuple

import httpx

from .errors import AuthError, ConfigError, InvalidResponseError, ServerError
from .types import CuttlegateConfig

_log = logging.getLogger(__name__)

_INITIAL_DELAY_S = 1.0
_MAX_DELAY_S = 30.0


@dataclass
class FlagChangeEvent:
    """A flag state change event received from the SSE stream.

    All six fields match the locked ``flag.state_changed`` wire format
    exactly. ``occurred_at`` is an ISO 8601 string as received from the
    server — no datetime parsing is applied; the caller decides whether
    to parse it.

    Instances are constructed by the background thread and delivered to
    ``on_change``. Thread safety of the callback is the caller's
    responsibility.
    """

    type: str           # always "flag.state_changed"
    project: str
    environment: str
    flag_key: str
    enabled: bool
    occurred_at: str    # ISO 8601 string


class StreamConnection:
    """Handle to an active SSE stream connection.

    Returned immediately by ``connect_stream()``. The background daemon
    thread is already running at construction time.

    Call ``close()`` to stop the thread cleanly. Safe to call before the
    first connection is established — the thread exits on its next loop
    iteration.
    """

    def __init__(self, thread: threading.Thread, stop_event: threading.Event) -> None:
        self._thread = thread
        self._stop_event = stop_event

    def close(self) -> None:
        """Signal the background thread to stop and close the SSE connection.

        Returns immediately — does not wait for the thread to exit. The
        thread is a daemon, so the process can exit regardless.
        """
        self._stop_event.set()

    def is_alive(self) -> bool:
        """Return True if the background thread is still running."""
        return self._thread.is_alive()


def connect_stream(
    config: CuttlegateConfig,
    on_change: Callable[[FlagChangeEvent], None],
    on_error: Optional[Callable[[Exception], None]] = None,
) -> StreamConnection:
    """Connect to the SSE flag state stream and return a StreamConnection handle.

    Validates ``config`` synchronously — raises ``ConfigError`` immediately
    if any required field is missing or invalid. Does not make a network
    connection at call time.

    Starts a background daemon thread that opens the SSE connection and
    delivers events. Returns the ``StreamConnection`` handle immediately.

    Args:
        config: SDK configuration. Must have non-empty ``api_key``,
            ``server_url``, ``project``, and ``environment``.
        on_change: Called from the background thread for each
            ``flag.state_changed`` event. Thread safety is the caller's
            responsibility.
        on_error: Optional callback for non-fatal errors (malformed JSON,
            server errors before reconnect) and terminal errors (auth
            failures). If not provided, errors are silently dropped.

    Returns:
        A ``StreamConnection`` handle. Call ``.close()`` to stop the stream.

    Raises:
        ConfigError: if ``config`` is missing a required field.
    """
    _validate_stream_config(config)

    stop_event = threading.Event()
    thread = threading.Thread(
        target=_stream_loop,
        args=(config, on_change, on_error, stop_event),
        daemon=True,
        name="cuttlegate-sse",
    )
    thread.start()
    return StreamConnection(thread, stop_event)


# ---------------------------------------------------------------------------
# Internal — stream loop
# ---------------------------------------------------------------------------

def _stream_loop(
    config: CuttlegateConfig,
    on_change: Callable[[FlagChangeEvent], None],
    on_error: Optional[Callable[[Exception], None]],
    stop_event: threading.Event,
) -> None:
    """Background thread: connect, read SSE, reconnect on transient failure.

    Reconnect strategy: exponential backoff with full jitter.
    Attempt counter resets to 0 after each successful connection so that
    a flapping server doesn't produce 30-second delays after recovery.
    """
    url = (
        f"{config.server_url.rstrip('/')}"
        f"/api/v1/projects/{config.project}"
        f"/environments/{config.environment}/flags/stream"
    )
    headers = {
        "Authorization": f"Bearer {config.api_key}",
        "Accept": "text/event-stream",
    }
    attempt = 0

    while not stop_event.is_set():
        if attempt > 0:
            delay = _backoff_delay(attempt)
            # stop_event.wait(timeout) returns True when the event is set.
            # Use it instead of time.sleep so close() unblocks immediately.
            if stop_event.wait(timeout=delay):
                break

        if stop_event.is_set():
            break

        terminal, connected = _attempt_connection(
            url, headers, on_change, on_error, stop_event
        )
        if terminal:
            break

        # Reset attempt counter after a successful connection — the server
        # recovered, so start backoff from the beginning again.
        attempt = 0 if connected else attempt + 1


def _attempt_connection(
    url: str,
    headers: dict,
    on_change: Callable[[FlagChangeEvent], None],
    on_error: Optional[Callable[[Exception], None]],
    stop_event: threading.Event,
) -> Tuple[bool, bool]:
    """Open one SSE connection and read until it closes or an error occurs.

    Returns:
        (terminal, connected) — ``terminal`` True means stop retrying;
        ``connected`` True means a 200 was received and events were read
        (use to reset the backoff attempt counter).
    """
    try:
        with httpx.stream(
            "GET",
            url,
            headers=headers,
            timeout=None,
        ) as resp:
            if resp.status_code in (401, 403):
                _notify_error(on_error, AuthError(resp.status_code))
                return True, False  # terminal

            if resp.status_code >= 500:
                _notify_error(on_error, ServerError(resp.status_code))
                return False, False  # transient

            if resp.status_code != 200:
                _log.warning("cuttlegate: unexpected SSE status %s", resp.status_code)
                return False, False  # transient

            _read_sse(resp, on_change, on_error, stop_event)
            return False, True  # connected successfully; reconnect if needed

    except (httpx.TransportError, httpx.TimeoutException) as exc:
        _log.debug("cuttlegate: SSE transport error: %s", type(exc).__name__)

    return False, False  # transient


def _read_sse(
    resp: httpx.Response,
    on_change: Callable[[FlagChangeEvent], None],
    on_error: Optional[Callable[[Exception], None]],
    stop_event: threading.Event,
) -> None:
    """Iterate SSE lines from the response and dispatch events."""
    for line in resp.iter_lines():
        if stop_event.is_set():
            break

        # Skip keep-alive comments and blank lines.
        if line.startswith(":") or not line.strip():
            continue

        if line.startswith("data:"):
            raw = line[len("data:"):].strip()
            if not raw:
                continue
            _handle_data_line(raw, on_change, on_error)


def _handle_data_line(
    raw: str,
    on_change: Callable[[FlagChangeEvent], None],
    on_error: Optional[Callable[[Exception], None]],
) -> None:
    """Parse one SSE data line and call on_change or on_error as appropriate."""
    try:
        payload = json.loads(raw)
    except json.JSONDecodeError as exc:
        _notify_error(
            on_error,
            InvalidResponseError(f"SSE data is not valid JSON: {exc}"),
        )
        return

    if not isinstance(payload, dict):
        _notify_error(on_error, InvalidResponseError("SSE event is not a JSON object"))
        return

    event_type = payload.get("type")
    if event_type != "flag.state_changed":
        # Unknown event type — silently ignore for forward compatibility.
        return

    try:
        event = FlagChangeEvent(
            type=payload["type"],
            project=payload["project"],
            environment=payload["environment"],
            flag_key=payload["flag_key"],
            enabled=bool(payload["enabled"]),
            occurred_at=payload["occurred_at"],
        )
    except KeyError as exc:
        _notify_error(
            on_error,
            InvalidResponseError(f"SSE event missing field {exc}"),
        )
        return

    on_change(event)


# ---------------------------------------------------------------------------
# Internal — helpers
# ---------------------------------------------------------------------------

def _backoff_delay(attempt: int) -> float:
    """Exponential backoff with full jitter.

    delay = random() * min(initial * 2^attempt, max)
    """
    base = min(_INITIAL_DELAY_S * (2 ** attempt), _MAX_DELAY_S)
    return random.random() * base


def _notify_error(
    on_error: Optional[Callable[[Exception], None]],
    exc: Exception,
) -> None:
    """Call on_error if provided; otherwise drop silently."""
    if on_error is not None:
        try:
            on_error(exc)
        except Exception:
            _log.debug("cuttlegate: on_error callback raised", exc_info=True)


def _validate_stream_config(config: CuttlegateConfig) -> None:
    """Raise ConfigError if any field required for streaming is missing."""
    if not config.api_key:
        raise ConfigError("api_key is required")
    if not config.server_url:
        raise ConfigError("server_url is required")
    if not config.project:
        raise ConfigError("project is required")
    if not config.environment:
        raise ConfigError("environment is required")
