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

// fakeEnvironmentService is a test double for the environmentService interface.
type fakeEnvironmentService struct {
	envs      map[string]*domain.Environment // key: projectID+"/"+slug
	byID      map[string]*domain.Environment
	projects  map[string]string // slug → ID (for Create)
	mutateErr error             // if set, mutating methods return this error
}

func newFakeEnvironmentService(projectSlugs ...string) *fakeEnvironmentService {
	svc := &fakeEnvironmentService{
		envs:     make(map[string]*domain.Environment),
		byID:     make(map[string]*domain.Environment),
		projects: make(map[string]string),
	}
	for _, s := range projectSlugs {
		svc.projects[s] = "proj-" + s
	}
	return svc
}

func ek(projectID, slug string) string { return projectID + "/" + slug }

func (f *fakeEnvironmentService) Create(_ context.Context, projectID, name, envSlug string) (*domain.Environment, error) {
	if f.mutateErr != nil {
		return nil, f.mutateErr
	}
	env := domain.Environment{Name: name, Slug: envSlug}
	if err := env.Validate(); err != nil {
		return nil, err
	}
	k := ek(projectID, envSlug)
	if _, exists := f.envs[k]; exists {
		return nil, domain.ErrConflict
	}
	env.ID = "env-" + envSlug
	env.ProjectID = projectID
	env.CreatedAt = time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	f.envs[k] = &env
	f.byID[env.ID] = &env
	return &env, nil
}

func (f *fakeEnvironmentService) GetBySlug(_ context.Context, projectID, slug string) (*domain.Environment, error) {
	e, ok := f.envs[ek(projectID, slug)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *e
	return &cp, nil
}

func (f *fakeEnvironmentService) ListByProject(_ context.Context, projectID string) ([]*domain.Environment, error) {
	result := make([]*domain.Environment, 0)
	for _, e := range f.envs {
		if e.ProjectID == projectID {
			cp := *e
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (f *fakeEnvironmentService) UpdateName(_ context.Context, projectID, slug, name string) (*domain.Environment, error) {
	if f.mutateErr != nil {
		return nil, f.mutateErr
	}
	k := ek(projectID, slug)
	e, ok := f.envs[k]
	if !ok {
		return nil, domain.ErrNotFound
	}
	e.Name = name
	cp := *e
	return &cp, nil
}

func (f *fakeEnvironmentService) DeleteBySlug(_ context.Context, projectID, slug string) error {
	if f.mutateErr != nil {
		return f.mutateErr
	}
	k := ek(projectID, slug)
	e, ok := f.envs[k]
	if !ok {
		return domain.ErrNotFound
	}
	delete(f.byID, e.ID)
	delete(f.envs, k)
	return nil
}

func newEnvMux(svc *fakeEnvironmentService, resolver *fakeFlagProjectResolver, auth func(http.Handler) http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	httpadapter.NewEnvironmentHandler(svc, resolver).RegisterRoutes(mux, auth)
	return mux
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestEnvironmentHandler_List_EmptyReturnsWrappedArray(t *testing.T) {
	mux := newEnvMux(newFakeEnvironmentService("acme"), newFakeResolver("acme"), noopAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme/environments", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	envs, ok := body["environments"]
	if !ok {
		t.Fatal("missing 'environments' key")
	}
	arr, ok := envs.([]any)
	if !ok || len(arr) != 0 {
		t.Errorf("expected empty array, got %v", envs)
	}
}

func TestEnvironmentHandler_List_UnknownProject_Returns404(t *testing.T) {
	mux := newEnvMux(newFakeEnvironmentService(), newFakeResolver(), noopAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/ghost/environments", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestEnvironmentHandler_Create_Succeeds(t *testing.T) {
	mux := newEnvMux(newFakeEnvironmentService("acme"), newFakeResolver("acme"), noopAuth)
	body := `{"name":"Production","slug":"prod"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201", rec.Code)
	}
	var e map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&e); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, field := range []string{"id", "project_id", "name", "slug", "created_at"} {
		if _, ok := e[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}
	if _, ok := e["id"].(string); !ok {
		t.Error("'id' must be a string")
	}
}

func TestEnvironmentHandler_Create_DuplicateSlug_Returns409(t *testing.T) {
	svc := newFakeEnvironmentService("acme")
	mux := newEnvMux(svc, newFakeResolver("acme"), noopAuth)
	body := `{"name":"Production","slug":"prod"}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if i == 1 && rec.Code != http.StatusConflict {
			t.Fatalf("second create: got %d, want 409", rec.Code)
		}
	}
}

func TestEnvironmentHandler_Create_SameSlugDifferentProjects_Returns201(t *testing.T) {
	svc := newFakeEnvironmentService("acme", "other")
	resolver := newFakeResolver("acme", "other")
	mux := newEnvMux(svc, resolver, noopAuth)
	body := `{"name":"Production","slug":"prod"}`

	for _, project := range []string{"acme", "other"} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+project+"/environments", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Errorf("project %s: got %d, want 201", project, rec.Code)
		}
	}
}

func TestEnvironmentHandler_Create_UsesProjectIDNotSlug(t *testing.T) {
	mux := newEnvMux(newFakeEnvironmentService("acme"), newFakeResolver("acme"), noopAuth)
	body := `{"name":"Staging","slug":"staging"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201", rec.Code)
	}
	var e map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&e); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// The project_id must be the resolved UUID, not the URL slug.
	pid, _ := e["project_id"].(string)
	if pid == "acme" {
		t.Error("project_id is the URL slug 'acme' instead of the resolved project ID")
	}
	if pid != "proj-acme" {
		t.Errorf("project_id: got %q, want %q", pid, "proj-acme")
	}
}

func TestEnvironmentHandler_Create_UnknownProject_Returns404(t *testing.T) {
	mux := newEnvMux(newFakeEnvironmentService(), newFakeResolver(), noopAuth)
	body := `{"name":"Production","slug":"prod"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/ghost/environments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestEnvironmentHandler_Get_Succeeds(t *testing.T) {
	svc := newFakeEnvironmentService("acme")
	resolver := newFakeResolver("acme")
	mux := newEnvMux(svc, resolver, noopAuth)

	// seed
	seedReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments",
		strings.NewReader(`{"name":"Production","slug":"prod"}`))
	seedReq.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(httptest.NewRecorder(), seedReq)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme/environments/prod", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
}

func TestEnvironmentHandler_Get_UnknownEnvSlug_Returns404(t *testing.T) {
	mux := newEnvMux(newFakeEnvironmentService("acme"), newFakeResolver("acme"), noopAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme/environments/ghost", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestEnvironmentHandler_Delete_Returns204(t *testing.T) {
	svc := newFakeEnvironmentService("acme")
	resolver := newFakeResolver("acme")
	mux := newEnvMux(svc, resolver, noopAuth)

	seedReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments",
		strings.NewReader(`{"name":"Production","slug":"prod"}`))
	seedReq.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(httptest.NewRecorder(), seedReq)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/acme/environments/prod", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", rec.Code)
	}
}

func TestEnvironmentHandler_Delete_UnknownEnv_Returns404(t *testing.T) {
	mux := newEnvMux(newFakeEnvironmentService("acme"), newFakeResolver("acme"), noopAuth)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/acme/environments/ghost", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ── Auth ──────────────────────────────────────────────────────────────────────

func TestEnvironmentHandler_Unauthenticated_Returns401(t *testing.T) {
	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/projects/acme/environments"},
		{http.MethodPost, "/api/v1/projects/acme/environments"},
		{http.MethodGet, "/api/v1/projects/acme/environments/prod"},
		{http.MethodPatch, "/api/v1/projects/acme/environments/prod"},
		{http.MethodDelete, "/api/v1/projects/acme/environments/prod"},
	}
	mux := newEnvMux(newFakeEnvironmentService("acme"), newFakeResolver("acme"), requireAuth401)
	for _, tc := range routes {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: got %d, want 401", tc.method, tc.path, rec.Code)
		}
	}
}

// ── RBAC ──────────────────────────────────────────────────────────────────────

func TestEnvironmentHandler_Create_Forbidden_Returns403(t *testing.T) {
	svc := newFakeEnvironmentService("acme")
	svc.mutateErr = domain.ErrForbidden
	mux := newEnvMux(svc, newFakeResolver("acme"), noopAuth)

	body := `{"name":"Production","slug":"prod"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments", strings.NewReader(body))
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

func TestEnvironmentHandler_Delete_Forbidden_Returns403(t *testing.T) {
	svc := newFakeEnvironmentService("acme")
	svc.mutateErr = domain.ErrForbidden
	mux := newEnvMux(svc, newFakeResolver("acme"), noopAuth)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/acme/environments/prod", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

// ── PATCH (Update) ────────────────────────────────────────────────────────────

func TestEnvironmentHandler_Update_Succeeds(t *testing.T) {
	svc := newFakeEnvironmentService("acme")
	resolver := newFakeResolver("acme")
	mux := newEnvMux(svc, resolver, noopAuth)

	// seed
	seedReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments",
		strings.NewReader(`{"name":"Staging","slug":"staging"}`))
	seedReq.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(httptest.NewRecorder(), seedReq)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/environments/staging",
		strings.NewReader(`{"name":"Pre-Production"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["name"] != "Pre-Production" {
		t.Errorf("name: got %v, want Pre-Production", body["name"])
	}
	if body["slug"] != "staging" {
		t.Errorf("slug should remain staging, got %v", body["slug"])
	}
}

func TestEnvironmentHandler_Update_EmptyName_Returns400(t *testing.T) {
	svc := newFakeEnvironmentService("acme")
	resolver := newFakeResolver("acme")
	mux := newEnvMux(svc, resolver, noopAuth)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/environments/staging",
		strings.NewReader(`{"name":""}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestEnvironmentHandler_Update_NotFound_Returns404(t *testing.T) {
	svc := newFakeEnvironmentService("acme")
	resolver := newFakeResolver("acme")
	mux := newEnvMux(svc, resolver, noopAuth)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/environments/ghost",
		strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

func TestEnvironmentHandler_Update_Forbidden_Returns403(t *testing.T) {
	svc := newFakeEnvironmentService("acme")
	svc.mutateErr = domain.ErrForbidden
	mux := newEnvMux(svc, newFakeResolver("acme"), noopAuth)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/environments/staging",
		strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

func TestEnvironmentHandler_Update_NilName_ReturnsCurrentState(t *testing.T) {
	svc := newFakeEnvironmentService("acme")
	resolver := newFakeResolver("acme")
	mux := newEnvMux(svc, resolver, noopAuth)

	// seed
	seedReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments",
		strings.NewReader(`{"name":"Staging","slug":"staging"}`))
	seedReq.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(httptest.NewRecorder(), seedReq)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/environments/staging",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["name"] != "Staging" {
		t.Errorf("name should remain Staging, got %v", body["name"])
	}
}
