---
sidebar_label: Getting Started
sidebar_position: 1
---

# Getting Started

This guide takes you from zero to your first SDK-powered flag evaluation. You need Docker installed — nothing else.

## 1. Start the stack

Clone the repository and bring up the full stack with Docker Compose:

```bash
git clone https://github.com/karo/cuttlegate.git
cd cuttlegate
docker compose up --build
```

This starts four services:

| Service | Port | Purpose |
|---|---|---|
| **db** | 5432 | PostgreSQL database |
| **dex** | 5556 | Local OIDC identity provider |
| **migrate** | — | Applies database migrations, then exits |
| **server** | 8080 | Cuttlegate API + embedded web UI |

Wait until you see the server log line indicating it is listening on `:8080`.

## 2. Log in

Open [http://localhost:8080](http://localhost:8080) in your browser. Click **Log in** and authenticate with the pre-configured test user:

| Field | Value |
|---|---|
| Email | `admin@example.com` |
| Password | `password` |

After login you are redirected to the Cuttlegate dashboard. The docker-compose configuration maps the Dex `name` claim to the Cuttlegate role, so the test user has **admin** access.

## 3. Create a project

In the dashboard, click **New Project**. Give it a name (e.g. "My App") and a slug (e.g. `my-app`). The slug is permanent — you will use it in API calls and SDK configuration.

## 4. Create an environment

Inside your project, navigate to **Environments** and create one called `production` with the slug `production`. Environments let you toggle flags independently per deployment stage.

## 5. Create a feature flag

Navigate to **Flags** within your project and create a new flag:

- **Key:** `dark-mode` (this is the identifier you use in code)
- **Type:** Boolean

The flag is created in the **disabled** state.

## 6. Create an API key

Navigate to the `production` environment, then open **API Keys**. Click **Create API Key**.

Copy the key immediately — it is shown only once. It looks like `cg_...`.

API keys are scoped to a single project + environment pair and are used by the SDK to authenticate evaluation requests.

## 7. Install the SDK

In your application, install the Cuttlegate JavaScript/TypeScript SDK:

```bash
npm install @cuttlegate/sdk
```

The SDK requires Node.js 20 or later. It also works in the browser via the `browser` export.

## 8. Evaluate a flag

Create a file called `demo.mjs` and paste:

```javascript
import { createClient } from '@cuttlegate/sdk';

const cg = createClient({
  baseUrl: 'http://localhost:8080',
  token: 'cg_YOUR_API_KEY_HERE',   // paste the key from step 6
  project: 'my-app',               // your project slug
  environment: 'production',       // your environment slug
});

const result = await cg.evaluateFlag('dark-mode', {
  user_id: 'user-1',
  attributes: {},
});

console.log('dark-mode enabled:', result.enabled);
console.log('reason:', result.reason);
```

Run it:

```bash
node demo.mjs
```

You should see:

```
dark-mode enabled: false
reason: disabled
```

The flag is disabled, so the SDK returns `enabled: false` with reason `disabled`.

## 9. Toggle the flag and verify

Go back to the Cuttlegate UI, find the `dark-mode` flag in the `production` environment, and toggle it **on**.

Run the script again:

```bash
node demo.mjs
```

```
dark-mode enabled: true
reason: default
```

The SDK now returns `enabled: true`. The `reason` is `default` because no targeting rules are configured — the flag is simply on for everyone.

## What's next

- **Targeting rules** — serve different variants to different users based on attributes
- **Real-time updates** — the SDK supports SSE streaming for instant flag changes without polling
- **Testing** — use `@cuttlegate/sdk/testing` to mock flags in your test suite without a running server (see [Testing](/docs/js/testing))
- **Multiple environments** — create `staging` and `development` environments with separate API keys

## SDK reference

See the full [JavaScript/TypeScript SDK documentation](/docs/js) for the complete API, including `evaluate()` for bulk evaluation and structured error handling.
