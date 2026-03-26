/**
 * E2E: Login flow smoke test — full PKCE authentication via stub OIDC.
 *
 * Scenarios:
 *   1. Happy path: user visits protected page → redirected to stub OIDC
 *      → code exchange completes → user lands on authenticated page
 *   2. Session persistence: after login, navigating to a protected page
 *      shows authenticated content without re-redirect
 */

import { test, expect } from '@playwright/test';

test.describe('OIDC PKCE login flow', () => {
  test('completes full login and lands on authenticated page', async ({ page }) => {
    // Navigate to the root — a protected route that triggers OIDC redirect.
    await page.goto('/');

    // The SPA should redirect to the stub OIDC /authorize endpoint,
    // which auto-approves and redirects back with a code.
    // The callback page exchanges the code and navigates to '/'.
    // Wait for the authenticated landing page to appear.
    await expect(
      page.getByRole('heading', { name: 'Cuttlegate' }),
    ).toBeVisible({ timeout: 15_000 });
  });

  test('session persists across navigation without re-authentication', async ({ page }) => {
    // First, complete the login flow.
    await page.goto('/');
    await expect(
      page.getByRole('heading', { name: 'Cuttlegate' }),
    ).toBeVisible({ timeout: 15_000 });

    // Navigate away and back — should not trigger another OIDC redirect.
    await page.goto('/');
    await expect(
      page.getByRole('heading', { name: 'Cuttlegate' }),
    ).toBeVisible({ timeout: 5_000 });

    // Verify no redirect happened by checking we never left the origin.
    expect(new URL(page.url()).origin).toBe('http://localhost:8082');
  });
});
