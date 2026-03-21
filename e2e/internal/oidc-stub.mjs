/**
 * OIDC stub server for E2E tests — full PKCE flow.
 *
 * Generates an RSA-2048 keypair at startup, serves OIDC discovery, JWKS,
 * and implements the authorization code + PKCE flow:
 *
 *   GET  /.well-known/openid-configuration  — OIDC discovery document
 *   GET  /.well-known/jwks.json             — RSA public key in JWK Set format
 *   GET  /authorize                         — authorization endpoint (PKCE, redirects with code)
 *   POST /token                             — token endpoint (validates code_verifier, issues JWT)
 *
 * Environment variables:
 *   OIDC_STUB_PORT   — port to listen on (default: 8081)
 *   OIDC_STATE_FILE  — path to write state JSON (default: ../.e2e-oidc.json)
 */

import { createServer } from 'http';
import { generateKeyPairSync, createPublicKey, createHash, sign as cryptoSign } from 'crypto';
import { writeFileSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));

const PORT = parseInt(process.env.OIDC_STUB_PORT || '8081', 10);
const STATE_FILE =
  process.env.OIDC_STATE_FILE || resolve(__dirname, '../.e2e-oidc.json');

const issuer = `http://localhost:${PORT}`;

const { privateKey, publicKey } = generateKeyPairSync('rsa', {
  modulusLength: 2048,
  publicKeyEncoding: { type: 'spki', format: 'pem' },
  privateKeyEncoding: { type: 'pkcs8', format: 'pem' },
});

const jwk = createPublicKey(publicKey).export({ format: 'jwk' });
jwk.kid = 'e2e-1';
jwk.use = 'sig';
jwk.alg = 'RS256';

const discoveryDoc = JSON.stringify({
  issuer,
  authorization_endpoint: `${issuer}/authorize`,
  token_endpoint: `${issuer}/token`,
  jwks_uri: `${issuer}/.well-known/jwks.json`,
  response_types_supported: ['code'],
  subject_types_supported: ['public'],
  id_token_signing_alg_values_supported: ['RS256'],
  grant_types_supported: ['authorization_code'],
  code_challenge_methods_supported: ['S256'],
});

const jwksDoc = JSON.stringify({ keys: [jwk] });

// In-memory store for pending authorization codes.
// Map<code, { codeChallenge, redirectUri, clientId, nonce }>
const pendingCodes = new Map();

function generateCode() {
  const bytes = new Uint8Array(32);
  globalThis.crypto.getRandomValues(bytes);
  return Buffer.from(bytes).toString('base64url');
}

function base64url(buffer) {
  return Buffer.from(buffer).toString('base64url');
}

function signJWT(payload) {
  const header = { alg: 'RS256', typ: 'JWT', kid: 'e2e-1' };
  const segments = [
    base64url(JSON.stringify(header)),
    base64url(JSON.stringify(payload)),
  ];
  const signingInput = segments.join('.');
  const signature = cryptoSign('sha256', Buffer.from(signingInput), privateKey);
  return `${signingInput}.${base64url(signature)}`;
}

function setCORS(res) {
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type');
}

function handleAuthorize(req, res) {
  const url = new URL(req.url, issuer);
  const params = url.searchParams;

  const responseType = params.get('response_type');
  const clientId = params.get('client_id');
  const redirectUri = params.get('redirect_uri');
  const state = params.get('state');
  const codeChallenge = params.get('code_challenge');
  const codeChallengeMethod = params.get('code_challenge_method');
  const nonce = params.get('nonce');

  if (responseType !== 'code') {
    res.writeHead(400, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'unsupported_response_type' }));
    return;
  }

  if (!codeChallenge || codeChallengeMethod !== 'S256') {
    res.writeHead(400, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'invalid_request', error_description: 'PKCE S256 required' }));
    return;
  }

  if (!redirectUri) {
    res.writeHead(400, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'invalid_request', error_description: 'redirect_uri required' }));
    return;
  }

  const code = generateCode();
  pendingCodes.set(code, {
    codeChallenge,
    redirectUri,
    clientId,
    nonce,
  });

  // Auto-approve: redirect back immediately with the code.
  const redirect = new URL(redirectUri);
  redirect.searchParams.set('code', code);
  if (state) redirect.searchParams.set('state', state);

  res.writeHead(302, { Location: redirect.toString() });
  res.end();
}

function handleToken(req, res) {
  let body = '';
  req.on('data', (chunk) => { body += chunk; });
  req.on('end', () => {
    const params = new URLSearchParams(body);

    const grantType = params.get('grant_type');
    if (grantType !== 'authorization_code') {
      res.writeHead(400, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'unsupported_grant_type' }));
      return;
    }

    const code = params.get('code');
    const codeVerifier = params.get('code_verifier');
    const redirectUri = params.get('redirect_uri');

    const pending = pendingCodes.get(code);
    if (!pending) {
      res.writeHead(400, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'invalid_grant', error_description: 'unknown or expired code' }));
      return;
    }

    // Single-use: delete immediately.
    pendingCodes.delete(code);

    if (redirectUri !== pending.redirectUri) {
      res.writeHead(400, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'invalid_grant', error_description: 'redirect_uri mismatch' }));
      return;
    }

    // PKCE: verify code_verifier against stored code_challenge (S256).
    if (!codeVerifier) {
      res.writeHead(400, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'invalid_grant', error_description: 'code_verifier required' }));
      return;
    }

    const computedChallenge = base64url(createHash('sha256').update(codeVerifier).digest());
    if (computedChallenge !== pending.codeChallenge) {
      res.writeHead(400, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'invalid_grant', error_description: 'code_verifier mismatch' }));
      return;
    }

    const now = Math.floor(Date.now() / 1000);
    const sub = 'e2e-user';

    const idTokenPayload = {
      iss: issuer,
      sub,
      aud: pending.clientId,
      exp: now + 3600,
      iat: now,
      email: `${sub}@example.com`,
      name: sub,
      role: 'admin',
    };
    if (pending.nonce) idTokenPayload.nonce = pending.nonce;

    const accessTokenPayload = {
      iss: issuer,
      sub,
      aud: pending.clientId,
      exp: now + 3600,
      iat: now,
      scope: 'openid profile email',
      role: 'admin',
    };

    const idToken = signJWT(idTokenPayload);
    const accessToken = signJWT(accessTokenPayload);

    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      access_token: accessToken,
      token_type: 'Bearer',
      expires_in: 3600,
      id_token: idToken,
    }));
  });
}

const server = createServer((req, res) => {
  setCORS(res);

  // Handle CORS preflight.
  if (req.method === 'OPTIONS') {
    res.writeHead(204);
    res.end();
    return;
  }

  if (req.url === '/.well-known/openid-configuration') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(discoveryDoc);
  } else if (req.url === '/.well-known/jwks.json') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(jwksDoc);
  } else if (req.url?.startsWith('/authorize')) {
    handleAuthorize(req, res);
  } else if (req.url === '/token' && req.method === 'POST') {
    handleToken(req, res);
  } else {
    res.writeHead(404);
    res.end();
  }
});

server.listen(PORT, () => {
  // Write state AFTER the server is bound so consumers know it's ready.
  writeFileSync(STATE_FILE, JSON.stringify({ issuer, privateKeyPem: privateKey }));
  console.log(`[oidc-stub] listening on ${issuer}`);
});

process.on('SIGTERM', () => server.close());
process.on('SIGINT', () => server.close());
