package dbadapter

import (
	"context"
	"database/sql"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// PostgresAuditRepository implements ports.AuditRepository using PostgreSQL.
type PostgresAuditRepository struct {
	db *sql.DB
}

// NewPostgresAuditRepository constructs a PostgresAuditRepository.
func NewPostgresAuditRepository(db *sql.DB) *PostgresAuditRepository {
	return &PostgresAuditRepository{db: db}
}

var _ ports.AuditRepository = (*PostgresAuditRepository)(nil)

func (r *PostgresAuditRepository) Record(ctx context.Context, event *domain.AuditEvent) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_events (id, project_id, actor_id, action, entity_type, entity_id, entity_key, before_state, after_state, occurred_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		event.ID, event.ProjectID, event.ActorID, event.Action,
		event.EntityType, event.EntityID, event.EntityKey,
		event.BeforeState, event.AfterState, event.OccurredAt,
	)
	return err
}

func (r *PostgresAuditRepository) ListByProject(ctx context.Context, projectID string, filter domain.AuditFilter) ([]*domain.AuditEvent, error) {
	filter = domain.NormalizeAuditFilter(filter)

	query := `SELECT id, project_id, actor_id, action, entity_type, entity_id, entity_key, before_state, after_state, occurred_at
		FROM audit_events WHERE project_id = $1`
	args := []any{projectID}
	argIdx := 2

	if filter.FlagKey != "" {
		query += ` AND entity_key = $` + itoa(argIdx)
		args = append(args, filter.FlagKey)
		argIdx++
	}
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

	events := make([]*domain.AuditEvent, 0)
	for rows.Next() {
		var e domain.AuditEvent
		if err := rows.Scan(
			&e.ID, &e.ProjectID, &e.ActorID, &e.Action,
			&e.EntityType, &e.EntityID, &e.EntityKey,
			&e.BeforeState, &e.AfterState, &e.OccurredAt,
		); err != nil {
			return nil, err
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}

// itoa converts a small int to a string without importing strconv.
func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
