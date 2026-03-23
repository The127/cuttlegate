# Install just: https://github.com/casey/just#installation

# Activate git hooks from .githooks/ (run once after cloning)
install-hooks:
    git config core.hooksPath .githooks
    @echo "Hooks installed."

# Run the linter (Go only — for SDK linting use just lint-sdk)
lint:
    golangci-lint run ./...

# Type-check the frontend SPA (catches unterminated strings, missing imports, type errors)
lint-web:
    cd web && npx tsc --noEmit -p tsconfig.app.json

# Run ESLint over the JS/TS SDK
lint-sdk:
    cd sdk/js && npm run lint

# Build the JS/TS SDK
build-sdk:
    cd sdk/js && npm run build

# Run the Go SDK tests (sdk/go is a separate module — must cd in)
test-sdk-go:
    cd sdk/go && go test -race ./...

# Run the JS/TS SDK tests
test-sdk-js:
    cd sdk/js && npm test

# Run the Python SDK tests
# Requires dev deps: pip install -e '.[dev]' from sdk/python/
test-sdk-python:
    cd sdk/python && python -m pytest

# Run all SDK test suites in sequence (Go → JS/TS → Python)
# Prerequisites: node_modules in sdk/js/ (npm install); pip install -e '.[dev]' in sdk/python/
test-sdk: test-sdk-go test-sdk-js test-sdk-python

# Run frontend component tests (Vitest + Testing Library)
test-frontend:
    cd web && npx vitest run

# Build the server binary into build/server
build:
    go build -o build/server ./cmd/server

# Run lint and all tests in sequence — mirrors CI exactly
ci: lint lint-web test test-integration test-sdk

# Run E2E tests against the full stack
# Builds the SPA, embeds it into the server binary, then Playwright starts Postgres, OIDC stub, and server.
# Requires Docker or Podman socket (same requirement as test-integration).
test-e2e:
    cd web && npm ci && npx vite build
    rm -rf cmd/server/web && mkdir -p cmd/server/web && cp -r web/dist cmd/server/web/dist
    go build -tags frontend -o e2e/bin/server ./cmd/server
    rm -rf cmd/server/web
    cd e2e && npm ci && npx playwright test

# Run all unit tests (includes architecture import rules)
# -p 4: run up to 4 packages in parallel. Safe for unit tests (no shared state).
# Chosen for GitHub Actions 2-core (4 logical) runners; adjust if runner spec changes.
test:
    go test -p 4 ./...

# Run integration tests against a real Postgres (requires Docker or Podman socket)
# Locally with Podman: systemctl --user start podman.socket
# CI (GitHub Actions ubuntu-latest): Docker daemon is available automatically
# -p 2: run up to 2 packages in parallel. Conservative limit — each integration test
# package may spin up Postgres containers, so keep this low to avoid exhausting the
# 2-core CI runner. Currently only one integration package exists (adapters/db), so
# this is a guardrail for when a second package appears.
test-integration:
    go test -tags=integration -p 2 ./...

# Apply all pending database migrations (requires DATABASE_URL)
migrate-up:
    go run ./cmd/migrate up

# Roll back all database migrations (requires DATABASE_URL)
migrate-down:
    go run ./cmd/migrate down

# Start everything for local dev: Postgres (5433), Keyline OIDC (5002), Go server with hot reload, and Vite (5173)
# Requires: docker or podman-compose
# Port 5433 is used instead of 5432 to avoid conflicts with other local Postgres instances.
# Override the server port via env: ADDR=:9090 just dev
dev:
    #!/usr/bin/env bash
    set -euo pipefail
    docker compose up -d db keyline-db keyline keyline-ui
    echo "Waiting for Postgres..."
    until docker compose exec -T db pg_isready -U cuttlegate -d cuttlegate -q; do sleep 1; done
    echo "Waiting for Keyline OIDC..."
    until curl -sf http://localhost:5002/oidc/cuttlegate/.well-known/openid-configuration > /dev/null 2>&1; do sleep 1; done
    echo "Keyline ready."
    trap 'kill 0' EXIT
    api_port="${ADDR:-:8080}"
    api_port="${api_port#:}"
    OIDC_ISSUER=http://localhost:5002/oidc/cuttlegate \
    OIDC_CLIENT_ID=cuttlegate \
    OIDC_REDIRECT_URI=http://localhost:5173/auth/callback \
    OIDC_ROLE_CLAIM=application_roles \
    DATABASE_URL=postgres://cuttlegate:cuttlegate@localhost:5433/cuttlegate?sslmode=disable \
    AUTO_MIGRATE=true \
    go run github.com/air-verse/air@latest &
    cd web && npm install --silent && API_PORT="$api_port" npm run dev &
    wait

# Smoke test the full docker-compose stack: boot, verify endpoints, tear down
smoke:
    ./scripts/smoke.sh

# Start the full stack (server + Postgres + Dex) via docker-compose
up:
    docker compose up --build

# Stop all containers
down:
    docker compose down

# Wipe the database volume and stop containers (use when migrations are dirty)
reset:
    docker compose down -v

# Start the docs dev server (hot-reload preview at http://localhost:3000/cuttlegate/)
docs-dev:
    cd site && npm run start

# Build the docs site (output to site/build/)
docs-build:
    cd site && npm ci && npm run build

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
    just index-web

# Remove worktrees for branches already merged to main, then prune stale metadata.
# Run after every merge to keep the worktree list clean.
clean-worktrees:
    #!/usr/bin/env bash
    set -euo pipefail
    git fetch --prune
    git worktree list --porcelain \
      | awk '/^worktree / { wt=$2 } /^branch / { branch=$2 } branch != "" && wt != "$(git rev-parse --show-toplevel)" { print wt, branch }' \
      | while read -r wt branch; do
          short="${branch#refs/heads/}"
          if git branch -r | grep -q "origin/${short}$"; then
              merged=$(git branch -r --merged origin/main | grep -c "origin/${short}$" || true)
          else
              merged=1
          fi
          if [[ "$merged" -gt 0 ]]; then
              echo "Removing worktree $wt (branch $short merged or gone)"
              git worktree remove --force "$wt" 2>/dev/null || true
              git branch -d "$short" 2>/dev/null || true
          fi
      done
    git worktree prune
    echo "Worktrees cleaned."

# Generate a frontend orientation index for AI sessions (writes to docs/frontend-index.md)
# Covers routes, components, hooks, and API surface of the web/ SPA.
index-web:
    #!/usr/bin/env bash
    set -euo pipefail
    [[ -x web/node_modules/.bin/tsx ]] || npm --prefix web install --silent
    web/node_modules/.bin/tsx web/scripts/gen-frontend-index.ts
