package app_test

import (
	"context"
	"testing"

	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// fakeAuditRepository is a no-op implementation for promotion tests.
type fakeAuditRepository struct{}

func (f *fakeAuditRepository) Record(_ context.Context, _ *domain.AuditEvent) error { return nil }
func (f *fakeAuditRepository) ListByProject(_ context.Context, _ string, _ domain.AuditFilter) ([]*domain.AuditEvent, error) {
	return nil, nil
}

// fakeUnitOfWork provides in-memory repository access within a single scope.
type fakeUnitOfWork struct {
	stateRepo *fakeFlagEnvironmentStateRepository
	ruleRepo  *fakeRuleRepository
	audit     *fakeAuditRepository
}

func (u *fakeUnitOfWork) FlagEnvironmentStateRepository() ports.FlagEnvironmentStateRepository {
	return u.stateRepo
}
func (u *fakeUnitOfWork) RuleRepository() ports.RuleRepository   { return u.ruleRepo }
func (u *fakeUnitOfWork) AuditRepository() ports.AuditRepository { return u.audit }
func (u *fakeUnitOfWork) Commit(_ context.Context) error         { return nil }
func (u *fakeUnitOfWork) Rollback(_ context.Context) error       { return nil }

// fakeUnitOfWorkFactory always returns the same fakeUnitOfWork.
type fakeUnitOfWorkFactory struct {
	uow *fakeUnitOfWork
}

func (f *fakeUnitOfWorkFactory) Begin(_ context.Context) (ports.UnitOfWork, error) {
	return f.uow, nil
}

// TestPromotion_PreservesRuleName verifies that rule Name is copied to the promoted rule,
// including the empty-string case (no null introduced on promotion).
func TestPromotion_PreservesRuleName(t *testing.T) {
	cases := []struct {
		desc         string
		sourceName   string
		expectedName string
	}{
		{"named rule", "VIP rule", "VIP rule"},
		{"empty name", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := authCtx("admin-1", domain.RoleAdmin)

			flagRepo := newFakeFlagRepository()
			flag := &domain.Flag{
				ID:                "flag-1",
				ProjectID:         "proj-1",
				Key:               "my-flag",
				DefaultVariantKey: "off",
			}
			_ = flagRepo.Create(ctx, flag)

			stateRepo := newFakeFlagEnvironmentStateRepository()
			_ = stateRepo.Upsert(ctx, &domain.FlagEnvironmentState{
				FlagID:        "flag-1",
				EnvironmentID: "env-src",
				Enabled:       true,
			})

			ruleRepo := newFakeRuleRepository()
			_ = ruleRepo.Upsert(ctx, &domain.Rule{
				ID:            "rule-1",
				FlagID:        "flag-1",
				EnvironmentID: "env-src",
				Name:          tc.sourceName,
				Priority:      0,
				Conditions:    validConditions,
				VariantKey:    "on",
				Enabled:       true,
			})

			uow := &fakeUnitOfWork{
				stateRepo: stateRepo,
				ruleRepo:  ruleRepo,
				audit:     &fakeAuditRepository{},
			}
			svc := app.NewPromotionService(&fakeUnitOfWorkFactory{uow: uow}, flagRepo)

			_, err := svc.PromoteFlagState(ctx, "proj-1", "env-src", "env-dst", "my-flag")
			if err != nil {
				t.Fatalf("PromoteFlagState: %v", err)
			}

			dstRules, err := ruleRepo.ListByFlagEnvironment(ctx, "flag-1", "env-dst")
			if err != nil {
				t.Fatalf("ListByFlagEnvironment: %v", err)
			}
			if len(dstRules) != 1 {
				t.Fatalf("expected 1 promoted rule, got %d", len(dstRules))
			}
			if dstRules[0].Name != tc.expectedName {
				t.Errorf("expected promoted rule Name = %q, got %q", tc.expectedName, dstRules[0].Name)
			}
		})
	}
}
