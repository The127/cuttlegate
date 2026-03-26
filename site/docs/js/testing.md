---
sidebar_label: Testing
sidebar_position: 3
---

# Testing Your Flag Integration

The `@cuttlegate/sdk/testing` module provides an in-process mock client for testing code that depends on feature flags. No running Cuttlegate server is required.

## Installation

The test helper ships with `@cuttlegate/sdk` — no additional package needed.

```bash
npm install --save-dev @cuttlegate/sdk
```

## Quick Start

```typescript
import { createMockClient } from '@cuttlegate/sdk/testing';

const mock = createMockClient();

// Enable a boolean flag
mock.enable('dark-mode');

// Set a variant flag
mock.setVariant('button-color', 'blue');

// Use it like the real client
const result = await mock.evaluate('dark-mode', { user_id: 'user-1', attributes: {} });
console.log(result.enabled); // true
```

## API

### `createMockClient(): MockCuttlegateClient`

Creates a new mock client. All flags are disabled by default.

### Flag Configuration

| Method | Description |
|---|---|
| `mock.enable(key)` | Enable a boolean flag |
| `mock.setVariant(key, value)` | Enable a flag with a specific variant value |
| `mock.disable(key)` | Remove a flag's configuration (returns to default disabled) |
| `mock.reset()` | Clear all flag configuration and evaluation tracking |

### Evaluation

The mock implements the same `CuttlegateClient` interface as the real client:

```typescript
// Evaluate a single flag
const result = await mock.evaluate('my-flag', { user_id: 'user-1', attributes: {} });
// { key, enabled, variant, reason, evaluatedAt }

// Evaluate all configured flags
const results = await mock.evaluateAll({ user_id: 'user-1', attributes: {} });
// EvalResult[] — only flags that were explicitly configured
```

**Default behaviour:** unconfigured flags return `{ enabled: false, variant: '', reason: 'mock_default' }`.

### Assertions

| Method | Description |
|---|---|
| `mock.assertEvaluated(key)` | Throws if the flag was never evaluated |
| `mock.assertNotEvaluated(key)` | Throws if the flag was evaluated |

Use assertions to verify your code actually checks the flags you expect:

```typescript
const mock = createMockClient();
mock.enable('premium-feature');

await myService.handleRequest(mock, request);

mock.assertEvaluated('premium-feature');
mock.assertNotEvaluated('admin-override');
```

## Example: Testing a Service

```typescript
import { describe, it, expect, beforeEach } from 'vitest';
import { createMockClient } from '@cuttlegate/sdk/testing';
import { FeatureService } from './feature-service';

describe('FeatureService', () => {
  let mock: ReturnType<typeof createMockClient>;
  let service: FeatureService;

  beforeEach(() => {
    mock = createMockClient();
    service = new FeatureService(mock);
  });

  it('shows dark mode when flag is enabled', async () => {
    mock.enable('dark-mode');
    const config = await service.getUIConfig({ user_id: 'user-1', attributes: {} });
    expect(config.darkMode).toBe(true);
  });

  it('hides dark mode by default', async () => {
    const config = await service.getUIConfig({ user_id: 'user-1', attributes: {} });
    expect(config.darkMode).toBe(false);
  });
});
```

## Type Safety

`MockCuttlegateClient` extends `CuttlegateClient`, so it can be used anywhere the real client is expected — no type casting required:

```typescript
import type { CuttlegateClient } from '@cuttlegate/sdk';

function myFunction(client: CuttlegateClient) {
  // works with both real and mock clients
}
```
