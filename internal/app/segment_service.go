package app

import (
	"context"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// SegmentService orchestrates user segment use cases.
type SegmentService struct {
	repo ports.SegmentRepository
}

// NewSegmentService constructs a SegmentService.
func NewSegmentService(repo ports.SegmentRepository) *SegmentService {
	return &SegmentService{repo: repo}
}

// Create validates and persists a new segment. Requires at least editor role.
func (s *SegmentService) Create(ctx context.Context, projectID, slug, name string) (*domain.Segment, error) {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return nil, err
	}
	seg := &domain.Segment{
		Slug:      slug,
		Name:      name,
		ProjectID: projectID,
	}
	if err := seg.Validate(); err != nil {
		return nil, err
	}
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	seg.ID = id
	seg.CreatedAt = time.Now().UTC()
	if err := s.repo.Create(ctx, seg); err != nil {
		return nil, err
	}
	return seg, nil
}

// GetBySlug returns a segment by its slug within a project. Requires at least viewer role.
func (s *SegmentService) GetBySlug(ctx context.Context, projectID, slug string) (*domain.Segment, error) {
	if _, err := requireRole(ctx, domain.RoleViewer); err != nil {
		return nil, err
	}
	return s.repo.GetBySlug(ctx, projectID, slug)
}

// List returns all segments in a project. Requires at least viewer role.
func (s *SegmentService) List(ctx context.Context, projectID string) ([]*domain.Segment, error) {
	if _, err := requireRole(ctx, domain.RoleViewer); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, projectID)
}

// ListWithCount returns all segments in a project with their member counts.
// Requires at least viewer role.
func (s *SegmentService) ListWithCount(ctx context.Context, projectID string) ([]*ports.SegmentWithCount, error) {
	if _, err := requireRole(ctx, domain.RoleViewer); err != nil {
		return nil, err
	}
	return s.repo.ListWithCount(ctx, projectID)
}

// UpdateName renames a segment. Requires at least editor role.
func (s *SegmentService) UpdateName(ctx context.Context, id, name string) error {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return err
	}
	if name == "" {
		return &domain.ValidationError{Field: "name", Message: "must not be empty"}
	}
	return s.repo.UpdateName(ctx, id, name)
}

// Delete removes a segment by its slug within a project. Requires at least editor role.
func (s *SegmentService) Delete(ctx context.Context, projectID, slug string) error {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return err
	}
	seg, err := s.repo.GetBySlug(ctx, projectID, slug)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, seg.ID)
}

// SetMembers bulk-replaces the member list for a segment. Requires at least editor role.
func (s *SegmentService) SetMembers(ctx context.Context, segmentID string, userKeys []string) error {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return err
	}
	return s.repo.SetMembers(ctx, segmentID, userKeys)
}

// ListMembers returns the user keys that are members of a segment. Requires at least viewer role.
func (s *SegmentService) ListMembers(ctx context.Context, segmentID string) ([]string, error) {
	if _, err := requireRole(ctx, domain.RoleViewer); err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx, segmentID)
}
