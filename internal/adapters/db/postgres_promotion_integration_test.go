//go:build integration

package dbadapter_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

// promotionFixture holds IDs for a project with two environments and two flags.
type promotionFixture struct {
	projID    string
	srcEnvID  string
	dstEnvID  string
	flag1ID   string
	flag1Key  string
	flag2ID   string
	flag2Key  string
}

// seedPromotionFixture creates a project, two environments, and two flags.
func seedPromotionFixture(t *testing.T, db *sql.DB, ctx context.Context, prefix string) promotionFixture {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	f := promotionFixture{
		projID:   "aaa00000-0000-4000-8000-" + prefix + "01",
		srcEnvID: "aaa00000-0000-4000-8000-" + prefix + "02",
		dstEnvID: "aaa00000-0000-4000-8000-" + prefix + "03",
		flag1ID:  "aaa00000-0000-4000-8000-" + prefix + "04",
		flag1Key: "flag-alpha",
		flag2ID:  "aaa00000-0000-4000-8000-" + prefix + "05",
		flag2Key: "flag-beta",
	}

	projRepo := dbadapter.NewPostgresProjectRepository(db)
	if err := projRepo.Create(ctx, domain.Project{
		ID: f.projID, Name: "Promotion Test", Slug: "prom-" + prefix, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	envRepo := dbadapter.NewPostgresEnvironmentRepository(db)
	for _, env := range []domain.Environment{
		{ID: f.srcEnvID, ProjectID: f.projID, Name: "Staging", Slug: "staging", CreatedAt: now},
		{ID: f.dstEnvID, ProjectID: f.projID, Name: "Production", Slug: "production", CreatedAt: now},
	} {
		if err := envRepo.Create(ctx, env); err != nil {
			t.Fatalf("seed env %s: %v", env.Slug, err)
		}
	}

	flagRepo := dbadapter.NewPostgresFlagRepository(db)
	for _, fl := range []*domain.Flag{
		{
			ID: f.flag1ID, ProjectID: f.projID, Key: f.flag1Key, Name: "Alpha",
			Type:              domain.FlagTypeBool,
			Variants:          []domain.Variant{{Key: "true", Name: "On"}, {Key: "false", Name: "Off"}},
			DefaultVariantKey: "false",
			CreatedAt:         now,
		},
		{
			ID: f.flag2ID, ProjectID: f.projID, Key: f.flag2Key, Name: "Beta",
			Type:              domain.FlagTypeBool,
			Variants:          []domain.Variant{{Key: "true", Name: "On"}, {Key: "false", Name: "Off"}},
			DefaultVariantKey: "false",
			CreatedAt:         now,
		},
	} {
		if err := flagRepo.Create(ctx, fl); err != nil {
			t.Fatalf("seed flag %s: %v", fl.Key, err)
		}
	}

	return f
}

func adminCtx() context.Context {
	return domain.NewAuthContext(context.Background(), domain.AuthContext{
		UserID: "user-1",
		Role:   domain.RoleAdmin,
	})
}

func editorCtx() context.Context {
	return domain.NewAuthContext(context.Background(), domain.AuthContext{
		UserID: "user-2",
		Role:   domain.RoleEditor,
	})
}

func newPromotionSvc(db *sql.DB) *app.PromotionService {
	return app.NewPromotionService(
		dbadapter.NewPostgresUnitOfWorkFactory(db),
		dbadapter.NewPostgresEnvironmentRepository(db),
		dbadapter.NewPostgresFlagRepository(db),
		dbadapter.NewPostgresAuditRepository(db),
	)
}

func TestPromotionService_PromoteFlagState_Success(t *testing.T) {
	db := newTestDB(t)
	ctx := adminCtx()
	f := seedPromotionFixture(t, db, context.Background(), "000000000")

	stateRepo := dbadapter.NewPostgresFlagEnvironmentStateRepository(db)
	ruleRepo := dbadapter.NewPostgresRuleRepository(db)

	// Seed source state: flag1 is enabled in staging.
	if err := stateRepo.Upsert(ctx, &domain.FlagEnvironmentState{
		FlagID: f.flag1ID, EnvironmentID: f.srcEnvID, Enabled: true,
	}); err != nil {
		t.Fatalf("seed src state: %v", err)
	}

	// Seed a rule in staging for flag1.
	srcRule := &domain.Rule{
		ID:            "bbb00000-0000-4000-8000-000000000001",
		FlagID:        f.flag1ID,
		EnvironmentID: f.srcEnvID,
		Priority:      1,
		Conditions:    []domain.Condition{{Attribute: "country", Operator: "eq", Values: []string{"US"}}},
		VariantKey:    "true",
		Enabled:       true,
		CreatedAt:     time.Now().UTC(),
	}
	if err := ruleRepo.Upsert(ctx, srcRule); err != nil {
		t.Fatalf("seed src rule: %v", err)
	}

	svc := newPromotionSvc(db)
	diff, err := svc.PromoteFlagState(ctx, f.projID, f.srcEnvID, f.dstEnvID, f.flag1Key)
	if err != nil {
		t.Fatalf("PromoteFlagState: %v", err)
	}

	// Verify diff.
	if diff.FlagKey != f.flag1Key {
		t.Errorf("FlagKey = %q, want %q", diff.FlagKey, f.flag1Key)
	}
	if !diff.EnabledAfter {
		t.Error("EnabledAfter should be true")
	}
	if diff.RulesAdded != 1 {
		t.Errorf("RulesAdded = %d, want 1", diff.RulesAdded)
	}
	if diff.RulesRemoved != 0 {
		t.Errorf("RulesRemoved = %d, want 0", diff.RulesRemoved)
	}

	// Verify target state was written.
	dstState, err := stateRepo.GetByFlagAndEnvironment(ctx, f.flag1ID, f.dstEnvID)
	if err != nil {
		t.Fatalf("get dst state: %v", err)
	}
	if dstState == nil || !dstState.Enabled {
		t.Error("expected flag enabled in production after promotion")
	}

	// Verify target rules were written (with new IDs).
	dstRules, err := ruleRepo.ListByFlagEnvironment(ctx, f.flag1ID, f.dstEnvID)
	if err != nil {
		t.Fatalf("list dst rules: %v", err)
	}
	if len(dstRules) != 1 {
		t.Fatalf("expected 1 dst rule, got %d", len(dstRules))
	}
	if dstRules[0].ID == srcRule.ID {
		t.Error("promoted rule should have a new ID, not the source rule ID")
	}
	if dstRules[0].EnvironmentID != f.dstEnvID {
		t.Errorf("promoted rule EnvironmentID = %q, want %q", dstRules[0].EnvironmentID, f.dstEnvID)
	}
}

func TestPromotionService_PromoteFlagState_ReplacesExistingTargetRules(t *testing.T) {
	db := newTestDB(t)
	ctx := adminCtx()
	f := seedPromotionFixture(t, db, context.Background(), "000000001")

	stateRepo := dbadapter.NewPostgresFlagEnvironmentStateRepository(db)
	ruleRepo := dbadapter.NewPostgresRuleRepository(db)

	// Seed state in both envs.
	for _, envID := range []string{f.srcEnvID, f.dstEnvID} {
		if err := stateRepo.Upsert(ctx, &domain.FlagEnvironmentState{
			FlagID: f.flag1ID, EnvironmentID: envID, Enabled: true,
		}); err != nil {
			t.Fatalf("seed state: %v", err)
		}
	}

	// Seed one rule in source, two rules in target.
	if err := ruleRepo.Upsert(ctx, &domain.Rule{
		ID: "ccc00000-0000-4000-8000-000000000001", FlagID: f.flag1ID,
		EnvironmentID: f.srcEnvID, Priority: 1, VariantKey: "true", Enabled: true,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed src rule: %v", err)
	}
	for i, id := range []string{"ddd00000-0000-4000-8000-000000000001", "ddd00000-0000-4000-8000-000000000002"} {
		if err := ruleRepo.Upsert(ctx, &domain.Rule{
			ID: id, FlagID: f.flag1ID, EnvironmentID: f.dstEnvID,
			Priority: i + 1, VariantKey: "true", Enabled: true,
			CreatedAt: time.Now().UTC(),
		}); err != nil {
			t.Fatalf("seed dst rule: %v", err)
		}
	}

	svc := newPromotionSvc(db)
	diff, err := svc.PromoteFlagState(ctx, f.projID, f.srcEnvID, f.dstEnvID, f.flag1Key)
	if err != nil {
		t.Fatalf("PromoteFlagState: %v", err)
	}

	if diff.RulesAdded != 1 {
		t.Errorf("RulesAdded = %d, want 1", diff.RulesAdded)
	}
	if diff.RulesRemoved != 2 {
		t.Errorf("RulesRemoved = %d, want 2", diff.RulesRemoved)
	}

	dstRules, err := ruleRepo.ListByFlagEnvironment(ctx, f.flag1ID, f.dstEnvID)
	if err != nil {
		t.Fatalf("list dst rules: %v", err)
	}
	if len(dstRules) != 1 {
		t.Errorf("expected 1 rule in production after promotion, got %d", len(dstRules))
	}
}

func TestPromotionService_PromoteAllFlags_Atomic(t *testing.T) {
	db := newTestDB(t)
	ctx := adminCtx()
	f := seedPromotionFixture(t, db, context.Background(), "000000002")

	stateRepo := dbadapter.NewPostgresFlagEnvironmentStateRepository(db)

	// Enable flag1 in source, disable flag2 in source.
	for _, s := range []*domain.FlagEnvironmentState{
		{FlagID: f.flag1ID, EnvironmentID: f.srcEnvID, Enabled: true},
		{FlagID: f.flag2ID, EnvironmentID: f.srcEnvID, Enabled: false},
	} {
		if err := stateRepo.Upsert(ctx, s); err != nil {
			t.Fatalf("seed state: %v", err)
		}
	}

	svc := newPromotionSvc(db)
	diffs, err := svc.PromoteAllFlags(ctx, f.projID, f.srcEnvID, f.dstEnvID)
	if err != nil {
		t.Fatalf("PromoteAllFlags: %v", err)
	}

	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(diffs))
	}

	// Verify production state matches source.
	for _, tc := range []struct {
		flagID  string
		wantOn  bool
	}{
		{f.flag1ID, true},
		{f.flag2ID, false},
	} {
		state, err := stateRepo.GetByFlagAndEnvironment(ctx, tc.flagID, f.dstEnvID)
		if err != nil {
			t.Fatalf("get state for %s: %v", tc.flagID, err)
		}
		if state == nil {
			t.Fatalf("expected state row for %s in production, got nil", tc.flagID)
		}
		if state.Enabled != tc.wantOn {
			t.Errorf("flag %s: Enabled = %v, want %v", tc.flagID, state.Enabled, tc.wantOn)
		}
	}
}

func TestPromotionService_RBAC_NonAdminRejected(t *testing.T) {
	db := newTestDB(t)
	ctx := editorCtx()
	f := seedPromotionFixture(t, db, context.Background(), "000000003")

	svc := newPromotionSvc(db)

	_, err := svc.PromoteFlagState(ctx, f.projID, f.srcEnvID, f.dstEnvID, f.flag1Key)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("PromoteFlagState with editor role: want ErrForbidden, got %v", err)
	}

	_, err = svc.PromoteAllFlags(ctx, f.projID, f.srcEnvID, f.dstEnvID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("PromoteAllFlags with editor role: want ErrForbidden, got %v", err)
	}
}

func TestPromotionService_SameEnvironmentRejected(t *testing.T) {
	db := newTestDB(t)
	ctx := adminCtx()
	f := seedPromotionFixture(t, db, context.Background(), "000000004")

	svc := newPromotionSvc(db)

	_, err := svc.PromoteFlagState(ctx, f.projID, f.srcEnvID, f.srcEnvID, f.flag1Key)
	var valErr *domain.ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("same-env PromoteFlagState: want ValidationError, got %v", err)
	}

	_, err = svc.PromoteAllFlags(ctx, f.projID, f.srcEnvID, f.srcEnvID)
	if !errors.As(err, &valErr) {
		t.Errorf("same-env PromoteAllFlags: want ValidationError, got %v", err)
	}
}

// Cross-project guard: the service accepts raw env IDs and does not independently verify that
// both environments belong to the given project. This guard lives in the HTTP handler, which
// resolves env slugs via EnvironmentRepository.GetBySlug(projectID, slug) — a project-scoped
// lookup that returns ErrNotFound if the slug belongs to a different project.
// See TestPromotionHandler_CrossProjectSlugRejected in promotion_handler_test.go.
