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
	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

// fakeEvaluationService is a test double for the evaluationService interface.
type fakeEvaluationService struct {
	view *app.EvalView
	err  error

	views       []app.EvalView
	evaluatedAt time.Time
	bulkErr     error
}

func (f *fakeEvaluationService) Evaluate(_ context.Context, _, _, _ string, _ domain.EvalContext) (*app.EvalView, error) {
	return f.view, f.err
}

func (f *fakeEvaluationService) EvaluateAll(_ context.Context, _, _ string, _ domain.EvalContext) ([]app.EvalView, time.Time, error) {
	return f.views, f.evaluatedAt, f.bulkErr
}

func newEvalMux(svc *fakeEvaluationService, auth func(http.Handler) http.Handler) *http.ServeMux {
	proj := &domain.Project{ID: "proj-1", Slug: "my-project", Name: "My Project", CreatedAt: time.Now()}
	env := &domain.Environment{ID: "env-1", ProjectID: "proj-1", Slug: "production", Name: "Production"}

	mux := http.NewServeMux()
	httpadapter.NewEvaluationHandler(
		svc,
		&fakeProjResolver{projects: map[string]*domain.Project{proj.Slug: proj}},
		&fakeEnvResolver{envs: map[string]*domain.Environment{"proj-1/production": env}},
	).RegisterRoutes(mux, auth)
	return mux
}

func TestEvaluationHandler_Evaluate_Disabled(t *testing.T) {
	svc := &fakeEvaluationService{view: &app.EvalView{
		Key:     "my-flag",
		Enabled: false,
		Value:   nil,
		Reason:  domain.ReasonDisabled,
		Type:    domain.FlagTypeBool,
	}}
	mux := newEvalMux(svc, noopAuth)

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/evaluate",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if resp["enabled"] != false {
		t.Errorf("expected enabled=false")
	}
	if resp["reason"] != "disabled" {
		t.Errorf("expected reason=disabled, got %v", resp["reason"])
	}
	if _, hasValue := resp["value"]; !hasValue {
		t.Errorf("value field must be present (as null)")
	}
	if resp["type"] != "bool" {
		t.Errorf("expected type=bool, got %v", resp["type"])
	}
}

func TestEvaluationHandler_Evaluate_RuleMatch(t *testing.T) {
	variant := "variant-a"
	svc := &fakeEvaluationService{view: &app.EvalView{
		Key:     "my-flag",
		Enabled: true,
		Value:   &variant,
		Reason:  domain.ReasonRuleMatch,
		Type:    domain.FlagTypeString,
	}}
	mux := newEvalMux(svc, noopAuth)

	body := `{"context":{"user_id":"u_1","attributes":{"plan":"pro"}}}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/evaluate",
		strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if resp["reason"] != "rule_match" {
		t.Errorf("expected reason=rule_match, got %v", resp["reason"])
	}
	if resp["value"] != "variant-a" {
		t.Errorf("expected value=variant-a, got %v", resp["value"])
	}
	if resp["type"] != "string" {
		t.Errorf("expected type=string, got %v", resp["type"])
	}
}

func TestEvaluationHandler_Evaluate_FlagNotFound(t *testing.T) {
	svc := &fakeEvaluationService{err: domain.ErrNotFound}
	mux := newEvalMux(svc, noopAuth)

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/my-project/environments/production/flags/missing/evaluate",
		strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestEvaluationHandler_Evaluate_MissingContext(t *testing.T) {
	svc := &fakeEvaluationService{}
	mux := newEvalMux(svc, noopAuth)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/evaluate",
		strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if resp["message"] != "context is required" {
		t.Errorf("expected 'context is required', got %v", resp["message"])
	}
}

func TestEvaluationHandler_Evaluate_MalformedBody(t *testing.T) {
	svc := &fakeEvaluationService{}
	mux := newEvalMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/evaluate",
		strings.NewReader(`not json`))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// Scenario A: nonexistent project slug must return 403, not 404 — existence oracle fix.
func TestEvaluationHandler_Evaluate_ProjectNotFound_Returns403(t *testing.T) {
	svc := &fakeEvaluationService{}
	mux := newEvalMux(svc, noopAuth)

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/nonexistent/environments/production/flags/my-flag/evaluate",
		strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// Scenario B: valid project but nonexistent env slug must return 403, not 404 — existence oracle fix.
func TestEvaluationHandler_Evaluate_EnvNotFound_Returns403(t *testing.T) {
	svc := &fakeEvaluationService{}
	mux := newEvalMux(svc, noopAuth)

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/my-project/environments/nonexistent/flags/my-flag/evaluate",
		strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// ── Auth ──────────────────────────────────────────────────────────────────────

func TestEvaluationHandler_Unauthenticated_Returns401(t *testing.T) {
	svc := &fakeEvaluationService{}
	mux := newEvalMux(svc, requireAuth401)

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/evaluate",
		strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ── RBAC ──────────────────────────────────────────────────────────────────────

func TestEvaluationHandler_Evaluate_Forbidden_Returns403(t *testing.T) {
	svc := &fakeEvaluationService{err: domain.ErrForbidden}
	mux := newEvalMux(svc, noopAuth)

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/evaluate",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	var b map[string]any
	json.NewDecoder(w.Body).Decode(&b) //nolint:errcheck
	if b["error"] != "forbidden" {
		t.Errorf("error code: got %v, want forbidden", b["error"])
	}
}

// ── Bulk evaluation ──────────────────────────────────────────────────────────

const bulkURL = "/api/v1/projects/my-project/environments/production/evaluate"

func TestEvaluationHandler_EvaluateAll_MultipleFlags(t *testing.T) {
	variantA := "variant-a"
	ts := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	svc := &fakeEvaluationService{
		views: []app.EvalView{
			{Key: "flag-1", Enabled: true, Value: &variantA, Reason: domain.ReasonRuleMatch, Type: domain.FlagTypeString},
			{Key: "flag-2", Enabled: false, Value: nil, Reason: domain.ReasonDisabled, Type: domain.FlagTypeBool},
			{Key: "flag-3", Enabled: true, Value: nil, Reason: domain.ReasonDefault, Type: domain.FlagTypeBool},
		},
		evaluatedAt: ts,
	}
	mux := newEvalMux(svc, noopAuth)

	body := `{"context":{"user_id":"u_1","attributes":{"plan":"pro"}}}`
	req := httptest.NewRequest(http.MethodPost, bulkURL, strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck

	flags, ok := resp["flags"].([]any)
	if !ok {
		t.Fatalf("expected flags array")
	}
	if len(flags) != 3 {
		t.Fatalf("expected 3 flags, got %d", len(flags))
	}

	first := flags[0].(map[string]any)
	if first["key"] != "flag-1" {
		t.Errorf("expected key=flag-1, got %v", first["key"])
	}
	if first["enabled"] != true {
		t.Errorf("expected enabled=true")
	}
	if first["value"] != "variant-a" {
		t.Errorf("expected value=variant-a, got %v", first["value"])
	}
	if first["reason"] != "rule_match" {
		t.Errorf("expected reason=rule_match, got %v", first["reason"])
	}
	if first["type"] != "string" {
		t.Errorf("expected type=string, got %v", first["type"])
	}

	if resp["evaluated_at"] != "2026-03-20T10:00:00Z" {
		t.Errorf("expected evaluated_at=2026-03-20T10:00:00Z, got %v", resp["evaluated_at"])
	}
}

func TestEvaluationHandler_EvaluateAll_EmptyProject(t *testing.T) {
	ts := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	svc := &fakeEvaluationService{
		views:       []app.EvalView{},
		evaluatedAt: ts,
	}
	mux := newEvalMux(svc, noopAuth)

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost, bulkURL, strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck

	flags, ok := resp["flags"].([]any)
	if !ok {
		t.Fatalf("expected flags array")
	}
	if len(flags) != 0 {
		t.Errorf("expected empty flags array, got %d items", len(flags))
	}
	if resp["evaluated_at"] == nil || resp["evaluated_at"] == "" {
		t.Errorf("evaluated_at must be present even for empty results")
	}
}

func TestEvaluationHandler_EvaluateAll_MissingContext(t *testing.T) {
	svc := &fakeEvaluationService{}
	mux := newEvalMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodPost, bulkURL, strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestEvaluationHandler_EvaluateAll_Unauthenticated(t *testing.T) {
	svc := &fakeEvaluationService{}
	mux := newEvalMux(svc, requireAuth401)

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost, bulkURL, strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestEvaluationHandler_EvaluateAll_ProjectNotFound_Returns403(t *testing.T) {
	svc := &fakeEvaluationService{}
	mux := newEvalMux(svc, noopAuth)

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/nonexistent/environments/production/evaluate",
		strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 (existence oracle), got %d", w.Code)
	}
}

func TestEvaluationHandler_EvaluateAll_EnvNotFound_Returns403(t *testing.T) {
	svc := &fakeEvaluationService{}
	mux := newEvalMux(svc, noopAuth)

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/my-project/environments/nonexistent/evaluate",
		strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 (existence oracle), got %d", w.Code)
	}
}

func TestEvaluationHandler_EvaluateAll_Forbidden_Returns403(t *testing.T) {
	svc := &fakeEvaluationService{bulkErr: domain.ErrForbidden}
	mux := newEvalMux(svc, noopAuth)

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost, bulkURL, strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}
