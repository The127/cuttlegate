/**
 * UI smoke test — verify the SPA renders API-seeded data correctly.
 *
 * Seeds a project, environment, and flag via the API factory, then navigates
 * through the SPA to verify the flag appears in the flag list. This proves the
 * full stack end-to-end: database -> API -> SPA rendering.
 */

import { test, expect } from '../fixtures';
import {
  issueToken,
  createProject,
  createEnvironment,
  createFlag,
} from '../internal/factory';

test.describe('UI smoke — project and flag', () => {
  const ts = Date.now();
  const projectName = `E2E UI Smoke ${ts}`;
  const projectSlug = `e2e-ui-smoke-${ts}`;
  const envName = 'Production';
  const envSlug = 'production';
  const flagKey = `smoke-flag-${ts}`;

  test.beforeAll(async () => {
    const token = issueToken('e2e-admin', 'admin');
    await createProject(token, projectName, projectSlug);
    await createEnvironment(token, projectSlug, envName, envSlug);
    await createFlag(token, projectSlug, flagKey);
  });

  test('flag appears in the flag list after navigating through the SPA', async ({ page }) => {
    // Navigate directly to the flag list page for the seeded project/environment.
    await page.goto(`/projects/${projectSlug}/environments/${envSlug}/flags`);

    // Verify the flag appears in the list (two links per row: key + name).
    await expect(page.getByRole('link', { name: flagKey }).first()).toBeVisible({ timeout: 15_000 });
  });
});
