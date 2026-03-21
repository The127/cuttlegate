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

