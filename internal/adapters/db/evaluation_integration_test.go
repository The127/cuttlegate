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
	"github.com/karo/cuttlegate/internal/domain/ports"
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

// newEvalSvcWithPublisher constructs an EvaluationService with a real
// PostgresEvaluationEventRepository wired as the publisher.
func newEvalSvcWithPublisher(db *sql.DB) (*app.EvaluationService, ports.EvaluationEventRepository) {
	eventRepo := dbadapter.NewPostgresEvaluationEventRepository(db)
	svc := app.NewEvaluationService(
		dbadapter.NewPostgresFlagRepository(db),
		dbadapter.NewPostgresFlagEnvironmentStateRepository(db),
		dbadapter.NewPostgresRuleRepository(db),
		dbadapter.NewPostgresSegmentRepository(db),
		eventRepo,
	)
	return svc, eventRepo
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

// waitForEvent polls until at least wantCount events appear for the given flag, or
// the deadline is exceeded. Used as a goroutine fence for fire-and-forget publishEvent
// calls — a short per-iteration sleep is fine; a fixed upfront sleep is not.
func waitForEvent(t *testing.T, repo ports.EvaluationEventRepository, projectID, environmentID, flagKey string, wantCount int) []*domain.EvaluationEvent {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		events, err := repo.ListByFlagEnvironment(context.Background(), projectID, environmentID, flagKey, ports.EvaluationFilter{Limit: 50})
		if err != nil {
			t.Fatalf("ListByFlagEnvironment: %v", err)
		}
		if len(events) >= wantCount {
			return events
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Final read for failure reporting.
	events, _ := repo.ListByFlagEnvironment(context.Background(), projectID, environmentID, flagKey, ports.EvaluationFilter{Limit: 50})
	t.Fatalf("timed out waiting for %d event(s) for flag %q; got %d", wantCount, flagKey, len(events))
	return nil
}

// TestEvaluationService_Evaluate_PublishesEvent covers @happy Scenario 1:
// a single Evaluate call produces exactly one evaluation event with correct fields.
func TestEvaluationService_Evaluate_PublishesEvent(t *testing.T) {
	db := newTestDB(t)
	ctx := viewerCtx()
	f := seedEvalFixture(t, db, context.Background(),
		"pub-single",
		"b1000000-0000-4000-8000-000000000001",
		"b1000000-0000-4000-8000-000000000002",
	)

	// Seed the user into the segment so the rule matches — gives us a deterministic
	// VariantKey and Reason to assert against.
	segRepo := dbadapter.NewPostgresSegmentRepository(db)
	if err := segRepo.SetMembers(context.Background(), f.segID, []string{"pub-user"}); err != nil {
		t.Fatalf("set members: %v", err)
	}

	svc, eventRepo := newEvalSvcWithPublisher(db)

	evalCtx := domain.EvalContext{
		UserID:     "pub-user",
		Attributes: map[string]string{"plan": "pro"},
	}
	result, err := svc.Evaluate(ctx, f.projID, f.envID, f.flagKey, evalCtx)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if result.Reason != domain.ReasonRuleMatch {
		t.Errorf("Reason = %q, want %q", result.Reason, domain.ReasonRuleMatch)
	}

	// Wait for the goroutine to persist the event.
	events := waitForEvent(t, eventRepo, f.projID, f.envID, f.flagKey, 1)

	// @happy Scenario 1: verify all required event fields.
	e := events[0]
	if e.FlagKey != f.flagKey {
		t.Errorf("FlagKey = %q, want %q", e.FlagKey, f.flagKey)
	}
	if e.ProjectID != f.projID {
		t.Errorf("ProjectID = %q, want %q", e.ProjectID, f.projID)
	}
	if e.EnvironmentID != f.envID {
		t.Errorf("EnvironmentID = %q, want %q", e.EnvironmentID, f.envID)
	}
	if e.UserID != evalCtx.UserID {
		t.Errorf("UserID = %q, want %q", e.UserID, evalCtx.UserID)
	}
	if e.VariantKey != "variant-a" {
		t.Errorf("VariantKey = %q, want %q", e.VariantKey, "variant-a")
	}
	if e.Reason != domain.ReasonRuleMatch {
		t.Errorf("Reason = %q, want %q", e.Reason, domain.ReasonRuleMatch)
	}
	if e.OccurredAt.IsZero() {
		t.Error("OccurredAt is zero")
	}
	if e.InputContext == "" || e.InputContext == "null" {
		t.Errorf("InputContext = %q, want non-empty JSON object", e.InputContext)
	}
}

// TestEvaluationService_EvaluateAll_PublishesEventPerFlag covers @happy Scenario 2:
// EvaluateAll produces exactly one event per flag.
func TestEvaluationService_EvaluateAll_PublishesEventPerFlag(t *testing.T) {
	db := newTestDB(t)
	ctx := viewerCtx()
	now := time.Now().UTC().Truncate(time.Microsecond)

	projID := "ea-pub-proj"
	envID := "ea-pub-env"

	projRepo := dbadapter.NewPostgresProjectRepository(db)
	if err := projRepo.Create(context.Background(), domain.Project{
		ID: projID, Name: "EvalAll Pub Test", Slug: "ea-pub-proj", CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	envRepo := dbadapter.NewPostgresEnvironmentRepository(db)
	if err := envRepo.Create(context.Background(), domain.Environment{
		ID: envID, ProjectID: projID, Name: "Test", Slug: "ea-pub-env", CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed environment: %v", err)
	}

	flagRepo := dbadapter.NewPostgresFlagRepository(db)
	stateRepo := dbadapter.NewPostgresFlagEnvironmentStateRepository(db)
	flagKeys := []string{"ea-flag-one", "ea-flag-two"}
	for i, key := range flagKeys {
		flagID := "ea-flag-id-" + key
		if err := flagRepo.Create(context.Background(), &domain.Flag{
			ID:                flagID,
			ProjectID:         projID,
			Key:               key,
			Name:              "EA Flag " + key,
			Type:              domain.FlagTypeBool,
			Variants:          []domain.Variant{{Key: "true", Name: "True"}, {Key: "false", Name: "False"}},
			DefaultVariantKey: "false",
			CreatedAt:         now.Add(time.Duration(i) * time.Millisecond),
		}); err != nil {
			t.Fatalf("seed flag %s: %v", key, err)
		}
		if err := stateRepo.Upsert(context.Background(), &domain.FlagEnvironmentState{
			FlagID: flagID, EnvironmentID: envID, Enabled: true,
		}); err != nil {
			t.Fatalf("seed state %s: %v", key, err)
		}
	}

	svc, eventRepo := newEvalSvcWithPublisher(db)

	evalCtx := domain.EvalContext{UserID: "ea-user"}
	views, _, err := svc.EvaluateAll(ctx, projID, envID, evalCtx)
	if err != nil {
		t.Fatalf("EvaluateAll: %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("EvaluateAll returned %d views, want 2", len(views))
	}

	// @happy Scenario 2: one event per flag — poll until all events land.
	for _, key := range flagKeys {
		events := waitForEvent(t, eventRepo, projID, envID, key, 1)
		if len(events) != 1 {
			t.Errorf("flag %q: got %d events, want 1", key, len(events))
			continue
		}
		e := events[0]
		if e.FlagKey != key {
			t.Errorf("event FlagKey = %q, want %q", e.FlagKey, key)
		}
		if e.ProjectID != projID {
			t.Errorf("event ProjectID = %q, want %q", e.ProjectID, projID)
		}
		if e.EnvironmentID != envID {
			t.Errorf("event EnvironmentID = %q, want %q", e.EnvironmentID, envID)
		}
		if e.UserID != evalCtx.UserID {
			t.Errorf("event UserID = %q, want %q", e.UserID, evalCtx.UserID)
		}
		if e.OccurredAt.IsZero() {
			t.Errorf("flag %q: OccurredAt is zero", key)
		}
	}
}
