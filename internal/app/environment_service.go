package app

import (
	"context"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// EnvironmentService orchestrates environment use cases.
type EnvironmentService struct {
	repo      ports.EnvironmentRepository
	flagRepo  ports.FlagRepository
	stateRepo ports.FlagEnvironmentStateRepository
}

// NewEnvironmentService constructs an EnvironmentService.
func NewEnvironmentService(repo ports.EnvironmentRepository, flagRepo ports.FlagRepository, stateRepo ports.FlagEnvironmentStateRepository) *EnvironmentService {
	return &EnvironmentService{repo: repo, flagRepo: flagRepo, stateRepo: stateRepo}
}

// Create assigns a UUID and creation timestamp, then persists the environment under projectID.
// Returns domain.ErrConflict if the environment slug already exists under that project.
// Requires at least editor role.
func (s *EnvironmentService) Create(ctx context.Context, projectID, name, envSlug string) (*domain.Environment, error) {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return nil, err
	}
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	e := domain.Environment{
		ID:        id,
		ProjectID: projectID,
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
	// Backfill: create a disabled state row for every existing flag in the project.
	flags, err := s.flagRepo.ListByProject(ctx, projectID)
	if err != nil {
		_ = s.repo.Delete(ctx, e.ID)
		return nil, err
	}
	if len(flags) > 0 {
		states := make([]*domain.FlagEnvironmentState, len(flags))
		for i, f := range flags {
			states[i] = &domain.FlagEnvironmentState{
				FlagID:        f.ID,
				EnvironmentID: e.ID,
				Enabled:       false,
			}
		}
		if err := s.stateRepo.CreateBatch(ctx, states); err != nil {
			_ = s.repo.Delete(ctx, e.ID)
			return nil, err
		}
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

// UpdateName changes the name of an environment identified by project ID and slug.
// Slug is immutable — only the display name is editable.
// Requires admin role.
func (s *EnvironmentService) UpdateName(ctx context.Context, projectID, slug, name string) (*domain.Environment, error) {
	if _, err := requireRole(ctx, domain.RoleAdmin); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, &domain.ValidationError{Field: "name", Message: "must not be empty"}
	}
	e, err := s.repo.GetBySlug(ctx, projectID, slug)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateName(ctx, e.ID, name); err != nil {
		return nil, err
	}
	e.Name = name
	return e, nil
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
