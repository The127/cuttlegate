package dbadapter

import (
	"context"
	"database/sql"
	"sync"

	"github.com/The127/cuttlegate/internal/domain/ports"
)

// PostgresUnitOfWork implements ports.UnitOfWork by wrapping a *sql.Tx
// and constructing transaction-scoped repository instances.
type PostgresUnitOfWork struct {
	tx   *sql.Tx
	done bool
	mu   sync.Mutex
}

var _ ports.UnitOfWork = (*PostgresUnitOfWork)(nil)

func (u *PostgresUnitOfWork) FlagEnvironmentStateRepository() ports.FlagEnvironmentStateRepository {
	return newFlagEnvStateRepoFromTx(u.tx)
}

func (u *PostgresUnitOfWork) RuleRepository() ports.RuleRepository {
	return NewPostgresRuleRepository(u.tx)
}

func (u *PostgresUnitOfWork) AuditRepository() ports.AuditRepository {
	return NewPostgresAuditRepository(u.tx)
}

func (u *PostgresUnitOfWork) Commit(ctx context.Context) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.done {
		return sql.ErrTxDone
	}
	u.done = true
	return u.tx.Commit()
}

func (u *PostgresUnitOfWork) Rollback(_ context.Context) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.done {
		return nil // no-op after Commit or previous Rollback
	}
	u.done = true
	return u.tx.Rollback()
}

// PostgresUnitOfWorkFactory implements ports.UnitOfWorkFactory using a *sql.DB.
type PostgresUnitOfWorkFactory struct {
	db *sql.DB
}

var _ ports.UnitOfWorkFactory = (*PostgresUnitOfWorkFactory)(nil)

// NewPostgresUnitOfWorkFactory constructs a PostgresUnitOfWorkFactory.
func NewPostgresUnitOfWorkFactory(db *sql.DB) *PostgresUnitOfWorkFactory {
	return &PostgresUnitOfWorkFactory{db: db}
}

func (f *PostgresUnitOfWorkFactory) Begin(ctx context.Context) (ports.UnitOfWork, error) {
	tx, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &PostgresUnitOfWork{tx: tx}, nil
}
