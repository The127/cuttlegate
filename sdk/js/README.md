# @cuttlegate/sdk

JavaScript/TypeScript client for the [Cuttlegate](https://github.com/karo/cuttlegate) feature-flag service.

Works in Node.js and any environment with a global `fetch` (browsers, Deno, Bun, Cloudflare Workers).

## Install

```sh
npm install @cuttlegate/sdk
```

## Getting started

```ts
import { createClient } from '@cuttlegate/sdk';

const client = createClient({
  baseUrl: 'https://flags.example.com',
  token: 'cg_your_api_key_here',
  project: 'my-project',
  environment: 'production',
});

const context = {
  user_id: 'user-123',
  attributes: { plan: 'pro' },
};

// Evaluate all flags for the context:
const results = await client.evaluate(context);
for (const result of results) {
  // Use valueKey — the primary field for the variant key.
  // For bool flags it is "true" or "false"; for all other types it is the variant key.
  console.log(`${result.key}: enabled=${result.enabled} valueKey=${result.valueKey}`);
}

// Evaluate a single flag by key:
const flag = await client.evaluateFlag('dark-mode', context);
console.log('dark-mode enabled:', flag.enabled);
console.log('dark-mode variant:', flag.valueKey); // "true" or "false" for bool flags
```

## Result fields

`evaluate()` returns `EvaluationResult[]`. `evaluateFlag()` returns `FlagResult`. Both carry:

| Field | Type | Notes |
|---|---|---|
| `valueKey` | `string` | **Primary field.** The variant key. `"true"` or `"false"` for bool flags; the variant key string for all other types. |
| `enabled` | `boolean` | Whether the flag is enabled for this context. |
| `reason` | `string` | Why this result was returned: `"targeting_rule"`, `"default"`, `"disabled"`, or `"rollout"`. |
| `value` | `string \| null` | **Deprecated.** `null` for bool flags. Use `valueKey` instead. |

### Migration from `value` to `valueKey`

If your code reads `result.value` today:

```ts
// Before (deprecated — null for bool flags):
console.log(result.value);

// After (primary field — always present):
console.log(result.valueKey);
```

For bool flags the `value` field is `null`. Read `valueKey` instead:

```ts
// Before (WRONG — value is null for bool flags):
const enabled = result.value === 'true';

// After:
const enabled = result.valueKey === 'true';
// Or simply:
const enabled = result.enabled;
```

## Error handling

All errors are instances of `CuttlegateError` with a structured `code`:

```ts
import { createClient, CuttlegateError } from '@cuttlegate/sdk';

try {
  const results = await client.evaluate(context);
} catch (err) {
  if (err instanceof CuttlegateError) {
    switch (err.code) {
      case 'unauthorized':
        console.error('Invalid or expired token');
        break;
      case 'forbidden':
        console.error('Token does not have access to this project');
        break;
      case 'timeout':
        console.error('Request timed out');
        break;
      case 'network_error':
        console.error('Network failure:', err.message);
        break;
      case 'invalid_response':
        console.error('Server returned an unexpected response');
        break;
      default:
        console.error('Unexpected error:', err.message);
    }
  }
}
```

`createClient` throws `CuttlegateError` with code `invalid_config` synchronously if any required config field is missing.

## Testing without a live server

Use the `@cuttlegate/sdk/testing` export to test flag integrations in-process:

```ts
import { createMockClient } from '@cuttlegate/sdk/testing';

const mock = createMockClient();
mock.enable('dark-mode');
mock.setVariant('banner-text', 'holiday');

// Inject mock wherever a CuttlegateClient is expected.
const result = await mock.evaluateFlag('banner-text', context);
console.log(result.valueKey); // "holiday"

mock.assertEvaluated('dark-mode');
mock.assertNotEvaluated('other-flag');
mock.reset();
```

Available mock helpers: `enable`, `disable`, `setVariant`, `assertEvaluated`, `assertNotEvaluated`, `reset`.

## Configuration reference

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `baseUrl` | `string` | yes | — | Base URL of the Cuttlegate server |
| `token` | `string` | yes | — | API key from the Cuttlegate UI (`cg_...`) |
| `project` | `string` | yes | — | Project slug |
| `environment` | `string` | yes | — | Environment slug (e.g. `"production"`) |
| `timeout` | `number` | no | 5000 | Request timeout in milliseconds |
| `fetch` | `typeof fetch` | no | `globalThis.fetch` | Custom fetch implementation |
