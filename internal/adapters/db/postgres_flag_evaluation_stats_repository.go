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

// bucketInterval maps bucketSize to the PostgreSQL interval string for generate_series.
var bucketInterval = map[string]string{
	"day":  "1 day",
	"hour": "1 hour",
}

// GetBuckets returns time-bucketed evaluation counts for the given flag in a
// project+environment, starting from since. bucketSize must be "day" or "hour".
// Every time slot from since to now is present in the result (zero-filled via
// generate_series + LEFT JOIN). Rows with no events have Total==0 and Variants==nil.
func (r *PostgresFlagEvaluationStatsRepository) GetBuckets(ctx context.Context, projectID, environmentID, flagKey string, since time.Time, bucketSize string) ([]domain.EvaluationBucket, error) {
	interval, ok := bucketInterval[bucketSize]
	if !ok {
		return nil, errors.New("invalid bucketSize: " + bucketSize)
	}
	// bucketSize is "day" or "hour" — both are valid PostgreSQL date_trunc fields.
	trunc := bucketSize

	// generate_series produces one row per bucket slot.
	// LEFT JOIN brings in actual events grouped by (bucket, variant_key).
	// Rows with no events have NULL variant_key and NULL count.
	// The interval and trunc field are enumerated constants — safe to inline.
	query := `
		SELECT
			gs.bucket_ts,
			e.variant_key,
			COUNT(e.id) AS cnt
		FROM (
			SELECT generate_series(
				date_trunc('` + trunc + `', $1::timestamptz),
				date_trunc('` + trunc + `', NOW() AT TIME ZONE 'UTC'),
				'` + interval + `'::interval
			) AS bucket_ts
		) gs
		LEFT JOIN evaluation_events e
			ON date_trunc('` + trunc + `', e.occurred_at AT TIME ZONE 'UTC') = gs.bucket_ts
			AND e.project_id     = $2
			AND e.environment_id = $3
			AND e.flag_key       = $4
			AND e.occurred_at   >= $1
		GROUP BY gs.bucket_ts, e.variant_key
		ORDER BY gs.bucket_ts ASC, e.variant_key ASC`

	rows, err := r.db.QueryContext(ctx, query,
		since,         // $1 — window start
		projectID,     // $2
		environmentID, // $3
		flagKey,       // $4
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	// Aggregate (bucket_ts, variant_key|NULL, count) rows into per-bucket maps.
	bucketMap := make(map[time.Time]*domain.EvaluationBucket)
	var order []time.Time

	for rows.Next() {
		var ts time.Time
		var variantKey sql.NullString
		var cnt int64
		if err := rows.Scan(&ts, &variantKey, &cnt); err != nil {
			return nil, err
		}
		ts = ts.UTC()
		b, exists := bucketMap[ts]
		if !exists {
			b = &domain.EvaluationBucket{Timestamp: ts, Variants: map[string]int64{}}
			bucketMap[ts] = b
			order = append(order, ts)
		}
		if variantKey.Valid && variantKey.String != "" && cnt > 0 {
			b.Variants[variantKey.String] = cnt
			b.Total += cnt
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]domain.EvaluationBucket, len(order))
	for i, ts := range order {
		result[i] = *bucketMap[ts]
	}
	return result, nil
}
