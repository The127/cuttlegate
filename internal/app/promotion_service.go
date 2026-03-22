package app

import (
	"context"
	"log"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// FlagPromotionDiff describes the state changes applied to a single flag during promotion.
type FlagPromotionDiff struct {
	FlagKey       string
	EnabledBefore bool
	EnabledAfter  bool
	RulesAdded    int
	RulesRemoved  int
}

// PromotionService orchestrates flag state promotion between environments.
type PromotionService struct {
	uowFactory ports.UnitOfWorkFactory
	flagRepo   ports.FlagRepository
}

// NewPromotionService constructs a PromotionService.
func NewPromotionService(
	uowFactory ports.UnitOfWorkFactory,
	flagRepo ports.FlagRepository,
) *PromotionService {
	return &PromotionService{uowFactory: uowFactory, flagRepo: flagRepo}
}

// PromoteFlagState copies a single flag's enabled state and targeting rules from
// sourceEnvID to targetEnvID atomically. Requires admin role.
func (s *PromotionService) PromoteFlagState(ctx context.Context, projectID, sourceEnvID, targetEnvID, flagKey string) (*FlagPromotionDiff, error) {
	if err := validatePromotion(ctx, sourceEnvID, targetEnvID); err != nil {
		return nil, err
	}
	flag, err := s.flagRepo.GetByKey(ctx, projectID, flagKey)
	if err != nil {
		return nil, err
	}

	uow, err := s.uowFactory.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer uow.Rollback(ctx) //nolint:errcheck

	diff, err := promoteFlag(ctx, uow, flag, projectID, sourceEnvID, targetEnvID)
	if err != nil {
		return nil, err
	}
	return diff, uow.Commit(ctx)
}

// PromoteAllFlags copies enabled state and targeting rules for every flag in the project
// from sourceEnvID to targetEnvID in a single atomic transaction. Requires admin role.
func (s *PromotionService) PromoteAllFlags(ctx context.Context, projectID, sourceEnvID, targetEnvID string) ([]*FlagPromotionDiff, error) {
	if err := validatePromotion(ctx, sourceEnvID, targetEnvID); err != nil {
		return nil, err
	}
	flags, err := s.flagRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	uow, err := s.uowFactory.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer uow.Rollback(ctx) //nolint:errcheck

	diffs := make([]*FlagPromotionDiff, 0, len(flags))
	for _, f := range flags {
		diff, err := promoteFlag(ctx, uow, f, projectID, sourceEnvID, targetEnvID)
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, diff)
	}
	return diffs, uow.Commit(ctx)
}

// validatePromotion enforces the admin role requirement and same-environment guard.
func validatePromotion(ctx context.Context, sourceEnvID, targetEnvID string) error {
	if _, err := requireRole(ctx, domain.RoleAdmin); err != nil {
		return err
	}
	if sourceEnvID == targetEnvID {
		return &domain.ValidationError{Field: "target_env", Message: "source and target environments must differ"}
	}
	return nil
}

// promoteFlag copies a single flag's state and rules within the given UnitOfWork.
// The audit record is written inside the transaction so it is atomic with the state change.
func promoteFlag(ctx context.Context, uow ports.UnitOfWork, flag *domain.Flag, projectID, sourceEnvID, targetEnvID string) (*FlagPromotionDiff, error) {
	stateRepo := uow.FlagEnvironmentStateRepository()
	ruleRepo := uow.RuleRepository()

	srcState, err := stateRepo.GetByFlagAndEnvironment(ctx, flag.ID, sourceEnvID)
	if err != nil {
		return nil, err
	}
	srcEnabled := srcState != nil && srcState.Enabled

	dstState, err := stateRepo.GetByFlagAndEnvironment(ctx, flag.ID, targetEnvID)
	if err != nil {
		return nil, err
	}
	dstEnabledBefore := dstState != nil && dstState.Enabled

	if err := stateRepo.Upsert(ctx, &domain.FlagEnvironmentState{
		FlagID:        flag.ID,
		EnvironmentID: targetEnvID,
		Enabled:       srcEnabled,
	}); err != nil {
		return nil, err
	}

	srcRules, err := ruleRepo.ListByFlagEnvironment(ctx, flag.ID, sourceEnvID)
	if err != nil {
		return nil, err
	}

	dstRules, err := ruleRepo.ListByFlagEnvironment(ctx, flag.ID, targetEnvID)
	if err != nil {
		return nil, err
	}

	if err := ruleRepo.DeleteByFlagEnvironment(ctx, flag.ID, targetEnvID); err != nil {
		return nil, err
	}

	// Source and target rules are independent: promote with new IDs so a future
	// delete of a source rule cannot accidentally reference a target rule.
	for _, r := range srcRules {
		newID, err := newUUID()
		if err != nil {
			return nil, err
		}
		if err := ruleRepo.Upsert(ctx, &domain.Rule{
			ID:            newID,
			FlagID:        flag.ID,
			EnvironmentID: targetEnvID,
			Name:          r.Name,
			Priority:      r.Priority,
			Conditions:    r.Conditions,
			VariantKey:    r.VariantKey,
			Enabled:       r.Enabled,
			CreatedAt:     time.Now().UTC(),
		}); err != nil {
			return nil, err
		}
	}

	if err := recordPromotionAudit(ctx, uow.AuditRepository(), projectID, flag); err != nil {
		log.Printf("promotion audit: failed to record for flag %s: %v", flag.Key, err)
	}

	return &FlagPromotionDiff{
		FlagKey:       flag.Key,
		EnabledBefore: dstEnabledBefore,
		EnabledAfter:  srcEnabled,
		RulesAdded:    len(srcRules),
		RulesRemoved:  len(dstRules),
	}, nil
}

func recordPromotionAudit(ctx context.Context, auditRepo ports.AuditRepository, projectID string, flag *domain.Flag) error {
	ac, _ := domain.AuthContextFrom(ctx)
	id, err := newUUID()
	if err != nil {
		return err
	}
	return auditRepo.Record(ctx, &domain.AuditEvent{
		ID:         id,
		ProjectID:  projectID,
		ActorID:    ac.UserID,
		Action:     "flag.promoted",
		EntityType: "flag",
		EntityID:   flag.ID,
		EntityKey:  flag.Key,
		OccurredAt: time.Now().UTC(),
	})
}
