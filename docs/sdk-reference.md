# SDK Cross-Reference

Comparison of the Go, JavaScript, and Python SDKs. Each SDK has its own README with full details:
- [JS SDK](../sdk/js/README.md) | [JS Migration Guide](../sdk/js/MIGRATION.md)
- Go SDK: `sdk/go/`
- Python SDK: `sdk/python/`

## Evaluation Methods

| Operation | Go | JavaScript | Python |
|---|---|---|---|
| Single flag | `Evaluate(ctx, key, evalCtx) (EvalResult, error)` | `evaluate(key, context): Promise<EvalResult>` | `evaluate(key, ctx) -> EvalResult` |
| All flags | `EvaluateAll(ctx, evalCtx) (map[string]EvalResult, error)` | `evaluateAll(context): Promise<EvalResult[]>` | `evaluate_all(ctx) -> dict[str, EvalResult]` |
| Bool convenience | `Bool(ctx, key, evalCtx) (bool, error)` | `bool(key, context): Promise<boolean>` | `bool(key, ctx) -> bool` |
| String convenience | `String(ctx, key, evalCtx) (string, error)` | `string(key, context): Promise<string>` | `string(key, ctx) -> str` |
| **Deprecated** | `EvaluateFlag()` | `evaluateFlag()` | — |

## EvalResult Fields

| Field | Go | JavaScript | Python | Wire (`value_key`) |
|---|---|---|---|---|
| Flag key | `Key string` | `key: string` | `key: str` | `key` |
| Enabled | `Enabled bool` | `enabled: boolean` | `enabled: bool` | `enabled` |
| Variant | `Variant string` | `variant: string` | `variant: str` | `value_key` |
| Reason | `Reason string` | `reason: string` | `reason: str` | `reason` |
| Evaluated at | `EvaluatedAt string` | `evaluatedAt: string` | `evaluated_at: str` | `evaluated_at` |
| **Deprecated** | `Value string` | `value: string \| null` | `value: str` | `value` |

> **Wire mapping:** The wire format uses `value_key`. All SDKs map this to `variant`. The deprecated `value` field is `null` for bool flags. Use `variant` in new code.

## Configuration

| Field | Go | JavaScript | Python |
|---|---|---|---|
| Server URL | `BaseURL string` | `baseUrl: string` | `server_url: str` |
| Auth token | `ServiceToken string` | `token: string` | `api_key: str` |
| Project | `Project string` | `project: string` | `project: str` |
| Environment | `Environment string` | `environment: string` | `environment: str` |
| Timeout | `Timeout time.Duration` (10s) | `timeout?: number` (5000ms) | `timeout_ms: int` (10000) |
| Custom HTTP | `HTTPClient *http.Client` | `fetch?: typeof fetch` | — |

## Context Type

| Field | Go | JavaScript | Python |
|---|---|---|---|
| User ID | `UserID string` | `user_id: string` | `user_id: str` |
| Attributes | `Attributes map[string]any` | `attributes: Record<string, string>` | `attributes: dict[str, Any]` |

## Streaming / Real-Time

| Aspect | Go | JavaScript | Python |
|---|---|---|---|
| Method | `Subscribe(ctx, key)` | `connectStream(config, opts)` | `connect_stream(config, opts)` |
| Returns | `(<-chan FlagUpdate, <-chan error, error)` | `StreamConnection` (sync) | `StreamConnection` |
| Event type | `FlagUpdate` struct | `FlagStateChangedEvent` | `FlagStateChangedEvent` |
| Reconnect | Automatic with backoff | Automatic with exponential backoff + jitter | Automatic with backoff |

## Cached Client

| Aspect | Go | JavaScript | Python |
|---|---|---|---|
| Creation | `NewCachedClient(Config)` | `createCachedClient(config, opts?)` | `CachedClient(config)` |
| Ready signal | `Bootstrap(ctx, evalCtx) error` | `client.ready: Promise<void>` | Synchronous (constructor blocks) |
| Context | Passed to `Bootstrap()` | `opts.context` (default: anonymous) | Passed to `Bootstrap()` on construction |
| Subscribe | — | `subscribe(key, cb): unsubscribe` | — |
| Close | — | `close()` | `close()` |
| Cache miss | — | Returns `not_found` (no HTTP fallback) | Falls back to live HTTP |
| Thread safety | — | Single-threaded (JS) | Lock-based (daemon thread for SSE) |

## Error Taxonomy

| Scenario | Go | JavaScript | Python |
|---|---|---|---|
| Invalid config | Returned as `error` | `CuttlegateError` code: `invalid_config` (thrown sync) | `ConfigError` (ValueError subclass) |
| Auth failure (401) | `AuthError{StatusCode: 401}` | `CuttlegateError` code: `unauthorized` | `AuthError(status_code=401)` |
| Forbidden (403) | `AuthError{StatusCode: 403}` | `CuttlegateError` code: `forbidden` | `AuthError(status_code=403)` |
| Flag not found | `NotFoundError{Key: "..."}` | `EvalResult` with `reason: "not_found"` | `FlagNotFoundError(key="...")` |
| Timeout | Returned as `error` | `CuttlegateError` code: `timeout` | `SDKError` |
| Network error | Returned as `error` | `CuttlegateError` code: `network_error` | `SDKError` |
| Bad response | Returned as `error` | `CuttlegateError` code: `invalid_response` | `InvalidResponseError` |
| Server 5xx | `ServerError{StatusCode: 500}` | `CuttlegateError` code: `network_error` | `ServerError(status_code=500)` |

## Async Model

| SDK | Model |
|---|---|
| Go | Synchronous with `context.Context` for cancellation. Channels for streaming. |
| JavaScript | Promise-based (async/await). `connectStream` returns synchronously. |
| Python | Synchronous. Background daemon thread for SSE in `CachedClient`. |
