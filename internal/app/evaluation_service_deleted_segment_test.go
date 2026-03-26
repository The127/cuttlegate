package app_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/The127/cuttlegate/internal/app"
	"github.com/The127/cuttlegate/internal/domain"
	"github.com/The127/cuttlegate/internal/domain/ports"
)

// capturingAuditRepository records all audit events in memory.
// Safe for concurrent access (though the deleted-segment path is synchronous).
type capturingAuditRepository struct {
	mu     sync.Mutex
	events []*domain.AuditEvent
}

var _ ports.AuditRepository = (*capturingAuditRepository)(nil)

func (r *capturingAuditRepository) Record(_ context.Context, event *domain.AuditEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *event
	r.events = append(r.events, &cp)
	return nil
}

func (r *capturingAuditRepository) ListByProject(_ context.Context, _ string, _ domain.AuditFilter) ([]*domain.AuditEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*domain.AuditEvent, len(r.events))
	copy(out, r.events)
	return out, nil
}

// newEvalSvcWithAudit constructs an EvaluationService with a capturing audit repo.
func newEvalSvcWithAudit(
	flagRepo *fakeFlagRepository,
	stateRepo *fakeFlagEnvironmentStateRepository,
	ruleRepo *countingRuleRepository,
	segRepo *fakeSegmentRepository,
	auditRepo ports.AuditRepository,
) *app.EvaluationService {
	return app.NewEvaluationService(flagRepo, stateRepo, ruleRepo, segRepo, nil).
		WithAuditRepo(auditRepo)
}

// seedSegmentRule adds a rule with an in_segment condition referencing segmentSlug.
func seedSegmentRule(t *testing.T, ruleRepo *countingRuleRepository, flagID, envID, segmentSlug string) {
	t.Helper()
	rule := &domain.Rule{
		ID:            "rule-seg-" + segmentSlug,
		FlagID:        flagID,
		EnvironmentID: envID,
		Priority:      1,
		Conditions: []domain.Condition{
			{Operator: domain.OperatorInSegment, Values: []string{segmentSlug}},
		},
		VariantKey: "true",
		Enabled:    true,
		CreatedAt:  time.Now().UTC(),
	}
	if err := ruleRepo.Upsert(context.Background(), rule); err != nil {
		t.Fatalf("seedSegmentRule: %v", err)
	}
}

// TestEvaluate_DeletedSegment_EmitsAuditEvent covers the @happy BDD scenario:
// a deleted segment reference produces an audit event with the correct fields.
func TestEvaluate_DeletedSegment_EmitsAuditEvent(t *testing.T) {
	const (
		projectID   = "proj-1"
		envID       = "env-1"
		segmentSlug = "beta"
	)

	flagRepo := newFakeFlagRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	ruleRepo := newCountingRuleRepository()
	segRepo := newFakeSegmentRepository() // segment "beta" is NOT seeded — simulates deletion
	auditRepo := &capturingAuditRepository{}

	flag := seedFlag(t, flagRepo, "flag-id-1", projectID, "my-flag")
	seedState(t, stateRepo, flag.ID, envID, true)
	seedSegmentRule(t, ruleRepo, flag.ID, envID, segmentSlug)

	svc := newEvalSvcWithAudit(flagRepo, stateRepo, ruleRepo, segRepo, auditRepo)
	ctx := authCtx("viewer-1", domain.RoleViewer)

	view, err := svc.Evaluate(ctx, projectID, envID, flag.Key, domain.EvalContext{UserID: "user-1"})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// @happy: evaluation result is unchanged — falls through to default variant.
	if view.ValueKey != flag.DefaultVariantKey {
		t.Errorf("expected default variant %q, got %q", flag.DefaultVariantKey, view.ValueKey)
	}

	// @happy: audit event was emitted.
	events, _ := auditRepo.ListByProject(context.Background(), projectID, domain.AuditFilter{})
	if len(events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(events))
	}

	ev := events[0]
	if ev.Action != "segment_not_found" {
		t.Errorf("expected Action %q, got %q", "segment_not_found", ev.Action)
	}
	if ev.EntityType != "segment" {
		t.Errorf("expected EntityType %q, got %q", "segment", ev.EntityType)
	}
	if ev.EntityKey != segmentSlug {
		t.Errorf("expected EntityKey %q, got %q", segmentSlug, ev.EntityKey)
	}

	// Verify AfterState JSON contains required fields.
	var payload struct {
		FlagID        string `json:"flag_id"`
		EnvironmentID string `json:"environment_id"`
		SegmentSlug   string `json:"segment_slug"`
		Reason        string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(ev.AfterState), &payload); err != nil {
		t.Fatalf("AfterState is not valid JSON: %v — raw: %q", err, ev.AfterState)
	}
	if payload.FlagID != flag.ID {
		t.Errorf("payload.flag_id: expected %q, got %q", flag.ID, payload.FlagID)
	}
	if payload.EnvironmentID != envID {
		t.Errorf("payload.environment_id: expected %q, got %q", envID, payload.EnvironmentID)
	}
	if payload.SegmentSlug != segmentSlug {
		t.Errorf("payload.segment_slug: expected %q, got %q", segmentSlug, payload.SegmentSlug)
	}
	if payload.Reason != "segment_not_found" {
		t.Errorf("payload.reason: expected %q, got %q", "segment_not_found", payload.Reason)
	}
}

// TestEvaluate_ExistingSegment_NoAuditEvent covers the @edge BDD scenario:
// an existing segment reference produces no segment_not_found audit event.
func TestEvaluate_ExistingSegment_NoAuditEvent(t *testing.T) {
	const (
		projectID   = "proj-1"
		envID       = "env-1"
		segmentSlug = "beta"
	)

	flagRepo := newFakeFlagRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	ruleRepo := newCountingRuleRepository()
	segRepo := newFakeSegmentRepository()
	auditRepo := &capturingAuditRepository{}

	flag := seedFlag(t, flagRepo, "flag-id-1", projectID, "my-flag")
	seedState(t, stateRepo, flag.ID, envID, true)
	seedSegmentRule(t, ruleRepo, flag.ID, envID, segmentSlug)

	// Seed the segment so it exists.
	if err := segRepo.Create(context.Background(), &domain.Segment{
		ID:        "seg-id-1",
		ProjectID: projectID,
		Slug:      segmentSlug,
		Name:      "Beta Users",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed segment: %v", err)
	}

	svc := newEvalSvcWithAudit(flagRepo, stateRepo, ruleRepo, segRepo, auditRepo)
	ctx := authCtx("viewer-1", domain.RoleViewer)

	if _, err := svc.Evaluate(ctx, projectID, envID, flag.Key, domain.EvalContext{UserID: "user-1"}); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// @edge: no audit event emitted for an existing segment.
	events, _ := auditRepo.ListByProject(context.Background(), projectID, domain.AuditFilter{})
	if len(events) != 0 {
		t.Errorf("expected 0 audit events, got %d", len(events))
	}
}

// TestEvaluate_DeletedSegment_NilAuditRepo_NoError verifies that when no
// auditRepo is configured, a deleted segment still silently skips without error.
func TestEvaluate_DeletedSegment_NilAuditRepo_NoError(t *testing.T) {
	const (
		projectID   = "proj-1"
		envID       = "env-1"
		segmentSlug = "beta"
	)

	flagRepo := newFakeFlagRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	ruleRepo := newCountingRuleRepository()
	segRepo := newFakeSegmentRepository() // segment not seeded

	flag := seedFlag(t, flagRepo, "flag-id-1", projectID, "my-flag")
	seedState(t, stateRepo, flag.ID, envID, true)
	seedSegmentRule(t, ruleRepo, flag.ID, envID, segmentSlug)

	// No WithAuditRepo — nil guard must prevent a panic.
	svc := app.NewEvaluationService(flagRepo, stateRepo, ruleRepo, segRepo, nil)
	ctx := authCtx("viewer-1", domain.RoleViewer)

	view, err := svc.Evaluate(ctx, projectID, envID, flag.Key, domain.EvalContext{UserID: "user-1"})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if view.ValueKey != flag.DefaultVariantKey {
		t.Errorf("expected default variant %q, got %q", flag.DefaultVariantKey, view.ValueKey)
	}
}

// TestEvaluate_DeletedSegment_AuditVisibleViaList covers the @happy BDD scenario:
// the audit event is visible via AuditService.List.
func TestEvaluate_DeletedSegment_AuditVisibleViaList(t *testing.T) {
	const (
		projectID   = "proj-1"
		envID       = "env-1"
		segmentSlug = "beta"
	)

	flagRepo := newFakeFlagRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	ruleRepo := newCountingRuleRepository()
	segRepo := newFakeSegmentRepository()
	auditRepo := &capturingAuditRepository{}

	flag := seedFlag(t, flagRepo, "flag-id-1", projectID, "my-flag")
	seedState(t, stateRepo, flag.ID, envID, true)
	seedSegmentRule(t, ruleRepo, flag.ID, envID, segmentSlug)

	svc := newEvalSvcWithAudit(flagRepo, stateRepo, ruleRepo, segRepo, auditRepo)
	ctx := authCtx("viewer-1", domain.RoleViewer)

	if _, err := svc.Evaluate(ctx, projectID, envID, flag.Key, domain.EvalContext{UserID: "user-1"}); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// Verify via AuditService.List (same repo, different service).
	auditSvc := app.NewAuditService(auditRepo)
	events, err := auditSvc.ListByProject(ctx, projectID, domain.AuditFilter{})
	if err != nil {
		t.Fatalf("AuditService.ListByProject: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event via AuditService, got %d", len(events))
	}
	if events[0].Action != "segment_not_found" {
		t.Errorf("expected Action %q, got %q", "segment_not_found", events[0].Action)
	}
}
