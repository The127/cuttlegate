package dbadapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/lib/pq"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// PostgresFlagRepository implements ports.FlagRepository using PostgreSQL.
type PostgresFlagRepository struct {
	db *sql.DB
}

// NewPostgresFlagRepository constructs a PostgresFlagRepository.
func NewPostgresFlagRepository(db *sql.DB) *PostgresFlagRepository {
	return &PostgresFlagRepository{db: db}
}

var _ ports.FlagRepository = (*PostgresFlagRepository)(nil)

// dbVariant is the JSON-serialisable form of domain.Variant stored in the variants jsonb column.
type dbVariant struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

func marshalVariants(variants []domain.Variant) ([]byte, error) {
	dv := make([]dbVariant, len(variants))
	for i, v := range variants {
		dv[i] = dbVariant{Key: v.Key, Name: v.Name}
	}
	return json.Marshal(dv)
}

func unmarshalVariants(data []byte) ([]domain.Variant, error) {
	var dv []dbVariant
	if err := json.Unmarshal(data, &dv); err != nil {
		return nil, err
	}
	variants := make([]domain.Variant, len(dv))
	for i, v := range dv {
		variants[i] = domain.Variant{Key: v.Key, Name: v.Name}
	}
	return variants, nil
}

func (r *PostgresFlagRepository) Create(ctx context.Context, flag *domain.Flag) error {
	variants, err := marshalVariants(flag.Variants)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO flags (id, project_id, key, name, type, variants, default_variant_key, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		flag.ID, flag.ProjectID, flag.Key, flag.Name, string(flag.Type),
		variants, flag.DefaultVariantKey, flag.CreatedAt,
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

func (r *PostgresFlagRepository) GetByKey(ctx context.Context, projectID, key string) (*domain.Flag, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, key, name, type, variants, default_variant_key, created_at
		 FROM flags WHERE project_id = $1 AND key = $2`,
		projectID, key,
	)
	return scanFlag(row)
}

func (r *PostgresFlagRepository) ListByProject(ctx context.Context, projectID string) ([]*domain.Flag, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, key, name, type, variants, default_variant_key, created_at
		 FROM flags WHERE project_id = $1 ORDER BY created_at ASC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	flags := make([]*domain.Flag, 0)
	for rows.Next() {
		f, err := scanFlagRow(rows)
		if err != nil {
			return nil, err
		}
		flags = append(flags, f)
	}
	return flags, rows.Err()
}

func (r *PostgresFlagRepository) Update(ctx context.Context, flag *domain.Flag) error {
	variants, err := marshalVariants(flag.Variants)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE flags SET name = $1, variants = $2, default_variant_key = $3 WHERE id = $4`,
		flag.Name, variants, flag.DefaultVariantKey, flag.ID,
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

func (r *PostgresFlagRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM flags WHERE id = $1`, id)
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

// scanFlag scans a single *sql.Row into a Flag.
func scanFlag(row *sql.Row) (*domain.Flag, error) {
	var f domain.Flag
	var flagType string
	var variantsJSON []byte
	if err := row.Scan(&f.ID, &f.ProjectID, &f.Key, &f.Name, &flagType, &variantsJSON, &f.DefaultVariantKey, &f.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	f.Type = domain.FlagType(flagType)
	variants, err := unmarshalVariants(variantsJSON)
	if err != nil {
		return nil, err
	}
	f.Variants = variants
	return &f, nil
}

// scanFlagRow scans a *sql.Rows (iterator) into a Flag.
func scanFlagRow(rows *sql.Rows) (*domain.Flag, error) {
	var f domain.Flag
	var flagType string
	var variantsJSON []byte
	if err := rows.Scan(&f.ID, &f.ProjectID, &f.Key, &f.Name, &flagType, &variantsJSON, &f.DefaultVariantKey, &f.CreatedAt); err != nil {
		return nil, err
	}
	f.Type = domain.FlagType(flagType)
	variants, err := unmarshalVariants(variantsJSON)
	if err != nil {
		return nil, err
	}
	f.Variants = variants
	return &f, nil
}
