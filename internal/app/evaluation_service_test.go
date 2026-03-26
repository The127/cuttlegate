package app_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/The127/cuttlegate/internal/app"
	"github.com/The127/cuttlegate/internal/domain"
	"github.com/The127/cuttlegate/internal/domain/ports"
)

// stubErrorPublisher always returns an error from Publish.
// It uses a sync.WaitGroup so tests can drain the fire-and-forget goroutine
// before asserting — no time.Sleep.
// Usage: wg.Add(1) before Evaluate, stub.Wait() after Evaluate.
type stubErrorPublisher struct {
	wg sync.WaitGroup
}

var _ ports.EvaluationEventPublisher = (*stubErrorPublisher)(nil)

func (s *stubErrorPublisher) Publish(_ context.Context, _ *domain.EvaluationEvent) error {
	defer s.wg.Done()
	return errors.New("publish error")
}

func (s *stubErrorPublisher) Wait() { s.wg.Wait() }

// stubOKPublisher always returns nil from Publish.
// Same WaitGroup drain pattern as stubErrorPublisher.
type stubOKPublisher struct {
	wg sync.WaitGroup
}

var _ ports.EvaluationEventPublisher = (*stubOKPublisher)(nil)

func (s *stubOKPublisher) Publish(_ context.Context, _ *domain.EvaluationEvent) error {
	defer s.wg.Done()
	return nil
}

func (s *stubOKPublisher) Wait() { s.wg.Wait() }

// countingRuleRepository wraps fakeRuleRepository and counts ListByEnvironment calls.
type countingRuleRepository struct {
	*fakeRuleRepository
	listByEnvironmentCalls int
}

func newCountingRuleRepository() *countingRuleRepository {
	return &countingRuleRepository{fakeRuleRepository: newFakeRuleRepository()}
}

func (c *countingRuleRepository) ListByEnvironment(ctx context.Context, environmentID string) ([]*domain.Rule, error) {
	c.listByEnvironmentCalls++
	return c.fakeRuleRepository.ListByEnvironment(ctx, environmentID)
}

// newEvalSvc constructs an EvaluationService with in-memory fakes.
// publisher is nil — fire-and-forget events are skipped in unit tests.
func newEvalSvc(flagRepo *fakeFlagRepository, stateRepo *fakeFlagEnvironmentStateRepository, ruleRepo *countingRuleRepository) *app.EvaluationService {
	return app.NewEvaluationService(flagRepo, stateRepo, ruleRepo, newFakeSegmentRepository(), nil)
}

func seedFlag(t *testing.T, flagRepo *fakeFlagRepository, id, projectID, key string) *domain.Flag {
	t.Helper()
	f := &domain.Flag{
		ID:                id,
		ProjectID:         projectID,
		Key:               key,
		Name:              key,
		Type:              domain.FlagTypeBool,
		Variants:          []domain.Variant{{Key: "true", Name: "On"}, {Key: "false", Name: "Off"}},
		DefaultVariantKey: "false",
		CreatedAt:         time.Now().UTC(),
	}
	if err := flagRepo.Create(context.Background(), f); err != nil {
		t.Fatalf("seed flag %s: %v", key, err)
	}
	return f
}

func seedState(t *testing.T, stateRepo *fakeFlagEnvironmentStateRepository, flagID, envID string, enabled bool) {
	t.Helper()
	if err := stateRepo.Upsert(context.Background(), &domain.FlagEnvironmentState{
		FlagID:        flagID,
		EnvironmentID: envID,
		Enabled:       enabled,
	}); err != nil {
		t.Fatalf("seed state flagID=%s: %v", flagID, err)
	}
}

// TestEvaluateAll_BatchLoadsRules verifies that EvaluateAll calls ListByEnvironment
// exactly once regardless of the number of flags, and returns correct results.
func TestEvaluateAll_BatchLoadsRules(t *testing.T) {
	const (
		projectID = "proj-1"
		envID     = "env-1"
	)

	flagRepo := newFakeFlagRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	ruleRepo := newCountingRuleRepository()

	flag1 := seedFlag(t, flagRepo, "flag-id-1", projectID, "flag-one")
	flag2 := seedFlag(t, flagRepo, "flag-id-2", projectID, "flag-two")

	// flag1 is enabled; flag2 is disabled.
	seedState(t, stateRepo, flag1.ID, envID, true)
	seedState(t, stateRepo, flag2.ID, envID, false)

	// Seed a rule for flag1 that does not match (attribute condition).
	rule := &domain.Rule{
		ID:            "rule-id-1",
		FlagID:        flag1.ID,
		EnvironmentID: envID,
		Priority:      1,
		Conditions: []domain.Condition{
			{Attribute: "plan", Operator: domain.OperatorEq, Values: []string{"pro"}},
		},
		VariantKey: "true",
		Enabled:    true,
		CreatedAt:  time.Now().UTC(),
	}
	if err := ruleRepo.Upsert(context.Background(), rule); err != nil {
		t.Fatalf("seed rule: %v", err)
	}

	svc := newEvalSvc(flagRepo, stateRepo, ruleRepo)
	ctx := authCtx("viewer-1", domain.RoleViewer)

	views, _, err := svc.EvaluateAll(ctx, projectID, envID, domain.EvalContext{UserID: "user-1"})
	if err != nil {
		t.Fatalf("EvaluateAll: %v", err)
	}

	// @happy: ListByEnvironment called exactly once — no N+1.
	if ruleRepo.listByEnvironmentCalls != 1 {
		t.Errorf("expected ListByEnvironment called once, got %d", ruleRepo.listByEnvironmentCalls)
	}

	// @happy: results contain both flags.
	if len(views) != 2 {
		t.Fatalf("expected 2 views, got %d", len(views))
	}

	// Build a map for deterministic assertion (flag loop order is map-iteration order).
	byKey := make(map[string]app.EvalView, len(views))
	for _, v := range views {
		byKey[v.Key] = v
	}

	// @happy: flag1 is enabled (state enabled, rule does not match for plan=pro).
	v1, ok := byKey["flag-one"]
	if !ok {
		t.Fatal("expected view for flag-one")
	}
	if !v1.Enabled {
		t.Errorf("expected flag-one to be enabled, got disabled (reason=%v)", v1.Reason)
	}

	// @happy: flag2 is disabled (state disabled).
	v2, ok := byKey["flag-two"]
	if !ok {
		t.Fatal("expected view for flag-two")
	}
	if v2.Enabled {
		t.Errorf("expected flag-two to be disabled, got enabled (reason=%v)", v2.Reason)
	}
}

// TestEvaluateAll_NoFlags_ReturnsEmptySlice verifies behaviour with zero flags.
func TestEvaluateAll_NoFlags_ReturnsEmptySlice(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	ruleRepo := newCountingRuleRepository()

	svc := newEvalSvc(flagRepo, stateRepo, ruleRepo)
	ctx := authCtx("viewer-1", domain.RoleViewer)

	views, _, err := svc.EvaluateAll(ctx, "empty-proj", "env-1", domain.EvalContext{})
	if err != nil {
		t.Fatalf("EvaluateAll: %v", err)
	}
	if len(views) != 0 {
		t.Errorf("expected 0 views, got %d", len(views))
	}

	// @edge: batch load still called once even with zero flags.
	if ruleRepo.listByEnvironmentCalls != 1 {
		t.Errorf("expected ListByEnvironment called once, got %d", ruleRepo.listByEnvironmentCalls)
	}
}

// TestEvaluationService_Evaluate_PublisherError_DoesNotFail verifies ADR 0021:
// a publisher error inside the fire-and-forget goroutine must not affect the
// EvalView returned to the caller.
//
// @happy (Scenario 1 from grooming BDD)
func TestEvaluationService_Evaluate_PublisherError_DoesNotFail(t *testing.T) {
	const (
		projectID = "proj-pub-err"
		envID     = "env-pub-err"
	)

	flagRepo := newFakeFlagRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	ruleRepo := newCountingRuleRepository()

	flag := seedFlag(t, flagRepo, "flag-pub-err-id", projectID, "my-flag")
	seedState(t, stateRepo, flag.ID, envID, true)

	stub := &stubErrorPublisher{}
	svc := app.NewEvaluationService(flagRepo, stateRepo, ruleRepo, newFakeSegmentRepository(), stub)

	ctx := authCtx("viewer-1", domain.RoleViewer)

	// Add(1) before Evaluate so the WaitGroup counter is set before the goroutine runs.
	stub.wg.Add(1)
	view, err := svc.Evaluate(ctx, projectID, envID, "my-flag", domain.EvalContext{UserID: "user-1"})
	stub.Wait() // drain the goroutine before asserting

	if err != nil {
		t.Fatalf("Evaluate returned error when publisher failed: %v", err)
	}
	if view == nil {
		t.Fatal("Evaluate returned nil EvalView when publisher failed")
	}
	if view.Key != "my-flag" {
		t.Errorf("expected Key=my-flag, got %q", view.Key)
	}
	if view.ValueKey == "" {
		t.Errorf("expected non-empty ValueKey")
	}
	if view.Reason == "" {
		t.Errorf("expected non-empty Reason")
	}
}

// TestEvaluationService_Evaluate_NilPublisher_DoesNotFail verifies that the
// nil-publisher guard in publishEvent is preserved: no goroutine is spawned and
// Evaluate still returns a valid EvalView.
//
// @edge (Scenario 2 from grooming BDD)
func TestEvaluationService_Evaluate_NilPublisher_DoesNotFail(t *testing.T) {
	const (
		projectID = "proj-nil-pub"
		envID     = "env-nil-pub"
	)

	flagRepo := newFakeFlagRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	ruleRepo := newCountingRuleRepository()

	flag := seedFlag(t, flagRepo, "flag-nil-pub-id", projectID, "my-flag")
	seedState(t, stateRepo, flag.ID, envID, true)

	// nil publisher — newEvalSvc uses nil already, but we call NewEvaluationService
	// directly here to make the nil-guard contract explicit.
	svc := app.NewEvaluationService(flagRepo, stateRepo, ruleRepo, newFakeSegmentRepository(), nil)

	ctx := authCtx("viewer-1", domain.RoleViewer)
	view, err := svc.Evaluate(ctx, projectID, envID, "my-flag", domain.EvalContext{UserID: "user-1"})

	if err != nil {
		t.Fatalf("Evaluate returned error with nil publisher: %v", err)
	}
	if view == nil {
		t.Fatal("Evaluate returned nil EvalView with nil publisher")
	}
	if view.Key != "my-flag" {
		t.Errorf("expected Key=my-flag, got %q", view.Key)
	}
}

// TestEvaluationService_Evaluate_PublisherOK_DoesNotFail verifies that a
// successful publish does not corrupt the EvalView returned to the caller.
//
// @edge (Scenario 3 from grooming BDD)
func TestEvaluationService_Evaluate_PublisherOK_DoesNotFail(t *testing.T) {
	const (
		projectID = "proj-pub-ok"
		envID     = "env-pub-ok"
	)

	flagRepo := newFakeFlagRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	ruleRepo := newCountingRuleRepository()

	flag := seedFlag(t, flagRepo, "flag-pub-ok-id", projectID, "my-flag")
	seedState(t, stateRepo, flag.ID, envID, true)

	stub := &stubOKPublisher{}
	svc := app.NewEvaluationService(flagRepo, stateRepo, ruleRepo, newFakeSegmentRepository(), stub)

	ctx := authCtx("viewer-1", domain.RoleViewer)

	stub.wg.Add(1)
	view, err := svc.Evaluate(ctx, projectID, envID, "my-flag", domain.EvalContext{UserID: "user-1"})
	stub.Wait()

	if err != nil {
		t.Fatalf("Evaluate returned error when publisher succeeded: %v", err)
	}
	if view == nil {
		t.Fatal("Evaluate returned nil EvalView when publisher succeeded")
	}
	if view.Key != "my-flag" {
		t.Errorf("expected Key=my-flag, got %q", view.Key)
	}
	if view.ValueKey == "" {
		t.Errorf("expected non-empty ValueKey")
	}
}
