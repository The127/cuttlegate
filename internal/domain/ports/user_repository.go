package ports

import (
	"context"

	"github.com/karo/cuttlegate/internal/domain"
)

// UserRepository is the port for the local user profile cache.
// It stores name and email sourced from verified OIDC token claims.
// It must not be used for authorization decisions — roles are owned by
// project_members and the OIDC role claim.
type UserRepository interface {
	// Upsert inserts or updates the cached profile for the given user.
	// Called on every authenticated request by the RequireBearer middleware.
	Upsert(ctx context.Context, user *domain.User) error
	// GetByID returns the cached profile for the given OIDC sub.
	// Returns (nil, nil) if no profile has been cached for this sub.
	GetByID(ctx context.Context, id string) (*domain.User, error)
}
