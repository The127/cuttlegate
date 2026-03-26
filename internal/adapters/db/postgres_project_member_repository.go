package dbadapter

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"github.com/The127/cuttlegate/internal/domain"
	"github.com/The127/cuttlegate/internal/domain/ports"
)

// PostgresProjectMemberRepository implements ports.ProjectMemberRepository using PostgreSQL.
type PostgresProjectMemberRepository struct {
	db *sql.DB
}

// NewPostgresProjectMemberRepository constructs a PostgresProjectMemberRepository.
func NewPostgresProjectMemberRepository(db *sql.DB) *PostgresProjectMemberRepository {
	return &PostgresProjectMemberRepository{db: db}
}

var _ ports.ProjectMemberRepository = (*PostgresProjectMemberRepository)(nil)

func (r *PostgresProjectMemberRepository) conn(ctx context.Context) DBTX {
	return TenantDBTX(ctx, r.db)
}

func (r *PostgresProjectMemberRepository) AddMember(ctx context.Context, m *domain.ProjectMember) error {
	_, err := r.conn(ctx).ExecContext(ctx,
		`INSERT INTO project_members (project_id, user_id, role, created_at) VALUES ($1, $2, $3, $4)`,
		m.ProjectID, m.UserID, string(m.Role), m.CreatedAt,
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

func (r *PostgresProjectMemberRepository) ListMembers(ctx context.Context, projectID string) ([]*domain.ProjectMember, error) {
	rows, err := r.conn(ctx).QueryContext(ctx,
		`SELECT project_id, user_id, role, created_at FROM project_members WHERE project_id = $1 ORDER BY created_at`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]*domain.ProjectMember, 0)
	for rows.Next() {
		var m domain.ProjectMember
		var role string
		if err := rows.Scan(&m.ProjectID, &m.UserID, &role, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Role = domain.Role(role)
		members = append(members, &m)
	}
	return members, rows.Err()
}

func (r *PostgresProjectMemberRepository) UpdateRole(ctx context.Context, projectID, userID string, role domain.Role) error {
	res, err := r.conn(ctx).ExecContext(ctx,
		`UPDATE project_members SET role = $1 WHERE project_id = $2 AND user_id = $3`,
		string(role), projectID, userID,
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

func (r *PostgresProjectMemberRepository) RemoveMember(ctx context.Context, projectID, userID string) error {
	res, err := r.conn(ctx).ExecContext(ctx,
		`DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`,
		projectID, userID,
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
