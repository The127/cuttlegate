---
sidebar_label: Python SDK
sidebar_position: 3
---

# Python SDK

The Cuttlegate Python SDK evaluates feature flags in your Python services. It requires Python 3.11 or later.

## Install

```bash
pip install cuttlegate
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

ctx = EvalContext(user_id="user-123", attributes={"plan": "pro"})

# Boolean flag
enabled = client.bool("dark-mode", ctx)
print("dark-mode enabled:", enabled)

# String variant
result = client.evaluate("banner-text", ctx)
print("variant:", result.variant)   # e.g. "holiday"
print("reason:", result.reason)     # e.g. "targeting_rule"
```

`CuttlegateClient` validates config immediately and raises `ConfigError` on any missing field. No network call is made at construction time.

## Evaluation methods

| Method | Returns | Notes |
|---|---|---|
| `bool(key, ctx)` | `bool` | `True` if variant is `"true"` |
| `string(key, ctx)` | `str` | Returns the variant string |
| `evaluate(key, ctx)` | `EvalResult` | Full result with `variant`, `reason`, `enabled` |
| `evaluate_all(ctx)` | `dict[str, EvalResult]` | All flags in one HTTP round trip |

`evaluate_all` is the most efficient option when you need multiple flags — one HTTP request regardless of how many flags exist.

## EvalResult fields

| Field | Type | Notes |
|---|---|---|
| `key` | `str` | Flag key |
| `enabled` | `bool` | Whether the flag is enabled for this context |
| `variant` | `str` | **Primary field.** `"true"` or `"false"` for bool flags; variant key for all others. |
| `reason` | `str` | `"targeting_rule"`, `"default"`, `"disabled"`, or `"percentage_rollout"` |
| `evaluated_at` | `str` | ISO 8601 evaluation timestamp |
| `value` | `str` | **Deprecated.** Use `variant` instead. |

## Real-time streaming

`connect_stream` opens a background SSE connection for live flag changes:

```python
from cuttlegate import connect_stream, CuttlegateConfig, FlagChangeEvent

def on_change(event: FlagChangeEvent) -> None:
    print(f"flag {event.flag_key} changed: enabled={event.enabled}")

stream = connect_stream(config, on_change=on_change, on_error=print)

# ... application runs ...

stream.close()
```

The SDK reconnects automatically on transient failures (5xx, network errors) with exponential backoff. Auth errors (401/403) are terminal.

## Production use — CachedClient

For hot request paths, use `CachedClient`. It seeds an in-memory cache on construction and keeps it fresh via a background SSE connection:

```python
from cuttlegate import CachedClient, CuttlegateConfig, EvalContext

cache = CachedClient(config)

ctx = EvalContext(user_id="user-123")
enabled = cache.bool("dark-mode", ctx)  # cache hit — no HTTP call
```

## Testing

Use `MockCuttlegateClient` for unit tests without a live server:

```python
from cuttlegate import MockCuttlegateClient, EvalContext

mock = MockCuttlegateClient(flags={"dark-mode": True})
ctx = EvalContext(user_id="u1")
assert mock.bool("dark-mode", ctx) is True
mock.assert_evaluated("dark-mode")
```

## Error handling

All errors are typed — catch specific classes:

```python
from cuttlegate import AuthError, FlagNotFoundError, ServerError, SDKError

try:
    result = client.evaluate("my-flag", ctx)
except FlagNotFoundError as exc:
    print(f"flag {exc.key!r} not found")
except AuthError as exc:
    print(f"auth failed: HTTP {exc.status_code}")
except SDKError as exc:
    print(f"sdk error: {exc}")
```

| Error | When raised |
|---|---|
| `ConfigError` | Missing or invalid config field |
| `AuthError` | 401 or 403 from server |
| `FlagNotFoundError` | Key absent from response |
| `ServerError` | 5xx from server |
| `InvalidResponseError` | Malformed JSON or SSE event |

## Configuration

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `api_key` | `str` | yes | — | API key (`cg_...`) |
| `server_url` | `str` | yes | — | Cuttlegate server URL |
| `project` | `str` | yes | — | Project slug |
| `environment` | `str` | yes | — | Environment slug |
| `timeout_ms` | `int` | no | `10000` | Request timeout in milliseconds |

For the full API reference and additional examples (Flask integration, async client, OpenFeature provider), see the [SDK README](https://github.com/The127/cuttlegate/blob/main/sdk/python/README.md).
