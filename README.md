# Cuttlegate

A feature flag service built on a ports & adapters architecture.

## Getting started

```sh
# Install dependencies
go mod download

# Run tests
just test

# Build the server
just build

# Apply database migrations (requires DATABASE_URL)
just migrate-up
```

## Development

