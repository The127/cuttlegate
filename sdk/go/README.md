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
        ServiceToken: "svc_your_token_here",
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
    results, err := client.Evaluate(ctx, evalCtx)
    if err != nil {
        log.Fatal(err)
    }
    for _, r := range results {
        fmt.Printf("%s: enabled=%v reason=%s\n", r.Key, r.Enabled, r.Reason)
    }
}
```

## Typed errors

All methods return typed errors — no string-only errors.

```go
result, err := client.Evaluate(ctx, evalCtx)
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
    ServiceToken: "svc_your_token_here",
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
| `ServiceToken` | `string` | yes | — | Service account token for auth |
| `Project` | `string` | yes | — | Project slug |
| `Environment` | `string` | yes | — | Environment slug (e.g. `"production"`) |
| `HTTPClient` | `*http.Client` | no | nil | Custom HTTP client; if set, `Timeout` is ignored |
| `Timeout` | `time.Duration` | no | 10s | Request timeout when `HTTPClient` is nil |
