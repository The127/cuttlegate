# Configuration Reference

All server configuration is read from environment variables at startup. No config files are used.

## Required

| Variable | Type | Description | Example |
|---|---|---|---|
| `OIDC_ISSUER` | URL | OIDC provider base URL for token discovery (server-side). Must serve `/.well-known/openid-configuration`. | `http://dex:5556/dex` |

## Optional

| Variable | Type | Default | Description | Example |
|---|---|---|---|---|
| `DATABASE_URL` | DSN | *(none)* | PostgreSQL connection string. When unset, the server starts without database-backed routes (health-only mode). | `postgres://user:pass@localhost:5432/cuttlegate?sslmode=disable` |
| `ADDR` | string | `:8080` | Listen address for the HTTP server. | `:3000` |
| `AUTO_MIGRATE` | bool | `false` | Run database migrations at startup. **Not safe for production** — rolling restarts can race between old pods and a migrated schema. Use the standalone `migrate` binary or the `migrate` docker-compose service instead. | `true` |
| `OIDC_AUDIENCE` | string | *(skip check)* | Expected `aud` claim in Bearer tokens. When empty, audience validation is skipped. | `cuttlegate` |
| `OIDC_ROLE_CLAIM` | string | `role` | JWT claim name carrying the Cuttlegate role (`admin`, `editor`, `viewer`). If the claim is missing or unrecognised, the user defaults to `viewer`. | `groups` |
| `OIDC_CLIENT_ID` | string | *(empty)* | OIDC `client_id` for the SPA, returned by `GET /api/v1/config`. | `cuttlegate` |
| `OIDC_REDIRECT_URI` | string | *(empty)* | OIDC redirect URI for the SPA, returned by `GET /api/v1/config`. | `http://localhost:8080/callback` |
| `OIDC_SPA_AUTHORITY` | URL | `OIDC_ISSUER` | OIDC authority URL returned to the SPA (browser-reachable). Use when the server reaches the OIDC provider at a different URL than the browser (e.g. Docker internal vs `localhost`). | `http://localhost:5556/dex` |
| `EVAL_RATE_LIMIT` | int | `600` | Maximum flag evaluation requests per user per rate-limit window. | `1000` |
| `EVAL_RATE_LIMIT_WINDOW` | duration | `1m` | Window size for evaluation rate limiting. Uses Go `time.ParseDuration` format. | `30s` |

## Health Endpoints

| Endpoint | Auth | Description |
|---|---|---|
| `GET /healthz` | None | Liveness probe. Always returns `200 {"status":"ok"}`. |
| `GET /readyz` | None | Readiness probe. Returns `200 {"status":"ok"}` when the database is reachable, `503 {"status":"not_ready","reason":"..."}` otherwise. |

## Docker Compose

The `docker-compose.yml` in the repo root starts the full stack:

```bash
docker compose up --build    # or: just up
```

Services:

| Service | Image | Ports | Purpose |
|---|---|---|---|
| `db` | `postgres:17` | `5432` | PostgreSQL database |
| `dex` | `dexidp/dex:v2.41.1` | `5556` | Stub OIDC provider for local dev |
| `migrate` | *(built from Dockerfile)* | — | Runs `cmd/migrate up`, then exits |
| `server` | *(built from Dockerfile)* | `8080` | Cuttlegate server with embedded SPA |

### Test user (Dex)

The stub OIDC provider (Dex) is pre-configured with a test user:

| Field | Value |
|---|---|
| Email | `admin@example.com` |
| Password | `password` |
| Role | `admin` (via `OIDC_ROLE_CLAIM=name` — Dex sets the `name` claim to `admin` from the static password `username` field) |

The docker-compose configuration sets `OIDC_ROLE_CLAIM=name` so that the Dex test user's `username: admin` maps to the `admin` role. This is a local development convenience — production deployments should configure their OIDC provider to include a dedicated `role` claim.
