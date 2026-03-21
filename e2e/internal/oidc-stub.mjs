/**
 * Minimal OIDC stub server for E2E tests.
 *
 * Generates an RSA-2048 keypair at startup, serves the OIDC discovery document
 * and JWKS endpoint, and writes the issuer URL and private key PEM to a state
 * file so that test data factories can issue signed JWTs.
 *
 * Serves:
 *   GET /.well-known/openid-configuration  — OIDC discovery document
 *   GET /.well-known/jwks.json             — RSA public key in JWK Set format
 *
 * Environment variables:
 *   OIDC_STUB_PORT   — port to listen on (default: 8081)
 *   OIDC_STATE_FILE  — path to write state JSON (default: ../.e2e-oidc.json)
 */

import { createServer } from 'http';
import { generateKeyPairSync, createPublicKey } from 'crypto';
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
});

const jwksDoc = JSON.stringify({ keys: [jwk] });

const server = createServer((req, res) => {
  if (req.url === '/.well-known/openid-configuration') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(discoveryDoc);
  } else if (req.url === '/.well-known/jwks.json') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(jwksDoc);
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
