package app

import (
	"context"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// EnvironmentService orchestrates environment use cases.
type EnvironmentService struct {
	repo    ports.EnvironmentRepository
	project ports.ProjectRepository
}

// NewEnvironmentService constructs an EnvironmentService.
func NewEnvironmentService(repo ports.EnvironmentRepository, project ports.ProjectRepository) *EnvironmentService {
	return &EnvironmentService{repo: repo, project: project}
}

// Create validates the project exists by slug, assigns a UUID and creation timestamp, then persists the environment.
// Returns domain.ErrNotFound if the project slug does not exist.
// Returns domain.ErrConflict if the environment slug already exists under that project.
// Requires at least editor role.
func (s *EnvironmentService) Create(ctx context.Context, projectSlug, name, envSlug string) (*domain.Environment, error) {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return nil, err
	}
	proj, err := s.project.GetBySlug(ctx, projectSlug)
	if err != nil {
		return nil, err
	}
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	e := domain.Environment{
		ID:        id,
		ProjectID: proj.ID,
		Name:      name,
		Slug:      envSlug,
		CreatedAt: time.Now().UTC(),
	}
	if err := e.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	return &e, nil
}

// GetBySlug retrieves an environment by project ID and slug.
func (s *EnvironmentService) GetBySlug(ctx context.Context, projectID, slug string) (*domain.Environment, error) {
	return s.repo.GetBySlug(ctx, projectID, slug)
}

// ListByProject returns all environments for a project, ordered by creation time.
func (s *EnvironmentService) ListByProject(ctx context.Context, projectID string) ([]*domain.Environment, error) {
	return s.repo.ListByProject(ctx, projectID)
}

// DeleteBySlug removes an environment identified by project ID and slug.
// Requires at least editor role.
func (s *EnvironmentService) DeleteBySlug(ctx context.Context, projectID, slug string) error {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return err
	}
	e, err := s.repo.GetBySlug(ctx, projectID, slug)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, e.ID)
}

// Delete removes an environment by ID.
// Requires at least editor role.
func (s *EnvironmentService) Delete(ctx context.Context, id string) error {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
