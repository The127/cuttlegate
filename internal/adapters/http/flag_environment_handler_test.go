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

// fakeFlagEnvService is a test double for the flagEnvService interface.
type fakeFlagEnvService struct {
	views      []*app.FlagEnvironmentView
	singleView *app.FlagEnvironmentView
	err        error
}

func (f *fakeFlagEnvService) ListByEnvironment(_ context.Context, _, _ string) ([]*app.FlagEnvironmentView, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.views, nil
}

func (f *fakeFlagEnvService) GetByKeyAndEnvironment(_ context.Context, _, _, _ string) (*app.FlagEnvironmentView, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.singleView, nil
}

func (f *fakeFlagEnvService) SetEnabled(_ context.Context, _ app.SetEnabledParams) error {
	return f.err
}

// fakeProjResolver is a minimal projectResolver for flag environment handler tests.
type fakeProjResolver struct {
	projects map[string]*domain.Project
}

func (f *fakeProjResolver) GetBySlug(_ context.Context, slug string) (*domain.Project, error) {
	p, ok := f.projects[slug]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

// fakeEnvResolver is a minimal environmentResolver.
type fakeEnvResolver struct {
	envs map[string]*domain.Environment // key: projectID+"/"+slug
}

func (f *fakeEnvResolver) GetBySlug(_ context.Context, projectID, slug string) (*domain.Environment, error) {
	e, ok := f.envs[projectID+"/"+slug]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return e, nil
}

func newFlagEnvMux(svc *fakeFlagEnvService, auth func(http.Handler) http.Handler) *http.ServeMux {
	proj := &fakeProjResolver{projects: map[string]*domain.Project{
		"acme": {ID: "proj-acme", Name: "Acme", Slug: "acme"},
	}}
	envs := &fakeEnvResolver{envs: map[string]*domain.Environment{
		"proj-acme/prod":    {ID: "env-prod", ProjectID: "proj-acme", Slug: "prod", Name: "Prod"},
		"proj-acme/dev":     {ID: "env-dev", ProjectID: "proj-acme", Slug: "dev", Name: "Dev"},
		"proj-acme/staging": {ID: "env-staging", ProjectID: "proj-acme", Slug: "staging", Name: "Staging"},
	}}
	mux := http.NewServeMux()
	httpadapter.NewFlagEnvironmentHandler(svc, proj, envs).RegisterRoutes(mux, auth)
	return mux
}

func boolFlag(id, key string, enabled bool) *app.FlagEnvironmentView {
	return &app.FlagEnvironmentView{
		Flag: &domain.Flag{
			ID:                id,
			ProjectID:         "proj-acme",
			Key:               key,
			Name:              key,
			Type:              domain.FlagTypeBool,
			Variants:          []domain.Variant{{Key: "true", Name: "On"}, {Key: "false", Name: "Off"}},
			DefaultVariantKey: "false",
			CreatedAt:         time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
		},
		Enabled: enabled,
	}
}

// ── Scenario 4: list includes enabled state ───────────────────────────────────

func TestFlagEnvironmentHandler_List_IncludesEnabledState(t *testing.T) {
	svc := &fakeFlagEnvService{
		views: []*app.FlagEnvironmentView{
			boolFlag("flag-1", "dark-mode", true),
		},
	}
	mux := newFlagEnvMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme/environments/prod/flags", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	arr, ok := body["flags"].([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("expected 1 flag, got %v", body["flags"])
	}
	flag := arr[0].(map[string]any)
	if flag["key"] != "dark-mode" {
		t.Errorf("key: got %v", flag["key"])
	}
	if flag["enabled"] != true {
		t.Errorf("enabled: got %v, want true", flag["enabled"])
	}
}

// ── Scenario 5: empty environment returns empty array ─────────────────────────

func TestFlagEnvironmentHandler_List_EmptyReturnsWrappedArray(t *testing.T) {
	svc := &fakeFlagEnvService{views: []*app.FlagEnvironmentView{}}
	mux := newFlagEnvMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme/environments/staging/flags", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	arr, ok := body["flags"].([]any)
	if !ok {
		t.Fatal("response missing 'flags' array")
	}
	if len(arr) != 0 {
		t.Errorf("expected empty array, got %d items", len(arr))
	}
}

// ── Scenario 8: unknown environment returns 404 ───────────────────────────────

func TestFlagEnvironmentHandler_List_UnknownEnvironment_Returns404(t *testing.T) {
	mux := newFlagEnvMux(&fakeFlagEnvService{}, noopAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme/environments/ghost/flags", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ── Scenario 9: PATCH toggles enabled state ───────────────────────────────────

func TestFlagEnvironmentHandler_SetEnabled_Succeeds(t *testing.T) {
	svc := &fakeFlagEnvService{}
	mux := newFlagEnvMux(svc, noopAuth)

	body := `{"enabled":true}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/environments/dev/flags/dark-mode", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["enabled"] != true {
		t.Errorf("enabled: got %v, want true", resp["enabled"])
	}
}

func TestFlagEnvironmentHandler_SetEnabled_MissingField_Returns400(t *testing.T) {
	mux := newFlagEnvMux(&fakeFlagEnvService{}, noopAuth)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/environments/dev/flags/dark-mode", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestFlagEnvironmentHandler_SetEnabled_NotFound_Returns404(t *testing.T) {
	svc := &fakeFlagEnvService{err: domain.ErrNotFound}
	mux := newFlagEnvMux(svc, noopAuth)
	body := `{"enabled":true}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/environments/dev/flags/ghost", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ── GET single flag with env state ───────────────────────────────────────────

func TestFlagEnvironmentHandler_GetByKey_ReturnsFlag(t *testing.T) {
	svc := &fakeFlagEnvService{
		singleView: boolFlag("flag-1", "dark-mode", true),
	}
	mux := newFlagEnvMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme/environments/prod/flags/dark-mode", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["key"] != "dark-mode" {
		t.Errorf("key: got %v", body["key"])
	}
	if body["enabled"] != true {
		t.Errorf("enabled: got %v, want true", body["enabled"])
	}
}

func TestFlagEnvironmentHandler_GetByKey_NotFound_Returns404(t *testing.T) {
	svc := &fakeFlagEnvService{err: domain.ErrNotFound}
	mux := newFlagEnvMux(svc, noopAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme/environments/prod/flags/ghost", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ── Auth: all routes require authentication ───────────────────────────────────

func TestFlagEnvironmentHandler_Unauthenticated_Returns401(t *testing.T) {
	routes := []struct {
		method, path, body string
	}{
		{http.MethodGet, "/api/v1/projects/acme/environments/prod/flags", ""},
		{http.MethodGet, "/api/v1/projects/acme/environments/prod/flags/dark-mode", ""},
		{http.MethodPatch, "/api/v1/projects/acme/environments/prod/flags/dark-mode", `{"enabled":true}`},
	}
	mux := newFlagEnvMux(&fakeFlagEnvService{}, requireAuth401)
	for _, tc := range routes {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: got %d, want 401", tc.method, tc.path, rec.Code)
		}
	}
}

// ── RBAC ──────────────────────────────────────────────────────────────────────

func TestFlagEnvironmentHandler_SetEnabled_Forbidden_Returns403(t *testing.T) {
	svc := &fakeFlagEnvService{err: domain.ErrForbidden}
	mux := newFlagEnvMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/environments/prod/flags/dark-mode",
		strings.NewReader(`{"enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
	var b map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&b); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if b["error"] != "forbidden" {
		t.Errorf("error code: got %v, want forbidden", b["error"])
	}
}
