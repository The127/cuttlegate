# Cuttlegate Go SDK

Go client for the [Cuttlegate](https://github.com/karo/cuttlegate) feature-flag service.

## Install

```sh
go get github.com/karo/cuttlegate/sdk/go
```

## Getting started

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

    // Evaluate a single flag:
    result, err := client.EvaluateFlag(ctx, "dark-mode", evalCtx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("dark-mode enabled:", result.Enabled)

    // Evaluate all flags for the environment:
    results, err := client.EvaluateAll(ctx, evalCtx)
    if err != nil {
        log.Fatal(err)
    }
    for key, r := range results {
        fmt.Printf("%s: enabled=%v reason=%s\n", key, r.Enabled, r.Reason)
    }
}
```

## Production use — CachedClient

For production applications where flag evaluation is in the hot request path,
use `CachedClient`. It seeds a local in-memory cache via `EvaluateAll` on
startup and keeps it fresh via a single background SSE connection. `Bool` and
`String` read from cache with no network round trip; they fall back to a live
HTTP call only on a cache miss (e.g. a new flag added after startup).

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
- `Bootstrap(ctx, evalCtx)` calls `EvaluateAll` once to seed the cache, then
  starts **one** background goroutine managing **one** SSE connection for all
  flags (see ADR-0025). The goroutine reconnects with exponential backoff on
  transient errors (5xx, network); it stops permanently on auth errors (401/403).
- `Bool`/`String` check the cache first. On a cache miss (key not present after
  `Bootstrap`), they fall back to a live HTTP evaluation — the result is not
  stored in cache.
- Cancelling `ctx` stops the SSE goroutine cleanly. The cache remains readable
  after cancel; `Bool`/`String` fall back to live HTTP for any subsequent calls.
- `Bootstrap` may be called more than once to refresh the full cache. The
  previous goroutine is stopped before the new one starts.
- `*CachedClient` satisfies the `Client` interface — inject it wherever a
  `Client` is expected.

**When to use `CachedClient` vs `NewClient`:**

| Scenario | Use |
|---|---|
| Hot request path, many flag checks per request | `CachedClient` |
| Low-volume / batch jobs | `NewClient` |
| Tests | `cuttlegatetesting.MockClient` |

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

## Custom HTTP client

Pass your own `*http.Client` if you have existing transport configuration (TLS, proxies, retry middleware):

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

## Configuration reference

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `BaseURL` | `string` | yes | — | Base URL of the Cuttlegate server |
| `ServiceToken` | `string` | yes | — | API key from the Cuttlegate UI (`cg_...`) |
| `Project` | `string` | yes | — | Project slug |
| `Environment` | `string` | yes | — | Environment slug (e.g. `"production"`) |
| `HTTPClient` | `*http.Client` | no | nil | Custom HTTP client; if set, `Timeout` is ignored |
| `Timeout` | `time.Duration` | no | 10s | Request timeout when `HTTPClient` is nil |
