import { defineConfig } from '@playwright/test';
import { resolve } from 'path';

const STORAGE_STATE_PATH = resolve(__dirname, '.auth-storage-state.json');

export default defineConfig({
  testDir: './tests',
  globalSetup: './global-setup',
  globalTeardown: './global-teardown',
  use: {
    baseURL: 'http://localhost:8082',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  reporter: [['html', { open: 'never' }], ['list']],
  timeout: 60_000,
  retries: 0,
  // Run tests serially — shared server state, not worth parallelising yet.
  workers: 1,
  projects: [
    // Unauthenticated tests — run first, no dependencies.
    {
      name: 'smoke',
      testMatch: 'smoke.spec.ts',
    },
    // Auth setup — runs the OIDC login flow once and saves browser state.
    {
      name: 'auth-setup',
      testMatch: 'auth.setup.ts',
    },
    // Login flow tests — no pre-loaded auth state; they test the login itself.
    {
      name: 'login',
      testMatch: 'login.spec.ts',
    },
    // Authenticated tests — depend on auth-setup, reuse saved browser state.
    {
      name: 'authenticated',
      testMatch: 'ui-smoke.spec.ts',
      dependencies: ['auth-setup'],
      use: {
        storageState: STORAGE_STATE_PATH,
      },
    },
    // SSE live update test — authenticated, tests full reactive vertical.
    {
      name: 'flag-sse',
      testMatch: 'flag-sse.spec.ts',
      dependencies: ['auth-setup'],
      use: {
        storageState: STORAGE_STATE_PATH,
      },
    },
    // Accessibility tests — keyboard nav, focus management, ARIA structure.
    {
      name: 'accessibility',
      testMatch: 'accessibility.spec.ts',
      dependencies: ['auth-setup'],
      use: {
        storageState: STORAGE_STATE_PATH,
      },
    },
    // Critical path — authenticated navigation, flag key visibility, 404 error state.
    {
      name: 'critical-path',
      testMatch: 'critical-path.spec.ts',
      dependencies: ['auth-setup'],
      use: {
        storageState: STORAGE_STATE_PATH,
      },
    },
    // MCP API tests — API-level (no browser), tests MCP tool calls and capability tier enforcement.
    // The MCP server runs on :8083 (MCP_ADDR set in global-setup to avoid collision with OIDC stub on :8081).
    {
      name: 'mcp-api',
      testMatch: 'mcp.spec.ts',
      use: {
        baseURL: 'http://localhost:8083',
      },
    },
  ],
});
