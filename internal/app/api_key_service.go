package app

import (
	"context"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// APIKeyView is the external representation of an API key (no secret material).
type APIKeyView struct {
	ID            string
	ProjectID     string
	EnvironmentID string
	Name          string
	DisplayPrefix string
	CreatedAt     time.Time
	RevokedAt     *time.Time
}

// APIKeyCreateResult is returned from Create — includes the plaintext key
// which must be shown to the caller exactly once.
type APIKeyCreateResult struct {
	APIKeyView
	Plaintext string
}

// APIKeyService orchestrates API key use cases.
type APIKeyService struct {
	repo ports.APIKeyRepository
}

// NewAPIKeyService constructs an APIKeyService.
func NewAPIKeyService(repo ports.APIKeyRepository) *APIKeyService {
	return &APIKeyService{repo: repo}
}

// Create generates a new API key scoped to the given project and environment.
// Requires admin role.
func (s *APIKeyService) Create(ctx context.Context, projectID, environmentID, name string) (*APIKeyCreateResult, error) {
	if _, err := requireRole(ctx, domain.RoleAdmin); err != nil {
		return nil, err
	}

	id, err := newUUID()
	if err != nil {
		return nil, err
	}

	key, plaintext, err := domain.GenerateAPIKey(id, projectID, environmentID, name)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, key); err != nil {
		return nil, err
	}

	return &APIKeyCreateResult{
		APIKeyView: APIKeyView{
			ID:            key.ID,
			ProjectID:     key.ProjectID,
			EnvironmentID: key.EnvironmentID,
			Name:          key.Name,
			DisplayPrefix: key.DisplayPrefix,
			CreatedAt:     key.CreatedAt,
		},
		Plaintext: plaintext,
	}, nil
}

// List returns all API keys for a project and environment (no secret material).
// Requires viewer role.
func (s *APIKeyService) List(ctx context.Context, projectID, environmentID string) ([]APIKeyView, error) {
	if _, err := requireRole(ctx, domain.RoleViewer); err != nil {
		return nil, err
	}

	keys, err := s.repo.ListByEnvironment(ctx, projectID, environmentID)
	if err != nil {
		return nil, err
	}

	views := make([]APIKeyView, len(keys))
	for i, k := range keys {
		views[i] = APIKeyView{
			ID:            k.ID,
			ProjectID:     k.ProjectID,
			EnvironmentID: k.EnvironmentID,
			Name:          k.Name,
			DisplayPrefix: k.DisplayPrefix,
			CreatedAt:     k.CreatedAt,
			RevokedAt:     k.RevokedAt,
		}
	}
	return views, nil
}

// Revoke marks an API key as revoked. Requires admin role.
func (s *APIKeyService) Revoke(ctx context.Context, id string) error {
	if _, err := requireRole(ctx, domain.RoleAdmin); err != nil {
		return err
	}
	return s.repo.Revoke(ctx, id)
}

// Authenticate verifies a plaintext API key and returns the scoped project and
// environment IDs. Returns ErrForbidden if the key is invalid, revoked, or
// does not match any stored key. This method does not require an AuthContext —
// it is called before authentication is established.
func (s *APIKeyService) Authenticate(ctx context.Context, plaintext string) (projectID, environmentID string, err error) {
	hash := domain.HashAPIKey(plaintext)
	key, err := s.repo.GetByHash(ctx, hash)
	if err != nil {
		return "", "", domain.ErrForbidden
	}
	if key.Revoked() {
		return "", "", domain.ErrForbidden
	}
	return key.ProjectID, key.EnvironmentID, nil
}
