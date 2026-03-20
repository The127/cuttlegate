package dbadapter

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// PostgresProjectRepository implements ports.ProjectRepository using PostgreSQL.
type PostgresProjectRepository struct {
	db *sql.DB
}

// NewPostgresProjectRepository constructs a PostgresProjectRepository.
func NewPostgresProjectRepository(db *sql.DB) *PostgresProjectRepository {
	return &PostgresProjectRepository{db: db}
}

var _ ports.ProjectRepository = (*PostgresProjectRepository)(nil)

// pgUniqueViolation is the PostgreSQL error code for unique constraint violations.
const pgUniqueViolation = pq.ErrorCode("23505")

func (r *PostgresProjectRepository) Create(ctx context.Context, p domain.Project) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO projects (id, name, slug, created_at) VALUES ($1, $2, $3, $4)`,
		p.ID, p.Name, p.Slug, p.CreatedAt,
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

func (r *PostgresProjectRepository) GetBySlug(ctx context.Context, slug string) (*domain.Project, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, slug, created_at FROM projects WHERE slug = $1`,
		slug,
	)
	var p domain.Project
	if err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *PostgresProjectRepository) List(ctx context.Context) ([]*domain.Project, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, slug, created_at FROM projects ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]*domain.Project, 0)
	for rows.Next() {
		var p domain.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, &p)
	}
	return projects, rows.Err()
}

func (r *PostgresProjectRepository) UpdateName(ctx context.Context, id, name string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE projects SET name = $1 WHERE id = $2`, name, id)
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

func (r *PostgresProjectRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id)
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
