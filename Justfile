# Install just: https://github.com/casey/just#installation

# Activate git hooks from .githooks/ (run once after cloning)
install-hooks:
    git config core.hooksPath .githooks
    @echo "Hooks installed."

# Run the linter
lint:
    golangci-lint run ./...

# Build the server binary into build/server
build:
    go build -o build/server ./cmd/server

# Run lint and all tests in sequence — mirrors CI exactly
ci: lint test test-integration

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
