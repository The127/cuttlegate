package dbadapter

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
)

// pgUniqueViolation is the PostgreSQL error code for unique constraint violations.
const pgUniqueViolation = pq.ErrorCode("23505") //nolint:staticcheck

// DBTX is the common interface satisfied by both *sql.DB and *sql.Tx.
// Repository adapters that participate in unit-of-work transactions accept
// this interface instead of *sql.DB directly.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
