# E2E Tests

End-to-end tests for Cuttlegate, using Playwright against the full stack.

## Running locally

```
just test-e2e
```

Requires Docker (or Podman with a socket). The command builds the server binary,
then Playwright's global setup starts Postgres, the OIDC stub, and the server
automatically.

## Test data factory

Use `e2e/internal/factory.ts` to create test resources via the API. Do not
click through the UI to set up state — it's slow, brittle, and not what E2E
tests are for.

### Issuing a token

```typescript
import { issueToken } from '../internal/factory';

const token = issueToken('e2e-user', 'admin');
// token is a valid JWT signed by the OIDC stub, accepted by the server
```

Available roles: `'admin'`, `'editor'`, `'viewer'`.

### Creating resources

```typescript
import { issueToken, createProject, createEnvironment, createFlag } from '../internal/factory';

const token = issueToken('e2e-admin', 'admin');

// Create a project
const project = await createProject(token, 'My Project', 'my-project');

// Create an environment inside that project (uses project slug, not ID)
const env = await createEnvironment(token, project.slug, 'Production', 'production');

// Create a flag inside that project
const flag = await createFlag(token, project.slug, 'my-flag');
```

### Adding a new factory method

1. Find the HTTP handler to understand the request body and response shape.
2. Add a typed response interface that mirrors the JSON.
3. Call `apiRequest()`:

```typescript
export async function createSegment(
  token: string,
  projectSlug: string,
  key: string,
): Promise<Segment> {
  const resp = await apiRequest(
    'POST',
    `/api/v1/projects/${projectSlug}/segments`,
    token,
    { key, name: key },
  );
  return resp.json() as Promise<Segment>;
}
```

4. Clean up resources in `afterAll` if your test creates state that would
   conflict with other tests.

## Architecture

```
global-setup.ts          starts Postgres (docker run), OIDC stub, Go server
global-teardown.ts       kills processes, stops container
internal/oidc-stub.mjs   minimal Node.js OIDC server — discovery doc + JWKS only
internal/factory.ts      REST API helpers for seeding test data
tests/                   Playwright test specs
```

The OIDC stub generates a fresh RSA-2048 keypair each run. The private key PEM
is written to `.e2e-oidc.json`. `issueToken()` reads this file to sign JWTs that
the Go server will accept via JWKS verification.
