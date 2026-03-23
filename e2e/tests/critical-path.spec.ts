/**
 * E2E: Critical path — authenticated flag navigation and key visibility.
 *
 * Covers:
 *   @happy — authenticated user navigates project → environment → flag list
 *            → flag detail; flag key visible in CopyableCode component header
 *   @edge  — navigating to a non-existent flag URL renders "Flag not found."
 *            error state (not blank, not an infinite loop)
 *
 * Auth state is pre-loaded from auth.setup.ts (storageState + sessionStorage).
 * All test data is created via the API factory before browser navigation begins.
 */

import { test, expect } from '../fixtures';
import {
  issueToken,
  createProject,
  createEnvironment,
  createFlag,
} from '../internal/factory';

test.describe('Critical path — flag navigation and key visibility', () => {
  const ts = Date.now();
  const projectName = `E2E Critical Path ${ts}`;
  const projectSlug = `e2e-cp-${ts}`;
  const envName = 'Production';
  const envSlug = 'production';
  const flagKey = `e2e-flag-${ts}`;

  test.beforeAll(async () => {
    // Use 'e2e-user' — matches the sub claim from the OIDC stub's browser
    // login flow, so the browser-authenticated user owns this project.
    const token = issueToken('e2e-user', 'admin');
    await createProject(token, projectName, projectSlug);
    await createEnvironment(token, projectSlug, envName, envSlug);
    await createFlag(token, projectSlug, flagKey);
  });

  test('@happy flag key is visible in CopyableCode after navigating to flag detail', async ({ page }) => {
    // Auth state is pre-loaded — navigate directly to the app root.
    await page.goto('/');
    await expect(
      page.getByRole('heading', { name: 'Cuttlegate' }),
    ).toBeVisible({ timeout: 10_000 });

    // Select the project from the project switcher dropdown.
    await page.locator('#project-select').selectOption(projectSlug);

    // Select the environment from the environment switcher dropdown.
    await expect(page.locator('#env-select')).toBeVisible({ timeout: 5_000 });
    await page.locator('#env-select').selectOption(envSlug);

    // Wait for the flag list page to load.
    await expect(
      page.getByRole('heading', { name: 'Feature Flags' }),
    ).toBeVisible({ timeout: 5_000 });

    // Click through to the flag detail page.
    const flagLink = page.getByRole('link', { name: flagKey });
    await expect(flagLink).toBeVisible();
    await flagLink.click();

    // Verify the CopyableCode component renders the flag key in the detail header.
    // CopyableCode renders a <button> with aria-label="Copy <key>".
    await expect(
      page.getByRole('button', { name: `Copy ${flagKey}` }),
    ).toBeVisible({ timeout: 10_000 });
  });

  test('@edge non-existent flag URL renders error state, not blank page', async ({ page }) => {
    // Navigate directly to a flag URL that does not exist.
    await page.goto(
      `/projects/${projectSlug}/environments/${envSlug}/flags/flag-does-not-exist`,
    );

    // The flag detail route renders "Flag not found." on a 404.
    // The query does not retry on 404 (APIError status check), so the error
    // state appears quickly without looping.
    await expect(
      page.getByText('Flag not found.'),
    ).toBeVisible({ timeout: 10_000 });

    // The "Back to flags" link should also be present.
    await expect(
      page.getByRole('link', { name: 'Back to flags' }),
    ).toBeVisible();
  });
});
