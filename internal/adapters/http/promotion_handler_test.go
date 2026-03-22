package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/karo/cuttlegate/internal/adapters/http"
	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

// fakePromotionService is a test double for the promotionService interface.
type fakePromotionService struct {
	diff  *app.FlagPromotionDiff
	diffs []*app.FlagPromotionDiff
	err   error
}

func (f *fakePromotionService) PromoteFlagState(_ context.Context, _, _, _, _ string) (*app.FlagPromotionDiff, error) {
	return f.diff, f.err
}

func (f *fakePromotionService) PromoteAllFlags(_ context.Context, _, _, _ string) ([]*app.FlagPromotionDiff, error) {
	return f.diffs, f.err
}

func newPromotionMux(svc *fakePromotionService, auth func(http.Handler) http.Handler) *http.ServeMux {
	proj := &fakeProjResolver{projects: map[string]*domain.Project{
		"acme": {ID: "proj-acme", Name: "Acme", Slug: "acme"},
	}}
	envs := &fakeEnvResolver{envs: map[string]*domain.Environment{
		"proj-acme/staging":    {ID: "env-staging", ProjectID: "proj-acme", Slug: "staging", Name: "Staging"},
		"proj-acme/production": {ID: "env-prod", ProjectID: "proj-acme", Slug: "production", Name: "Production"},
	}}
	mux := http.NewServeMux()
	httpadapter.NewPromotionHandler(svc, proj, envs).RegisterRoutes(mux, auth)
	return mux
}

func singleDiff(key string, enabledBefore, enabledAfter bool, added, removed int) *app.FlagPromotionDiff {
	return &app.FlagPromotionDiff{
		FlagKey:       key,
		EnabledBefore: enabledBefore,
		EnabledAfter:  enabledAfter,
		RulesAdded:    added,
		RulesRemoved:  removed,
	}
}

// ── PromoteFlagState ────────────────────────────────────────────────────────────

func TestPromotionHandler_PromoteFlag_Success(t *testing.T) {
	svc := &fakePromotionService{diff: singleDiff("dark-mode", false, true, 1, 0)}
	mux := newPromotionMux(svc, noopAuth)

	body := `{"target_env_slug":"production"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments/staging/flags/dark-mode/promote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200\nbody: %s", rec.Code, rec.Body)
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["flag_key"] != "dark-mode" {
		t.Errorf("flag_key: got %v, want dark-mode", resp["flag_key"])
	}
	if resp["enabled_after"] != true {
		t.Errorf("enabled_after: got %v, want true", resp["enabled_after"])
	}
}

func TestPromotionHandler_PromoteFlag_MissingTargetEnvSlug_Returns400(t *testing.T) {
	svc := &fakePromotionService{}
	mux := newPromotionMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments/staging/flags/dark-mode/promote", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestPromotionHandler_PromoteFlag_UnknownTargetEnv_Returns404(t *testing.T) {
	svc := &fakePromotionService{}
	mux := newPromotionMux(svc, noopAuth)

	body := `{"target_env_slug":"nonexistent"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments/staging/flags/dark-mode/promote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404 (cross-project guard via slug resolution)", rec.Code)
	}
}

func TestPromotionHandler_PromoteFlag_Forbidden_Returns403(t *testing.T) {
	svc := &fakePromotionService{err: domain.ErrForbidden}
	mux := newPromotionMux(svc, noopAuth)

	body := `{"target_env_slug":"production"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments/staging/flags/dark-mode/promote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

func TestPromotionHandler_PromoteFlag_SameEnv_Returns400(t *testing.T) {
	svc := &fakePromotionService{err: &domain.ValidationError{Field: "target_env", Message: "source and target environments must differ"}}
	mux := newPromotionMux(svc, noopAuth)

	body := `{"target_env_slug":"production"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments/staging/flags/dark-mode/promote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

// ── PromoteAllFlags ───────────────────────────────────────────────────────────

func TestPromotionHandler_PromoteAllFlags_Success(t *testing.T) {
	svc := &fakePromotionService{diffs: []*app.FlagPromotionDiff{
		singleDiff("flag-a", false, true, 0, 0),
		singleDiff("flag-b", true, false, 0, 1),
	}}
	mux := newPromotionMux(svc, noopAuth)

	body := `{"target_env_slug":"production"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments/staging/promote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200\nbody: %s", rec.Code, rec.Body)
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	arr, ok := resp["flags"].([]any)
	if !ok {
		t.Fatal("expected 'flags' array in response")
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(arr))
	}
}

func TestPromotionHandler_PromoteAllFlags_MissingTargetEnvSlug_Returns400(t *testing.T) {
	svc := &fakePromotionService{}
	mux := newPromotionMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments/staging/promote", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestPromotionHandler_PromoteAllFlags_Forbidden_Returns403(t *testing.T) {
	svc := &fakePromotionService{err: domain.ErrForbidden}
	mux := newPromotionMux(svc, noopAuth)

	body := `{"target_env_slug":"production"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments/staging/promote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

// ── Auth ──────────────────────────────────────────────────────────────────────

func TestPromotionHandler_Unauthenticated_Returns401(t *testing.T) {
	routes := []string{
		"/api/v1/projects/acme/environments/staging/flags/dark-mode/promote",
		"/api/v1/projects/acme/environments/staging/promote",
	}
	svc := &fakePromotionService{}
	mux := newPromotionMux(svc, requireAuth401)
	for _, path := range routes {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"target_env_slug":"production"}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("POST %s: got %d, want 401", path, rec.Code)
		}
	}
}
