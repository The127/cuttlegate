package app

import (
	"context"
	"time"

	"github.com/The127/cuttlegate/internal/domain"
	"github.com/The127/cuttlegate/internal/domain/ports"
)

// EvaluationEventView is the read model for a single evaluation event.
type EvaluationEventView struct {
	ID              string
	FlagKey         string
	EnvironmentID   string
	UserID          string
	InputContext    string // JSON string
	MatchedRuleID   string
	MatchedRuleName string
	VariantKey      string
	Reason          string // API-facing reason string (may differ from domain constant)
	OccurredAt      time.Time
}

// EvaluationAuditService orchestrates read access to the evaluation audit trail.
type EvaluationAuditService struct {
	repo    ports.EvaluationEventRepository
	flagSvc flagKeyResolver
}

// flagKeyResolver is the subset of FlagRepository needed to resolve a flag key to a project flag.
type flagKeyResolver interface {
	GetByKey(ctx context.Context, projectID, key string) (*domain.Flag, error)
}

// NewEvaluationAuditService constructs an EvaluationAuditService.
func NewEvaluationAuditService(repo ports.EvaluationEventRepository, flagRepo flagKeyResolver) *EvaluationAuditService {
	return &EvaluationAuditService{repo: repo, flagSvc: flagRepo}
}

// ListEvaluations returns paginated evaluation events for a flag in a project+environment,
// newest first. Requires at least viewer role. Returns ErrNotFound if the flag key does not
// exist in the project.
func (s *EvaluationAuditService) ListEvaluations(
	ctx context.Context,
	projectID, environmentID, flagKey string,
	filter ports.EvaluationFilter,
) ([]*EvaluationEventView, error) {
	if _, err := requireRole(ctx, domain.RoleViewer); err != nil {
		return nil, err
	}

	// Verify the flag exists in this project. Returns ErrNotFound if not.
	if _, err := s.flagSvc.GetByKey(ctx, projectID, flagKey); err != nil {
		return nil, err
	}

	events, err := s.repo.ListByFlagEnvironment(ctx, projectID, environmentID, flagKey, filter)
	if err != nil {
		return nil, err
	}

	views := make([]*EvaluationEventView, len(events))
	for i, e := range events {
		views[i] = &EvaluationEventView{
			ID:              e.ID,
			FlagKey:         e.FlagKey,
			EnvironmentID:   e.EnvironmentID,
			UserID:          e.UserID,
			InputContext:    e.InputContext,
			MatchedRuleID:   e.MatchedRuleID,
			MatchedRuleName: e.MatchedRuleName,
			VariantKey:      e.VariantKey,
			Reason:          apiReason(e.Reason),
			OccurredAt:      e.OccurredAt,
		}
	}
	return views, nil
}

// apiReason translates a domain EvalReason to the API-facing string.
// "disabled" → "flag_disabled" for clarity in the debugging UI.
func apiReason(r domain.EvalReason) string {
	if r == domain.ReasonDisabled {
		return "flag_disabled"
	}
	return string(r)
}
