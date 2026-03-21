package app

import (
	"context"
	"errors"

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
	flagRepo    ports.FlagRepository
	stateRepo   ports.FlagEnvironmentStateRepository
	ruleRepo    ports.RuleRepository
	segmentRepo ports.SegmentRepository
}

// NewEvaluationService constructs an EvaluationService.
func NewEvaluationService(
	flagRepo ports.FlagRepository,
	stateRepo ports.FlagEnvironmentStateRepository,
	ruleRepo ports.RuleRepository,
	segmentRepo ports.SegmentRepository,
) *EvaluationService {
	return &EvaluationService{
		flagRepo:    flagRepo,
		stateRepo:   stateRepo,
		ruleRepo:    ruleRepo,
		segmentRepo: segmentRepo,
	}
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

	// Pre-load segment membership for any in_segment / not_in_segment conditions.
	// Extract distinct segment slugs referenced by the rules, then check membership
	// once per slug using evalCtx.UserID as the user key.
	userSegments, err := s.resolveSegments(ctx, projectID, rules, evalCtx.UserID)
	if err != nil {
		return nil, err
	}

	result := domain.Evaluate(flag, state, rules, evalCtx, userSegments)

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

// resolveSegments extracts distinct segment slugs referenced by in_segment /
// not_in_segment conditions in rules, resolves each slug to an ID, and returns
// the set of segment slugs the given userKey belongs to.
// Segments that no longer exist are silently skipped (treated as non-membership).
func (s *EvaluationService) resolveSegments(ctx context.Context, projectID string, rules []*domain.Rule, userKey string) (map[string]struct{}, error) {
	// Collect distinct segment slugs from rule conditions.
	slugs := make(map[string]struct{})
	for _, rule := range rules {
		for _, c := range rule.Conditions {
			if c.Operator == domain.OperatorInSegment || c.Operator == domain.OperatorNotInSegment {
				if len(c.Values) > 0 {
					slugs[c.Values[0]] = struct{}{}
				}
			}
		}
	}
	if len(slugs) == 0 {
		return nil, nil
	}

	userSegments := make(map[string]struct{})
	for slug := range slugs {
		seg, err := s.segmentRepo.GetBySlug(ctx, projectID, slug)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				continue // segment was deleted — treat user as non-member
			}
			return nil, err
		}
		member, err := s.segmentRepo.IsMember(ctx, seg.ID, userKey)
		if err != nil {
			return nil, err
		}
		if member {
			userSegments[slug] = struct{}{}
		}
	}
	return userSegments, nil
}
