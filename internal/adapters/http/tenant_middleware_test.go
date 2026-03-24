package httpadapter_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
	httpadapter "github.com/karo/cuttlegate/internal/adapters/http"
	"github.com/karo/cuttlegate/internal/domain"
)

// stubProjectResolver returns a fixed project for any slug.
type stubProjectResolver struct {
	project *domain.Project
	err     error
}

func (s *stubProjectResolver) GetBySlug(_ context.Context, _ string) (*domain.Project, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.project, nil
}

func TestTenantRLS_SetsTenantTxInContext(t *testing.T) {
	// This test verifies that the middleware stores a *sql.Tx in context
	// that downstream handlers can retrieve via dbadapter.TenantTxFromContext.
	// We can't easily test the actual SET LOCAL without a real Postgres,
	// so we only test the context plumbing and the commit/rollback logic.

	// We need a real *sql.DB to call BeginTx — use an in-memory stub if
	// available, otherwise skip. Since we can't easily get a *sql.DB without
	// a real connection, this test is more of a compilation check + integration
	// test placeholder.
	t.Skip("requires real Postgres connection — covered by integration tests")
}

func TestTenantRLS_PassesThroughWithoutSlug(t *testing.T) {
	proj := &domain.Project{ID: "proj-1", Slug: "test-project"}
	resolver := &stubProjectResolver{project: proj}

	// nil db is fine since we won't reach BeginTx when slug is empty
	middleware := httpadapter.TenantRLS(nil, resolver)

	var called bool
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// No tenant tx should be set
		_, ok := dbadapter.TenantTxFromContext(r.Context())
		if ok {
			t.Error("expected no tenant tx when slug is empty")
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Request with no slug param
	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler was not called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestTenantRLS_ReturnsErrorWhenProjectNotFound(t *testing.T) {
	resolver := &stubProjectResolver{err: domain.ErrNotFound}

	middleware := httpadapter.TenantRLS(nil, resolver)

	var called bool
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	// Use a mux to set the slug path value
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/projects/{slug}/flags", handler)

	req := httptest.NewRequest("GET", "/api/v1/projects/nonexistent/flags", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if called {
		t.Error("handler should not have been called")
	}
	if rec.Code == http.StatusOK {
		t.Error("expected error status, got 200")
	}
}

func TestTenantRLS_BeginsTransaction(t *testing.T) {
	// Integration test — requires a real Postgres. Verify:
	// 1. A tenant-scoped tx is stored in context
	// 2. SET LOCAL app.project_id was called on it
	// 3. Queries within the tx respect RLS

	dsn := testDSN(t)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	proj := &domain.Project{ID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", Slug: "test-project"}
	resolver := &stubProjectResolver{project: proj}

	middleware := httpadapter.TenantRLS(db, resolver)

	var gotTx bool
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tx, ok := dbadapter.TenantTxFromContext(r.Context())
		gotTx = ok
		if !ok {
			t.Error("expected tenant tx in context")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Verify the GUC was set
		var projectID string
		err := tx.QueryRowContext(r.Context(), "SELECT current_setting('app.project_id', true)").Scan(&projectID)
		if err != nil {
			t.Errorf("query current_setting: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if projectID != proj.ID {
			t.Errorf("expected project_id=%q, got %q", proj.ID, projectID)
		}

		w.WriteHeader(http.StatusOK)
	}))

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/projects/{slug}/flags", handler)

	req := httptest.NewRequest("GET", "/api/v1/projects/test-project/flags", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if !gotTx {
		t.Error("handler did not receive tenant tx")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// testDSN returns a Postgres DSN for testing, or skips the test if unavailable.
func testDSN(t *testing.T) string {
	t.Helper()
	dsn := "postgres://cuttlegate:cuttlegate@localhost:5432/cuttlegate_test?sslmode=disable"
	// Try to connect — skip if not available
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("skipping integration test: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("skipping integration test: db unreachable: %v", err)
	}
	db.Close()
	return dsn
}
