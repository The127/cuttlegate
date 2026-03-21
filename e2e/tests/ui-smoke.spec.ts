/**
 * UI smoke test — verify the SPA renders API-seeded data correctly.
 *
 * Seeds a project, environment, and flag via the API factory, then navigates
 * through the SPA to verify the flag appears in the flag list. This proves the
 * full stack end-to-end: database -> API -> SPA rendering.
 *
 * Note: the BDD scenario says "creates a new project via the UI" but the SPA
 * does not yet have create-project/flag forms. Once those forms ship, this test
 * should be updated to exercise the UI creation flow directly.
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
    // Navigate to the app — auth state is pre-loaded from auth.setup.ts.
    await page.goto('/');
    await expect(
      page.getByRole('heading', { name: 'Cuttlegate' }),
    ).toBeVisible({ timeout: 10_000 });

    // Select the project from the project switcher dropdown.
    await page.locator('#project-select').selectOption(projectSlug);

    // Select the environment from the environment switcher dropdown.
    await expect(page.locator('#env-select')).toBeVisible({ timeout: 5_000 });
    await page.locator('#env-select').selectOption(envSlug);

    // Verify the flag list page loads and shows our flag.
    await expect(page.getByRole('heading', { name: 'Feature Flags' })).toBeVisible({
      timeout: 5_000,
    });
    await expect(page.getByRole('link', { name: flagKey })).toBeVisible();
  });
});
