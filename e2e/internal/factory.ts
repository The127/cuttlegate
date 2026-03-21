/**
 * Test data factory.
 *
 * Creates and manages Cuttlegate resources via the REST API for test setup.
 * Authentication uses tokens signed by the OIDC stub's private key.
 *
 * ## Adding a new factory method
 *
 * 1. Add a typed interface for the response (mirror the JSON shape from the handler).
 * 2. Call `apiRequest()` with the correct method, path, and body.
 * 3. The caller is responsible for cleanup — delete the resource in `afterAll` if needed.
 *
 * @example
 * ```typescript
 * const token = issueToken('e2e-admin', 'admin');
 * const project = await createProject(token, 'My Project', 'my-project');
 * const env = await createEnvironment(token, project.slug, 'Production', 'production');
 * const flag = await createFlag(token, project.slug, 'my-flag');
 * ```
 */

import jwt from 'jsonwebtoken';
import { existsSync, readFileSync } from 'fs';
import { resolve } from 'path';

const OIDC_STATE_FILE = resolve(__dirname, '../.e2e-oidc.json');
const BASE_URL = 'http://localhost:8082';

interface OIDCState {
  issuer: string;
  privateKeyPem: string;
}

function readOIDCState(): OIDCState {
  if (!existsSync(OIDC_STATE_FILE)) {
    throw new Error(
      `OIDC state not found at ${OIDC_STATE_FILE} — is globalSetup running?`,
    );
  }
  return JSON.parse(readFileSync(OIDC_STATE_FILE, 'utf8')) as OIDCState;
}

/**
 * Issues a signed JWT for a test user.
 * The token is accepted by the Go server because the OIDC stub's public key
 * is served from the JWKS endpoint that the server fetches on first use.
 */
export function issueToken(sub: string, role: 'admin' | 'editor' | 'viewer'): string {
  const { issuer, privateKeyPem } = readOIDCState();
  return jwt.sign(
    { sub, role, email: `${sub}@example.com`, name: sub },
    privateKeyPem,
    { algorithm: 'RS256', keyid: 'e2e-1', issuer, expiresIn: '1h' },
  );
}

async function apiRequest(
  method: string,
  path: string,
  token: string,
  body?: unknown,
): Promise<Response> {
  const resp = await fetch(`${BASE_URL}${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: body != null ? JSON.stringify(body) : undefined,
  });
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`API ${method} ${path} → ${resp.status}: ${text}`);
  }
  return resp;
}

export interface Project {
  id: string;
  slug: string;
  name: string;
}

export interface Environment {
  id: string;
  project_id: string;
  slug: string;
  name: string;
}

export interface Flag {
  id: string;
  project_id: string;
  key: string;
  name: string;
}

export async function createProject(
  token: string,
  name: string,
  slug: string,
): Promise<Project> {
  const resp = await apiRequest('POST', '/api/v1/projects', token, { name, slug });
  return resp.json() as Promise<Project>;
}

export async function createEnvironment(
  token: string,
  projectSlug: string,
  name: string,
  slug: string,
): Promise<Environment> {
  const resp = await apiRequest(
    'POST',
    `/api/v1/projects/${projectSlug}/environments`,
    token,
    { name, slug },
  );
  return resp.json() as Promise<Environment>;
}

export async function createFlag(
  token: string,
  projectSlug: string,
  key: string,
): Promise<Flag> {
  const resp = await apiRequest(
    'POST',
    `/api/v1/projects/${projectSlug}/flags`,
    token,
    { key, name: key, type: 'bool', variants: [{ key: 'true', name: 'Enabled' }, { key: 'false', name: 'Disabled' }], default_variant_key: 'false' },
  );
  return resp.json() as Promise<Flag>;
}

export async function toggleFlag(
  token: string,
  projectSlug: string,
  envSlug: string,
  flagKey: string,
  enabled: boolean,
): Promise<void> {
  await apiRequest(
    'PATCH',
    `/api/v1/projects/${projectSlug}/environments/${envSlug}/flags/${flagKey}`,
    token,
    { enabled },
  );
}
