---
sidebar_label: Go SDK
sidebar_position: 1
---

# Go SDK

The Cuttlegate Go SDK evaluates feature flags in your Go services. It requires Go 1.24 or later.

## Install

```bash
go get github.com/The127/cuttlegate/sdk/go
```

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    cuttlegate "github.com/The127/cuttlegate/sdk/go"
)

func main() {
    client, err := cuttlegate.NewClient(cuttlegate.Config{
        BaseURL:      "https://flags.example.com",
        ServiceToken: "cg_your_api_key_here", // API key from the Cuttlegate UI
        Project:      "my-project",
        Environment:  "production",
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    evalCtx := cuttlegate.EvalContext{
        UserID:     "user-123",
        Attributes: map[string]any{"plan": "pro"},
    }

    // Bool is the simplest way to evaluate a boolean flag
    enabled, err := client.Bool(ctx, "dark-mode", evalCtx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("dark-mode enabled:", enabled)
}
```

`NewClient` validates the configuration and returns an error if any required field is missing. No network calls are made at construction time.

## Evaluation methods

| Method | Returns | Notes |
|---|---|---|
| `Bool(ctx, key, evalCtx)` | `(bool, error)` | Preferred for boolean flags |
| `String(ctx, key, evalCtx)` | `(string, error)` | Returns the variant string |
| `Evaluate(ctx, key, evalCtx)` | `(EvalResult, error)` | Returns full result including `Reason` and `Variant` |
| `EvaluateAll(ctx, evalCtx)` | `(map[string]EvalResult, error)` | Evaluates all flags in one HTTP round trip |
| `EvaluateFlag(ctx, key, evalCtx)` | `(FlagResult, error)` | Deprecated — use `Bool` or `Evaluate` for new code |
| `Subscribe(ctx, key)` | `(<-chan FlagUpdate, <-chan error, error)` | Real-time SSE stream of flag state changes |

`EvaluateAll` is the most efficient method when you need multiple flags — one HTTP request regardless of how many flags exist.

## EvalResult fields

```go
type EvalResult struct {
    Key         string // flag key
    Enabled     bool   // whether the flag is enabled for this context
    Value       string // Deprecated — use Variant instead
    Variant     string // Primary field: variant key; "true"/"false" for bool flags
    Reason      string // "rule_match", "default", "disabled", or "percentage_rollout"
    EvaluatedAt string // ISO 8601 timestamp
}
```

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

## Real-time streaming

`Subscribe` opens an SSE connection and delivers flag state changes as they happen:

```go
updates, errs, err := client.Subscribe(ctx, "dark-mode")
if err != nil {
    log.Fatal(err)
}

for {
    select {
    case update := <-updates:
        fmt.Printf("flag %s changed: enabled=%v at %s\n",
            update.Key, update.Enabled, update.UpdatedAt)
    case err := <-errs:
        log.Printf("stream error: %v", err)
    case <-ctx.Done():
        return
    }
}
```

Both channels are closed when `ctx` is cancelled. A terminal `AuthError` is delivered on the error channel before it closes on 401/403.

## Testing without a live server

Use the `cuttlegatetesting` subpackage to mock flags in-process:

```go
import (
    cgt "github.com/The127/cuttlegate/sdk/go/testing"
)

func TestMyService(t *testing.T) {
    mock := cgt.NewMockClient()
    mock.Enable("dark-mode")
    mock.SetVariant("banner-text", "holiday")

    svc := mypackage.NewService(mock) // inject as cuttlegate.Client
    svc.GetFeatures(context.Background(), "user-123")

    if err := mock.AssertEvaluated("dark-mode"); err != nil {
        t.Error(err)
    }
}
```

Available mock helpers: `Enable`, `Disable`, `SetVariant`, `AssertEvaluated`, `AssertNotEvaluated`, `Reset`.

## Configuration reference

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `BaseURL` | `string` | yes | — | Base URL of the Cuttlegate server |
| `ServiceToken` | `string` | yes | — | API key from the Cuttlegate UI (`cg_...`) |
| `Project` | `string` | yes | — | Project slug |
| `Environment` | `string` | yes | — | Environment slug (e.g. `"production"`) |
| `HTTPClient` | `*http.Client` | no | nil | Custom HTTP client; if set, `Timeout` is ignored |
| `StreamHTTPClient` | `*http.Client` | no | nil | Custom HTTP client for SSE connections; must not have a short timeout |
| `Timeout` | `time.Duration` | no | 10s | Request timeout for evaluation calls when `HTTPClient` is nil |

## Custom HTTP client

Pass your own `*http.Client` if you have existing transport configuration (TLS, proxies, retry middleware):

```go
client, err := cuttlegate.NewClient(cuttlegate.Config{
    BaseURL:      "https://flags.example.com",
    ServiceToken: "cg_your_api_key_here",
    Project:      "my-project",
    Environment:  "production",
    HTTPClient:   myConfiguredHTTPClient,
})
```

If `HTTPClient` is nil, a default client with a 10-second timeout is used. The `Timeout` field only applies when `HTTPClient` is nil.
