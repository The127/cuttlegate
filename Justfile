# Install just: https://github.com/casey/just#installation

# Activate git hooks from .githooks/ (run once after cloning)
install-hooks:
    git config core.hooksPath .githooks
    @echo "Hooks installed."

# Run the linter (Go only — for SDK linting use just lint-sdk)
lint:
    golangci-lint run ./...

# Run ESLint over the JS/TS SDK
lint-sdk:
    cd sdk/js && npm run lint

# Build the JS/TS SDK
build-sdk:
    cd sdk/js && npm run build

# Run the JS/TS SDK tests
test-sdk:
    cd sdk/js && npm test

# Build the server binary into build/server
build:
    go build -o build/server ./cmd/server

# Run lint and all tests in sequence — mirrors CI exactly
ci: lint test test-integration

# Run E2E tests against the full stack
# Builds the server binary, then Playwright global setup starts Postgres, OIDC stub, and server.
# Requires Docker or Podman socket (same requirement as test-integration).
test-e2e:
    go build -o e2e/bin/server ./cmd/server
    cd e2e && npm ci && npx playwright test

# Run all unit tests (includes architecture import rules)
test:
    go test ./...

# Run integration tests against a real Postgres (requires Docker or Podman socket)
# Locally with Podman: systemctl --user start podman.socket
# CI (GitHub Actions ubuntu-latest): Docker daemon is available automatically
test-integration:
    go test -tags=integration ./...

# Apply all pending database migrations (requires DATABASE_URL)
migrate-up:
    go run ./cmd/migrate up

# Roll back all database migrations (requires DATABASE_URL)
migrate-down:
    go run ./cmd/migrate down

# Start everything for local dev: Postgres, Go server (hot reload), and Vite (port 5173)
# Requires: docker or podman-compose
# Requires a 'cuttlegate' application registered in Keyline (https://keyline.karo.gay)
dev:
    #!/usr/bin/env bash
    set -euo pipefail
    docker compose -f docker-compose.dev.yml up -d
    echo "Waiting for Postgres..."
    until bash -c 'echo > /dev/tcp/localhost/5432' 2>/dev/null; do sleep 1; done
    trap 'kill 0' EXIT
    OIDC_ISSUER=https://keyline-api.karo.gay/oidc/keyline \
    OIDC_CLIENT_ID=cuttlegate \
    OIDC_REDIRECT_URI=http://localhost:5173/auth/callback \
    DATABASE_URL=postgres://cuttlegate:cuttlegate@localhost:5432/cuttlegate?sslmode=disable \
    AUTO_MIGRATE=true \
    go run github.com/air-verse/air@latest &
    cd web && npm run dev &
    wait

# Stop the dev Postgres container
dev-down:
    docker compose -f docker-compose.dev.yml down

# Wipe the dev database volume and stop containers (use when migrations are dirty)
dev-reset:
    docker compose -f docker-compose.dev.yml down -v

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
