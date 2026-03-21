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
}

func (f *fakeEvaluationService) Evaluate(_ context.Context, _, _, _ string, _ domain.EvalContext) (*app.EvalView, error) {
	return f.view, f.err
}

func newEvalMux(svc *fakeEvaluationService) *http.ServeMux {
	return newEvalMuxWithCORS(svc, nil)
}

func newEvalMuxWithCORS(svc *fakeEvaluationService, corsOrigins []string) *http.ServeMux {
	proj := &domain.Project{ID: "proj-1", Slug: "my-project", Name: "My Project", CreatedAt: time.Now()}
	env := &domain.Environment{ID: "env-1", ProjectID: "proj-1", Slug: "production", Name: "Production"}

	mux := http.NewServeMux()
	noAuth := func(h http.Handler) http.Handler { return h }
	httpadapter.NewEvaluationHandler(
		svc,
		&fakeProjResolver{projects: map[string]*domain.Project{proj.Slug: proj}},
		&fakeEnvResolver{envs: map[string]*domain.Environment{"proj-1/production": env}},
		corsOrigins,
	).RegisterRoutes(mux, noAuth)
	return mux
}

func TestEvaluationHandler_Evaluate_Disabled(t *testing.T) {
	svc := &fakeEvaluationService{view: &app.EvalView{
		Key:     "my-flag",
		Enabled: false,
		Value:   nil,
		Reason:  domain.ReasonDisabled,
	}}
	mux := newEvalMux(svc)

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
}

func TestEvaluationHandler_Evaluate_RuleMatch(t *testing.T) {
	variant := "variant-a"
	svc := &fakeEvaluationService{view: &app.EvalView{
		Key:     "my-flag",
		Enabled: true,
		Value:   &variant,
		Reason:  domain.ReasonRuleMatch,
	}}
	mux := newEvalMux(svc)

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
}

func TestEvaluationHandler_Evaluate_FlagNotFound(t *testing.T) {
	svc := &fakeEvaluationService{err: domain.ErrNotFound}
	mux := newEvalMux(svc)

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
	mux := newEvalMux(svc)

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
	mux := newEvalMux(svc)

	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/my-project/environments/production/flags/my-flag/evaluate",
		strings.NewReader(`not json`))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestEvaluationHandler_Evaluate_ProjectNotFound(t *testing.T) {
	svc := &fakeEvaluationService{}
	mux := newEvalMux(svc)

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/nonexistent/environments/production/flags/my-flag/evaluate",
		strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

const evalPath = "/api/v1/projects/my-project/environments/production/flags/my-flag/evaluate"

func TestEvaluationHandler_CORS_Preflight_AllowedOrigin(t *testing.T) {
	svc := &fakeEvaluationService{}
	mux := newEvalMuxWithCORS(svc, []string{"https://app.example.com"})

	req := httptest.NewRequest(http.MethodOptions, evalPath, nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 preflight, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("expected ACAO=https://app.example.com, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("expected Access-Control-Allow-Methods to be set")
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("expected Access-Control-Allow-Headers to be set")
	}
	if got := w.Header().Get("Vary"); got == "" {
		t.Error("expected Vary header to be set")
	}
}

func TestEvaluationHandler_CORS_Preflight_DisallowedOrigin(t *testing.T) {
	svc := &fakeEvaluationService{}
	mux := newEvalMuxWithCORS(svc, []string{"https://app.example.com"})

	req := httptest.NewRequest(http.MethodOptions, evalPath, nil)
	req.Header.Set("Origin", "https://evil.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO header for disallowed origin, got %q", got)
	}
}

func TestEvaluationHandler_CORS_ActualRequest_AllowedOrigin(t *testing.T) {
	svc := &fakeEvaluationService{view: &app.EvalView{
		Key: "my-flag", Enabled: true, Reason: domain.ReasonDefault,
	}}
	mux := newEvalMuxWithCORS(svc, []string{"https://app.example.com"})

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost, evalPath, strings.NewReader(body))
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("expected ACAO=https://app.example.com, got %q", got)
	}
}

func TestEvaluationHandler_CORS_ActualRequest_DisallowedOrigin(t *testing.T) {
	svc := &fakeEvaluationService{view: &app.EvalView{
		Key: "my-flag", Enabled: true, Reason: domain.ReasonDefault,
	}}
	mux := newEvalMuxWithCORS(svc, []string{"https://app.example.com"})

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost, evalPath, strings.NewReader(body))
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO header for disallowed origin, got %q", got)
	}
}

func TestEvaluationHandler_CORS_NoOrigin_PassesThrough(t *testing.T) {
	svc := &fakeEvaluationService{view: &app.EvalView{
		Key: "my-flag", Enabled: true, Reason: domain.ReasonDefault,
	}}
	mux := newEvalMuxWithCORS(svc, []string{"https://app.example.com"})

	body := `{"context":{"user_id":"u_1","attributes":{}}}`
	req := httptest.NewRequest(http.MethodPost, evalPath, strings.NewReader(body))
	// No Origin header — server-side SDK
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO header when no Origin sent, got %q", got)
	}
}

func TestEvaluationHandler_CORS_Disabled_OptionsReturns405(t *testing.T) {
	svc := &fakeEvaluationService{}
	mux := newEvalMuxWithCORS(svc, nil) // CORS disabled

	req := httptest.NewRequest(http.MethodOptions, evalPath, nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// OPTIONS is not registered when CORS is disabled — mux returns 405
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 when CORS disabled, got %d", w.Code)
	}
}
