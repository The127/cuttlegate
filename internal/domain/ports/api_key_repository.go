package ports

import (
	"context"

	"github.com/karo/cuttlegate/internal/domain"
)

// APIKeyRepository is the port for persisting and retrieving API key entities.
type APIKeyRepository interface {
	Create(ctx context.Context, key *domain.APIKey) error
	GetByHash(ctx context.Context, hash [32]byte) (*domain.APIKey, error)
	ListByEnvironment(ctx context.Context, projectID, environmentID string) ([]*domain.APIKey, error)
	Revoke(ctx context.Context, id string) error
}
