# Cuttlegate JS SDK

JavaScript/TypeScript client for the [Cuttlegate](https://github.com/karo/cuttlegate) feature-flag service. Requires Node.js 20 or later. Works in any modern browser via a bundler.

## Install

```sh
npm install @cuttlegate/sdk
```

## Quick start

```ts
import { createClient } from '@cuttlegate/sdk';

const client = createClient({
  baseUrl: 'https://flags.example.com',
  token: 'cg_your_api_key_here',
  project: 'my-project',
  environment: 'production',
});

const result = await client.evaluateFlag('dark-mode', {
  user_id: 'user-123',
  attributes: { plan: 'pro' },
});

console.log('dark-mode enabled:', result.enabled);
console.log('variant key:', result.valueKey);
```

`createClient` validates the configuration synchronously — it throws a `CuttlegateError` with `code: 'invalid_config'` if any required field is missing. No network calls are made at construction time.

## Configuration

`createClient` and `connectStream` both accept a `CuttlegateConfig` object:

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `baseUrl` | `string` | yes | — | Base URL of the Cuttlegate server |
| `token` | `string` | yes | — | Service account token (`cg_...`) |
| `project` | `string` | yes | — | Project slug |
| `environment` | `string` | yes | — | Environment slug (e.g. `"production"`) |
| `timeout` | `number` | no | `5000` | Request timeout in milliseconds. Does not apply to SSE connections. |
| `fetch` | `typeof fetch` | no | `globalThis.fetch` | Custom fetch implementation |

## Evaluation methods

`CuttlegateClient` has two evaluation methods:

| Method | Returns | Notes |
|---|---|---|
| `evaluate(context)` | `Promise<EvaluationResult[]>` | All flags in one HTTP request |
| `evaluateFlag(key, context)` | `Promise<FlagResult>` | Single flag by key |

`evaluate` is the most efficient option when you need multiple flags — one HTTP round trip regardless of how many flags exist.

### EvalContext

Both methods accept an `EvalContext`:

| Field | Type | Notes |
|---|---|---|
| `user_id` | `string` | User identifier for targeting rules. Empty string is valid but will not match user-specific rules. |
| `attributes` | `Record<string, string>` | Arbitrary key-value pairs used in targeting rules. |

### EvaluationResult

`evaluate` returns `EvaluationResult[]`:

| Field | Type | Notes |
|---|---|---|
| `key` | `string` | Flag key |
| `enabled` | `boolean` | Whether the flag is enabled for this context |
| `valueKey` | `string` | **Primary field.** `"true"` or `"false"` for bool flags; the variant key string for all other types. Always present. |
| `value` | `string \| null` | **Deprecated.** `null` for bool flags. Use `valueKey` instead. |
| `reason` | `string` | Why this result was returned: `"targeting_rule"`, `"default"`, `"disabled"`, or `"percentage_rollout"` |
| `evaluatedAt` | `string` | ISO 8601 evaluation timestamp |

### FlagResult

`evaluateFlag` returns `FlagResult`:

| Field | Type | Notes |
|---|---|---|
| `enabled` | `boolean` | Whether the flag is enabled |
| `valueKey` | `string` | **Primary field.** Always present. |
| `value` | `string \| null` | **Deprecated.** `null` for bool flags. Use `valueKey`. |
| `reason` | `string` | Reason string, or `"not_found"` if the key does not exist in the project |

### Migrating from `value` to `valueKey`

```ts
// Before (deprecated — null for bool flags):
console.log(result.value);

// After (primary field — always present):
console.log(result.valueKey);

// For bool flags, read enabled directly:
const isOn = result.enabled;
```

## Real-time streaming

**`connectStream` returns `StreamConnection` synchronously — it does not return a Promise.**

```ts
import { connectStream } from '@cuttlegate/sdk';

const stream = connectStream(config, {
  onFlagChange(event) {
    console.log(`${event.flagKey} changed: enabled=${event.enabled}`);
  },
  onConnected(reconnect) {
    console.log(reconnect ? 'reconnected' : 'connected');
  },
  onDisconnect() {
    console.log('stream disconnected — will reconnect');
  },
  onError(err) {
    console.error('stream error:', err.code, err.message);
  },
});

// Later — close the stream:
stream.close();
```

### StreamOptions callbacks

| Callback | Signature | When called |
|---|---|---|
| `onFlagChange` | `(event: FlagStateChangedEvent) => void` | **Required.** A flag state changed. |
| `onConnected` | `(reconnect: boolean) => void` | Connection established; `reconnect` is `true` after a drop. |
| `onDisconnect` | `() => void` | Connection dropped; reconnect attempt will follow. |
| `onError` | `(error: CuttlegateError) => void` | Terminal error (401/403) or invalid event received. |

### FlagStateChangedEvent fields

| Field | Type | Notes |
|---|---|---|
| `type` | `'flag.state_changed'` | Event type discriminator |
| `flagKey` | `string` | Flag that changed |
| `enabled` | `boolean` | New enabled state |
| `project` | `string` | Project slug |
| `environment` | `string` | Environment slug |
| `occurredAt` | `string` | ISO 8601 timestamp |

### Reconnection policy

The SDK reconnects automatically on transient failures:

- **Initial backoff:** 1000 ms
- **Growth:** exponential (1s → 2s → 4s → …)
- **Cap:** 30 seconds
- **Jitter:** full jitter — each delay is a random value between 0 and the computed cap

**Auth errors are terminal.** When the server returns 401 or 403, `onError` is called with a `CuttlegateError` (`code: 'unauthorized'` or `'forbidden'`) and the stream stops. It will not reconnect automatically — fix the credential and call `connectStream` again.

Call `stream.close()` to stop the stream at any time.

## CachedClient

For production applications where flag evaluation is in the hot request path, use `createCachedClient`. It seeds an in-memory cache via an initial HTTP fetch and keeps it fresh via a single background SSE connection.

```ts
import { createCachedClient } from '@cuttlegate/sdk';

const client = createCachedClient(
  {
    baseUrl: 'https://flags.example.com',
    token: 'cg_your_api_key_here',
    project: 'my-project',
    environment: 'production',
  },
  {
    onError(err) {
      // Called on terminal SSE auth errors after hydration (401/403).
      // The cache retains its last-known values.
      console.error('stream auth error:', err.code);
    },
  },
);

// Await ready before evaluating — it resolves once the initial HTTP hydration
// succeeds and the cache is seeded.
await client.ready;

// evaluateFlag and evaluate serve from cache — no HTTP round trip.
const result = await client.evaluateFlag('dark-mode', {
  user_id: 'user-123',
  attributes: {},
});
console.log('dark-mode:', result.enabled);

// Stop the SSE connection when done (e.g. on server shutdown).
client.close();
```

### How it works

- `createCachedClient` validates config synchronously — throws `CuttlegateError` with `code: 'invalid_config'` on missing fields.
- On construction, the SSE connection opens immediately (before hydration), buffering events that arrive during the HTTP fetch.
- `client.ready` resolves when the HTTP hydration completes and the cache is seeded. Await it before calling `evaluateFlag` or `evaluate`. Calls made before `ready` resolves return `{ enabled: false, reason: 'not_found' }` from an empty cache.
- `client.ready` rejects with `CuttlegateError` if hydration fails: `code: 'unauthorized'`, `'forbidden'`, `'timeout'`, or `'network_error'`.
- `evaluateFlag` and `evaluate` always serve from cache — no live HTTP calls. Unknown flag keys return `{ enabled: false, valueKey: '', reason: 'not_found' }`.
- The cache is not user-specific. `evaluateFlag(key, context)` accepts a context for interface compatibility, but the cached result is the same for all users.
- On SSE reconnect, the SDK re-fetches all flags to close the missed-events gap.
- `close()` stops the SSE connection. Cached values are retained after close.

### CachedClient vs createClient

| Scenario | Use |
|---|---|
| Hot request path, many flag checks per request | `createCachedClient` |
| Low-volume scripts or batch jobs | `createClient` |
| Unit tests | `createMockClient` (see Testing) |

## Testing

Use `createMockClient` from `@cuttlegate/sdk/testing` to test flag integrations without a live server:

```ts
import { createMockClient } from '@cuttlegate/sdk/testing';

const mock = createMockClient();
mock.enable('dark-mode');
mock.setVariant('banner-text', 'holiday');

// Inject mock as CuttlegateClient wherever your code expects it
const result = await mock.evaluateFlag('dark-mode', { user_id: 'u1', attributes: {} });
// { enabled: true, valueKey: 'true', reason: 'mock' }

mock.assertEvaluated('dark-mode');
mock.assertNotEvaluated('other-flag');
```

### MockCuttlegateClient methods

| Method | Description |
|---|---|
| `enable(key)` | Sets the flag to `enabled: true`, `valueKey: 'true'` |
| `disable(key)` | **Removes** the key from the mock — see gotcha below |
| `setVariant(key, value)` | Sets the flag to enabled with the given variant string as `valueKey` |
| `assertEvaluated(key)` | Throws if the key was not evaluated since the last `reset()` |
| `assertNotEvaluated(key)` | Throws if the key was evaluated since the last `reset()` |
| `reset()` | Clears all flag state and evaluation history |

**`disable()` gotcha:** `disable(key)` removes the key from the mock — it does not set `enabled: false`. A subsequent `evaluateFlag(key, ctx)` returns `{ enabled: false, reason: 'mock_default' }`, not a `CuttlegateError`. To test that a flag is off with a specific reason, use `setVariant` or leave the key absent and assert on `reason === 'mock_default'`.

`MockCuttlegateClient` implements `CuttlegateClient` — inject it wherever a `CuttlegateClient` is expected.

## Error handling

All errors thrown by the SDK are instances of `CuttlegateError`:

```ts
import { CuttlegateError } from '@cuttlegate/sdk';

try {
  const result = await client.evaluateFlag('my-flag', { user_id: 'u1', attributes: {} });
} catch (err) {
  if (err instanceof CuttlegateError) {
    console.error(err.code, err.message);
  }
}
```

### Known error codes

| Code | When |
|---|---|
| `invalid_config` | `createClient`, `connectStream`, or `createCachedClient` called with a missing required field. Thrown synchronously. |
| `timeout` | HTTP request exceeded the configured `timeout`. |
| `network_error` | Network failure, DNS error, or unexpected HTTP status. |
| `unauthorized` | Server returned 401. For `connectStream`, this is terminal — no reconnect. |
| `forbidden` | Server returned 403. For `connectStream`, this is terminal — no reconnect. |
| `invalid_response` | Server response failed schema validation. |

`createClient` throws synchronously on invalid config — errors from the constructor do not need to be caught asynchronously:

```ts
// throws CuttlegateError { code: 'invalid_config', message: 'token is required' }
const client = createClient({ baseUrl: 'https://flags.example.com', project: 'x', environment: 'production' });
```

## Browser, Node, and React

`@cuttlegate/sdk` is the single import path for all environments.

**In Node.js** (ESM): import directly.

```ts
import { createClient } from '@cuttlegate/sdk';
```

**In bundler projects** (Vite, webpack, esbuild, Rollup): the bundler resolves the `browser` condition in the package exports map automatically. React components are exported from the browser entry point and are available from the same `@cuttlegate/sdk` import — no separate path needed.

```tsx
import { CuttlegateProvider, useFlag, useFlagVariant } from '@cuttlegate/sdk';
```

There is no `@cuttlegate/sdk/react` export path. Do not use it.

### CuttlegateProvider

Wrap your React app with `CuttlegateProvider` to make flag state available to all child components:

```tsx
import { CuttlegateProvider } from '@cuttlegate/sdk';

function App() {
  return (
    <CuttlegateProvider config={{
      baseUrl: 'https://flags.example.com',
      token: 'cg_your_api_key_here',
      project: 'my-project',
      environment: 'production',
    }}>
      <MyApp />
    </CuttlegateProvider>
  );
}
```

`CuttlegateProvider` fetches all flags on mount using an empty user context and opens an SSE connection for real-time updates. It closes the connection on unmount.

**User-specific targeting:** `CuttlegateProvider` and the hooks evaluate without a user context (`user_id: ''`). Targeting rules that match on a specific user ID will not apply. For user-specific flag evaluation, call `client.evaluateFlag(key, { user_id: '...', attributes: {} })` directly from your component.

### useFlag

```tsx
import { useFlag } from '@cuttlegate/sdk';

function DarkModeToggle() {
  const { enabled, loading } = useFlag('dark-mode');
  if (loading) return null;
  return <Toggle on={enabled} />;
}
```

`useFlag(key)` returns `{ enabled: boolean; loading: boolean }`. While loading, `enabled` is `false`. After the initial fetch, `enabled` updates reactively on SSE events.

Must be used inside a `CuttlegateProvider`.

### useFlagVariant

```tsx
import { useFlagVariant } from '@cuttlegate/sdk';

function Banner() {
  const { value, loading } = useFlagVariant('banner-text');
  if (loading || value === null) return null;
  return <p>{value}</p>;
}
```

`useFlagVariant(key)` returns `{ value: string | null; loading: boolean }`. `value` is `null` while loading or if the flag is not found.

Must be used inside a `CuttlegateProvider`.

### Testing utilities

```ts
import { createMockClient } from '@cuttlegate/sdk/testing';
```

`@cuttlegate/sdk/testing` is the only named sub-path export. All other exports are from `@cuttlegate/sdk`.

## Production guide

For production deployments where flags are checked on every request:

1. Use `createCachedClient` — single SSE connection, in-memory cache, no per-request HTTP.
2. Await `client.ready` before your server starts accepting traffic. This ensures the cache is seeded before any flag evaluation occurs.
3. Handle `onError` on the cached client — it fires on terminal SSE auth errors (401/403) after hydration. The cache retains stale values, so flag evaluation continues to work but will not receive new updates.
4. Call `client.close()` on graceful shutdown to clean up the SSE connection.

```ts
import { createCachedClient, CuttlegateError } from '@cuttlegate/sdk';

const client = createCachedClient(config, {
  onError(err: CuttlegateError) {
    // alert your observability system — flags are frozen at last-known values
    logger.error({ code: err.code }, 'Cuttlegate stream auth error');
  },
});

try {
  await client.ready;
} catch (err) {
  if (err instanceof CuttlegateError) {
    // hydration failed — decide whether to abort startup or continue with defaults
    throw err;
  }
}

// server is ready — client.evaluateFlag() serves from cache
```

For low-volume applications or scripts, use `createClient` — one HTTP request per evaluation call, no setup required.

---

For a narrative getting-started guide, see the [JS SDK guide](../../site/docs/js/index.md).
