<p align="center">
  <img src="web/public/logo.svg" width="120" alt="Cuttlegate logo — a friendly purple cuttlefish" />
</p>

<h1 align="center">Cuttlegate</h1>

<p align="center">
  A self-hosted feature flag service.<br/>
  Single binary. Postgres. OIDC. Real-time.
</p>

<p align="center">
  <a href="https://github.com/The127/cuttlegate/releases/latest"><img src="https://img.shields.io/github/v/release/The127/cuttlegate" alt="Latest release" /></a>
  <a href="https://github.com/The127/cuttlegate/actions"><img src="https://img.shields.io/github/actions/workflow/status/The127/cuttlegate/ci.yml?label=CI" alt="CI status" /></a>
  <a href="https://the127.github.io/cuttlegate"><img src="https://img.shields.io/badge/docs-GitHub%20Pages-blue" alt="Docs" /></a>
</p>

---

## What is Cuttlegate?

Cuttlegate is a feature flag server for teams who want full control over their flag infrastructure. No SaaS dependency, no vendor lock-in — just a Go binary, a Postgres database, and your existing OIDC provider.

**Core features:**

- **Boolean and string flags** with variants, rules, segments, and percentage rollouts
- **Project and environment scoping** — model your real deployment topology
- **OIDC authentication** — plug into Keycloak, Auth0, Dex, or any OIDC provider
- **RBAC** — admin, editor, and viewer roles per project
- **Real-time streaming** — SSE pushes flag changes to SDKs instantly
- **API keys** — scoped per project/environment with read, write, and destructive tiers
- **Prometheus metrics** — `/metrics` endpoint for production observability
- **Audit logging** — every flag change is recorded with who, what, and when

## Quick start

```sh
# Run with Docker Compose (includes Postgres and a dev OIDC provider)
docker compose up -d

# Open the UI
open http://localhost:8080
```

Or pull the image directly:

```sh
docker pull ghcr.io/the127/cuttlegate:latest
```

See the [Getting Started guide](https://the127.github.io/cuttlegate/docs/getting-started) for full setup instructions.

## SDKs

Evaluate flags from your services with first-party SDKs:

| SDK | Install | Docs |
|-----|---------|------|
| **Go** | `go get github.com/The127/cuttlegate/sdk/go` | [Go SDK docs](https://the127.github.io/cuttlegate/docs/go) |
| **JavaScript / TypeScript** | `npm install @cuttlegate/sdk` | [JS SDK docs](https://the127.github.io/cuttlegate/docs/js) |
| **Python** | `pip install cuttlegate` | [Python SDK docs](https://the127.github.io/cuttlegate/docs/python) |

All SDKs include:

- **CachedClient** — in-memory cache with background SSE updates for zero-latency reads
- **FlagStore** — pluggable persistence interface for offline bootstrap (bring your own storage)
- **OpenFeature provider** — standard-compliant integration
- **Test doubles** — mock clients for unit testing without a live server

## CLI

Manage flags from the terminal. Authenticates via OIDC device flow.

```sh
# Download from the latest release
curl -LO https://github.com/The127/cuttlegate/releases/latest/download/cuttlegate-linux-amd64
chmod +x cuttlegate-linux-amd64

# Authenticate
./cuttlegate-linux-amd64 login --issuer https://your-oidc-provider.com

# Configure defaults
./cuttlegate-linux-amd64 config set server https://flags.example.com
./cuttlegate-linux-amd64 config set project my-project
./cuttlegate-linux-amd64 config set environment production

# Use it
./cuttlegate-linux-amd64 flags list
./cuttlegate-linux-amd64 flags enable dark-mode
./cuttlegate-linux-amd64 eval dark-mode --user-id user-123 --attr plan=pro
```

Available for Linux, macOS, and Windows (amd64 and arm64).

## Configuration

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | — (required) | Postgres connection string |
| `OIDC_ISSUER` | — (required) | OIDC provider base URL for discovery |
| `OIDC_CLIENT_ID` | — (required) | OIDC client ID |
| `OIDC_REDIRECT_URI` | — (required) | OAuth callback URL |
| `OIDC_AUDIENCE` | — | Expected `aud` claim; empty skips the check |
| `OIDC_ROLE_CLAIM` | `role` | JWT claim carrying the Cuttlegate role (`admin`, `editor`, `viewer`) |
| `OIDC_MISSING_ROLE_POLICY` | `reject` | `reject` returns 401; `viewer` grants viewer role |
| `ADDR` | `:8080` | Listen address |
| `AUTO_MIGRATE` | `false` | Run database migrations on startup (dev only) |

## Development

```sh
# Prerequisites: Go 1.24+, Node.js 22+, just, Docker

# Activate git hooks
just install-hooks

# Full local dev stack (Postgres, Keyline OIDC, hot-reload)
just dev

# Run tests
just test              # unit tests
just test-integration  # integration tests (requires Postgres)
just test-sdk          # all SDK test suites
just test-frontend     # React component tests
just ci                # full CI pipeline locally

# Build
just build             # server binary
just build-cli         # CLI binary
```

## Documentation

Full documentation is hosted at [the127.github.io/cuttlegate](https://the127.github.io/cuttlegate).

## License

See [LICENSE](LICENSE) for details.
