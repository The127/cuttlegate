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

# Generate a codebase orientation index for AI sessions (writes to docs/codebase-index.md)
# Read this file at the start of a new session instead of grepping individual files.
index:
    #!/usr/bin/env bash
    set -euo pipefail
    OUT=docs/codebase-index.md
    {
      echo "# Codebase Index"
      echo ""
      echo "_Generated $(date -u '+%Y-%m-%d %H:%M UTC'). Read this at session start for orientation._"
      echo ""
      echo "## Packages"
      echo '```'
      go list ./... | sed 's|.*/cuttlegate/||' | sort
      echo '```'
      echo ""
      echo "## Domain ports — interfaces (internal/domain/ports/)"
      echo '```'
      grep -rh "type [A-Z][A-Za-z]* interface" internal/domain/ports/ 2>/dev/null | sed 's/^[[:space:]]*//' | sort
      echo '```'
      echo ""
      echo "## Domain types — structs (internal/domain/)"
      echo '```'
      find internal/domain -maxdepth 1 -name "*.go" ! -name "*_test.go" | xargs grep -h "^type [A-Z][A-Za-z]* struct" 2>/dev/null | sort
      echo '```'
      echo ""
      echo "## App services (internal/app/)"
      echo '```'
      find internal/app -maxdepth 1 -name "*.go" ! -name "*_test.go" | xargs grep -h "^type [A-Z][A-Za-z]* struct" 2>/dev/null | sort
      echo '```'
      echo ""
      echo "## HTTP handlers & middleware (internal/adapters/http/)"
      echo '```'
      find internal/adapters/http -maxdepth 1 -name "*.go" ! -name "*_test.go" | xargs grep -h "^type [A-Z][A-Za-z]* struct" 2>/dev/null | sort
      echo '```'
      echo ""
      echo "## DB adapters (internal/adapters/db/)"
      echo '```'
      find internal/adapters/db -maxdepth 1 -name "*.go" ! -name "*_test.go" | xargs grep -h "^type [A-Z][A-Za-z]* struct" 2>/dev/null | sort
      echo '```'
    } > "$OUT"
    echo "Index written to $OUT"
