/**
 * Playwright global teardown.
 *
 * Reads .e2e-state.json (written by global-setup) and shuts everything down:
 *   1. Go server process (SIGTERM by PID)
 *   2. OIDC stub process (SIGTERM by PID)
 *   3. Postgres container (docker stop)
 */

import { execFileSync } from 'child_process';
import { existsSync, readFileSync, rmSync } from 'fs';
import { resolve } from 'path';

const E2E_DIR = __dirname;
const STATE_FILE = resolve(E2E_DIR, '.e2e-state.json');
const OIDC_STATE_FILE = resolve(E2E_DIR, '.e2e-oidc.json');

interface State {
  serverPid: number;
  oidcStubPid: number;
  pgContainerId: string;
}

export default function globalTeardown(): void {
  if (!existsSync(STATE_FILE)) {
    console.log('[teardown] No state file — nothing to clean up');
    return;
  }

  const state: State = JSON.parse(readFileSync(STATE_FILE, 'utf8'));

  if (state.serverPid) {
    try {
      process.kill(state.serverPid, 'SIGTERM');
    } catch {
      // Process already gone.
    }
  }

  if (state.oidcStubPid) {
    try {
      process.kill(state.oidcStubPid, 'SIGTERM');
    } catch {
      // Process already gone.
    }
  }

  if (state.pgContainerId) {
    try {
      execFileSync('docker', ['stop', state.pgContainerId], { stdio: 'ignore' });
    } catch {
      // Container already stopped.
    }
  }

  for (const f of [STATE_FILE, OIDC_STATE_FILE]) {
    try {
      rmSync(f);
    } catch {
      // Already gone.
    }
  }

  console.log('[teardown] Done');
}
