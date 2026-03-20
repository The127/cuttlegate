package app

import (
	"context"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// FlagService orchestrates flag use cases.
type FlagService struct {
	repo ports.FlagRepository
}

// NewFlagService constructs a FlagService.
func NewFlagService(repo ports.FlagRepository) *FlagService {
	return &FlagService{repo: repo}
}

// Create validates the flag, assigns a UUID and creation timestamp, then persists it.
func (s *FlagService) Create(ctx context.Context, flag *domain.Flag) error {
	if err := flag.Validate(); err != nil {
		return err
	}
	id, err := newUUID()
	if err != nil {
		return err
	}
	flag.ID = id
	flag.CreatedAt = time.Now().UTC()
	return s.repo.Create(ctx, flag)
}

// GetByKey retrieves a flag by project ID and key.
func (s *FlagService) GetByKey(ctx context.Context, projectID, key string) (*domain.Flag, error) {
	return s.repo.GetByKey(ctx, projectID, key)
}

// ListByProject returns all flags for a project.
func (s *FlagService) ListByProject(ctx context.Context, projectID string) ([]*domain.Flag, error) {
	return s.repo.ListByProject(ctx, projectID)
}

// Update validates and persists updated flag fields. Only Name, Variants, and DefaultVariantKey are mutable.
func (s *FlagService) Update(ctx context.Context, flag *domain.Flag) error {
	if err := flag.Validate(); err != nil {
		return err
	}
	return s.repo.Update(ctx, flag)
}

// Delete removes a flag by ID.
func (s *FlagService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// DeleteByKey removes a flag identified by project ID and key.
func (s *FlagService) DeleteByKey(ctx context.Context, projectID, key string) error {
	f, err := s.repo.GetByKey(ctx, projectID, key)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, f.ID)
}
