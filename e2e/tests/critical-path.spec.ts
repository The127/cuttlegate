/**
 * E2E: Critical path — authenticated flag navigation and key visibility.
 *
 * Covers:
 *   @happy — authenticated user navigates to flag list → flag detail;
 *            flag key visible in CopyableCode component header
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
    const token = issueToken('e2e-user', 'admin');
    await createProject(token, projectName, projectSlug);
    await createEnvironment(token, projectSlug, envName, envSlug);
    await createFlag(token, projectSlug, flagKey);
  });

  test('@happy flag key is visible in CopyableCode after navigating to flag detail', async ({ page }) => {
    // Navigate directly to the flag list page.
    await page.goto(`/projects/${projectSlug}/environments/${envSlug}/flags`);

    // Wait for the flag to appear in the list (two links per row: key + name).
    const flagLink = page.getByRole('link', { name: flagKey }).first();
    await expect(flagLink).toBeVisible({ timeout: 15_000 });
    await flagLink.click();

    // Verify the CopyableCode component renders the flag key in the detail header.
    await expect(
      page.getByRole('button', { name: `Copy flag key ${flagKey}` }),
    ).toBeVisible({ timeout: 10_000 });
  });

  test('@edge non-existent flag URL renders error state, not blank page', async ({ page }) => {
    await page.goto(
      `/projects/${projectSlug}/environments/${envSlug}/flags/flag-does-not-exist`,
    );

    await expect(
      page.getByText('Flag not found.'),
    ).toBeVisible({ timeout: 10_000 });

    await expect(
      page.getByRole('link', { name: 'Back to flags' }),
    ).toBeVisible();
  });
});
