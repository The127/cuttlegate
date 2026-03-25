package app

import (
	"context"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// RuleService orchestrates targeting rule use cases.
type RuleService struct {
	repo ports.RuleRepository
}

// NewRuleService constructs a RuleService.
func NewRuleService(repo ports.RuleRepository) *RuleService {
	return &RuleService{repo: repo}
}

// Create validates the rule, assigns a UUID and creation timestamp, and persists it.
// Requires at least editor role.
func (s *RuleService) Create(ctx context.Context, flagID, environmentID string, priority int, conditions []domain.Condition, variantKey, name string, rollout []domain.RolloutEntry) (*domain.Rule, error) {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return nil, err
	}
	rule := &domain.Rule{
		FlagID:        flagID,
		EnvironmentID: environmentID,
		Name:          name,
		Priority:      priority,
		Conditions:    conditions,
		VariantKey:    variantKey,
		Rollout:       rollout,
		Enabled:       true,
	}
	if err := rule.Validate(); err != nil {
		return nil, err
	}
	existing, err := s.repo.ListByFlagEnvironment(ctx, flagID, environmentID)
	if err != nil {
		return nil, err
	}
	for _, r := range existing {
		if r.Priority == priority {
			return nil, domain.ErrPriorityConflict
		}
	}
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	rule.ID = id
	rule.CreatedAt = time.Now().UTC()
	if err := s.repo.Upsert(ctx, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

// List returns all rules for a flag+environment, ordered by priority ascending.
func (s *RuleService) List(ctx context.Context, flagID, environmentID string) ([]*domain.Rule, error) {
	return s.repo.ListByFlagEnvironment(ctx, flagID, environmentID)
}

// Update validates and persists updated rule fields.
// Requires at least editor role.
func (s *RuleService) Update(ctx context.Context, rule *domain.Rule) (*domain.Rule, error) {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return nil, err
	}
	if err := rule.Validate(); err != nil {
		return nil, err
	}
	existing, err := s.repo.ListByFlagEnvironment(ctx, rule.FlagID, rule.EnvironmentID)
	if err != nil {
		return nil, err
	}
	for _, r := range existing {
		if r.Priority == rule.Priority && r.ID != rule.ID {
			return nil, domain.ErrPriorityConflict
		}
	}
	if err := s.repo.Upsert(ctx, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

// Delete removes a rule by ID.
// Requires at least editor role.
func (s *RuleService) Delete(ctx context.Context, id string) error {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
