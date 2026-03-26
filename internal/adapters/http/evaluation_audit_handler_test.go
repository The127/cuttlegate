package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpadapter "github.com/The127/cuttlegate/internal/adapters/http"
	"github.com/The127/cuttlegate/internal/app"
	"github.com/The127/cuttlegate/internal/domain"
	"github.com/The127/cuttlegate/internal/domain/ports"
)

// rejectAllAuth is a test auth middleware that always returns 401.
func rejectAllAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized","message":"authentication required"}`)) //nolint:errcheck
	})
}

// fakeEvaluationAuditService is a test double for the evaluationAuditService interface.
type fakeEvaluationAuditService struct {
	views []*app.EvaluationEventView
	err   error
}

func (f *fakeEvaluationAuditService) ListEvaluations(_ context.Context, _, _, _ string, _ ports.EvaluationFilter) ([]*app.EvaluationEventView, error) {
	return f.views, f.err
}

func newAuditMux(svc *fakeEvaluationAuditService, auth func(http.Handler) http.Handler) *http.ServeMux {
	proj := &domain.Project{ID: "proj-1", Slug: "my-project", Name: "My Project", CreatedAt: time.Now()}
	env := &domain.Environment{ID: "env-1", ProjectID: "proj-1", Slug: "production", Name: "Production"}

	mux := http.NewServeMux()
	httpadapter.NewEvaluationAuditHandler(
		svc,
		&fakeProjResolver{projects: map[string]*domain.Project{proj.Slug: proj}},
		&fakeEnvResolver{envs: map[string]*domain.Environment{"proj-1/production": env}},
	).RegisterRoutes(mux, auth)
	return mux
}

// @happy: flag has evaluations; returns list with all seven fields.
func TestEvaluationAuditHandler_List_Happy(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	svc := &fakeEvaluationAuditService{
		views: []*app.EvaluationEventView{
			{
				ID:            "evt-1",
				FlagKey:       "my-flag",
				EnvironmentID: "env-1",
				UserID:        "user-1",
				InputContext:  `{"plan":"pro"}`,
				MatchedRuleID: "rule-001",
				VariantKey:    "variant-a",
				Reason:        "rule_match",
				OccurredAt:    now,
			},
		},
	}
	mux := newAuditMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/evaluations", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items, ok := resp["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("expected 1 item, got %v", resp["items"])
	}
	item := items[0].(map[string]interface{})
	if item["variant_key"] != "variant-a" {
		t.Errorf("variant_key: want variant-a, got %v", item["variant_key"])
	}
	if item["reason"] != "rule_match" {
		t.Errorf("reason: want rule_match, got %v", item["reason"])
	}
	// matched_rule must be an object (not null) when rule matched.
	rule, ok := item["matched_rule"].(map[string]interface{})
	if !ok {
		t.Errorf("matched_rule: want object, got %T %v", item["matched_rule"], item["matched_rule"])
	} else if rule["id"] != "rule-001" {
		t.Errorf("matched_rule.id: want rule-001, got %v", rule["id"])
	}
	// input_context must be a JSON object, not a string.
	if _, ok := item["input_context"].(map[string]interface{}); !ok {
		t.Errorf("input_context: want object, got %T", item["input_context"])
	}
}

// @happy: flag has no evaluations; returns empty items array.
func TestEvaluationAuditHandler_List_Empty(t *testing.T) {
	svc := &fakeEvaluationAuditService{views: []*app.EvaluationEventView{}}
	mux := newAuditMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/evaluations", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatalf("items must be an array, got %T", resp["items"])
	}
	if len(items) != 0 {
		t.Errorf("want 0 items, got %d", len(items))
	}
}

// @edge: no matched rule — matched_rule is null.
func TestEvaluationAuditHandler_List_NoRuleMatch(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	svc := &fakeEvaluationAuditService{
		views: []*app.EvaluationEventView{
			{
				ID:         "evt-2",
				FlagKey:    "my-flag",
				VariantKey: "default",
				Reason:     "no_match",
				OccurredAt: now,
			},
		},
	}
	mux := newAuditMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/evaluations", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items := resp["items"].([]interface{})
	item := items[0].(map[string]interface{})
	// matched_rule must be null (Go nil → JSON null) when no rule matched.
	if item["matched_rule"] != nil {
		t.Errorf("matched_rule: want null, got %v", item["matched_rule"])
	}
}

// @error-path: malformed before cursor returns 400 with error: invalid_cursor.
func TestEvaluationAuditHandler_List_InvalidCursor(t *testing.T) {
	svc := &fakeEvaluationAuditService{views: []*app.EvaluationEventView{}}
	mux := newAuditMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/evaluations?before=not-a-timestamp", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["error"] != "bad_request" {
		t.Errorf("error: want bad_request, got %q", body["error"])
	}
}

// @error-path: flag not found returns 404.
func TestEvaluationAuditHandler_List_FlagNotFound(t *testing.T) {
	svc := &fakeEvaluationAuditService{err: domain.ErrNotFound}
	mux := newAuditMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/no-such-flag/evaluations", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

// @auth-bypass: unauthenticated request returns 401.
func TestEvaluationAuditHandler_List_Unauthorized(t *testing.T) {
	svc := &fakeEvaluationAuditService{}
	mux := newAuditMux(svc, rejectAllAuth)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/evaluations", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}
