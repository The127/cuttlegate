/**
 * Unit tests for the OIDC stub's PKCE implementation.
 *
 * These test the stub server directly (no Playwright) to verify that
 * the authorization and token endpoints enforce PKCE correctly.
 */

import { createHash } from 'crypto';
import { describe, it, before, after } from 'node:test';
import assert from 'node:assert/strict';
import { spawn } from 'child_process';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';
import { existsSync, rmSync } from 'fs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const STUB_PORT = 8091;
const STATE_FILE = resolve(__dirname, '../.e2e-oidc-test.json');

function base64url(buffer) {
  return Buffer.from(buffer).toString('base64url');
}

function makeVerifierAndChallenge() {
  const verifier = base64url(Buffer.from(crypto.getRandomValues(new Uint8Array(32))));
  const challenge = base64url(createHash('sha256').update(verifier).digest());
  return { verifier, challenge };
}

async function getAuthCode(challenge) {
  const params = new URLSearchParams({
    response_type: 'code',
    client_id: 'test-client',
    redirect_uri: 'http://localhost:9999/callback',
    code_challenge: challenge,
    code_challenge_method: 'S256',
    state: 'test-state',
  });
  const resp = await fetch(`http://localhost:${STUB_PORT}/authorize?${params}`, {
    redirect: 'manual',
  });
  assert.equal(resp.status, 302);
  const location = new URL(resp.headers.get('location'));
  return location.searchParams.get('code');
}

async function exchangeCode(code, verifier) {
  const body = new URLSearchParams({
    grant_type: 'authorization_code',
    code,
    code_verifier: verifier,
    redirect_uri: 'http://localhost:9999/callback',
    client_id: 'test-client',
  });
  return fetch(`http://localhost:${STUB_PORT}/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: body.toString(),
  });
}

let stubProcess;

before(async () => {
  if (existsSync(STATE_FILE)) rmSync(STATE_FILE);

  stubProcess = spawn('node', [resolve(__dirname, 'oidc-stub.mjs')], {
    env: {
      ...process.env,
      OIDC_STUB_PORT: String(STUB_PORT),
      OIDC_STATE_FILE: STATE_FILE,
    },
    stdio: ['ignore', 'pipe', 'pipe'],
  });

  const deadline = Date.now() + 10_000;
  while (Date.now() < deadline) {
    if (existsSync(STATE_FILE)) break;
    await new Promise((r) => setTimeout(r, 200));
  }
  if (!existsSync(STATE_FILE)) throw new Error('Stub did not start');
});

after(() => {
  if (stubProcess) stubProcess.kill('SIGTERM');
  if (existsSync(STATE_FILE)) rmSync(STATE_FILE);
});

describe('OIDC stub PKCE', () => {
  it('accepts a valid code_verifier', async () => {
    const { verifier, challenge } = makeVerifierAndChallenge();
    const code = await getAuthCode(challenge);
    const resp = await exchangeCode(code, verifier);
    assert.equal(resp.status, 200);
    const body = await resp.json();
    assert.ok(body.access_token);
    assert.ok(body.id_token);
    assert.equal(body.token_type, 'Bearer');
  });

  it('rejects a mismatched code_verifier', async () => {
    const { challenge } = makeVerifierAndChallenge();
    const wrongVerifier = 'this-is-definitely-not-the-right-verifier';
    const code = await getAuthCode(challenge);
    const resp = await exchangeCode(code, wrongVerifier);
    assert.equal(resp.status, 400);
    const body = await resp.json();
    assert.equal(body.error, 'invalid_grant');
    assert.match(body.error_description, /code_verifier/);
  });

  it('rejects a replayed authorization code', async () => {
    const { verifier, challenge } = makeVerifierAndChallenge();
    const code = await getAuthCode(challenge);
    const resp1 = await exchangeCode(code, verifier);
    assert.equal(resp1.status, 200);
    const resp2 = await exchangeCode(code, verifier);
    assert.equal(resp2.status, 400);
    const body = await resp2.json();
    assert.equal(body.error, 'invalid_grant');
  });

  it('rejects code_challenge_method=plain', async () => {
    const params = new URLSearchParams({
      response_type: 'code',
      client_id: 'test-client',
      redirect_uri: 'http://localhost:9999/callback',
      code_challenge: 'some-challenge',
      code_challenge_method: 'plain',
      state: 'test-state',
    });
    const resp = await fetch(`http://localhost:${STUB_PORT}/authorize?${params}`, {
      redirect: 'manual',
    });
    assert.equal(resp.status, 400);
    const body = await resp.json();
    assert.equal(body.error, 'invalid_request');
  });

  it('rejects missing code_verifier', async () => {
    const { challenge } = makeVerifierAndChallenge();
    const code = await getAuthCode(challenge);
    const body = new URLSearchParams({
      grant_type: 'authorization_code',
      code,
      redirect_uri: 'http://localhost:9999/callback',
      client_id: 'test-client',
    });
    const resp = await fetch(`http://localhost:${STUB_PORT}/token`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: body.toString(),
    });
    assert.equal(resp.status, 400);
    const respBody = await resp.json();
    assert.equal(respBody.error, 'invalid_grant');
    assert.match(respBody.error_description, /code_verifier/);
  });
});
