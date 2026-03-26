package dbadapter

import (
	"context"
	"time"

	"github.com/The127/cuttlegate/internal/domain"
	"github.com/The127/cuttlegate/internal/domain/ports"
)

// PostgresEvaluationEventRepository implements ports.EvaluationEventRepository using PostgreSQL.
type PostgresEvaluationEventRepository struct {
	db DBTX
}

// NewPostgresEvaluationEventRepository constructs a PostgresEvaluationEventRepository.
func NewPostgresEvaluationEventRepository(db DBTX) *PostgresEvaluationEventRepository {
	return &PostgresEvaluationEventRepository{db: db}
}

var _ ports.EvaluationEventRepository = (*PostgresEvaluationEventRepository)(nil)

// Publish persists a single evaluation event. Best-effort — callers must not propagate
// errors to the eval response path.
func (r *PostgresEvaluationEventRepository) Publish(ctx context.Context, event *domain.EvaluationEvent) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO evaluation_events
		 (id, flag_key, project_id, environment_id, user_id, input_context,
		  matched_rule_id, matched_rule_name, variant_key, reason, occurred_at)
		 VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11)`,
		event.ID,
		event.FlagKey,
		event.ProjectID,
		event.EnvironmentID,
		event.UserID,
		event.InputContext,
		event.MatchedRuleID,
		event.MatchedRuleName,
		event.VariantKey,
		string(event.Reason),
		event.OccurredAt,
	)
	return err
}

// ListByFlagEnvironment returns evaluation events for a specific flag in a project+environment,
// newest first, with cursor-based pagination via filter.Before.
func (r *PostgresEvaluationEventRepository) ListByFlagEnvironment(
	ctx context.Context,
	projectID, environmentID, flagKey string,
	filter ports.EvaluationFilter,
) ([]*domain.EvaluationEvent, error) {
	filter = ports.NormalizeEvaluationFilter(filter)

	query := `SELECT id, flag_key, project_id, environment_id, user_id, input_context,
		matched_rule_id, matched_rule_name, variant_key, reason, occurred_at
		FROM evaluation_events
		WHERE project_id = $1 AND environment_id = $2 AND flag_key = $3`
	args := []any{projectID, environmentID, flagKey}
	argIdx := 4

	if !filter.Before.IsZero() {
		query += ` AND occurred_at < $` + itoa(argIdx)
		args = append(args, filter.Before)
		argIdx++
	}

	query += ` ORDER BY occurred_at DESC LIMIT $` + itoa(argIdx)
	args = append(args, filter.Limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*domain.EvaluationEvent, 0)
	for rows.Next() {
		var e domain.EvaluationEvent
		var reasonStr string
		if err := rows.Scan(
			&e.ID, &e.FlagKey, &e.ProjectID, &e.EnvironmentID,
			&e.UserID, &e.InputContext,
			&e.MatchedRuleID, &e.MatchedRuleName,
			&e.VariantKey, &reasonStr, &e.OccurredAt,
		); err != nil {
			return nil, err
		}
		e.Reason = domain.EvalReason(reasonStr)
		events = append(events, &e)
	}
	return events, rows.Err()
}

// DeleteOlderThan removes all evaluation events with occurred_at before cutoff.
// Used by the retention worker.
func (r *PostgresEvaluationEventRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM evaluation_events WHERE occurred_at < $1`,
		cutoff,
	)
	return err
}
