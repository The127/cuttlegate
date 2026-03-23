/**
 * Playwright global setup.
 *
 * Starts the full stack before any test runs:
 *   1. Postgres container (docker run)
 *   2. OIDC stub (node e2e/internal/oidc-stub.mjs) — writes .e2e-oidc.json when ready
 *   3. Go server binary — built by `just test-e2e` before Playwright is invoked
 *
 * Persists PIDs and container ID to .e2e-state.json for global-teardown.
 */

import { execFileSync, spawn } from 'child_process';
import { existsSync, readFileSync, rmSync, writeFileSync } from 'fs';
import { resolve } from 'path';

const E2E_DIR = __dirname;
const REPO_ROOT = resolve(E2E_DIR, '..');
const STATE_FILE = resolve(E2E_DIR, '.e2e-state.json');
const OIDC_STATE_FILE = resolve(E2E_DIR, '.e2e-oidc.json');

const OIDC_STUB_PORT = 8081;
const SERVER_PORT = 8082;
const MCP_PORT = 8083;
const PG_PORT = 5433;

interface State {
  serverPid: number;
  oidcStubPid: number;
  pgContainerId: string;
}

export default async function globalSetup(): Promise<void> {
  // Remove stale state files from a previous interrupted run.
  for (const f of [STATE_FILE, OIDC_STATE_FILE]) {
    if (existsSync(f)) rmSync(f);
  }

  // ── 1. Postgres ───────────────────────────────────────────────────────────

  console.log('[setup] Starting Postgres...');
  const pgContainerId = execFileSync('docker', [
    'run', '-d', '--rm',
    '-e', 'POSTGRES_USER=cuttlegate',
    '-e', 'POSTGRES_PASSWORD=cuttlegate',
    '-e', 'POSTGRES_DB=cuttlegate_e2e',
    '-p', `${PG_PORT}:5432`,
    'postgres:16-alpine',
  ]).toString().trim();

  await poll(() => {
    try {
      execFileSync('docker', ['exec', pgContainerId, 'pg_isready', '-U', 'cuttlegate'], {
        stdio: 'ignore',
      });
      return true;
    } catch {
      return false;
    }
  }, 30_000, 'Postgres (container)');

  // Verify host-side connectivity by running a real SQL query via psql.
  // pg_isready inside the container can pass before the port mapping is stable.
  await poll(() => {
    try {
      execFileSync('docker', [
        'exec', pgContainerId,
        'psql', '-U', 'cuttlegate', '-d', 'cuttlegate_e2e', '-c', 'SELECT 1',
      ], { stdio: 'ignore' });
      return true;
    } catch {
      return false;
    }
  }, 15_000, 'Postgres (query)');

  console.log(`[setup] Postgres ready (${pgContainerId.slice(0, 12)})`);
  const dbUrl =
    `postgres://cuttlegate:cuttlegate@127.0.0.1:${PG_PORT}/cuttlegate_e2e?sslmode=disable`;

  // ── 2. OIDC stub ──────────────────────────────────────────────────────────

  console.log('[setup] Starting OIDC stub...');
  const oidcStub = spawn('node', ['internal/oidc-stub.mjs'], {
    cwd: E2E_DIR,
    env: {
      ...process.env,
      OIDC_STUB_PORT: String(OIDC_STUB_PORT),
      OIDC_STATE_FILE,
    },
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  oidcStub.stdout?.on('data', (d: Buffer) => process.stdout.write(`[oidc-stub] ${d}`));
  oidcStub.stderr?.on('data', (d: Buffer) => process.stderr.write(`[oidc-stub] ${d}`));

  // The stub writes OIDC_STATE_FILE the moment it starts listening.
  await poll(() => existsSync(OIDC_STATE_FILE), 15_000, 'OIDC stub');

  const oidcState: { issuer: string } = JSON.parse(readFileSync(OIDC_STATE_FILE, 'utf8'));
  console.log(`[setup] OIDC stub ready at ${oidcState.issuer}`);

  // ── 3. Go server ──────────────────────────────────────────────────────────

  const serverBin = resolve(E2E_DIR, 'bin/server');
  if (!existsSync(serverBin)) {
    throw new Error(
      `Server binary not found at ${serverBin}. Run \`just test-e2e\` which builds it first.`
    );
  }

  const server = spawn(serverBin, [], {
    env: {
      ...process.env,
      OIDC_ISSUER: oidcState.issuer,
      OIDC_CLIENT_ID: 'cuttlegate',
      OIDC_REDIRECT_URI: `http://localhost:${SERVER_PORT}/auth/callback`,
      ADDR: `:${SERVER_PORT}`,
      MCP_ADDR: `:${MCP_PORT}`,
      DATABASE_URL: dbUrl,
      AUTO_MIGRATE: 'true',
    },
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  server.stdout?.on('data', (d: Buffer) => process.stdout.write(`[server] ${d}`));
  server.stderr?.on('data', (d: Buffer) => process.stderr.write(`[server] ${d}`));

  await poll(async () => {
    try {
      const resp = await fetch(`http://localhost:${SERVER_PORT}/healthz`);
      return resp.ok;
    } catch {
      return false;
    }
  }, 30_000, 'Go server /healthz');

  console.log(`[setup] Server ready on :${SERVER_PORT}`);

  // ── Persist state ─────────────────────────────────────────────────────────

  const state: State = {
    serverPid: server.pid!,
    oidcStubPid: oidcStub.pid!,
    pgContainerId,
  };
  writeFileSync(STATE_FILE, JSON.stringify(state, null, 2));
}

async function poll(
  check: () => boolean | Promise<boolean>,
  timeoutMs: number,
  name: string,
): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    if (await check()) return;
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`[setup] Timed out waiting for ${name} (${timeoutMs}ms)`);
}
