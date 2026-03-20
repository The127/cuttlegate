# Install just: https://github.com/casey/just#installation

# Run all unit tests (includes architecture import rules)
test:
    go test ./...

# Run integration tests against a real Postgres (requires Docker)
test-integration:
    go test -tags=integration ./...

# Apply all pending database migrations (requires DATABASE_URL)
migrate-up:
    go run ./cmd/migrate up

# Roll back all database migrations (requires DATABASE_URL)
migrate-down:
    go run ./cmd/migrate down
