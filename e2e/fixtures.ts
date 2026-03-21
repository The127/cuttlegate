/**
 * Shared Playwright fixtures for authenticated tests.
 *
 * Extends the base test with a page that has sessionStorage pre-loaded
 * from the auth setup project. Playwright's storageState handles cookies
 * and localStorage; this fixture fills the sessionStorage gap.
 */

import { test as base } from '@playwright/test';
import { readFileSync, existsSync } from 'fs';
import { resolve } from 'path';

const SESSION_STORAGE_PATH = resolve(__dirname, '.auth-session-storage.json');

export const test = base.extend({
  page: async ({ page }, use) => {
    if (existsSync(SESSION_STORAGE_PATH)) {
      const sessionData: Record<string, string> = JSON.parse(
        readFileSync(SESSION_STORAGE_PATH, 'utf8'),
      );

      // Inject sessionStorage before any page script runs.
      await page.addInitScript((data) => {
        for (const [key, value] of Object.entries(data)) {
          window.sessionStorage.setItem(key, value);
        }
      }, sessionData);
    }

    await use(page);
  },
});

export { expect } from '@playwright/test';
