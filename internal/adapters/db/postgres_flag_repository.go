package dbadapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

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

// conn returns the tenant-scoped *sql.Tx from context if present,
// otherwise falls back to the connection pool. This ensures queries
// run within the RLS-scoped transaction when tenant middleware is active.
func (r *PostgresFlagRepository) conn(ctx context.Context) DBTX {
	return TenantDBTX(ctx, r.db)
}

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
	_, err = r.conn(ctx).ExecContext(ctx,
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
	row := r.conn(ctx).QueryRowContext(ctx,
		`SELECT id, project_id, key, name, type, variants, default_variant_key, created_at
		 FROM flags WHERE project_id = $1 AND key = $2`,
		projectID, key,
	)
	return scanFlag(row)
}

func (r *PostgresFlagRepository) ListByProject(ctx context.Context, projectID string) ([]*domain.Flag, error) {
	rows, err := r.conn(ctx).QueryContext(ctx,
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

// sortColumnMap maps allowed FlagListFilter.SortBy values to SQL column names.
// This is the allowlist — only these values are accepted; everything else is
// rejected at the domain layer by Normalize().
var sortColumnMap = map[string]string{
	"key":        "key",
	"name":       "name",
	"type":       "type",
	"created_at": "created_at",
}

func (r *PostgresFlagRepository) ListByProjectPaginated(ctx context.Context, projectID string, filter domain.FlagListFilter) ([]*domain.Flag, int, error) {
	filter.Normalize()

	// Build the WHERE clause. Always filter by project_id.
	// Search is optional — when present, filter on key OR name via ILIKE with parameterized value.
	whereClause := "WHERE project_id = $1"
	args := []any{projectID}
	paramIdx := 2

	if filter.Search != "" {
		whereClause += fmt.Sprintf(" AND (key ILIKE $%d OR name ILIKE $%d)", paramIdx, paramIdx)
		args = append(args, "%"+filter.Search+"%")
		paramIdx++
	}

	// Count query
	var total int
	countQuery := "SELECT COUNT(*) FROM flags " + whereClause
	if err := r.conn(ctx).QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query with sort and pagination
	orderCol, ok := sortColumnMap[filter.SortBy]
	if !ok {
		orderCol = "created_at"
	}
	orderDir := "ASC"
	if filter.SortDir == "desc" {
		orderDir = "DESC"
	}

	offset := (filter.Page - 1) * filter.PerPage
	dataQuery := fmt.Sprintf(
		`SELECT id, project_id, key, name, type, variants, default_variant_key, created_at
		 FROM flags %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		whereClause, orderCol, orderDir, paramIdx, paramIdx+1,
	)
	args = append(args, filter.PerPage, offset)

	rows, err := r.conn(ctx).QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	flags := make([]*domain.Flag, 0)
	for rows.Next() {
		f, err := scanFlagRow(rows)
		if err != nil {
			return nil, 0, err
		}
		flags = append(flags, f)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return flags, total, nil
}

func (r *PostgresFlagRepository) Update(ctx context.Context, flag *domain.Flag) error {
	variants, err := marshalVariants(flag.Variants)
	if err != nil {
		return err
	}
	res, err := r.conn(ctx).ExecContext(ctx,
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
	res, err := r.conn(ctx).ExecContext(ctx, `DELETE FROM flags WHERE id = $1`, id)
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
