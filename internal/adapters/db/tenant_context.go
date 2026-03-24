package dbadapter

import (
	"context"
	"database/sql"
)

type tenantTxKey struct{}

// WithTenantTx stores a tenant-scoped *sql.Tx in the context.
// Repos that query RLS-protected tables should use TenantDBTX to
// retrieve it, falling back to the pool when no tx is present.
func WithTenantTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, tenantTxKey{}, tx)
}

// TenantTxFromContext retrieves the tenant-scoped *sql.Tx from context.
// Returns nil, false if no tenant tx is set.
func TenantTxFromContext(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(tenantTxKey{}).(*sql.Tx)
	return tx, ok
}

// TenantDBTX returns the tenant-scoped *sql.Tx from context if present,
// otherwise returns the provided fallback (typically the pool *sql.DB).
func TenantDBTX(ctx context.Context, fallback DBTX) DBTX {
	if tx, ok := TenantTxFromContext(ctx); ok {
		return tx
	}
	return fallback
}
