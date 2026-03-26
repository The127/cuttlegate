# Cuttlegate Python SDK

Python client for the [Cuttlegate](https://github.com/The127/cuttlegate) feature-flag service. Requires Python 3.11 or later.

## Install

```sh
pip install cuttlegate
```

For development (editable install from source):

```sh
git clone https://github.com/The127/cuttlegate
cd cuttlegate/sdk/python
pip install -e .
```

## Quick start

```python
import os
from cuttlegate import CuttlegateClient, CuttlegateConfig, EvalContext

client = CuttlegateClient(CuttlegateConfig(
    api_key=os.environ["CUTTLEGATE_API_KEY"],
    server_url="https://flags.example.com",
    project="my-project",
    environment="production",
))

ctx = EvalContext(user_id="user-123")
enabled = client.bool("dark-mode", ctx)
print("dark-mode enabled:", enabled)
```

`CuttlegateClient` validates config immediately and raises `ConfigError` on any missing field. No network call is made until the first evaluation method is called.

## Flag evaluation

### `evaluate_all` — bulk evaluation (prefer this for multiple flags)

`evaluate_all` makes one HTTP request regardless of how many flags exist. Use it when you need several flags for a single operation.

```python
ctx = EvalContext(user_id="user-123", attributes={"plan": "pro", "beta": True})
results = client.evaluate_all(ctx)

for key, result in results.items():
    print(f"{key}: enabled={result.enabled} variant={result.variant} reason={result.reason}")
```

### `evaluate` — single flag with full result

```python
result = client.evaluate("banner-text", ctx)
print("variant:", result.variant)   # e.g. "holiday"
print("reason:", result.reason)     # e.g. "rule_match"
print("enabled:", result.enabled)
```

Raises `FlagNotFoundError` if the key is absent from the response.

### `bool` — boolean flag convenience

```python
if client.bool("dark-mode", ctx):
    apply_dark_theme()
```

Returns `True` if the flag's variant is `"true"`. Raises `FlagNotFoundError` if the key is absent.

### `string` — string flag convenience

```python
theme = client.string("color-theme", ctx)  # e.g. "blue"
```

Returns the variant string. Raises `FlagNotFoundError` if the key is absent.

### EvalContext fields

| Field | Type | Notes |
|---|---|---|
| `user_id` | `str` | User identifier passed to targeting rules. |
| `attributes` | `dict[str, Any]` | Arbitrary key-value pairs for targeting rules. Values must be JSON-serialisable. |

### EvalResult fields

| Field | Type | Notes |
|---|---|---|
| `key` | `str` | Flag key |
| `enabled` | `bool` | Whether the flag is enabled for this context |
| `variant` | `str` | **Primary field.** `"true"` or `"false"` for bool flags; variant key string for all others. |
| `reason` | `str` | `"rule_match"`, `"default"`, `"disabled"`, or `"percentage_rollout"` |
| `evaluated_at` | `str` | ISO 8601 evaluation timestamp |
| `value` | `str` | **Deprecated.** Always `""` for wire responses. Use `variant`. |

## Real-time streaming

`connect_stream` opens a background SSE connection and calls `on_change` for every `flag.state_changed` event. It returns immediately with a `StreamConnection` handle.

```python
from cuttlegate import connect_stream, CuttlegateConfig, FlagChangeEvent

config = CuttlegateConfig(
    api_key=os.environ["CUTTLEGATE_API_KEY"],
    server_url="https://flags.example.com",
    project="my-project",
    environment="production",
)

def on_change(event: FlagChangeEvent) -> None:
    print(f"flag {event.flag_key} changed: enabled={event.enabled} at {event.occurred_at}")

def on_error(exc: Exception) -> None:
    print(f"stream error: {exc}")

stream = connect_stream(config, on_change=on_change, on_error=on_error)

# ... application runs ...

stream.close()  # signal the background thread to stop
```

`connect_stream` validates config synchronously and raises `ConfigError` immediately on invalid input. The network connection is made in the background thread — `connect_stream` does not block.

### FlagChangeEvent fields

| Field | Type | Notes |
|---|---|---|
| `type` | `str` | Always `"flag.state_changed"` |
| `project` | `str` | Project slug |
| `environment` | `str` | Environment slug |
| `flag_key` | `str` | The flag that changed |
| `enabled` | `bool` | New enabled state |
| `occurred_at` | `str` | ISO 8601 timestamp; no datetime parsing applied |

### Reconnection behaviour

The SDK reconnects automatically on transient failures (network errors, 5xx responses) using exponential backoff with full jitter:

- Initial delay: 1 second
- Maximum delay: 30 seconds
- Auth errors (401/403) are terminal — the background thread stops and calls `on_error`. Fix the credential and create a new `connect_stream` call.

`stream.close()` returns immediately. The background thread is a daemon, so process exit is not blocked if `close()` is not called.

## CachedClient — production use

For applications where flag evaluation is in the hot request path, use `CachedClient`. It seeds a local in-memory cache from `evaluate_all` on construction and keeps it current via a single background SSE connection. `bool()` and `string()` read from cache with no network round trip.

```python
from cuttlegate import CachedClient, CuttlegateConfig, EvalContext

config = CuttlegateConfig(
    api_key=os.environ["CUTTLEGATE_API_KEY"],
    server_url="https://flags.example.com",
    project="my-project",
    environment="production",
)

# Construction blocks on evaluate_all — do this at startup, not inside a request handler.
cache = CachedClient(config)

ctx = EvalContext(user_id="user-123")
enabled = cache.bool("dark-mode", ctx)   # cache hit — no HTTP call
theme = cache.string("color-theme", ctx) # cache hit — no HTTP call

# On shutdown:
cache.close()
```

**How it works:**

- `CachedClient(config)` calls `evaluate_all` synchronously to seed the cache, then starts one background SSE thread for all flags. Raises `SDKError` (or a subclass) if the seed call fails — no cache is stored and no thread is started.
- `bool()` and `string()` check the cache first. Keys added after construction fall back to a live HTTP call; the result is not stored in the cache.
- `close()` signals the background thread to stop. Non-blocking.
- `CachedClient` satisfies `CuttlegateClientProtocol` — inject it wherever a `CuttlegateClient` is expected.

**When to use CachedClient vs CuttlegateClient:**

| Scenario | Use |
|---|---|
| Hot request path, many flag checks per request | `CachedClient` |
| Low-volume scripts or batch jobs | `CuttlegateClient` |
| Tests | `MockCuttlegateClient` |

### Flask integration example

Construct `CachedClient` once at app-factory time, before `app.run()`. Do not construct it inside a request handler — the blocking `evaluate_all` call will add latency to the first request and the SSE thread will not be reused across requests.

```python
import os
from flask import Flask, g
from cuttlegate import CachedClient, CuttlegateConfig, EvalContext

config = CuttlegateConfig(
    api_key=os.environ["CUTTLEGATE_API_KEY"],
    server_url="https://flags.example.com",
    project="my-project",
    environment="production",
)

# Module-level — constructed once when the module is imported.
flags = CachedClient(config)

app = Flask(__name__)

@app.route("/")
def index():
    ctx = EvalContext(user_id=g.user_id, attributes={"plan": g.plan})
    if flags.bool("dark-mode", ctx):
        return render_dark()
    return render_light()
```

## Testing without a live server

Use `MockCuttlegateClient` to write unit tests without a running Cuttlegate instance. It implements `CuttlegateClientProtocol`, so it can substitute for `CuttlegateClient` or `CachedClient` in any type-hinted code.

```python
from cuttlegate import MockCuttlegateClient, CuttlegateClientProtocol, EvalContext

def send_welcome_email(client: CuttlegateClientProtocol, user_id: str) -> bool:
    ctx = EvalContext(user_id=user_id)
    return client.bool("welcome-email", ctx)


def test_send_welcome_email_when_flag_enabled():
    mock = MockCuttlegateClient(flags={"welcome-email": True})
    result = send_welcome_email(mock, user_id="u1")
    assert result is True
    mock.assert_evaluated("welcome-email")


def test_send_welcome_email_when_flag_disabled():
    mock = MockCuttlegateClient(flags={"welcome-email": False})
    result = send_welcome_email(mock, user_id="u1")
    assert result is False


def test_flag_mutation_with_set_flag():
    mock = MockCuttlegateClient()
    mock.set_flag("dark-mode", True)
    ctx = EvalContext(user_id="u1")
    assert mock.bool("dark-mode", ctx) is True

    mock.set_flag("dark-mode", False)
    assert mock.bool("dark-mode", ctx) is False


def test_reset_clears_state():
    mock = MockCuttlegateClient(flags={"my-flag": True})
    ctx = EvalContext(user_id="u1")
    mock.bool("my-flag", ctx)
    mock.reset()
    # After reset, flags are cleared and evaluation history is wiped.
    mock.assert_not_evaluated("my-flag")
```

### MockCuttlegateClient API

| Method | Notes |
|---|---|
| `__init__(flags=None)` | Pass `{"key": True}` for bool flags, `{"key": "variant"}` for string flags. |
| `bool(key, ctx)` | Returns `True` if value is `True` or `"true"`. Raises `FlagNotFoundError` for unknown keys. |
| `string(key, ctx)` | Returns the variant string. Raises `FlagNotFoundError` for unknown keys. |
| `evaluate(key, ctx)` | Returns `EvalResult(reason="mock")`. Raises `FlagNotFoundError` for unknown keys. |
| `evaluate_all(ctx)` | Returns all configured flags as `dict[str, EvalResult]`. Returns `{}` if empty. |
| `set_flag(key, value)` | Add or update a flag. Call between test cases when flag state changes. |
| `assert_evaluated(key)` | Raises `AssertionError` if the flag was not evaluated during the test. |
| `assert_not_evaluated(key)` | Raises `AssertionError` if the flag was evaluated during the test. |
| `reset()` | Clear all flag values and evaluation history. Use between test cases. |

## Error handling

All SDK errors are typed — catch the specific class rather than inspecting string messages.

```python
from cuttlegate import (
    CuttlegateClient,
    CuttlegateConfig,
    EvalContext,
    AuthError,
    FlagNotFoundError,
    SDKError,
)

client = CuttlegateClient(CuttlegateConfig(
    api_key=os.environ["CUTTLEGATE_API_KEY"],
    server_url="https://flags.example.com",
    project="my-project",
    environment="production",
))

try:
    result = client.evaluate("my-flag", EvalContext(user_id="u1"))
except FlagNotFoundError as exc:
    print(f"flag {exc.key!r} does not exist in this project")
except AuthError as exc:
    print(f"authentication failed: HTTP {exc.status_code}")
except SDKError as exc:
    print(f"sdk error: {exc}")
```

| Error | When raised | Key attributes |
|---|---|---|
| `ConfigError` | Missing or invalid field at construction | — |
| `AuthError` | 401 or 403 from the server | `.status_code` |
| `FlagNotFoundError` | Key absent from a 200 response | `.key` |
| `ServerError` | 5xx from the server | `.status_code` |
| `InvalidResponseError` | Malformed JSON or unexpected SSE event shape | `.message` |
| `SDKError` | Base class for all SDK-specific errors | — |

`ConfigError` inherits from `ValueError`, not `SDKError`, so it can be caught independently of network errors.

## Gotchas

**No async/await.** The Python SDK is synchronous. All evaluation methods make blocking HTTP calls. Async support is deferred to a future version. In ASGI applications (FastAPI, Starlette), run evaluation in a thread pool executor or use `CachedClient` to avoid blocking the event loop.

**Method names.** The methods are `bool()` and `string()` — not `get_bool()`/`get_string()`. This is intentional: the names match the Go and JS SDKs' naming convention and read naturally in most call sites.

**`CachedClient` blocks on construction.** `CachedClient(config)` calls `evaluate_all` synchronously before returning. Construct it at application startup (module level or app factory) — not inside a request handler.

**`api_key` is never logged.** `CuttlegateConfig` sets `repr=False` on `api_key` — it never appears in `repr()`, log output, or tracebacks. Always source it from an environment variable or secrets manager, never hardcode it.

**SSE thread lifecycle.** `connect_stream` starts a daemon thread. Daemon threads are killed when the main process exits — call `stream.close()` explicitly before shutdown if you need the thread to stop cleanly. `close()` is non-blocking.

**Cache staleness after new flags are added.** `CachedClient` only caches flags present at construction time. Flags added after startup fall back to a live HTTP call; the result is not stored in the cache. If this matters, restart the application to reseed the cache.

## `variant` field and `value` field deprecation

The primary result field is `variant` (maps from JSON `value_key`). The `value` field is **deprecated**:

```python
result = client.evaluate("my-flag", ctx)

# Correct — use variant:
print(result.variant)   # "true", "false", or a string variant key

# Deprecated — value is always "" on wire responses; do not use:
print(result.value)     # always ""
```

For bool flags, use `client.bool()` rather than checking `result.variant == "true"` manually.

This follows [ADR-0018](../../docs/adr/0018-eval-response-value-key-field.md): `value_key` is the v1 contract field; `variant` is its Python representation.

## API reference

See [`docs/api-reference.md`](docs/api-reference.md) for full method signatures, parameter types, and return types for all public classes.
