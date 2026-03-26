/**
 * E2E: Flag toggle triggers React UI update via SSE without page refresh.
 *
 * Exercises the full reactive vertical:
 *   1. Create project, environment, and flag via API
 *   2. Navigate to the flag detail page
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
    token = issueToken('e2e-user', 'admin');
    await createProject(token, `SSE E2E ${ts}`, projectSlug);
    await createEnvironment(token, projectSlug, 'Production', envSlug);
    await createFlag(token, projectSlug, flagKey);
  });

  test.fixme('flag toggle via API updates UI in real-time via SSE', async ({ page }) => {
    // Navigate directly to the flag list page.
    await page.goto(`/projects/${projectSlug}/environments/${envSlug}/flags`);

    // Wait for the flags list to load, then click through to the flag detail.
    // Two links per row (key + name) — use first().
    const flagLink = page.getByRole('link', { name: flagKey }).first();
    await expect(flagLink).toBeVisible({ timeout: 15_000 });
    await flagLink.click();

    // Wait for the flag detail page to load — the toggle button shows "Disabled".
    const toggleButton = page.getByRole('button', { name: /Enable flag in Production/i });
    await expect(toggleButton).toBeVisible({ timeout: 10_000 });
    await expect(toggleButton).toHaveText('Disabled');

    // Record the current URL to verify no page refresh occurs.
    const urlBefore = page.url();

    // Give the SSE connection time to establish before toggling.
    await page.waitForTimeout(5_000);

    // Toggle the flag via API — NOT via UI click.
    await toggleFlag(token, projectSlug, envSlug, flagKey, true);

    // Assert the UI updates to "Enabled" without a page refresh.
    // The button changes from "Enable flag in Production" → "Disable flag in Production".
    await expect(
      page.getByRole('button', { name: /Disable flag in Production/i }),
    ).toBeVisible({ timeout: 30_000 });

    // Verify no navigation occurred (same URL, no refresh).
    expect(page.url()).toBe(urlBefore);
  });
});
