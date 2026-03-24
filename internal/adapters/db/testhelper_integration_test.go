//go:build integration

// Integration test helper for DB adapter tests.
//
// To add an integration test for a new DB adapter:
//  1. Create adapters/db/<name>_integration_test.go with //go:build integration
//  2. Call db := newTestDB(t) to get a migrated *sql.DB backed by a real Postgres container
//  3. Group assertions that share setup in one test function — each newTestDB call starts a
//     container (~3–5s); avoid calling it for a single trivial assertion
//  4. No data cleanup needed in t.Cleanup — the container is destroyed when the test ends
//  5. Run with: just test-integration (requires Docker or Podman socket — see Justfile)

package dbadapter_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
	testcontainerspg "github.com/testcontainers/testcontainers-go/modules/postgres"

	dbmigrations "github.com/karo/cuttlegate/db"
)

// TestMain disables the Ryuk reaper before any test runs. Ryuk is testcontainers-go's
// cleanup daemon; it fails on Podman (which this project uses locally). Our tests already
// terminate containers explicitly via t.Cleanup, so Ryuk is not needed.
func TestMain(m *testing.M) {
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true") //nolint:errcheck
	os.Exit(m.Run())
}

// newTestDB starts a fresh Postgres container, runs all migrations against it,
// and returns a connected *sql.DB. The container is terminated when t completes.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	ctr, err := testcontainerspg.Run(ctx, "postgres:16-alpine",
		testcontainerspg.WithDatabase("cuttlegate"),
		testcontainerspg.WithUsername("cuttlegate"),
		testcontainerspg.WithPassword("cuttlegate"),
		testcontainerspg.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	// Cleanups run LIFO: db.Close() (registered below) runs before Terminate.
	// This ensures the connection is closed before the container stops.
	t.Cleanup(func() {
		if err := ctr.Terminate(ctx); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
	})

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}

	src, err := iofs.New(dbmigrations.FS, "migrations")
	if err != nil {
		t.Fatalf("migrations source: %v", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		t.Fatalf("migrate init: %v", err)
	}
	defer m.Close() //nolint:errcheck

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("migrate up: %v", err)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return db
}

// newTestDBWithDSN is like newTestDB but also returns the connection DSN,
// needed for creating additional connections as different roles (e.g. RLS tests).
func newTestDBWithDSN(t *testing.T) (*sql.DB, string) {
	t.Helper()
	ctx := context.Background()

	ctr, err := testcontainerspg.Run(ctx, "postgres:16-alpine",
		testcontainerspg.WithDatabase("cuttlegate"),
		testcontainerspg.WithUsername("cuttlegate"),
		testcontainerspg.WithPassword("cuttlegate"),
		testcontainerspg.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := ctr.Terminate(ctx); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
	})

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}

	src, err := iofs.New(dbmigrations.FS, "migrations")
	if err != nil {
		t.Fatalf("migrations source: %v", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		t.Fatalf("migrate init: %v", err)
	}
	defer m.Close() //nolint:errcheck

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("migrate up: %v", err)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return db, dsn
}
