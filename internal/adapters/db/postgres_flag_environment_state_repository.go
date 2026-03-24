package dbadapter

import (
	"context"
	"database/sql"
	"errors"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// PostgresFlagEnvironmentStateRepository implements ports.FlagEnvironmentStateRepository using PostgreSQL.
type PostgresFlagEnvironmentStateRepository struct {
	db       *sql.DB
	explicit DBTX // non-nil when constructed with an explicit tx (e.g. UnitOfWork)
}

// NewPostgresFlagEnvironmentStateRepository constructs a PostgresFlagEnvironmentStateRepository.
func NewPostgresFlagEnvironmentStateRepository(db *sql.DB) *PostgresFlagEnvironmentStateRepository {
	return &PostgresFlagEnvironmentStateRepository{db: db}
}

// newFlagEnvStateRepoFromTx constructs a repo scoped to an explicit transaction (used by UnitOfWork).
func newFlagEnvStateRepoFromTx(tx *sql.Tx) *PostgresFlagEnvironmentStateRepository {
	return &PostgresFlagEnvironmentStateRepository{explicit: tx}
}

var _ ports.FlagEnvironmentStateRepository = (*PostgresFlagEnvironmentStateRepository)(nil)

func (r *PostgresFlagEnvironmentStateRepository) conn(ctx context.Context) DBTX {
	if r.explicit != nil {
		return r.explicit
	}
	return TenantDBTX(ctx, r.db)
}

func (r *PostgresFlagEnvironmentStateRepository) CreateBatch(ctx context.Context, states []*domain.FlagEnvironmentState) error {
	for _, s := range states {
		if _, err := r.conn(ctx).ExecContext(ctx,
			`INSERT INTO flag_environment_states (flag_id, environment_id, enabled) VALUES ($1, $2, $3)`,
			s.FlagID, s.EnvironmentID, s.Enabled,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresFlagEnvironmentStateRepository) ListByEnvironment(ctx context.Context, environmentID string) ([]*domain.FlagEnvironmentState, error) {
	rows, err := r.conn(ctx).QueryContext(ctx,
		`SELECT flag_id, environment_id, enabled FROM flag_environment_states WHERE environment_id = $1`,
		environmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	states := make([]*domain.FlagEnvironmentState, 0)
	for rows.Next() {
		var s domain.FlagEnvironmentState
		if err := rows.Scan(&s.FlagID, &s.EnvironmentID, &s.Enabled); err != nil {
			return nil, err
		}
		states = append(states, &s)
	}
	return states, rows.Err()
}

func (r *PostgresFlagEnvironmentStateRepository) GetByFlagAndEnvironment(ctx context.Context, flagID, environmentID string) (*domain.FlagEnvironmentState, error) {
	var s domain.FlagEnvironmentState
	err := r.conn(ctx).QueryRowContext(ctx,
		`SELECT flag_id, environment_id, enabled FROM flag_environment_states WHERE flag_id = $1 AND environment_id = $2`,
		flagID, environmentID,
	).Scan(&s.FlagID, &s.EnvironmentID, &s.Enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresFlagEnvironmentStateRepository) Upsert(ctx context.Context, state *domain.FlagEnvironmentState) error {
	_, err := r.conn(ctx).ExecContext(ctx,
		`INSERT INTO flag_environment_states (flag_id, environment_id, enabled) VALUES ($1, $2, $3)
		 ON CONFLICT (flag_id, environment_id) DO UPDATE SET enabled = EXCLUDED.enabled`,
		state.FlagID, state.EnvironmentID, state.Enabled,
	)
	return err
}

func (r *PostgresFlagEnvironmentStateRepository) SetEnabled(ctx context.Context, flagID, environmentID string, enabled bool) error {
	res, err := r.conn(ctx).ExecContext(ctx,
		`UPDATE flag_environment_states SET enabled = $1 WHERE flag_id = $2 AND environment_id = $3`,
		enabled, flagID, environmentID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
