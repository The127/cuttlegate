package httpadapter

import (
	"database/sql"
	"log/slog"
	"net/http"

	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
)

// TenantRLS returns middleware that sets the Postgres app.project_id GUC
// per request so that row-level security policies scope queries to the
// resolved project.
//
// It resolves the project slug from the URL (r.PathValue("slug")), looks
// up the project ID via the provided projectResolver, begins a transaction,
// calls SET LOCAL app.project_id, and stores the *sql.Tx in the request
// context. Downstream repos retrieve it via dbadapter.TenantDBTX.
//
// The transaction is committed if the handler returns without error and
// rolled back otherwise.
func TenantRLS(db *sql.DB, projects projectResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slug := r.PathValue("slug")
			if slug == "" {
				// No project slug in this route — pass through.
				next.ServeHTTP(w, r)
				return
			}

			proj, err := projects.GetBySlug(r.Context(), slug)
			if err != nil {
				WriteError(w, err)
				return
			}

			tx, err := db.BeginTx(r.Context(), nil)
			if err != nil {
				slog.ErrorContext(r.Context(), "tenant_rls: begin tx", "err", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if err := dbadapter.SetTenantContext(r.Context(), tx, proj.ID); err != nil {
				_ = tx.Rollback()
				slog.ErrorContext(r.Context(), "tenant_rls: set tenant context", "err", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			ctx := dbadapter.WithTenantTx(r.Context(), tx)
			// Use a response interceptor to detect if the handler wrote an error.
			rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r.WithContext(ctx))

			if rw.status >= 400 {
				_ = tx.Rollback()
			} else {
				if err := tx.Commit(); err != nil {
					slog.ErrorContext(r.Context(), "tenant_rls: commit", "err", err)
				}
			}
		})
	}
}

// statusRecorder wraps an http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.wroteHeader = true
	}
	return r.ResponseWriter.Write(b)
}
