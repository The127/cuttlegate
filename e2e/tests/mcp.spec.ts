/**
 * MCP tool call E2E tests — capability tier enforcement.
 *
 * Uses the Playwright `request` fixture for direct HTTP calls against the MCP
 * server (baseURL: http://localhost:8083, configured in playwright.config.ts as
 * the `mcp-api` project). No browser is involved.
 *
 * ## MCP protocol (2024-11-05 subset used here)
 *
 *   GET  /sse             — establishes an SSE stream; server sends an
 *                           `event: endpoint` with `data: /message?session_id=<id>`
 *   POST /message?session_id=<id>  — JSON-RPC 2.0 dispatch
 *
 * Auth: `Authorization: Bearer <plaintext-api-key>` on both requests.
 *
 * ## Scenarios covered
 *
 *   1 @happy     list_flags with read-tier key returns the flag list
 *   2 @happy     evaluate_flag with read-tier key returns a valid result
 *   3 @error-path enable_flag with read-tier key → insufficient_capability
 *   4 @happy     enable_flag with write-tier key succeeds + audit event emitted
 *   5 @auth-bypass unauthenticated request → {"error":"unauthenticated"}
 *   6 @edge      tools/list with read-tier key omits write-tier tools
 *   7 @edge      tools/list with write-tier key includes write-tier tools
 *   8 @edge      key downgraded mid-session — write call rejected (ADR 0028 live lookup)
 */

import { test, expect } from '@playwright/test';
import {
  issueToken,
  createProject,
  createEnvironment,
  createFlag,
  toggleFlag,
  createAPIKey,
  updateAPIKeyTier,
  listAuditEvents,
} from '../internal/factory';

// ── MCP helpers ──────────────────────────────────────────────────────────────

const MCP_BASE = 'http://localhost:8083';

/**
 * Opens an SSE connection to /sse, reads the first `endpoint` event to obtain
 * the session ID, then immediately aborts the stream (we only need the ID).
 *
 * Returns the session ID string (e.g. "a1b2c3...").
 */
async function openMCPSession(apiKey: string): Promise<string> {
  const ac = new AbortController();
  const resp = await fetch(`${MCP_BASE}/sse`, {
    headers: { Authorization: `Bearer ${apiKey}` },
    signal: ac.signal,
  });
  if (!resp.ok) {
    ac.abort();
    const text = await resp.text().catch(() => String(resp.status));
    throw new Error(`SSE /sse returned ${resp.status}: ${text}`);
  }

  const reader = resp.body!.getReader();
  const decoder = new TextDecoder();
  let buf = '';
  let sessionID = '';

  try {
    while (!sessionID) {
      const { value, done } = await reader.read();
      if (done) break;
      buf += decoder.decode(value, { stream: true });
      const lines = buf.split('\n');
      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const dataVal = line.slice('data: '.length).trim();
          const match = dataVal.match(/session_id=([a-f0-9]+)/);
          if (match) {
            sessionID = match[1];
          }
        }
      }
    }
  } finally {
    reader.cancel().catch(() => { /* ignore */ });
    ac.abort();
  }

  if (!sessionID) {
    throw new Error('SSE stream closed without emitting an endpoint event');
  }
  return sessionID;
}

interface JSONRPCResult {
  jsonrpc: string;
  id: number;
  result?: unknown;
  error?: { code: number; message: string };
}

/**
 * Sends a JSON-RPC 2.0 request to POST /message and returns the parsed body.
 */
async function mcpCall(
  sessionID: string,
  apiKey: string,
  method: string,
  params: unknown,
  id = 1,
): Promise<JSONRPCResult> {
  const resp = await fetch(`${MCP_BASE}/message?session_id=${sessionID}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${apiKey}`,
    },
    body: JSON.stringify({ jsonrpc: '2.0', id, method, params }),
  });
  return resp.json() as Promise<JSONRPCResult>;
}

/**
 * Extracts the text payload from an MCP tool result.
 * MCP wraps all tool results as: { content: [{ type: 'text', text: '...' }] }
 */
function toolText(result: unknown): unknown {
  const r = result as { content?: Array<{ type: string; text: string }> };
  if (!r?.content?.[0]?.text) return result;
  try {
    return JSON.parse(r.content[0].text);
  } catch {
    return r.content[0].text;
  }
}

// ── Shared test state ─────────────────────────────────────────────────────────

interface TestFixtures {
  projectSlug: string;
  envSlug: string;
  flagKey: string;
  adminToken: string;
  readKey: string;
  writeKey: string;
  writeKeyId: string;
}

async function setupTestState(): Promise<TestFixtures> {
  const ts = Date.now();
  const adminToken = issueToken('mcp-e2e-admin', 'admin');

  const projectSlug = `mcp-proj-${ts}`;
  const envSlug = 'mcp-env';
  const flagKey = `mcp-flag-${ts}`;

  await createProject(adminToken, `MCP E2E ${ts}`, projectSlug);
  await createEnvironment(adminToken, projectSlug, 'MCP Env', envSlug);
  await createFlag(adminToken, projectSlug, flagKey);

  const [readAPIKey, writeAPIKey] = await Promise.all([
    createAPIKey(adminToken, projectSlug, envSlug, { name: 'e2e-read', capability_tier: 'read' }),
    createAPIKey(adminToken, projectSlug, envSlug, { name: 'e2e-write', capability_tier: 'write' }),
  ]);

  return {
    projectSlug,
    envSlug,
    flagKey,
    adminToken,
    readKey: readAPIKey.key,
    writeKey: writeAPIKey.key,
    writeKeyId: writeAPIKey.id,
  };
}

// ── Tests ─────────────────────────────────────────────────────────────────────

test.describe('MCP capability tier enforcement', () => {
  let fixtures: TestFixtures;

  test.beforeAll(async () => {
    fixtures = await setupTestState();
  });

  test('Scenario 1 @happy: list_flags with read-tier key returns flag list', async () => {
    const sessionID = await openMCPSession(fixtures.readKey);
    const resp = await mcpCall(sessionID, fixtures.readKey, 'tools/call', {
      name: 'list_flags',
      arguments: {
        project_slug: fixtures.projectSlug,
        environment_slug: fixtures.envSlug,
      },
    });

    expect(resp.error).toBeUndefined();
    const payload = toolText(resp.result);
    expect(Array.isArray(payload)).toBe(true);
    const flags = payload as Array<{ key: string }>;
    expect(flags.some((f) => f.key === fixtures.flagKey)).toBe(true);
  });

  test('Scenario 2 @happy: evaluate_flag with read-tier key returns a valid result', async () => {
    const sessionID = await openMCPSession(fixtures.readKey);
    const resp = await mcpCall(sessionID, fixtures.readKey, 'tools/call', {
      name: 'evaluate_flag',
      arguments: {
        project_slug: fixtures.projectSlug,
        environment_slug: fixtures.envSlug,
        key: fixtures.flagKey,
        eval_context: { user_id: 'test-user', attributes: {} },
      },
    });

    expect(resp.error).toBeUndefined();
    const payload = toolText(resp.result) as { key: string; enabled: boolean };
    expect(payload.key).toBe(fixtures.flagKey);
    expect(typeof payload.enabled).toBe('boolean');
  });

  test('Scenario 3 @error-path: enable_flag with read-tier key returns insufficient_capability', async () => {
    const sessionID = await openMCPSession(fixtures.readKey);
    const resp = await mcpCall(sessionID, fixtures.readKey, 'tools/call', {
      name: 'enable_flag',
      arguments: {
        project_slug: fixtures.projectSlug,
        environment_slug: fixtures.envSlug,
        key: fixtures.flagKey,
      },
    });

    expect(resp.error).toBeUndefined();
    const payload = toolText(resp.result) as {
      error: string;
      required: string;
      provided: string;
    };
    expect(payload).toMatchObject({
      error: 'insufficient_capability',
      required: 'write',
      provided: 'read',
    });
  });

  test('Scenario 4 @happy: enable_flag with write-tier key succeeds and emits audit event', async () => {
    await toggleFlag(fixtures.adminToken, fixtures.projectSlug, fixtures.envSlug, fixtures.flagKey, false);

    const sessionID = await openMCPSession(fixtures.writeKey);
    const resp = await mcpCall(sessionID, fixtures.writeKey, 'tools/call', {
      name: 'enable_flag',
      arguments: {
        project_slug: fixtures.projectSlug,
        environment_slug: fixtures.envSlug,
        key: fixtures.flagKey,
      },
    });

    expect(resp.error).toBeUndefined();
    const payload = toolText(resp.result) as { key: string; enabled: boolean };
    expect(payload.enabled).toBe(true);

    const entries = await listAuditEvents(fixtures.adminToken, fixtures.projectSlug);
    const auditEntry = entries.find(
      (e) => e.action === 'flag.state_changed' && e.flag_key === fixtures.flagKey,
    );
    expect(auditEntry).toBeDefined();
  });

  test('Scenario 5 @auth-bypass: unauthenticated request returns {"error":"unauthenticated"}', async () => {
    const sseResp = await fetch(`${MCP_BASE}/sse`);
    expect(sseResp.status).toBe(401);
    const sseBody = await sseResp.json() as { error: string };
    expect(sseBody).toMatchObject({ error: 'unauthenticated' });

    const msgResp = await fetch(`${MCP_BASE}/message?session_id=fake`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        jsonrpc: '2.0',
        id: 1,
        method: 'tools/call',
        params: { name: 'list_flags', arguments: {} },
      }),
    });
    expect(msgResp.status).toBe(401);
    const msgBody = await msgResp.json() as { error: string };
    expect(msgBody).toMatchObject({ error: 'unauthenticated' });
  });

  test('Scenario 6 @edge: tools/list with read-tier key does not include write-tier tools', async () => {
    const sessionID = await openMCPSession(fixtures.readKey);
    const resp = await mcpCall(sessionID, fixtures.readKey, 'tools/list', {});

    expect(resp.error).toBeUndefined();
    const result = resp.result as { tools: Array<{ name: string }> };
    const names = result.tools.map((t) => t.name);

    expect(names).toContain('list_flags');
    expect(names).toContain('evaluate_flag');
    expect(names).not.toContain('enable_flag');
    expect(names).not.toContain('disable_flag');
  });

  test('Scenario 7 @edge: tools/list with write-tier key includes write-tier tools', async () => {
    const sessionID = await openMCPSession(fixtures.writeKey);
    const resp = await mcpCall(sessionID, fixtures.writeKey, 'tools/list', {});

    expect(resp.error).toBeUndefined();
    const result = resp.result as { tools: Array<{ name: string }> };
    const names = result.tools.map((t) => t.name);

    expect(names).toContain('list_flags');
    expect(names).toContain('evaluate_flag');
    expect(names).toContain('enable_flag');
    expect(names).toContain('disable_flag');
  });

  test('Scenario 8 @edge: key downgraded mid-session is rejected on subsequent write call', async () => {
    // Ensure the flag is disabled so enable_flag is a valid write operation.
    await toggleFlag(fixtures.adminToken, fixtures.projectSlug, fixtures.envSlug, fixtures.flagKey, false);

    // 1. Open an MCP session with the write-tier key and confirm write works.
    const sessionID = await openMCPSession(fixtures.writeKey);
    const writeResp = await mcpCall(sessionID, fixtures.writeKey, 'tools/call', {
      name: 'enable_flag',
      arguments: {
        project_slug: fixtures.projectSlug,
        environment_slug: fixtures.envSlug,
        key: fixtures.flagKey,
      },
    });
    expect(writeResp.error).toBeUndefined();
    const writePayload = toolText(writeResp.result) as { enabled: boolean };
    expect(writePayload.enabled).toBe(true);

    // 2. Admin downgrades the key's tier to read via the PATCH endpoint.
    await updateAPIKeyTier(
      fixtures.adminToken,
      fixtures.projectSlug,
      fixtures.envSlug,
      fixtures.writeKeyId,
      'read',
    );

    // Reset flag so the same write operation is valid again.
    await toggleFlag(fixtures.adminToken, fixtures.projectSlug, fixtures.envSlug, fixtures.flagKey, false);

    // 3. Same session, same key — write call should now be rejected.
    const rejectedResp = await mcpCall(sessionID, fixtures.writeKey, 'tools/call', {
      name: 'enable_flag',
      arguments: {
        project_slug: fixtures.projectSlug,
        environment_slug: fixtures.envSlug,
        key: fixtures.flagKey,
      },
    });
    expect(rejectedResp.error).toBeUndefined();
    const rejectedPayload = toolText(rejectedResp.result) as {
      error: string;
      required: string;
      provided: string;
    };
    expect(rejectedPayload).toMatchObject({
      error: 'insufficient_capability',
      required: 'write',
      provided: 'read',
    });
  });
});
