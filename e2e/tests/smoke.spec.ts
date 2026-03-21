/**
 * Smoke tests — prove the E2E harness is working end-to-end.
 *
 * These tests exercise:
 *   1. The Go server health endpoint (no auth)
 *   2. The test data factory (authenticated API calls via stub-issued JWTs)
 *
 * If these pass, the harness is ready for feature-specific E2E tests.
 */

import { test, expect } from '@playwright/test';
import {
  issueToken,
  createProject,
  createEnvironment,
  createFlag,
} from '../internal/factory';

test('GET /healthz returns 200', async ({ request }) => {
  const resp = await request.get('/healthz');
  expect(resp.status()).toBe(200);
});

test('factory creates project, environment, and flag via authenticated API', async () => {
  const token = issueToken('e2e-admin', 'admin');
  const ts = Date.now();

  const project = await createProject(token, `E2E Project ${ts}`, `e2e-project-${ts}`);
  expect(project.id).toBeTruthy();
  expect(project.slug).toBe(`e2e-project-${ts}`);

  const env = await createEnvironment(token, project.slug, 'E2E Env', 'e2e-env');
  expect(env.id).toBeTruthy();
  expect(env.slug).toBe('e2e-env');

  const flag = await createFlag(token, project.slug, `e2e-flag-${ts}`);
  expect(flag.id).toBeTruthy();
  expect(flag.key).toBe(`e2e-flag-${ts}`);
});
