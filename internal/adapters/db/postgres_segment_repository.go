package dbadapter

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// PostgresSegmentRepository implements ports.SegmentRepository using PostgreSQL.
type PostgresSegmentRepository struct {
	db *sql.DB
}

// NewPostgresSegmentRepository constructs a PostgresSegmentRepository.
func NewPostgresSegmentRepository(db *sql.DB) *PostgresSegmentRepository {
	return &PostgresSegmentRepository{db: db}
}

var _ ports.SegmentRepository = (*PostgresSegmentRepository)(nil)

func (r *PostgresSegmentRepository) conn(ctx context.Context) DBTX {
	return TenantDBTX(ctx, r.db)
}

func (r *PostgresSegmentRepository) Create(ctx context.Context, segment *domain.Segment) error {
	_, err := r.conn(ctx).ExecContext(ctx,
		`INSERT INTO segments (id, slug, name, project_id, created_at) VALUES ($1, $2, $3, $4, $5)`,
		segment.ID, segment.Slug, segment.Name, segment.ProjectID, segment.CreatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pgUniqueViolation {
			return domain.ErrConflict
		}
		return err
	}
	return nil
}

func (r *PostgresSegmentRepository) GetBySlug(ctx context.Context, projectID, slug string) (*domain.Segment, error) {
	row := r.conn(ctx).QueryRowContext(ctx,
		`SELECT id, slug, name, project_id, created_at FROM segments WHERE project_id = $1 AND slug = $2`,
		projectID, slug,
	)
	var s domain.Segment
	if err := row.Scan(&s.ID, &s.Slug, &s.Name, &s.ProjectID, &s.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *PostgresSegmentRepository) List(ctx context.Context, projectID string) ([]*domain.Segment, error) {
	rows, err := r.conn(ctx).QueryContext(ctx,
		`SELECT id, slug, name, project_id, created_at FROM segments WHERE project_id = $1 ORDER BY created_at`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	segments := make([]*domain.Segment, 0)
	for rows.Next() {
		var s domain.Segment
		if err := rows.Scan(&s.ID, &s.Slug, &s.Name, &s.ProjectID, &s.CreatedAt); err != nil {
			return nil, err
		}
		segments = append(segments, &s)
	}
	return segments, rows.Err()
}

func (r *PostgresSegmentRepository) ListWithCount(ctx context.Context, projectID string) ([]*ports.SegmentWithCount, error) {
	rows, err := r.conn(ctx).QueryContext(ctx, `
		SELECT s.id, s.slug, s.name, s.project_id, s.created_at, COUNT(sm.user_key) AS member_count
		FROM segments s
		LEFT JOIN segment_members sm ON sm.segment_id = s.id
		WHERE s.project_id = $1
		GROUP BY s.id
		ORDER BY s.created_at`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*ports.SegmentWithCount, 0)
	for rows.Next() {
		var s domain.Segment
		var count int
		if err := rows.Scan(&s.ID, &s.Slug, &s.Name, &s.ProjectID, &s.CreatedAt, &count); err != nil {
			return nil, err
		}
		items = append(items, &ports.SegmentWithCount{Segment: &s, MemberCount: count})
	}
	return items, rows.Err()
}

func (r *PostgresSegmentRepository) UpdateName(ctx context.Context, id, name string) error {
	res, err := r.conn(ctx).ExecContext(ctx, `UPDATE segments SET name = $1 WHERE id = $2`, name, id)
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

func (r *PostgresSegmentRepository) Delete(ctx context.Context, id string) error {
	res, err := r.conn(ctx).ExecContext(ctx, `DELETE FROM segments WHERE id = $1`, id)
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

// SetMembers bulk-replaces all members of a segment in a single transaction.
// An empty userKeys slice clears all members.
func (r *PostgresSegmentRepository) SetMembers(ctx context.Context, segmentID string, userKeys []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM segment_members WHERE segment_id = $1`, segmentID); err != nil {
		return err
	}
	for _, key := range userKeys {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO segment_members (segment_id, user_key) VALUES ($1, $2)`,
			segmentID, key,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *PostgresSegmentRepository) ListMembers(ctx context.Context, segmentID string) ([]string, error) {
	rows, err := r.conn(ctx).QueryContext(ctx,
		`SELECT user_key FROM segment_members WHERE segment_id = $1 ORDER BY added_at`,
		segmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]string, 0)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		members = append(members, key)
	}
	return members, rows.Err()
}

func (r *PostgresSegmentRepository) IsMember(ctx context.Context, segmentID string, userKey string) (bool, error) {
	var exists bool
	err := r.conn(ctx).QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM segment_members WHERE segment_id = $1 AND user_key = $2)`,
		segmentID, userKey,
	).Scan(&exists)
	return exists, err
}
