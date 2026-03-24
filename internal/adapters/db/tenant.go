package dbadapter

import (
	"context"
	"database/sql"
	"fmt"
)

// SetTenantContext sets the app.project_id GUC on the given connection or
// transaction so that Postgres RLS policies restrict access to the specified
// project. Must be called at the start of each request that touches
// tenant-scoped tables.
//
// Use SET LOCAL to scope the setting to the current transaction. For
// non-transactional queries, use ExecContext on a *sql.Conn obtained via
// db.Conn(ctx).
func SetTenantContext(ctx context.Context, execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}, projectID string) error {
	// Use a parameterised format string — the GUC value is set via
	// SET LOCAL which does not support $1 placeholders, so we must
	// interpolate. The projectID is a UUID validated at the domain layer,
	// but we still use fmt.Sprintf with %q for safety.
	query := fmt.Sprintf("SET LOCAL app.project_id = %s", quoteLiteral(projectID))
	_, err := execer.ExecContext(ctx, query)
	return err
}

// quoteLiteral quotes a string as a Postgres literal, escaping single quotes.
func quoteLiteral(s string) string {
	// Replace any single quotes with doubled single quotes, then wrap.
	escaped := ""
	for _, c := range s {
		if c == '\'' {
			escaped += "''"
		} else {
			escaped += string(c)
		}
	}
	return "'" + escaped + "'"
}
