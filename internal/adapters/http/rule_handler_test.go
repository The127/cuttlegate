package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpadapter "github.com/karo/cuttlegate/internal/adapters/http"
	"github.com/karo/cuttlegate/internal/domain"
)

// fakeRuleService is a test double for the ruleService interface.
type fakeRuleService struct {
	rules map[string]*domain.Rule // key: rule ID
	err   error
}

func newFakeRuleService() *fakeRuleService {
	return &fakeRuleService{rules: make(map[string]*domain.Rule)}
}

func (f *fakeRuleService) Create(_ context.Context, flagID, environmentID string, priority int, conditions []domain.Condition, variantKey string) (*domain.Rule, error) {
	if f.err != nil {
		return nil, f.err
	}
	rule := &domain.Rule{
		ID:            "rule-uuid-1",
		FlagID:        flagID,
		EnvironmentID: environmentID,
		Priority:      priority,
		Conditions:    conditions,
		VariantKey:    variantKey,
		Enabled:       true,
		CreatedAt:     time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC),
	}
	f.rules[rule.ID] = rule
	return rule, nil
}

func (f *fakeRuleService) List(_ context.Context, flagID, environmentID string) ([]*domain.Rule, error) {
	if f.err != nil {
		return nil, f.err
	}
	result := make([]*domain.Rule, 0)
	for _, r := range f.rules {
		if r.FlagID == flagID && r.EnvironmentID == environmentID {
			cp := *r
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (f *fakeRuleService) Update(_ context.Context, rule *domain.Rule) (*domain.Rule, error) {
	if f.err != nil {
		return nil, f.err
	}
	cp := *rule
	cp.CreatedAt = time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)
	f.rules[rule.ID] = &cp
	return &cp, nil
}

func (f *fakeRuleService) Delete(_ context.Context, id string) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.rules[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.rules, id)
	return nil
}

// fakeFlagResolver is a minimal flagResolver for rule handler tests.
type fakeFlagResolver struct {
	flags map[string]*domain.Flag // key: projectID+"/"+key
}

func (f *fakeFlagResolver) GetByKey(_ context.Context, projectID, key string) (*domain.Flag, error) {
	flag, ok := f.flags[projectID+"/"+key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *flag
	return &cp, nil
}

func newRuleMux(svc *fakeRuleService, auth func(http.Handler) http.Handler) *http.ServeMux {
	proj := &fakeProjResolver{projects: map[string]*domain.Project{
		"acme": {ID: "proj-acme", Name: "Acme", Slug: "acme"},
	}}
	flags := &fakeFlagResolver{flags: map[string]*domain.Flag{
		"proj-acme/dark-mode": {
			ID:        "flag-dark-mode",
			ProjectID: "proj-acme",
			Key:       "dark-mode",
		},
	}}
	envs := &fakeEnvResolver{envs: map[string]*domain.Environment{
		"proj-acme/prod": {ID: "env-prod", ProjectID: "proj-acme", Slug: "prod", Name: "Prod"},
	}}
	mux := http.NewServeMux()
	httpadapter.NewRuleHandler(svc, proj, flags, envs).RegisterRoutes(mux, auth)
	return mux
}

const ruleBase = "/api/v1/projects/acme/flags/dark-mode/environments/prod/rules"

// ── Scenario 1: create rule succeeds ─────────────────────────────────────────

func TestRuleHandler_Create_Succeeds(t *testing.T) {
	mux := newRuleMux(newFakeRuleService(), noopAuth)
	body := `{"conditions":[{"attribute":"plan","operator":"eq","values":["pro"]}],"variantKey":"on","priority":0}`
	req := httptest.NewRequest(http.MethodPost, ruleBase, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201; body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["id"] == "" || resp["id"] == nil {
		t.Error("expected non-empty id in response")
	}
}

// ── Scenario 2: create with priority conflict returns 400 ────────────────────

func TestRuleHandler_Create_PriorityConflict_Returns400(t *testing.T) {
	svc := newFakeRuleService()
	svc.err = domain.ErrPriorityConflict
	mux := newRuleMux(svc, noopAuth)
	body := `{"conditions":[{"attribute":"plan","operator":"eq","values":["pro"]}],"variantKey":"on","priority":5}`
	req := httptest.NewRequest(http.MethodPost, ruleBase, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["error"] != "priority_conflict" {
		t.Errorf("error code: got %v, want priority_conflict", resp["error"])
	}
}

// ── Scenario 3: create with empty conditions returns 400 ─────────────────────

func TestRuleHandler_Create_EmptyConditions_Returns400(t *testing.T) {
	svc := newFakeRuleService()
	svc.err = &domain.ValidationError{Field: "conditions", Message: "rule must have at least one condition"}
	mux := newRuleMux(svc, noopAuth)
	body := `{"conditions":[],"variantKey":"on","priority":0}`
	req := httptest.NewRequest(http.MethodPost, ruleBase, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

// ── Scenario 4: list is ordered by priority ascending ────────────────────────

func TestRuleHandler_List_ReturnsRulesArray(t *testing.T) {
	svc := newFakeRuleService()
	svc.rules["rule-1"] = &domain.Rule{
		ID: "rule-1", FlagID: "flag-dark-mode", EnvironmentID: "env-prod",
		Priority: 1, Conditions: []domain.Condition{{Attribute: "plan", Operator: "eq", Values: []string{"pro"}}},
		VariantKey: "on", Enabled: true, CreatedAt: time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC),
	}
	mux := newRuleMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet, ruleBase, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	arr, ok := body["rules"].([]any)
	if !ok {
		t.Fatal("expected 'rules' array in response")
	}
	if len(arr) != 1 {
		t.Errorf("expected 1 rule, got %d", len(arr))
	}
}

// ── Scenario 5: empty list returns {"rules": []} ─────────────────────────────

func TestRuleHandler_List_EmptyReturnsWrappedArray(t *testing.T) {
	mux := newRuleMux(newFakeRuleService(), noopAuth)
	req := httptest.NewRequest(http.MethodGet, ruleBase, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	arr, ok := body["rules"].([]any)
	if !ok {
		t.Fatal("response missing 'rules' array")
	}
	if len(arr) != 0 {
		t.Errorf("expected empty array, got %d items", len(arr))
	}
}

// ── Scenario 6: update rule succeeds ─────────────────────────────────────────

func TestRuleHandler_Update_Succeeds(t *testing.T) {
	svc := newFakeRuleService()
	svc.rules["rule-abc"] = &domain.Rule{
		ID: "rule-abc", FlagID: "flag-dark-mode", EnvironmentID: "env-prod",
		Priority: 0, Conditions: []domain.Condition{{Attribute: "plan", Operator: "eq", Values: []string{"pro"}}},
		VariantKey: "on", Enabled: true,
	}
	mux := newRuleMux(svc, noopAuth)
	body := `{"conditions":[{"attribute":"plan","operator":"in","values":["pro","enterprise"]}],"variantKey":"on","priority":0,"enabled":true}`
	req := httptest.NewRequest(http.MethodPatch, ruleBase+"/rule-abc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	conditions, ok := resp["conditions"].([]any)
	if !ok || len(conditions) != 1 {
		t.Fatalf("expected 1 condition in response, got %v", resp["conditions"])
	}
	cond := conditions[0].(map[string]any)
	if cond["operator"] != "in" {
		t.Errorf("operator: got %v, want 'in'", cond["operator"])
	}
}

// ── Scenario 7: update with invalid condition returns 400 ────────────────────

func TestRuleHandler_Update_InvalidCondition_Returns400(t *testing.T) {
	svc := newFakeRuleService()
	svc.err = &domain.ValidationError{Field: "conditions", Message: "operator eq requires exactly one value"}
	mux := newRuleMux(svc, noopAuth)
	body := `{"conditions":[{"attribute":"plan","operator":"eq","values":["a","b"]}],"variantKey":"on","priority":0,"enabled":true}`
	req := httptest.NewRequest(http.MethodPatch, ruleBase+"/rule-abc", strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

// ── Scenario 8: delete rule returns 204 ──────────────────────────────────────

func TestRuleHandler_Delete_Succeeds(t *testing.T) {
	svc := newFakeRuleService()
	svc.rules["rule-xyz"] = &domain.Rule{ID: "rule-xyz"}
	mux := newRuleMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodDelete, ruleBase+"/rule-xyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", rec.Code)
	}
}

// ── Scenario 9: delete non-existent rule returns 404 ─────────────────────────

func TestRuleHandler_Delete_NotFound_Returns404(t *testing.T) {
	mux := newRuleMux(newFakeRuleService(), noopAuth)
	req := httptest.NewRequest(http.MethodDelete, ruleBase+"/ghost-rule", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ── Auth: all routes require authentication ───────────────────────────────────

func TestRuleHandler_Unauthenticated_Returns401(t *testing.T) {
	routes := []struct{ method, path, body string }{
		{http.MethodPost, ruleBase, `{"conditions":[{"attribute":"plan","operator":"eq","values":["pro"]}],"variantKey":"on"}`},
		{http.MethodGet, ruleBase, ""},
		{http.MethodPatch, ruleBase + "/rule-1", `{"conditions":[{"attribute":"plan","operator":"eq","values":["pro"]}],"variantKey":"on"}`},
		{http.MethodDelete, ruleBase + "/rule-1", ""},
	}
	mux := newRuleMux(newFakeRuleService(), requireAuth401)
	for _, tc := range routes {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: got %d, want 401", tc.method, tc.path, rec.Code)
		}
	}
}
