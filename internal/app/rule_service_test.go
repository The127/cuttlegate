package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

// fakeRuleRepository is an in-memory implementation of ports.RuleRepository.
type fakeRuleRepository struct {
	rules map[string]*domain.Rule // key: rule ID
}

func newFakeRuleRepository() *fakeRuleRepository {
	return &fakeRuleRepository{rules: make(map[string]*domain.Rule)}
}

func (f *fakeRuleRepository) ListByFlagEnvironment(_ context.Context, flagID, environmentID string) ([]*domain.Rule, error) {
	result := make([]*domain.Rule, 0)
	for _, r := range f.rules {
		if r.FlagID == flagID && r.EnvironmentID == environmentID {
			cp := *r
			result = append(result, &cp)
		}
	}
	// Sort by priority ascending (simple insertion sort for test data).
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].Priority < result[j-1].Priority; j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}
	return result, nil
}

func (f *fakeRuleRepository) Upsert(_ context.Context, rule *domain.Rule) error {
	cp := *rule
	f.rules[rule.ID] = &cp
	return nil
}

func (f *fakeRuleRepository) Delete(_ context.Context, id string) error {
	if _, ok := f.rules[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.rules, id)
	return nil
}

func (f *fakeRuleRepository) ListByEnvironment(_ context.Context, environmentID string) ([]*domain.Rule, error) {
	result := make([]*domain.Rule, 0)
	for _, r := range f.rules {
		if r.EnvironmentID == environmentID {
			cp := *r
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (f *fakeRuleRepository) DeleteByFlagEnvironment(_ context.Context, flagID, environmentID string) error {
	for id, r := range f.rules {
		if r.FlagID == flagID && r.EnvironmentID == environmentID {
			delete(f.rules, id)
		}
	}
	return nil
}

func newRuleSvc() *app.RuleService {
	return app.NewRuleService(newFakeRuleRepository())
}

var validConditions = []domain.Condition{
	{Attribute: "plan", Operator: domain.OperatorEq, Values: []string{"pro"}},
}

// ── Create scenarios ──────────────────────────────────────────────────────────

func TestRuleService_Create_Succeeds(t *testing.T) {
	svc := newRuleSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	rule, err := svc.Create(ctx, "flag-1", "env-1", 0, validConditions, "on", "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rule.ID == "" {
		t.Error("expected non-empty ID after Create")
	}
	if rule.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestRuleService_Create_EmptyConditions_ReturnsError(t *testing.T) {
	svc := newRuleSvc()
	_, err := svc.Create(authCtx("editor-1", domain.RoleEditor), "flag-1", "env-1", 0, nil, "on", "", nil)
	if err == nil {
		t.Error("expected validation error for empty conditions, got nil")
	}
}

func TestRuleService_Create_InvalidConditionOperator_ReturnsError(t *testing.T) {
	svc := newRuleSvc()
	conditions := []domain.Condition{
		{Attribute: "plan", Operator: "fuzzy", Values: []string{"pro"}},
	}
	_, err := svc.Create(authCtx("editor-1", domain.RoleEditor), "flag-1", "env-1", 0, conditions, "on", "", nil)
	if err == nil {
		t.Error("expected validation error for unknown operator, got nil")
	}
}

func TestRuleService_Create_MissingVariantKey_ReturnsError(t *testing.T) {
	svc := newRuleSvc()
	_, err := svc.Create(authCtx("editor-1", domain.RoleEditor), "flag-1", "env-1", 0, validConditions, "", "", nil)
	if err == nil {
		t.Error("expected validation error for empty variant key, got nil")
	}
}

func TestRuleService_Create_DuplicatePriority_ReturnsPriorityConflict(t *testing.T) {
	svc := newRuleSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	_, err := svc.Create(ctx, "flag-1", "env-1", 5, validConditions, "on", "", nil)
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err = svc.Create(ctx, "flag-1", "env-1", 5, validConditions, "off", "", nil)
	if !errors.Is(err, domain.ErrPriorityConflict) {
		t.Errorf("expected ErrPriorityConflict, got %v", err)
	}
}

func TestRuleService_Create_DuplicatePriority_DifferentEnvironment_Succeeds(t *testing.T) {
	svc := newRuleSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	_, err := svc.Create(ctx, "flag-1", "env-1", 5, validConditions, "on", "", nil)
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err = svc.Create(ctx, "flag-1", "env-2", 5, validConditions, "off", "", nil)
	if err != nil {
		t.Errorf("expected success for same priority in different environment, got %v", err)
	}
}

// ── List scenarios ────────────────────────────────────────────────────────────

func TestRuleService_List_Empty_ReturnsEmptySlice(t *testing.T) {
	svc := newRuleSvc()
	rules, err := svc.List(context.Background(), "flag-1", "env-1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if rules == nil {
		t.Error("expected non-nil slice, got nil")
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestRuleService_List_OrderedByPriorityAscending(t *testing.T) {
	svc := newRuleSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	_, err := svc.Create(ctx, "flag-1", "env-1", 2, validConditions, "on", "", nil)
	if err != nil {
		t.Fatalf("Create priority 2: %v", err)
	}
	_, err = svc.Create(ctx, "flag-1", "env-1", 1, validConditions, "off", "", nil)
	if err != nil {
		t.Fatalf("Create priority 1: %v", err)
	}
	rules, err := svc.List(context.Background(), "flag-1", "env-1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].Priority != 1 || rules[1].Priority != 2 {
		t.Errorf("expected priority [1,2], got [%d,%d]", rules[0].Priority, rules[1].Priority)
	}
}

// ── Update scenarios ──────────────────────────────────────────────────────────

func TestRuleService_Update_Succeeds(t *testing.T) {
	svc := newRuleSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	rule, err := svc.Create(ctx, "flag-1", "env-1", 0, validConditions, "on", "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	rule.VariantKey = "off"
	updated, err := svc.Update(ctx, rule)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.VariantKey != "off" {
		t.Errorf("expected variantKey 'off', got %q", updated.VariantKey)
	}
}

func TestRuleService_Update_InvalidRule_ReturnsError(t *testing.T) {
	svc := newRuleSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	rule, err := svc.Create(ctx, "flag-1", "env-1", 0, validConditions, "on", "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	rule.Conditions = nil // make invalid
	_, err = svc.Update(ctx, rule)
	if err == nil {
		t.Error("expected validation error for empty conditions, got nil")
	}
}

func TestRuleService_Update_DuplicatePriority_ReturnsPriorityConflict(t *testing.T) {
	svc := newRuleSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	_, err := svc.Create(ctx, "flag-1", "env-1", 1, validConditions, "on", "", nil)
	if err != nil {
		t.Fatalf("Create rule at priority 1: %v", err)
	}
	rule2, err := svc.Create(ctx, "flag-1", "env-1", 2, validConditions, "off", "", nil)
	if err != nil {
		t.Fatalf("Create rule at priority 2: %v", err)
	}
	rule2.Priority = 1
	_, err = svc.Update(ctx, rule2)
	if !errors.Is(err, domain.ErrPriorityConflict) {
		t.Errorf("expected ErrPriorityConflict, got %v", err)
	}
}

func TestRuleService_Update_SamePriority_SameRule_Succeeds(t *testing.T) {
	svc := newRuleSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	rule, err := svc.Create(ctx, "flag-1", "env-1", 5, validConditions, "on", "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	rule.VariantKey = "off"
	_, err = svc.Update(ctx, rule)
	if err != nil {
		t.Errorf("expected no error updating rule without changing priority, got %v", err)
	}
}

// ── Delete scenarios ──────────────────────────────────────────────────────────

func TestRuleService_Delete_Succeeds(t *testing.T) {
	svc := newRuleSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	rule, err := svc.Create(ctx, "flag-1", "env-1", 0, validConditions, "on", "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Delete(ctx, rule.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	rules, _ := svc.List(context.Background(), "flag-1", "env-1")
	if len(rules) != 0 {
		t.Errorf("expected 0 rules after delete, got %d", len(rules))
	}
}

func TestRuleService_Delete_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := newRuleSvc()
	err := svc.Delete(authCtx("editor-1", domain.RoleEditor), "nonexistent-id")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── RBAC scenarios ────────────────────────────────────────────────────────────

func TestRuleService_Create_ViewerReturnsForbidden(t *testing.T) {
	svc := newRuleSvc()
	_, err := svc.Create(authCtx("viewer-1", domain.RoleViewer), "flag-1", "env-1", 0, validConditions, "on", "", nil)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestRuleService_Delete_ViewerReturnsForbidden(t *testing.T) {
	svc := newRuleSvc()
	err := svc.Delete(authCtx("viewer-1", domain.RoleViewer), "rule-1")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestRuleService_Create_NoAuthContextReturnsForbidden(t *testing.T) {
	svc := newRuleSvc()
	_, err := svc.Create(context.Background(), "flag-1", "env-1", 0, validConditions, "on", "", nil)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden for missing auth, got %v", err)
	}
}
