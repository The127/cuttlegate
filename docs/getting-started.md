# Getting Started

This guide walks you through running Cuttlegate locally using Docker Compose.

## Prerequisites

- Docker (with Compose v2)
- Ports 8080, 5433, 5002, 5003 available on localhost

## Start the stack

```bash
docker compose up --build
```

This starts:

- `db` — PostgreSQL database
- `migrate` — applies schema migrations, then exits
- `keyline` — OIDC provider for local dev
- `server` — Cuttlegate API server + embedded SPA on port 8080

Wait for all services to report healthy. The server is ready when you see:

```
server-1  | listening on :8080
```

## Verify the server is up

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{"status":"ok"}
```

If the database is not yet reachable you will see:

```json
{"status":"degraded","reason":"database"}
```

with HTTP status 503. Wait a few seconds and retry — the database may still be starting.

## Open the UI

Navigate to [http://localhost:8080](http://localhost:8080) in your browser. You will be redirected to Keyline (the local OIDC provider) to log in.

Default credentials for the local Keyline instance are configured in `deploy/keyline/config.yml`.

## Configuration

All server configuration is via environment variables. See [docs/configuration.md](configuration.md) for the full reference.

## Next steps

- [Configuration reference](configuration.md)
- [MCP server getting started](getting-started-mcp.md)
- [API contract](api-contract.md)
