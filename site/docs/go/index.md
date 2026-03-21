---
sidebar_label: Go SDK
sidebar_position: 1
---

# Go SDK

Documentation for the Cuttlegate Go SDK is coming soon.

The Go SDK will allow you to evaluate feature flags in your Go services with a single function call.

```go
// Coming soon
client, _ := cuttlegate.NewClient("https://your-instance", apiKey)
result, _ := client.EvaluateFlag(ctx, "my-flag", cuttlegate.EvalContext{
    UserID: "user-123",
})
```

In the meantime, see the [API reference](https://github.com/karo/cuttlegate) for the raw HTTP evaluation endpoint.
