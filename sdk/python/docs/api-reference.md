# Cuttlegate Python SDK — API Reference

All public names are importable from the top-level `cuttlegate` package:

```python
from cuttlegate import (
    CuttlegateClient, CachedClient, MockCuttlegateClient,
    CuttlegateClientProtocol, CuttlegateConfig, EvalContext, EvalResult,
    connect_stream, StreamConnection, FlagChangeEvent,
    ConfigError, AuthError, FlagNotFoundError, NotFoundError,
    ServerError, InvalidResponseError, SDKError,
)
```

---

## Configuration

### `CuttlegateConfig`

```python
@dataclass
class CuttlegateConfig:
    api_key: str          # repr=False — never logged
    server_url: str       # must be http or https URL
    project: str
    environment: str
    timeout_ms: int = 10_000
```

| Field | Required | Default | Notes |
|---|---|---|---|
| `api_key` | yes | — | Suppressed in `repr()` and tracebacks. Source from env var or secrets manager. |
| `server_url` | yes | — | Must be an `http://` or `https://` URL. Trailing slash is stripped. |
| `project` | yes | — | Project slug. |
| `environment` | yes | — | Environment slug (e.g. `"production"`). |
| `timeout_ms` | no | `10_000` | HTTP timeout in milliseconds for evaluation requests. Does not apply to SSE connections. |

---

## Evaluation context and result

### `EvalContext`

```python
@dataclass
class EvalContext:
    user_id: str
    attributes: dict[str, Any] = field(default_factory=dict)
```

| Field | Type | Notes |
|---|---|---|
| `user_id` | `str` | User identifier. Empty string is valid but will not match user-specific targeting rules. |
| `attributes` | `dict[str, Any]` | Arbitrary key-value pairs for targeting rules. Values must be JSON-serialisable (str, int, float, bool, None). |

### `EvalResult`

```python
@dataclass
class EvalResult:
    key: str
    enabled: bool
    variant: str          # primary — maps from JSON value_key
    reason: str
    evaluated_at: str
    value: str = ""       # deprecated — use variant; always "" on wire responses
```

| Field | Type | Notes |
|---|---|---|
| `key` | `str` | Flag key. |
| `enabled` | `bool` | Whether the flag is enabled for the evaluated context. |
| `variant` | `str` | **Primary field.** `"true"` or `"false"` for bool flags; variant key string for all others. Maps from JSON `value_key`. |
| `reason` | `str` | `"rule_match"`, `"default"`, `"disabled"`, or `"percentage_rollout"`. |
| `evaluated_at` | `str` | ISO 8601 evaluation timestamp. |
| `value` | `str` | **Deprecated.** Always `""` for wire responses (JSON `null` coerced to `""`). Use `variant`. |

---

## CuttlegateClient

Synchronous flag evaluation client backed by `httpx`. No network call at construction time.

```python
class CuttlegateClient:
    def __init__(self, config: CuttlegateConfig) -> None: ...
    def evaluate_all(self, ctx: EvalContext) -> dict[str, EvalResult]: ...
    def evaluate(self, key: str, ctx: EvalContext) -> EvalResult: ...
    def bool(self, key: str, ctx: EvalContext) -> bool: ...
    def string(self, key: str, ctx: EvalContext) -> str: ...
```

### `__init__(config)`

Validates config synchronously. Raises `ConfigError` on any invalid or missing field. No network call is made.

### `evaluate_all(ctx) -> dict[str, EvalResult]`

Evaluate all flags for `ctx`. Makes one HTTP POST request regardless of flag count.

Prefer this method when evaluating multiple flags in the same operation.

**Raises:**
- `ValueError` — if `ctx` is `None`
- `AuthError` — on HTTP 401 or 403
- `SDKError` — on HTTP 404 (project/environment not found), 5xx, network error, or malformed response

### `evaluate(key, ctx) -> EvalResult`

Evaluate a single flag by key. Calls `evaluate_all` internally — does not use a per-flag endpoint.

**Raises:**
- `ValueError` — if `ctx` is `None`
- `FlagNotFoundError` — if `key` is absent from the 200 response
- `AuthError` — on HTTP 401 or 403
- `SDKError` — on HTTP 404, 5xx, network error, or malformed response

### `bool(key, ctx) -> bool`

Returns `True` if `result.variant == "true"`. Calls `evaluate` internally.

**Raises:** same as `evaluate`.

### `string(key, ctx) -> str`

Returns `result.variant`. Calls `evaluate` internally.

**Raises:** same as `evaluate`.

---

## CachedClient

In-memory flag cache backed by a single background SSE connection. Implements `CuttlegateClientProtocol`.

Construction blocks on `evaluate_all` — construct at application startup, not inside a request handler.

```python
class CachedClient:
    def __init__(self, config: CuttlegateConfig) -> None: ...
    def bool(self, key: str, ctx: EvalContext) -> bool: ...
    def string(self, key: str, ctx: EvalContext) -> str: ...
    def evaluate(self, key: str, ctx: EvalContext) -> EvalResult: ...
    def evaluate_all(self, ctx: EvalContext) -> dict[str, EvalResult]: ...
    def close(self) -> None: ...
    def is_alive(self) -> bool: ...
```

### `__init__(config)`

Seeds the cache by calling `evaluate_all` synchronously. If the seed call raises, the exception propagates directly — no cache is stored and no background thread is started.

After seeding, starts one background daemon thread managing one SSE connection for all flags.

**Raises:** `SDKError` (or subclass) — if `evaluate_all` fails during construction.

### `bool(key, ctx) -> bool`

Returns the cached enabled state for `key`. If the key is not in the cache (added after construction), falls back to a live HTTP call via the inner `CuttlegateClient`. The fallback result is not stored in the cache.

**Raises:**
- `FlagNotFoundError` — if `key` is absent from both cache and server
- `AuthError` — on HTTP 401/403 (live fallback path only)
- `SDKError` — on network or server error (live fallback path only)

### `string(key, ctx) -> str`

Returns the cached variant string for `key`. Falls back to live HTTP for unknown keys.

**Raises:** same as `bool`.

### `evaluate(key, ctx) -> EvalResult`

Returns a copy of the cached `EvalResult` for `key`. Falls back to live HTTP for unknown keys.

Note: only `enabled` is kept current via SSE events. `variant`, `reason`, and `evaluated_at` are from bootstrap time. For a fully fresh result, use `CuttlegateClient.evaluate()` directly.

**Raises:** same as `bool`.

### `evaluate_all(ctx) -> dict[str, EvalResult]`

Returns a snapshot copy of the full cache. `ctx` is ignored — the cache is not user-specific. Thread-safe.

### `close() -> None`

Signals the background SSE thread to stop. Non-blocking — returns immediately. The thread is a daemon, so process exit is not blocked if `close()` is not called.

### `is_alive() -> bool`

Returns `True` if the background SSE thread is still running.

---

## Streaming

### `connect_stream`

```python
def connect_stream(
    config: CuttlegateConfig,
    on_change: Callable[[FlagChangeEvent], None],
    on_error: Optional[Callable[[Exception], None]] = None,
) -> StreamConnection: ...
```

Validates `config` synchronously and raises `ConfigError` immediately on invalid input. Starts a background daemon thread and returns a `StreamConnection` handle. Does not block on network.

**Parameters:**

| Parameter | Type | Notes |
|---|---|---|
| `config` | `CuttlegateConfig` | SDK configuration. |
| `on_change` | `Callable[[FlagChangeEvent], None]` | Called from the background thread on each `flag.state_changed` event. Thread safety of the callback is the caller's responsibility. |
| `on_error` | `Optional[Callable[[Exception], None]]` | Called on non-fatal errors (malformed events, server errors before reconnect) and terminal errors (auth failures). Errors are silently dropped if not provided. |

**Raises:** `ConfigError` — if any required config field is missing.

**Returns:** `StreamConnection`

### `StreamConnection`

```python
class StreamConnection:
    def close(self) -> None: ...
    def is_alive(self) -> bool: ...
```

Handle returned by `connect_stream`. The background thread is already running at construction time.

#### `close() -> None`

Signals the background thread to stop. Returns immediately — does not wait for the thread to exit.

#### `is_alive() -> bool`

Returns `True` if the background thread is still running.

### `FlagChangeEvent`

```python
@dataclass
class FlagChangeEvent:
    type: str           # always "flag.state_changed"
    project: str
    environment: str
    flag_key: str
    enabled: bool
    occurred_at: str    # ISO 8601 string; no datetime parsing applied
```

| Field | Type | Notes |
|---|---|---|
| `type` | `str` | Always `"flag.state_changed"`. |
| `project` | `str` | Project slug. |
| `environment` | `str` | Environment slug. |
| `flag_key` | `str` | The flag key that changed. |
| `enabled` | `bool` | New enabled state. |
| `occurred_at` | `str` | ISO 8601 timestamp as received from the server. No datetime parsing applied — parse with `datetime.fromisoformat()` if needed. |

---

## Protocol

### `CuttlegateClientProtocol`

PEP 544 structural protocol. Type-hint consumer variables as `CuttlegateClientProtocol` to accept `CuttlegateClient`, `CachedClient`, or `MockCuttlegateClient` without subclassing.

```python
class CuttlegateClientProtocol(Protocol):
    def bool(self, key: str, ctx: EvalContext) -> bool: ...
    def string(self, key: str, ctx: EvalContext) -> str: ...
    def evaluate(self, key: str, ctx: EvalContext) -> EvalResult: ...
    def evaluate_all(self, ctx: EvalContext) -> dict[str, EvalResult]: ...
```

---

## Testing

### `MockCuttlegateClient`

In-memory implementation of `CuttlegateClientProtocol`. No live server required. Ships as part of the `cuttlegate` distribution — no separate test dependency needed.

```python
class MockCuttlegateClient:
    def __init__(self, flags: dict[str, Any] | None = None) -> None: ...
    def bool(self, key: str, ctx: EvalContext) -> bool: ...
    def string(self, key: str, ctx: EvalContext) -> str: ...
    def evaluate(self, key: str, ctx: EvalContext) -> EvalResult: ...
    def evaluate_all(self, ctx: EvalContext) -> dict[str, EvalResult]: ...
    def set_flag(self, key: str, value: Any) -> None: ...
    def assert_evaluated(self, key: str) -> None: ...
    def assert_not_evaluated(self, key: str) -> None: ...
    def reset(self) -> None: ...
```

#### Construction

```python
mock = MockCuttlegateClient(flags={
    "dark-mode": True,          # bool flag — bool() returns True
    "color-theme": "blue",      # string flag — string() returns "blue"
})
```

Values are `bool` or `str`. `True` and `"true"` both produce `bool() == True`.

#### `set_flag(key, value) -> None`

Add or update a flag. Useful within a single test when flag state needs to change.

#### `assert_evaluated(key) -> None`

Raises `AssertionError` if the flag was not evaluated during the test. The error message includes the flag key.

#### `assert_not_evaluated(key) -> None`

Raises `AssertionError` if the flag was evaluated during the test.

#### `reset() -> None`

Clears all flag values and evaluation history. Use between test cases when reusing a mock instance.

---

## Errors

All error types are importable from the top-level `cuttlegate` package.

| Class | Base | When raised | Key attributes |
|---|---|---|---|
| `SDKError` | `Exception` | Base class for all SDK-specific errors | — |
| `ConfigError` | `ValueError` | Missing or invalid config at construction | — |
| `AuthError` | `SDKError` | HTTP 401 or 403 | `.status_code: int` |
| `FlagNotFoundError` | `SDKError` | Flag key absent from a 200 response | `.key: str` |
| `NotFoundError` | — | Deprecated alias for `FlagNotFoundError` | — |
| `ServerError` | `SDKError` | HTTP 5xx | `.status_code: int` |
| `InvalidResponseError` | `SDKError` | Malformed JSON or unexpected SSE event shape | `.message: str` |

`ConfigError` inherits from `ValueError`, not `SDKError`. Catch it separately from network-related errors.

`NotFoundError` is a backward-compatible alias for `FlagNotFoundError`. Prefer `FlagNotFoundError` in new code.

### Catching errors

```python
from cuttlegate import AuthError, FlagNotFoundError, SDKError

try:
    result = client.evaluate("my-flag", ctx)
except FlagNotFoundError as exc:
    # exc.key contains the missing flag key
    print(f"flag {exc.key!r} not found")
except AuthError as exc:
    # exc.status_code is 401 or 403
    print(f"auth failed: HTTP {exc.status_code}")
except SDKError as exc:
    # Catches ServerError, InvalidResponseError, and other SDK errors
    print(f"sdk error: {exc}")
```
