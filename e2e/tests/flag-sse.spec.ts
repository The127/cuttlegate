/**
 * E2E: Flag toggle triggers React UI update via SSE without page refresh.
 *
 * Exercises the full reactive vertical:
 *   1. Create project, environment, and flag via API
 *   2. Navigate to the flag detail page via SPA
 *   3. Assert the flag shows "Disabled"
 *   4. Toggle the flag via API (not UI) — this tests the SSE delivery path
 *   5. Assert the UI updates to "Enabled" without a page refresh
 */

import { test, expect } from '../fixtures';
import {
  issueToken,
  createProject,
  createEnvironment,
  createFlag,
  toggleFlag,
} from '../internal/factory';

test.describe('Flag SSE live update', () => {
  const ts = Date.now();
  const projectSlug = `sse-e2e-${ts}`;
  const envSlug = 'production';
  const flagKey = `sse-flag-${ts}`;
  let token: string;

  test.beforeAll(async () => {
    // Use 'e2e-user' — matches the sub claim from the OIDC stub's browser flow,
    // so the browser-authenticated user is a member of the project.
    token = issueToken('e2e-user', 'admin');
    await createProject(token, `SSE E2E ${ts}`, projectSlug);
    await createEnvironment(token, projectSlug, 'Production', envSlug);
    await createFlag(token, projectSlug, flagKey);
  });

  test('flag toggle via API updates UI in real-time via SSE', async ({ page }) => {
    // Navigate to the app (auth state pre-loaded from auth.setup.ts).
    await page.goto('/');
    await expect(
      page.getByRole('heading', { name: 'Cuttlegate' }),
    ).toBeVisible({ timeout: 10_000 });

    // Select the project and environment via the switcher dropdowns.
    await page.locator('#project-select').selectOption(projectSlug);
    await expect(page.locator('#env-select')).toBeVisible({ timeout: 5_000 });
    await page.locator('#env-select').selectOption(envSlug);

    // Wait for the flags list to load, then click through to the flag detail.
    await expect(page.getByRole('heading', { name: 'Feature Flags' })).toBeVisible({
      timeout: 5_000,
    });
    const flagLink = page.getByRole('link', { name: flagKey });
    await expect(flagLink).toBeVisible();
    await flagLink.click();

    // Wait for the flag detail page to load — the toggle button shows "Disabled".
    const toggleButton = page.getByRole('button', { name: 'Enable flag', exact: true });
    await expect(toggleButton).toBeVisible({ timeout: 10_000 });
    await expect(toggleButton).toHaveText('Disabled');

    // Record the current URL to verify no page refresh occurs.
    const urlBefore = page.url();

    // Give the SSE connection a moment to establish before toggling.
    await page.waitForTimeout(2_000);

    // Toggle the flag via API — NOT via UI click.
    // This sends a PATCH that triggers server-side SSE broadcast.
    await toggleFlag(token, projectSlug, envSlug, flagKey, true);

    // Assert the UI updates to "Enabled" without a page refresh.
    // After the SSE event, TanStack Query refetches and the button changes
    // from "Enable flag" (aria-label) to "Disable flag".
    await expect(
      page.getByRole('button', { name: 'Disable flag', exact: true }),
    ).toHaveText('Enabled', { timeout: 10_000 });

    // Verify no navigation occurred (same URL, no refresh).
    expect(page.url()).toBe(urlBefore);
  });
});
