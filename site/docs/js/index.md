---
sidebar_label: JavaScript / TypeScript SDK
sidebar_position: 2
---

# JavaScript / TypeScript SDK

Documentation for the Cuttlegate JavaScript and TypeScript SDK is coming soon.

The JS/TS SDK will support both Node.js server-side evaluation and browser-safe client usage.

```typescript
// Coming soon
import { CuttlegateClient } from '@cuttlegate/sdk';

const client = new CuttlegateClient({ baseUrl: 'https://your-instance', apiKey });
const result = await client.evaluateFlag('my-flag', { userId: 'user-123' });
```

In the meantime, see the [API reference](https://github.com/karo/cuttlegate) for the raw HTTP evaluation endpoint.
