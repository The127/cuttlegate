package app

import (
	"context"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// EvalView is the result of evaluating a flag for a given user context.
type EvalView struct {
	Key     string
	Enabled bool
	Value   *string // nil for bool flags
	Reason  domain.EvalReason
}

// EvaluationService orchestrates flag evaluation use cases.
type EvaluationService struct {
	flagRepo  ports.FlagRepository
	stateRepo ports.FlagEnvironmentStateRepository
	ruleRepo  ports.RuleRepository
}

// NewEvaluationService constructs an EvaluationService.
func NewEvaluationService(
	flagRepo ports.FlagRepository,
	stateRepo ports.FlagEnvironmentStateRepository,
	ruleRepo ports.RuleRepository,
) *EvaluationService {
	return &EvaluationService{flagRepo: flagRepo, stateRepo: stateRepo, ruleRepo: ruleRepo}
}

// Evaluate evaluates a flag for the given user context. Requires at least viewer role.
// Returns ErrNotFound if the flag key does not exist in the project.
func (s *EvaluationService) Evaluate(ctx context.Context, projectID, environmentID, flagKey string, evalCtx domain.EvalContext) (*EvalView, error) {
	if _, err := requireRole(ctx, domain.RoleViewer); err != nil {
		return nil, err
	}

	flag, err := s.flagRepo.GetByKey(ctx, projectID, flagKey)
	if err != nil {
		return nil, err
	}

	state, err := s.stateRepo.GetByFlagAndEnvironment(ctx, flag.ID, environmentID)
	if err != nil {
		return nil, err
	}

	rules, err := s.ruleRepo.ListByFlagEnvironment(ctx, flag.ID, environmentID)
	if err != nil {
		return nil, err
	}

	result := domain.Evaluate(flag, state, rules, evalCtx)

	view := &EvalView{
		Key:     flag.Key,
		Enabled: result.Reason != domain.ReasonDisabled,
		Reason:  result.Reason,
	}
	if flag.Type != domain.FlagTypeBool {
		view.Value = &result.VariantKey
	}
	return view, nil
}
