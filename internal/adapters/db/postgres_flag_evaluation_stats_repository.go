package dbadapter

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// PostgresFlagEvaluationStatsRepository implements ports.FlagEvaluationStatsRepository using PostgreSQL.
type PostgresFlagEvaluationStatsRepository struct {
	db DBTX
}

// NewPostgresFlagEvaluationStatsRepository constructs a PostgresFlagEvaluationStatsRepository.
func NewPostgresFlagEvaluationStatsRepository(db DBTX) *PostgresFlagEvaluationStatsRepository {
	return &PostgresFlagEvaluationStatsRepository{db: db}
}

var _ ports.FlagEvaluationStatsRepository = (*PostgresFlagEvaluationStatsRepository)(nil)

// Upsert increments evaluation_count and updates last_evaluated_at for the
// given flag+environment. Creates the row if it does not exist.
func (r *PostgresFlagEvaluationStatsRepository) Upsert(ctx context.Context, flagID, environmentID, flagKey string, evaluatedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO flag_evaluation_stats (flag_id, environment_id, flag_key, evaluation_count, last_evaluated_at)
		VALUES ($1, $2, $3, 1, $4)
		ON CONFLICT (flag_id, environment_id) DO UPDATE
		SET evaluation_count  = flag_evaluation_stats.evaluation_count + 1,
		    last_evaluated_at = EXCLUDED.last_evaluated_at,
		    flag_key          = EXCLUDED.flag_key`,
		flagID, environmentID, flagKey, evaluatedAt,
	)
	return err
}

// GetByFlagEnvironment returns stats for the given flag+environment pair.
// If no evaluations have been recorded it returns a zero-count struct rather
// than ErrNotFound.
func (r *PostgresFlagEvaluationStatsRepository) GetByFlagEnvironment(ctx context.Context, flagID, environmentID string) (*domain.FlagEvaluationStats, error) {
	var s domain.FlagEvaluationStats
	err := r.db.QueryRowContext(ctx, `
		SELECT flag_id, environment_id, flag_key, evaluation_count, last_evaluated_at
		FROM flag_evaluation_stats
		WHERE flag_id = $1 AND environment_id = $2`,
		flagID, environmentID,
	).Scan(&s.FlagID, &s.EnvironmentID, &s.FlagKey, &s.EvaluationCount, &s.LastEvaluatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return &domain.FlagEvaluationStats{
			FlagID:        flagID,
			EnvironmentID: environmentID,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}
