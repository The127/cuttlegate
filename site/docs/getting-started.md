---
sidebar_label: Getting Started
sidebar_position: 1
---

# Getting Started

This guide takes you from zero to your first SDK-powered flag evaluation — with targeting rules, segments, and a real audit trail.

**Prerequisites:** Docker, Git, and either Node 20+ (for the JS path) or Go 1.24+ (for the Go path). Nothing else.

By the end you will have:

- A flag evaluating via the JS or Go SDK
- A targeting rule that enables the flag for a specific user segment
- An audit trail showing exactly which rule matched

## 1. Start the stack

Clone the repository and bring up the full stack with Docker Compose:

```bash
git clone https://github.com/The127/cuttlegate.git
cd cuttlegate
docker compose up --build
```

:::note First run takes 2–5 minutes
On a cold Docker cache, `docker compose up --build` pulls several images before starting. This is normal. Wait for the log line `server listening on :8080` before proceeding.
:::

This starts six services:

| Service | Port | Purpose |
|---|---|---|
| **db** | 5433 | PostgreSQL database (Cuttlegate) |
| **migrate** | — | Applies database migrations, then exits |
| **server** | 8080 | Cuttlegate API + embedded web UI |
| **keyline-db** | — | PostgreSQL database (identity provider) |
| **keyline** | 5002 | Local OIDC identity provider |
| **keyline-ui** | 5003 | Identity provider admin UI |

Wait until you see `server listening on :8080` in the compose output before continuing.

## 2. Log in

Open [http://localhost:8080](http://localhost:8080) in your browser. Click **Log in** and authenticate with the pre-configured test user:

| Field | Value |
|---|---|
| Email | `admin@cuttlegate.local` |
| Password | `password` |

After login you are redirected to the Cuttlegate dashboard. The pre-seeded admin account has **admin** access.

## 3. Create a project

In the dashboard, click **New Project**. Give it a name (e.g. "My App") and a slug (e.g. `my-app`). The slug is permanent — you will use it in API calls and SDK configuration.

## 4. Create an environment

Inside your project, navigate to **Environments** and create one called `production` with the slug `production`. Environments let you toggle flags independently per deployment stage.

## 5. Create a feature flag

Navigate to **Flags** within your project and create a new flag:

- **Key:** `dark-mode` (this is the identifier you use in code)
- **Type:** Boolean

The flag is created in the **disabled** state. Toggle it **on** for the `production` environment before continuing.

## 6. Create an API key

Navigate to the `production` environment, then open **API Keys**. Click **Create API Key**.

Copy the key immediately — it is shown only once. It looks like `cg_...`.

API keys are scoped to a single project + environment pair and are used by the SDK to authenticate evaluation requests.

## 7. Create a segment and add members

Segments are named groups of users. Membership is explicit — you add user IDs to a segment, and targeting rules match on whether the evaluating user is in that list.

Navigate to **Segments** within your project and click **New Segment**:

- **Name:** `Pro users`
- **Key:** `pro-users`

Save the segment. Now add the demo user to it. In the UI, open the `pro-users` segment and click **Add Member**, entering `user-1`.

Alternatively, use the API directly:

```bash
curl -s -X PUT http://localhost:8080/api/v1/projects/my-app/segments/pro-users/members \
  -H "Authorization: Bearer cg_YOUR_API_KEY_HERE" \
  -H "Content-Type: application/json" \
  -d '{"members": ["user-1"]}'
```

Any SDK call that evaluates with `user_id: "user-1"` will now match the `pro-users` segment.

## 8. Add a targeting rule

Navigate back to the `dark-mode` flag and open the `production` environment settings. Click **Add Rule**:

- **Name:** `Pro users get dark mode`
- **Condition:** `in_segment: pro-users`
- **Serve:** Enabled

Save the rule. The flag now behaves as follows:

- Users in the `pro-users` segment (`user-1`) → **enabled**, reason `rule_match`
- All other users → **enabled**, reason `default` (the flag is on; no rule matched their context)

:::tip Fallthrough behaviour
When a flag is toggled on and no targeting rule matches a user, the flag falls through to its default state — enabled for everyone. Rules narrow the audience; they do not gate it unless the flag's default is off.
:::

## 9. Evaluate the flag with SDK

Choose the SDK for your language. Both examples use the `cg_...` key from step 6 — the field name differs between SDKs (see callout below).

### JavaScript / TypeScript

Install the SDK:

```bash
npm install @cuttlegate/sdk
```

Create `demo.mjs`:

```javascript
import { createClient } from '@cuttlegate/sdk';

const cg = createClient({
  baseUrl: 'http://localhost:8080',
  token: 'cg_YOUR_API_KEY_HERE',   // paste the key from step 6
  project: 'my-app',
  environment: 'production',
});

// Evaluate as a segment member — matches the targeting rule
const proResult = await cg.evaluateFlag('dark-mode', {
  user_id: 'user-1',
  attributes: { plan: 'pro' },
});
console.log('pro user enabled:', proResult.enabled);  // true
console.log('reason:', proResult.reason);             // rule_match

// Evaluate as a free user — no rule match, falls through to default
const freeResult = await cg.evaluateFlag('dark-mode', {
  user_id: 'user-2',
  attributes: { plan: 'free' },
});
console.log('free user enabled:', freeResult.enabled); // true
console.log('reason:', freeResult.reason);             // default
```

Run it:

```bash
node demo.mjs
```

### Go

Install the SDK:

```bash
go get github.com/The127/cuttlegate/sdk/go
```

Create `main.go`:

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
        BaseURL:      "http://localhost:8080",
        ServiceToken: "cg_YOUR_API_KEY_HERE", // paste the key from step 6
        Project:      "my-app",
        Environment:  "production",
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Bool is the simplest way to evaluate a boolean flag
    enabled, err := client.Bool(ctx, "dark-mode", cuttlegate.EvalContext{
        UserID:     "user-1",
        Attributes: map[string]any{"plan": "pro"},
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("dark-mode enabled (pro user):", enabled) // true

    // Evaluate returns the full result including the reason
    result, err := client.Evaluate(ctx, "dark-mode", cuttlegate.EvalContext{
        UserID:     "user-1",
        Attributes: map[string]any{"plan": "pro"},
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("reason:", result.Reason) // rule_match
}
```

Run it:

```bash
go run main.go
```

:::info Same API key, different field name
The `cg_...` key from step 6 is the same credential in both SDKs. The JS SDK passes it as `token`; the Go SDK passes it as `ServiceToken`. If you copy a value from one SDK example into the other, the field name is what changes — not the key itself.
:::

## 10. View the audit trail

Go back to the Cuttlegate UI. Navigate to the `dark-mode` flag and open **Audit Trail**.

You will see the evaluation events from your SDK calls. Each entry shows:

- The flag key and environment
- Whether the flag was enabled
- The **reason** — one of:
  - `rule_match` — a targeting rule matched the user
  - `default` — no rule matched; the flag fell through to its default state
  - `segment_deleted` — a targeting rule referenced a segment that no longer exists; the rule was skipped and evaluation fell through to the next matching rule or the flag's default
- The **matched rule name** — visible for `rule_match` evaluations; not present for `default` or `segment_deleted`

This makes it straightforward to verify that your rules are matching the right users in production.

:::warning Stale rule configuration
If you see `segment_deleted` in the audit trail, a targeting rule is referencing a segment that no longer exists. The evaluation continued — falling through to the next rule or the flag's default — but the stale rule is doing nothing. Navigate to the flag's rules and update or remove any rule that references the deleted segment.
:::

## What's next

- **Real-time updates** — the SDK supports SSE streaming for instant flag changes without polling. See [JS SDK](/docs/js) or [Go SDK](/docs/go).
- **Testing** — use `@cuttlegate/sdk/testing` (JS) or the `cuttlegatetesting` package (Go) to mock flags in your test suite without a running server.
- **Multiple environments** — create `staging` and `development` environments with separate API keys and independent flag states.
- **Percentage rollouts** — gradually roll out a flag to a percentage of users without defining a segment.

## SDK reference

- [JavaScript / TypeScript SDK](/docs/js) — full API, bulk evaluation, SSE streaming, testing utilities
- [Go SDK](/docs/go) — `Bool`, `Evaluate`, `EvaluateAll`, `Subscribe`, typed errors, testing mock
