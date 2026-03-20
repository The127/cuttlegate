package dbadapter

import (
	"context"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// NoOpRuleRepository is a stub that returns no rules.
// Used until the postgres implementation and migration are available (#TODO).
type NoOpRuleRepository struct{}

var _ ports.RuleRepository = (*NoOpRuleRepository)(nil)

func (r *NoOpRuleRepository) ListByFlagEnvironment(_ context.Context, _, _ string) ([]*domain.Rule, error) {
	return nil, nil
}

func (r *NoOpRuleRepository) Upsert(_ context.Context, _ *domain.Rule) error {
	return nil
}

func (r *NoOpRuleRepository) Delete(_ context.Context, _ string) error {
	return nil
}
