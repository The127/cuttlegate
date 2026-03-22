//go:build integration

package dbadapter_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

// viewerCtx returns a context carrying a viewer role for EvaluationService calls.
func viewerCtx() context.Context {
	return domain.NewAuthContext(context.Background(), domain.AuthContext{
		UserID: "test-viewer",
		Role:   domain.RoleViewer,
	})
}

// newEvalSvcFromDB constructs an EvaluationService backed by real Postgres adapters.
// publisher is nil — fire-and-forget events are not tested here.
func newEvalSvcFromDB(db *sql.DB) *app.EvaluationService {
	return app.NewEvaluationService(
		dbadapter.NewPostgresFlagRepository(db),
		dbadapter.NewPostgresFlagEnvironmentStateRepository(db),
		dbadapter.NewPostgresRuleRepository(db),
		dbadapter.NewPostgresSegmentRepository(db),
		nil,
	)
}

// evalFixture holds IDs for the segment evaluation integration test setup.
// Projects, environments, and flags use TEXT PKs; segments and rules use UUID PKs.
type evalFixture struct {
	projID  string
	envID   string
	flagID  string
	flagKey string
	segID   string // valid UUID
	segSlug string
	ruleID  string // valid UUID
}

// seedEvalFixture creates a project, environment, flag (with two variants), segment,
// and a rule that requires segment membership to return "variant-a".
// The textPrefix parameter is used to namespace TEXT PKs and slug/key values.
// segID and ruleID must be valid UUIDs (those columns are UUID type in Postgres).
func seedEvalFixture(t *testing.T, db *sql.DB, ctx context.Context, textPrefix, segID, ruleID string) evalFixture {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	f := evalFixture{
		projID:  "eval-proj-" + textPrefix,
		envID:   "eval-env-" + textPrefix,
		flagID:  "eval-flag-id-" + textPrefix,
		flagKey: "eval-flag-" + textPrefix,
		segID:   segID,
		segSlug: "beta-" + textPrefix,
		ruleID:  ruleID,
	}

	projRepo := dbadapter.NewPostgresProjectRepository(db)
	if err := projRepo.Create(ctx, domain.Project{
		ID: f.projID, Name: "Eval Test", Slug: "eval-" + textPrefix, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	envRepo := dbadapter.NewPostgresEnvironmentRepository(db)
	if err := envRepo.Create(ctx, domain.Environment{
		ID: f.envID, ProjectID: f.projID, Name: "Test", Slug: "test-" + textPrefix, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed environment: %v", err)
	}

	flagRepo := dbadapter.NewPostgresFlagRepository(db)
	if err := flagRepo.Create(ctx, &domain.Flag{
		ID:        f.flagID,
		ProjectID: f.projID,
		Key:       f.flagKey,
		Name:      "Eval Flag",
		// Use string type so EvalView.Value is populated — enables asserting the variant key directly.
		Type:              domain.FlagTypeString,
		Variants:          []domain.Variant{{Key: "variant-a", Name: "Variant A"}, {Key: "default", Name: "Default"}},
		DefaultVariantKey: "default",
		CreatedAt:         now,
	}); err != nil {
		t.Fatalf("seed flag: %v", err)
	}

	stateRepo := dbadapter.NewPostgresFlagEnvironmentStateRepository(db)
	if err := stateRepo.Upsert(ctx, &domain.FlagEnvironmentState{
		FlagID: f.flagID, EnvironmentID: f.envID, Enabled: true,
	}); err != nil {
		t.Fatalf("seed flag state: %v", err)
	}

	segRepo := dbadapter.NewPostgresSegmentRepository(db)
	if err := segRepo.Create(ctx, &domain.Segment{
		ID: f.segID, Slug: f.segSlug, Name: "Beta", ProjectID: f.projID, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed segment: %v", err)
	}

	ruleRepo := dbadapter.NewPostgresRuleRepository(db)
	if err := ruleRepo.Upsert(ctx, &domain.Rule{
		ID:            f.ruleID,
		FlagID:        f.flagID,
		EnvironmentID: f.envID,
		Priority:      1,
		Conditions: []domain.Condition{
			{Operator: domain.OperatorInSegment, Values: []string{f.segSlug}},
		},
		VariantKey: "variant-a",
		Enabled:    true,
		CreatedAt:  now,
	}); err != nil {
		t.Fatalf("seed rule: %v", err)
	}

	return f
}

// TestEvaluationService_SegmentMember_GetsRuleVariant covers the @happy scenario:
// a user who is a member of the referenced segment gets the rule-matched variant.
func TestEvaluationService_SegmentMember_GetsRuleVariant(t *testing.T) {
	db := newTestDB(t)
	ctx := viewerCtx()
	f := seedEvalFixture(t, db, context.Background(),
		"seg-member",
		"a1000000-0000-4000-8000-000000000001",
		"a1000000-0000-4000-8000-000000000002",
	)

	segRepo := dbadapter.NewPostgresSegmentRepository(db)
	if err := segRepo.SetMembers(context.Background(), f.segID, []string{"user-in-segment"}); err != nil {
		t.Fatalf("set members: %v", err)
	}

	result, err := newEvalSvcFromDB(db).Evaluate(ctx, f.projID, f.envID, f.flagKey, domain.EvalContext{UserID: "user-in-segment"})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// @happy: segment member gets the rule-matched variant.
	if result.Reason != domain.ReasonRuleMatch {
		t.Errorf("Reason = %q, want %q", result.Reason, domain.ReasonRuleMatch)
	}
	if result.Value == nil || *result.Value != "variant-a" {
		v := "<nil>"
		if result.Value != nil {
			v = *result.Value
		}
		t.Errorf("Value = %q, want %q", v, "variant-a")
	}
}

// TestEvaluationService_NonMember_GetsDefault covers the @edge scenario:
// a user NOT in the segment falls through to the default variant.
func TestEvaluationService_NonMember_GetsDefault(t *testing.T) {
	db := newTestDB(t)
	ctx := viewerCtx()
	f := seedEvalFixture(t, db, context.Background(),
		"non-member",
		"a2000000-0000-4000-8000-000000000001",
		"a2000000-0000-4000-8000-000000000002",
	)

	segRepo := dbadapter.NewPostgresSegmentRepository(db)
	if err := segRepo.SetMembers(context.Background(), f.segID, []string{"someone-else"}); err != nil {
		t.Fatalf("set members: %v", err)
	}

	result, err := newEvalSvcFromDB(db).Evaluate(ctx, f.projID, f.envID, f.flagKey, domain.EvalContext{UserID: "user-outside"})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// @edge: non-member falls through to the default variant.
	if result.Reason != domain.ReasonDefault {
		t.Errorf("Reason = %q, want %q", result.Reason, domain.ReasonDefault)
	}
	if result.Value == nil || *result.Value != "default" {
		v := "<nil>"
		if result.Value != nil {
			v = *result.Value
		}
		t.Errorf("Value = %q, want %q", v, "default")
	}
}

// TestEvaluationService_DeletedSegment_FallsBackGracefully covers the @edge scenario:
// when the segment referenced by a rule no longer exists, evaluation succeeds and
// returns the default variant — not an error.
func TestEvaluationService_DeletedSegment_FallsBackGracefully(t *testing.T) {
	db := newTestDB(t)
	ctx := viewerCtx()
	f := seedEvalFixture(t, db, context.Background(),
		"deleted-seg",
		"a3000000-0000-4000-8000-000000000001",
		"a3000000-0000-4000-8000-000000000002",
	)

	segRepo := dbadapter.NewPostgresSegmentRepository(db)
	if err := segRepo.Delete(context.Background(), f.segID); err != nil {
		t.Fatalf("delete segment: %v", err)
	}

	result, err := newEvalSvcFromDB(db).Evaluate(ctx, f.projID, f.envID, f.flagKey, domain.EvalContext{UserID: "any-user"})
	if err != nil {
		t.Fatalf("Evaluate returned error after segment deleted: %v", err)
	}

	// @edge: deleted segment → non-membership fallback → default variant, no error.
	if result.Reason != domain.ReasonDefault {
		t.Errorf("Reason = %q, want %q", result.Reason, domain.ReasonDefault)
	}
}

// TestEvaluationService_MultiConditionRule_PartialMembership covers the @edge scenario:
// a rule with two in_segment conditions uses AND semantics — if the user is only in
// one of the two required segments, the rule must not match and the default is returned.
func TestEvaluationService_MultiConditionRule_PartialMembership(t *testing.T) {
	db := newTestDB(t)
	ctx := viewerCtx()
	now := time.Now().UTC().Truncate(time.Microsecond)

	projID := "eval-proj-multicond"
	envID := "eval-env-multicond"
	flagID := "eval-flag-id-multicond"
	flagKey := "mc-flag"
	segBetaID := "a4000000-0000-4000-8000-000000000001"
	segAlphaID := "a4000000-0000-4000-8000-000000000002"
	ruleID := "a4000000-0000-4000-8000-000000000003"
	betaSlug := "mc-beta"
	alphaSlug := "mc-alpha"

	projRepo := dbadapter.NewPostgresProjectRepository(db)
	if err := projRepo.Create(context.Background(), domain.Project{
		ID: projID, Name: "Multi-cond Test", Slug: "mc-proj", CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	envRepo := dbadapter.NewPostgresEnvironmentRepository(db)
	if err := envRepo.Create(context.Background(), domain.Environment{
		ID: envID, ProjectID: projID, Name: "Test", Slug: "mc-env", CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed env: %v", err)
	}

	flagRepo := dbadapter.NewPostgresFlagRepository(db)
	if err := flagRepo.Create(context.Background(), &domain.Flag{
		ID: flagID, ProjectID: projID, Key: flagKey, Name: "MC Flag",
		Type:              domain.FlagTypeString,
		Variants:          []domain.Variant{{Key: "variant-a", Name: "A"}, {Key: "default", Name: "Default"}},
		DefaultVariantKey: "default",
		CreatedAt:         now,
	}); err != nil {
		t.Fatalf("seed flag: %v", err)
	}

	stateRepo := dbadapter.NewPostgresFlagEnvironmentStateRepository(db)
	if err := stateRepo.Upsert(context.Background(), &domain.FlagEnvironmentState{
		FlagID: flagID, EnvironmentID: envID, Enabled: true,
	}); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	segRepo := dbadapter.NewPostgresSegmentRepository(db)
	for _, seg := range []*domain.Segment{
		{ID: segBetaID, Slug: betaSlug, Name: "Beta", ProjectID: projID, CreatedAt: now},
		{ID: segAlphaID, Slug: alphaSlug, Name: "Alpha", ProjectID: projID, CreatedAt: now},
	} {
		if err := segRepo.Create(context.Background(), seg); err != nil {
			t.Fatalf("seed segment %s: %v", seg.Slug, err)
		}
	}

	// User is in beta but NOT in alpha.
	if err := segRepo.SetMembers(context.Background(), segBetaID, []string{"partial-user"}); err != nil {
		t.Fatalf("set beta members: %v", err)
	}

	// Rule requires membership in BOTH beta AND alpha (AND semantics).
	ruleRepo := dbadapter.NewPostgresRuleRepository(db)
	if err := ruleRepo.Upsert(context.Background(), &domain.Rule{
		ID:            ruleID,
		FlagID:        flagID,
		EnvironmentID: envID,
		Priority:      1,
		Conditions: []domain.Condition{
			{Operator: domain.OperatorInSegment, Values: []string{betaSlug}},
			{Operator: domain.OperatorInSegment, Values: []string{alphaSlug}},
		},
		VariantKey: "variant-a",
		Enabled:    true,
		CreatedAt:  now,
	}); err != nil {
		t.Fatalf("seed rule: %v", err)
	}

	result, err := newEvalSvcFromDB(db).Evaluate(ctx, projID, envID, flagKey, domain.EvalContext{UserID: "partial-user"})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// @edge: user in beta but not alpha — AND condition fails — default returned.
	if result.Reason != domain.ReasonDefault {
		t.Errorf("Reason = %q, want %q; partial segment membership should not satisfy multi-condition rule", result.Reason, domain.ReasonDefault)
	}
}
