package app

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/The127/cuttlegate/internal/domain"
	"github.com/The127/cuttlegate/internal/domain/ports"
)

// ProjectService orchestrates project use cases.
type ProjectService struct {
	repo    ports.ProjectRepository
	members ports.ProjectMemberRepository
}

// NewProjectService constructs a ProjectService.
func NewProjectService(repo ports.ProjectRepository, members ports.ProjectMemberRepository) *ProjectService {
	return &ProjectService{repo: repo, members: members}
}

// Create assigns a UUID and creation timestamp, then persists the project.
// Requires at least editor role.
func (s *ProjectService) Create(ctx context.Context, name, slug string) (*domain.Project, error) {
	ac, err := requireRole(ctx, domain.RoleEditor)
	if err != nil {
		return nil, err
	}
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	p := domain.Project{
		ID:        id,
		Name:      name,
		Slug:      slug,
		CreatedAt: time.Now().UTC(),
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	// Auto-add the creator as an admin member of the new project.
	m := domain.ProjectMember{
		ProjectID: p.ID,
		UserID:    ac.UserID,
		Role:      domain.RoleAdmin,
		CreatedAt: p.CreatedAt,
	}
	if err := s.members.AddMember(ctx, &m); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetBySlug retrieves a project by its slug.
func (s *ProjectService) GetBySlug(ctx context.Context, slug string) (*domain.Project, error) {
	return s.repo.GetBySlug(ctx, slug)
}

// List returns all projects.
func (s *ProjectService) List(ctx context.Context) ([]*domain.Project, error) {
	return s.repo.List(ctx)
}

// UpdateName changes the name of a project identified by slug. Slug is immutable.
// Requires admin role.
func (s *ProjectService) UpdateName(ctx context.Context, slug, name string) (*domain.Project, error) {
	if _, err := requireRole(ctx, domain.RoleAdmin); err != nil {
		return nil, err
	}
	p, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	p.Name = name
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateName(ctx, p.ID, name); err != nil {
		return nil, err
	}
	return p, nil
}

// DeleteBySlug removes a project by slug.
// Requires admin role.
func (s *ProjectService) DeleteBySlug(ctx context.Context, slug string) error {
	if _, err := requireRole(ctx, domain.RoleAdmin); err != nil {
		return err
	}
	p, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, p.ID)
}

// Delete removes a project by ID.
// Requires admin role.
func (s *ProjectService) Delete(ctx context.Context, id string) error {
	if _, err := requireRole(ctx, domain.RoleAdmin); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
