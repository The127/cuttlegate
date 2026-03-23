# Cuttlegate Go SDK

Go client for the [Cuttlegate](https://github.com/karo/cuttlegate) feature-flag service. Requires Go 1.22 or later.

## Install

```sh
go get github.com/karo/cuttlegate/sdk/go
```

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    cuttlegate "github.com/karo/cuttlegate/sdk/go"
)

func main() {
    client, err := cuttlegate.NewClient(cuttlegate.Config{
        BaseURL:      "https://flags.example.com",
        ServiceToken: "cg_your_api_key_here",
        Project:      "my-project",
        Environment:  "production",
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    evalCtx := cuttlegate.EvalContext{
        UserID: "user-123",
        // Attributes values must be JSON-serialisable.
        // Use string, number, bool, or nil — not channels or functions.
        Attributes: map[string]any{
            "plan": "pro",
            "beta": true,
        },
    }

    // Evaluate a boolean flag:
    enabled, err := client.Bool(ctx, "dark-mode", evalCtx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("dark-mode enabled:", enabled)

    // Evaluate a string flag — use Variant for the raw variant key:
    result, err := client.Evaluate(ctx, "banner-text", evalCtx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("banner-text variant:", result.Variant) // e.g. "holiday"

    // Evaluate all flags in one HTTP request:
    results, err := client.EvaluateAll(ctx, evalCtx)
    if err != nil {
        log.Fatal(err)
    }
    for key, r := range results {
        fmt.Printf("%s: enabled=%v variant=%s reason=%s\n", key, r.Enabled, r.Variant, r.Reason)
    }
}
```

`NewClient` validates the configuration and returns an error if any required field is missing. No network calls are made at construction time.

## Evaluation methods

| Method | Returns | Notes |
|---|---|---|
| `Bool(ctx, key, evalCtx)` | `(bool, error)` | Preferred for boolean flags |
| `String(ctx, key, evalCtx)` | `(string, error)` | Returns the string variant value; empty for bool flags — use `Bool` instead |
| `Evaluate(ctx, key, evalCtx)` | `(EvalResult, error)` | Returns full result including `Variant`, `Reason`, `Enabled` |
| `EvaluateAll(ctx, evalCtx)` | `(map[string]EvalResult, error)` | Evaluates all flags in one HTTP round trip |
| `EvaluateFlag(ctx, key, evalCtx)` | `(FlagResult, error)` | **Deprecated** — use `Bool`, `String`, or `Evaluate` for new code |
| `Subscribe(ctx, key)` | `(<-chan FlagUpdate, <-chan error, error)` | Real-time SSE stream of flag state changes |

`EvaluateAll` is the most efficient method when you need multiple flags — one HTTP request regardless of how many flags exist.

## Result types

### EvalResult

`Evaluate` and `EvaluateAll` return `EvalResult`:

| Field | Type | Notes |
|---|---|---|
| `Key` | `string` | Flag key |
| `Enabled` | `bool` | Whether the flag is enabled for this context |
| `Variant` | `string` | **Primary field.** The variant key. `"true"` or `"false"` for bool flags; the variant key string for all other types. |
| `Reason` | `string` | Why this result was returned: `"targeting_rule"`, `"default"`, `"disabled"`, or `"percentage_rollout"` |
| `Value` | `string` | **Deprecated.** Empty for bool flags. Use `Variant` instead. |
| `EvaluatedAt` | `string` | ISO 8601 evaluation timestamp |

**`String()` method note:** `String(ctx, key, evalCtx)` returns `result.Value`, which is empty for bool flags. Do not call `String()` on a bool flag — use `Bool()` instead, and use `Variant` if you need the raw `"true"`/`"false"` key.

### FlagResult (deprecated)

`EvaluateFlag` returns `FlagResult`. This type is deprecated — use `Evaluate`, `Bool`, or `String` for new code. Those methods return a structured `NotFoundError` for missing keys; `EvaluateFlag` encodes not-found as `Reason: "not_found"` with a nil error, making programmatic handling harder.

| Field | Type | Notes |
|---|---|---|
| `Enabled` | `bool` | Whether the flag is enabled |
| `Variant` | `string` | The variant key |
| `Value` | `string` | **Deprecated.** Empty for bool flags. Use `Variant`. |
| `Reason` | `string` | `"not_found"` if the key is absent; otherwise a standard reason string |

### Migration from `Value` to `Variant`

If your code reads `result.Value` today:

```go
// Before (deprecated — empty for bool flags):
fmt.Println(result.Value)

// After (primary field — always present):
fmt.Println(result.Variant)
```

For bool flags, use the typed helper:

```go
// Before:
enabled := result.Value == "true" // WRONG — Value is empty for bool flags

// After:
enabled, err := client.Bool(ctx, "dark-mode", evalCtx)
```

## Subscribe — real-time streaming

`Subscribe` opens an SSE connection and delivers flag state changes as they happen:

```go
updates, errs, err := client.Subscribe(ctx, "dark-mode")
if err != nil {
    log.Fatal(err)
}

for {
    select {
    case update, ok := <-updates:
        if !ok {
            return // context cancelled — stream closed
        }
        fmt.Printf("flag %s changed: enabled=%v at %s\n",
            update.Key, update.Enabled, update.UpdatedAt)
    case err, ok := <-errs:
        if !ok {
            return // context cancelled — stream closed
        }
        log.Printf("stream error: %v", err)
    case <-ctx.Done():
        return
    }
}
```

### Reconnection policy

The SDK reconnects automatically on transient failures:

- **Initial backoff:** 100 ms
- **Growth:** exponential (100 ms → 200 ms → 400 ms → …)
- **Cap:** 30 seconds
- **Jitter:** ±10% applied to each delay

**Auth errors are terminal.** When the server returns 401 or 403, an `*AuthError` is sent to the error channel and both channels are closed. The stream does not reconnect — fix the credential and create a new `Subscribe` call.

**Context cancellation** closes both the update channel and the error channel cleanly. No further values are sent after cancellation.

Multiple independent `Subscribe` calls on the same key return independent streams — cancelling one does not affect the others.

## Production use — CachedClient

For production applications where flag evaluation is in the hot request path, use `CachedClient`. It seeds a local in-memory cache via `EvaluateAll` on startup and keeps it fresh via a single background SSE connection. `Bool` and `String` read from cache with no network round trip; they fall back to a live HTTP call only on a cache miss (e.g. a new flag added after startup).

```go
package main

import (
    "context"
    "fmt"
    "log"

    cuttlegate "github.com/karo/cuttlegate/sdk/go"
)

func main() {
    cc, err := cuttlegate.NewCachedClient(cuttlegate.Config{
        BaseURL:      "https://flags.example.com",
        ServiceToken: "cg_your_api_key_here",
        Project:      "my-project",
        Environment:  "production",
    })
    if err != nil {
        log.Fatal(err)
    }

    // ctx controls the background SSE goroutine lifetime.
    // Cancel it on shutdown to stop the goroutine cleanly.
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    evalCtx := cuttlegate.EvalContext{UserID: "user-123"}

    // Bootstrap seeds the cache via EvaluateAll and starts the SSE goroutine.
    if err := cc.Bootstrap(ctx, evalCtx); err != nil {
        log.Fatalf("Bootstrap: %v", err)
    }

    // Bool and String read from cache — no network call on cache hit.
    enabled, err := cc.Bool(ctx, "dark-mode", evalCtx)
    if err != nil {
        log.Printf("dark-mode eval: %v", err)
    }
    fmt.Println("dark-mode:", enabled)

    banner, err := cc.String(ctx, "banner-text", evalCtx)
    if err != nil {
        log.Printf("banner-text eval: %v", err)
    }
    fmt.Println("banner-text:", banner)
}
```

**How it works:**

- `NewCachedClient` validates config — no network calls.
- `Bootstrap(ctx, evalCtx)` calls `EvaluateAll` once to seed the cache, then starts **one** background goroutine managing **one** SSE connection for all flags. The goroutine reconnects with exponential backoff on transient errors (5xx, network); it stops permanently on auth errors (401/403).
- `Bool`/`String` check the cache first. On a cache miss (key not present after `Bootstrap`), they fall back to a live HTTP evaluation — the result is not stored in cache.
- Cancelling `ctx` stops the SSE goroutine cleanly. The cache remains readable after cancel; `Bool`/`String` fall back to live HTTP for any subsequent calls.
- `Bootstrap` may be called more than once to refresh the full cache. The previous goroutine is stopped before the new one starts.
- `*CachedClient` satisfies the `Client` interface — inject it wherever a `Client` is expected.

**When to use `CachedClient` vs `NewClient`:**

| Scenario | Use |
|---|---|
| Hot request path, many flag checks per request | `CachedClient` |
| Low-volume / batch jobs | `NewClient` |
| Tests | `cuttlegatetesting.MockClient` |

## Testing without a live server

Use the `cuttlegatetesting` subpackage to test flag integrations in-process:

```go
import (
    cgt "github.com/karo/cuttlegate/sdk/go/testing"
)

func TestMyService(t *testing.T) {
    mock := cgt.NewMockClient()
    mock.Enable("dark-mode")
    mock.SetVariant("banner-text", "holiday")

    svc := mypackage.NewService(mock) // inject as cuttlegate.Client
    result := svc.GetFeatures(context.Background(), "user-123")

    if err := mock.AssertEvaluated("dark-mode"); err != nil {
        t.Error(err)
    }
}
```

Available mock helpers: `Enable`, `Disable`, `SetVariant`, `AssertEvaluated`, `AssertNotEvaluated`, `Reset`.

**`Subscribe` limitation:** `MockClient.Subscribe` returns immediately-closed channels. It satisfies the `Client` interface for type-checking purposes but does not simulate a real-time stream. If your test needs to exercise the `Subscribe` code path, use an `httptest.Server` that serves SSE responses instead.

## Typed errors

All methods return typed errors — no string-only errors.

```go
results, err := client.EvaluateAll(ctx, evalCtx)
if err != nil {
    var authErr *cuttlegate.AuthError
    var notFoundErr *cuttlegate.NotFoundError
    var serverErr *cuttlegate.ServerError

    switch {
    case errors.As(err, &authErr):
        log.Printf("auth failed: %d %s", authErr.StatusCode, authErr.Message)
    case errors.As(err, &notFoundErr):
        log.Printf("%s %q not found", notFoundErr.Resource, notFoundErr.Key)
    case errors.As(err, &serverErr):
        log.Printf("server error: %d", serverErr.StatusCode)
    default:
        log.Printf("unexpected error: %v", err)
    }
}
```

| Error type | When returned | Key fields |
|---|---|---|
| `*AuthError` | 401 or 403 from the server | `StatusCode`, `Message` |
| `*NotFoundError` | Flag key or project not found | `Resource` (`"flag"` or `"project"`), `Key` |
| `*ServerError` | 5xx from the server | `StatusCode`, `Message` |

**Detecting a missing flag key:**

```go
result, err := client.Evaluate(ctx, "my-flag", evalCtx)
if err != nil {
    var notFound *cuttlegate.NotFoundError
    if errors.As(err, &notFound) && notFound.Resource == "flag" {
        // flag key does not exist in this project
    }
}
```

## Configuration reference

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `BaseURL` | `string` | yes | — | Base URL of the Cuttlegate server |
| `ServiceToken` | `string` | yes | — | API key from the Cuttlegate UI (`cg_...`) |
| `Project` | `string` | yes | — | Project slug |
| `Environment` | `string` | yes | — | Environment slug (e.g. `"production"`) |
| `HTTPClient` | `*http.Client` | no | nil | Custom HTTP client for evaluation requests; if set, `Timeout` is ignored |
| `StreamHTTPClient` | `*http.Client` | no | nil | Custom HTTP client for SSE connections (`Subscribe`, `CachedClient`). Must **not** have a short timeout — SSE connections are long-lived. If nil, a client with no timeout is used. |
| `Timeout` | `time.Duration` | no | 10s | Request timeout for evaluation calls when `HTTPClient` is nil. Does not apply to SSE connections. |

**Custom HTTP client example:**

```go
client, err := cuttlegate.NewClient(cuttlegate.Config{
    BaseURL:      "https://flags.example.com",
    ServiceToken: "cg_your_api_key_here",
    Project:      "my-project",
    Environment:  "production",
    HTTPClient:   myConfiguredHTTPClient, // Timeout field is ignored when HTTPClient is set
})
```

If `HTTPClient` is nil, a default client with a 10-second timeout is used. The `Timeout` field only applies when `HTTPClient` is nil.

## Gotchas

**Context cancellation and channel ranging.** When the context passed to `Subscribe` is cancelled, both the update channel and the error channel are closed. If you range over a channel without a `select` that also checks `ctx.Done()`, you may attempt to read from a closed channel after cancellation. Prefer the `select` pattern shown in the Subscribe section above.

**`String()` on bool flags returns empty string.** See the Result types section for details — use `Bool()` instead.

**Zero-value `UpdatedAt` on `FlagUpdate`.** If the server sends a malformed `occurred_at` timestamp in an SSE event, `FlagUpdate.UpdatedAt` will be the zero value of `time.Time`. The update is still delivered — check `UpdatedAt.IsZero()` if the timestamp matters to your application.

**`EvaluateFlag` does not return `NotFoundError`.** Unlike `Evaluate`, `Bool`, and `String`, `EvaluateFlag` encodes a missing key as `Reason: "not_found"` and returns a nil error. This makes programmatic error handling harder — prefer the typed-error methods for new code.

**`EvaluateAll` is cheaper than N individual calls.** `Bool` and `Evaluate` each make one HTTP round trip. If you need several flags for a single request, call `EvaluateAll` once and read from the resulting map — it is always cheaper than multiple individual calls.

---

For a narrative getting-started guide, see the [Go SDK guide](../../site/docs/go/index.md).
