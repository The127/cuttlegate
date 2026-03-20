package dbadapter

import (
	"context"
	"database/sql"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// PostgresFlagEnvironmentStateRepository implements ports.FlagEnvironmentStateRepository using PostgreSQL.
type PostgresFlagEnvironmentStateRepository struct {
	db *sql.DB
}

// NewPostgresFlagEnvironmentStateRepository constructs a PostgresFlagEnvironmentStateRepository.
func NewPostgresFlagEnvironmentStateRepository(db *sql.DB) *PostgresFlagEnvironmentStateRepository {
	return &PostgresFlagEnvironmentStateRepository{db: db}
}

var _ ports.FlagEnvironmentStateRepository = (*PostgresFlagEnvironmentStateRepository)(nil)

func (r *PostgresFlagEnvironmentStateRepository) CreateBatch(ctx context.Context, states []*domain.FlagEnvironmentState) error {
	if len(states) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO flag_environment_states (flag_id, environment_id, enabled) VALUES ($1, $2, $3)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close() //nolint:errcheck

	for _, s := range states {
		if _, err := stmt.ExecContext(ctx, s.FlagID, s.EnvironmentID, s.Enabled); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *PostgresFlagEnvironmentStateRepository) ListByEnvironment(ctx context.Context, environmentID string) ([]*domain.FlagEnvironmentState, error) {
	rows, err := r.db.QueryContext(ctx,
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

func (r *PostgresFlagEnvironmentStateRepository) SetEnabled(ctx context.Context, flagID, environmentID string, enabled bool) error {
	res, err := r.db.ExecContext(ctx,
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
