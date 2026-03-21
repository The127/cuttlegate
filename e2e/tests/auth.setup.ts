/**
 * Playwright auth setup project.
 *
 * Runs once before all test files. Navigates through the OIDC stub login flow,
 * then saves the full browser state (cookies, localStorage, AND sessionStorage)
 * so subsequent tests can skip re-authentication.
 *
 * oidc-client-ts stores tokens in sessionStorage, which Playwright's built-in
 * storageState does not capture. We save sessionStorage separately and restore
 * it via the authenticated fixture in fixtures.ts.
 */

import { test as setup, expect } from '@playwright/test';
import { resolve } from 'path';

const AUTH_STATE_DIR = resolve(__dirname, '..');
export const STORAGE_STATE_PATH = resolve(AUTH_STATE_DIR, '.auth-storage-state.json');
export const SESSION_STORAGE_PATH = resolve(AUTH_STATE_DIR, '.auth-session-storage.json');

setup('authenticate via OIDC stub', async ({ page }) => {
  // Navigate to the app — triggers OIDC redirect to stub, which auto-approves.
  await page.goto('/');

  // Wait for the authenticated landing page.
  await expect(
    page.getByRole('heading', { name: 'Cuttlegate' }),
  ).toBeVisible({ timeout: 15_000 });

  // Save sessionStorage (oidc-client-ts stores tokens here).
  const sessionStorage = await page.evaluate(() => {
    const entries: Record<string, string> = {};
    for (let i = 0; i < window.sessionStorage.length; i++) {
      const key = window.sessionStorage.key(i);
      if (key) entries[key] = window.sessionStorage.getItem(key)!;
    }
    return entries;
  });

  const { writeFileSync } = await import('fs');
  writeFileSync(SESSION_STORAGE_PATH, JSON.stringify(sessionStorage, null, 2));

  // Save cookies + localStorage via Playwright's built-in mechanism.
  await page.context().storageState({ path: STORAGE_STATE_PATH });
});
