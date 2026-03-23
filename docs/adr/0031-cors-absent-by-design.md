# ADR 0031: CORS absent by design — evaluation endpoint is not browser-called

**Date:** 2026-03-23
**Status:** Accepted
**Issue:** #312

## Context

Cuttlegate has no CORS configuration. No `Access-Control-Allow-Origin` or related headers are set on any endpoint. This was flagged during the Sprint 24 security checklist (#312) as a potential oversight.

Two possible interpretations:

1. CORS was forgotten and should be added.
2. CORS is intentionally absent because no browser-to-Cuttlegate cross-origin request is expected.

The deployment model determines which interpretation is correct.

**Current deployment model:** Cuttlegate is deployed as a self-hosted Docker Compose stack. The management SPA (`web/`) is served by Vite in development and by a static file server (or the same origin) in production. SDK clients — Go, JS/TS, Python — call the evaluation endpoint server-side (Node.js, backend services) or from native mobile clients. They are not called from a browser page on a different origin.

**The flag evaluation endpoint** is the only endpoint that SDK clients call at high frequency. It authenticates via API key (ADR 0012). A browser calling this endpoint cross-origin would need to include the API key in the request — this is a misuse pattern (keys should not be in browser-side JavaScript).

**The management API** is called by the SPA. In a standard deployment the SPA and API share an origin (same host, different port proxied by nginx, or same container). Cross-origin management API calls are not a supported deployment pattern.

## Decision

CORS is intentionally absent from Cuttlegate's HTTP server.

- The evaluation endpoint is not designed to be called cross-origin from a browser. Operator-side SDK integrations run server-side.
- The management SPA is deployed on the same origin as the API in the standard Docker Compose setup.
- Adding permissive CORS headers (e.g. `Access-Control-Allow-Origin: *`) to the evaluation endpoint would be a security regression — it would enable browser pages to call the evaluation endpoint with an API key embedded in client-side JavaScript.

If a future deployment model requires cross-origin access (e.g. a hosted SaaS version where the SPA is served from a CDN on a different origin), CORS configuration should be added at that time with explicit allow-list of trusted origins — not as a wildcard.

## Rationale

**Absent CORS is not missing CORS.** A server that does not set `Access-Control-Allow-Origin` is not misconfigured — browsers will simply reject cross-origin requests to it, which is the correct behaviour for endpoints that are not designed to be called cross-origin.

**API keys in browser JavaScript is a known antipattern.** If CORS were open on the evaluation endpoint, an operator could be tempted to call it directly from a browser SPA, embedding the API key in client-side code. That key would be extractable by any user of the page. The absence of CORS removes this footgun.

**The standard deployment keeps SPA and API on the same origin.** nginx reverse-proxy or a single container serving both is the documented pattern. Same-origin requests do not require CORS headers.

## Consequences

- No CORS configuration is required in `cmd/server/main.go` or any HTTP adapter.
- If a deployment topology requires cross-origin access, the operator must configure a reverse proxy (nginx, Caddy) to add CORS headers at the edge — Cuttlegate does not provide this configuration.
- The getting-started guide and deployment documentation should note that the SPA and API must share an origin in the standard deployment, and that CORS is not configured.
- Any future issue that proposes adding CORS to the evaluation endpoint must explicitly address the API key exposure risk described above.
