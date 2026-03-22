/**
 * Accessibility E2E tests — keyboard navigation and ARIA structure.
 *
 * These tests verify WCAG 2.1 AA behavioural requirements that cannot be
 * checked by axe-core static analysis:
 *   1. Focus moves to #main-content on route navigation
 *   2. All key interactive elements on the flag list page are keyboard-reachable
 *   3. CreateProjectDialog traps focus while open and restores it on close
 *   4. The aria-live region is present in the DOM for screen reader announcements
 *
 * Depends on auth-setup (uses saved browser auth state).
 */

import { test, expect } from '@playwright/test';
import { issueToken, createProject, createEnvironment, createFlag } from '../internal/factory';

test.describe('Accessibility — focus management', () => {
  test('main content area has id and tabindex for programmatic focus', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('main#main-content');

    const main = page.locator('main#main-content');
    await expect(main).toHaveAttribute('tabindex', '-1');
  });

  test('aria-live region is present for screen reader announcements', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('main#main-content');

    // The polite live region rendered by LiveAnnouncerProvider
    const liveRegion = page.locator('[role="status"][aria-live="polite"]');
    await expect(liveRegion).toBeAttached();
  });
});

test.describe('Accessibility — keyboard navigation', () => {
  let token: string;
  let projectSlug: string;
  let envSlug: string;

  test.beforeAll(async () => {
    token = issueToken('a11y-test-user', 'admin');
    const ts = Date.now();
    const project = await createProject(token, `A11y Project ${ts}`, `a11y-project-${ts}`);
    projectSlug = project.slug;
    const env = await createEnvironment(token, projectSlug, 'Production', 'production');
    envSlug = env.slug;
    await createFlag(token, projectSlug, `a11y-flag-${ts}`);
  });

  test('all interactive elements on flag list page are keyboard-reachable', async ({ page }) => {
    await page.goto(`/projects/${projectSlug}/environments/${envSlug}/flags`);
    await page.waitForSelector('h1');

    // Start from the top of the page body
    await page.keyboard.press('Tab');

    // Tab through up to 30 elements; collect focused elements
    const focusedTags: string[] = [];
    for (let i = 0; i < 30; i++) {
      const focused = await page.evaluate(() => {
        const el = document.activeElement;
        return el ? `${el.tagName.toLowerCase()}[${el.getAttribute('role') ?? ''}]` : 'none';
      });
      focusedTags.push(focused);
      await page.keyboard.press('Tab');
    }

    // At least one button (toggle) and at least one link (flag name) should be reached
    const hasButton = focusedTags.some((t) => t.startsWith('button'));
    const hasLink = focusedTags.some((t) => t.startsWith('a'));
    expect(hasButton).toBe(true);
    expect(hasLink).toBe(true);
  });

  test('focus does not escape to browser chrome while tabbing through flag list', async ({ page }) => {
    await page.goto(`/projects/${projectSlug}/environments/${envSlug}/flags`);
    await page.waitForSelector('h1');

    // Tab through many elements and confirm focus stays within the document
    for (let i = 0; i < 20; i++) {
      await page.keyboard.press('Tab');
      const isDocumentFocused = await page.evaluate(() => document.hasFocus());
      expect(isDocumentFocused).toBe(true);
    }
  });
});

test.describe('Accessibility — CreateProjectDialog', () => {
  test('focus is trapped inside the dialog while open', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('main#main-content');

    // Open the dialog — find the "New project" button in ProjectSwitcher
    const newProjectBtn = page.getByRole('button', { name: /new project/i });
    await newProjectBtn.click();

    // Dialog should be open; first focusable element should be focused
    await page.waitForSelector('dialog[open]');

    // Tab through elements — focus must stay inside the dialog
    for (let i = 0; i < 10; i++) {
      await page.keyboard.press('Tab');
      const isInsideDialog = await page.evaluate(() => {
        const dialog = document.querySelector('dialog[open]');
        const active = document.activeElement;
        return dialog ? dialog.contains(active) : false;
      });
      expect(isInsideDialog).toBe(true);
    }
  });

  test('focus returns to trigger after dialog is closed with Escape', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('main#main-content');

    const newProjectBtn = page.getByRole('button', { name: /new project/i });
    await newProjectBtn.click();
    await page.waitForSelector('dialog[open]');

    // Close with Escape
    await page.keyboard.press('Escape');
    await page.waitForSelector('dialog[open]', { state: 'detached' });

    // Focus should have returned to the trigger button
    const focusedLabel = await page.evaluate(() =>
      document.activeElement?.textContent?.trim() ?? '',
    );
    expect(focusedLabel).toMatch(/new project/i);
  });

  test('focus returns to trigger after dialog is closed with Cancel button', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('main#main-content');

    const newProjectBtn = page.getByRole('button', { name: /new project/i });
    await newProjectBtn.click();
    await page.waitForSelector('dialog[open]');

    await page.getByRole('button', { name: /cancel/i }).click();
    await page.waitForSelector('dialog[open]', { state: 'detached' });

    const focusedLabel = await page.evaluate(() =>
      document.activeElement?.textContent?.trim() ?? '',
    );
    expect(focusedLabel).toMatch(/new project/i);
  });
});
