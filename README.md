# Cuttlegate

A feature flag service built on a ports & adapters architecture.

## Getting started

```sh
# Install dependencies
go mod download

# Activate git hooks (required once after cloning)
just install-hooks

# Run tests
just test

# Build the server
just build

# Apply database migrations (requires DATABASE_URL)
just migrate-up
```

## Development

Install `golangci-lint` (pinned via `tools.go`):

```sh
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
```

Then run:

```sh
just lint   # run linter
just test   # run tests
just ci     # lint + all tests — mirrors CI exactly (requires Docker for postgres)
```

See [CHANGELOG.md](CHANGELOG.md) for the release history.

## Configuration

Key environment variables (see `docker-compose.yml` for the full list):

| Variable | Default | Description |
|---|---|---|
| `OIDC_ISSUER` | — (required) | OIDC provider base URL for discovery |
| `OIDC_AUDIENCE` | — | Expected `aud` claim; empty skips the check |
| `OIDC_ROLE_CLAIM` | `role` | JWT claim name carrying the Cuttlegate role (`admin`, `editor`, `viewer`) |
| `OIDC_MISSING_ROLE_POLICY` | `reject` | What to do when a valid token has no role claim. `reject` returns 401; `viewer` grants viewer role and logs a warning. Any other value is a startup error. |
| `DATABASE_URL` | — | Postgres connection string |
| `ADDR` | `:8080` | Listen address |
