---
sidebar_label: JavaScript / TypeScript SDK
sidebar_position: 2
---

# JavaScript / TypeScript SDK

The Cuttlegate JS/TS SDK evaluates feature flags via the Cuttlegate HTTP API. It works in Node.js (v20+) and in the browser.

## Installation

```bash
npm install @cuttlegate/sdk
```

## Quick start

```typescript
import { createClient } from '@cuttlegate/sdk';

const cg = createClient({
  baseUrl: 'http://localhost:8080',
  token: 'cg_YOUR_API_KEY',
  project: 'my-project',
  environment: 'production',
});

// Evaluate a single flag
const result = await cg.evaluate('dark-mode', {
  user_id: 'user-123',
  attributes: { plan: 'pro' },
});

console.log(result.enabled);  // true or false
console.log(result.variant);  // "true", "false", or a variant key
console.log(result.reason);   // "disabled", "default", "rule_match", etc.
```

## Configuration

`createClient(config)` accepts:

| Option | Type | Required | Description |
|---|---|---|---|
| `baseUrl` | `string` | Yes | Cuttlegate server URL (e.g. `http://localhost:8080`) |
| `token` | `string` | Yes | API key (`cg_...`) or OIDC Bearer token |
| `project` | `string` | Yes | Project slug |
| `environment` | `string` | Yes | Environment slug to evaluate against |
| `timeout` | `number` | No | Request timeout in ms (default: 5000) |
| `fetch` | `typeof fetch` | No | Custom fetch implementation |

## Evaluating flags

### Single flag

```typescript
const result = await cg.evaluate('my-flag', {
  user_id: 'user-123',
  attributes: { plan: 'pro', country: 'DE' },
});
// { key, enabled, variant, reason, evaluatedAt }
```

If the flag does not exist, the SDK throws `CuttlegateError` with `code: 'not_found'`.

### Convenience methods

```typescript
const enabled = await cg.bool('dark-mode', ctx);   // returns boolean
const variant = await cg.string('banner-text', ctx); // returns variant string
```

### Bulk evaluation

```typescript
const results = await cg.evaluateAll({
  user_id: 'user-123',
  attributes: { plan: 'pro' },
});
// EvalResult[] — one entry per flag in the project/environment
```

## Error handling

The SDK throws `CuttlegateError` with a machine-readable `code`:

```typescript
import { CuttlegateError } from '@cuttlegate/sdk';

try {
  await cg.evaluate('my-flag', ctx);
} catch (err) {
  if (err instanceof CuttlegateError) {
    switch (err.code) {
      case 'unauthorized':     // invalid or expired token
      case 'forbidden':        // token lacks access to this project/environment
      case 'not_found':        // flag key does not exist
      case 'timeout':          // request exceeded timeout
      case 'network_error':    // server unreachable or non-2xx response
      case 'invalid_response': // response didn't match expected schema
    }
  }
}
```

## Testing

See [Testing](/docs/js/testing) for the in-process mock client that lets you test flag-dependent code without a running server.

## Gotchas

See [Gotchas & Known Behaviours](/docs/js/gotchas) for non-obvious runtime behaviours you may encounter when using the SDK.
