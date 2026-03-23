package dbadapter

import (
	"context"
	"database/sql"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// PostgresAPIKeyRepository implements ports.APIKeyRepository using PostgreSQL.
type PostgresAPIKeyRepository struct {
	db *sql.DB
}

// NewPostgresAPIKeyRepository constructs a PostgresAPIKeyRepository.
func NewPostgresAPIKeyRepository(db *sql.DB) *PostgresAPIKeyRepository {
	return &PostgresAPIKeyRepository{db: db}
}

var _ ports.APIKeyRepository = (*PostgresAPIKeyRepository)(nil)

func (r *PostgresAPIKeyRepository) Create(ctx context.Context, key *domain.APIKey) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO api_keys (id, project_id, environment_id, name, key_hash, display_prefix, capability_tier, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		key.ID, key.ProjectID, key.EnvironmentID, key.Name, key.KeyHash[:], key.DisplayPrefix, key.CapabilityTier, key.CreatedAt,
	)
	return err
}

func (r *PostgresAPIKeyRepository) GetByHash(ctx context.Context, hash [32]byte) (*domain.APIKey, error) {
	var key domain.APIKey
	var hashBytes []byte
	err := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, environment_id, name, key_hash, display_prefix, capability_tier, created_at, revoked_at
		 FROM api_keys WHERE key_hash = $1`,
		hash[:],
	).Scan(&key.ID, &key.ProjectID, &key.EnvironmentID, &key.Name, &hashBytes, &key.DisplayPrefix, &key.CapabilityTier, &key.CreatedAt, &key.RevokedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	copy(key.KeyHash[:], hashBytes)
	return &key, nil
}

func (r *PostgresAPIKeyRepository) ListByEnvironment(ctx context.Context, projectID, environmentID string) ([]*domain.APIKey, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, environment_id, name, key_hash, display_prefix, capability_tier, created_at, revoked_at
		 FROM api_keys WHERE project_id = $1 AND environment_id = $2 ORDER BY created_at`,
		projectID, environmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make([]*domain.APIKey, 0)
	for rows.Next() {
		var key domain.APIKey
		var hashBytes []byte
		if err := rows.Scan(&key.ID, &key.ProjectID, &key.EnvironmentID, &key.Name, &hashBytes, &key.DisplayPrefix, &key.CapabilityTier, &key.CreatedAt, &key.RevokedAt); err != nil {
			return nil, err
		}
		copy(key.KeyHash[:], hashBytes)
		keys = append(keys, &key)
	}
	return keys, rows.Err()
}

func (r *PostgresAPIKeyRepository) Revoke(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE api_keys SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`,
		id,
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
