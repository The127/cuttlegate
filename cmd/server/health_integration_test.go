//go:build integration

package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
	testcontainerspg "github.com/testcontainers/testcontainers-go/modules/postgres"

	dbmigrations "github.com/The127/cuttlegate/db"
)

func TestMain(m *testing.M) {
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true") //nolint:errcheck
	os.Exit(m.Run())
}

// newHealthTestDB starts a fresh Postgres container, runs migrations, and returns a *sql.DB.
// The container is terminated when t completes.
func newHealthTestDB(t *testing.T) *sql.DB {
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

	return db
}

// TestHealthHandler_Healthy covers the @happy scenario: DB reachable, 200 {"status":"ok"}.
func TestHealthHandler_Healthy(t *testing.T) {
	db := newHealthTestDB(t)

	handler := healthHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	res := rec.Result()
	defer res.Body.Close() //nolint:errcheck

	if res.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", res.StatusCode, http.StatusOK)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", ct, "application/json")
	}
	body := rec.Body.String()
	want := `{"status":"ok"}`
	if body != want {
		t.Errorf("body: got %q, want %q", body, want)
	}
}
