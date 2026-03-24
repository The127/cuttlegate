package dbadapter

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// PostgresEnvironmentRepository implements ports.EnvironmentRepository using PostgreSQL.
type PostgresEnvironmentRepository struct {
	db *sql.DB
}

// NewPostgresEnvironmentRepository constructs a PostgresEnvironmentRepository.
func NewPostgresEnvironmentRepository(db *sql.DB) *PostgresEnvironmentRepository {
	return &PostgresEnvironmentRepository{db: db}
}

var _ ports.EnvironmentRepository = (*PostgresEnvironmentRepository)(nil)

func (r *PostgresEnvironmentRepository) Create(ctx context.Context, env domain.Environment) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO environments (id, project_id, name, slug, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		env.ID, env.ProjectID, env.Name, env.Slug, env.CreatedAt,
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

func (r *PostgresEnvironmentRepository) GetBySlug(ctx context.Context, projectID, slug string) (*domain.Environment, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, slug, created_at FROM environments
		 WHERE project_id = $1 AND slug = $2`,
		projectID, slug,
	)
	var e domain.Environment
	if err := row.Scan(&e.ID, &e.ProjectID, &e.Name, &e.Slug, &e.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}

func (r *PostgresEnvironmentRepository) ListByProject(ctx context.Context, projectID string) ([]*domain.Environment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, name, slug, created_at FROM environments
		 WHERE project_id = $1 ORDER BY created_at ASC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	envs := make([]*domain.Environment, 0)
	for rows.Next() {
		var e domain.Environment
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.Name, &e.Slug, &e.CreatedAt); err != nil {
			return nil, err
		}
		envs = append(envs, &e)
	}
	return envs, rows.Err()
}

func (r *PostgresEnvironmentRepository) UpdateName(ctx context.Context, id, name string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE environments SET name = $1 WHERE id = $2`, name, id)
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

func (r *PostgresEnvironmentRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM environments WHERE id = $1`, id)
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
