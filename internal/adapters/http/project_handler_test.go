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

// fakeProjectService is a test double for the projectService interface.
type fakeProjectService struct {
	projects  map[string]*domain.Project // keyed by slug
	mutateErr error                      // if set, mutating methods return this error
}

func newFakeProjectService() *fakeProjectService {
	return &fakeProjectService{projects: make(map[string]*domain.Project)}
}

func (f *fakeProjectService) Create(_ context.Context, name, slug string) (*domain.Project, error) {
	if f.mutateErr != nil {
		return nil, f.mutateErr
	}
	p := domain.Project{Name: name, Slug: slug}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if _, exists := f.projects[slug]; exists {
		return nil, domain.ErrConflict
	}
	p.ID = "test-uuid-" + slug
	p.CreatedAt = time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	f.projects[slug] = &p
	return &p, nil
}

func (f *fakeProjectService) GetBySlug(_ context.Context, slug string) (*domain.Project, error) {
	p, ok := f.projects[slug]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (f *fakeProjectService) List(_ context.Context) ([]*domain.Project, error) {
	result := make([]*domain.Project, 0, len(f.projects))
	for _, p := range f.projects {
		cp := *p
		result = append(result, &cp)
	}
	return result, nil
}

func (f *fakeProjectService) UpdateName(_ context.Context, slug, name string) (*domain.Project, error) {
	if f.mutateErr != nil {
		return nil, f.mutateErr
	}
	p, ok := f.projects[slug]
	if !ok {
		return nil, domain.ErrNotFound
	}
	candidate := domain.Project{Name: name, Slug: p.Slug}
	if err := candidate.Validate(); err != nil {
		return nil, err
	}
	p.Name = name
	cp := *p
	return &cp, nil
}

func (f *fakeProjectService) DeleteBySlug(_ context.Context, slug string) error {
	if f.mutateErr != nil {
		return f.mutateErr
	}
	if _, ok := f.projects[slug]; !ok {
		return domain.ErrNotFound
	}
	delete(f.projects, slug)
	return nil
}

// noopAuth is a middleware that passes requests through without authentication.
func noopAuth(next http.Handler) http.Handler { return next }

// requireAuth401 is a middleware that always rejects with 401.
func requireAuth401(_ http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
}

func newTestMux(svc *fakeProjectService, auth func(http.Handler) http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	httpadapter.NewProjectHandler(svc).RegisterRoutes(mux, auth)
	return mux
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestProjectHandler_List_EmptyReturnsWrappedArray(t *testing.T) {
	mux := newTestMux(newFakeProjectService(), noopAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	projects, ok := body["projects"]
	if !ok {
		t.Fatal("response missing 'projects' key")
	}
	if projects == nil {
		t.Error("'projects' must not be null")
	}
	arr, ok := projects.([]any)
	if !ok || len(arr) != 0 {
		t.Errorf("expected empty array, got %v", projects)
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestProjectHandler_Create_Succeeds(t *testing.T) {
	mux := newTestMux(newFakeProjectService(), noopAuth)
	body := `{"name":"Acme","slug":"acme"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201", rec.Code)
	}
	var p map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := p["id"].(string); !ok {
		t.Error("'id' must be a string")
	}
	if p["slug"] != "acme" {
		t.Errorf("slug: got %v", p["slug"])
	}
	if _, ok := p["created_at"].(string); !ok {
		t.Error("'created_at' must be a string (ISO 8601)")
	}
}

func TestProjectHandler_Create_DuplicateSlug_Returns409(t *testing.T) {
	svc := newFakeProjectService()
	svc.projects["acme"] = &domain.Project{ID: "x", Name: "Acme", Slug: "acme", CreatedAt: time.Now()}
	mux := newTestMux(svc, noopAuth)

	body := `{"name":"Acme2","slug":"acme"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status: got %d, want 409", rec.Code)
	}
}

func TestProjectHandler_Create_MissingFields_Returns400(t *testing.T) {
	mux := newTestMux(newFakeProjectService(), noopAuth)
	body := `{"name":"Acme"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestProjectHandler_Get_Succeeds(t *testing.T) {
	svc := newFakeProjectService()
	svc.projects["acme"] = &domain.Project{ID: "uid-1", Name: "Acme", Slug: "acme", CreatedAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)}
	mux := newTestMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
}

func TestProjectHandler_Get_UnknownSlug_Returns404(t *testing.T) {
	mux := newTestMux(newFakeProjectService(), noopAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/ghost", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Error("error body missing 'error' key")
	}
	if _, ok := body["message"]; !ok {
		t.Error("error body missing 'message' key")
	}
}

// ── PATCH ─────────────────────────────────────────────────────────────────────

func TestProjectHandler_Patch_UpdatesName(t *testing.T) {
	svc := newFakeProjectService()
	svc.projects["acme"] = &domain.Project{ID: "uid-1", Name: "Acme", Slug: "acme", CreatedAt: time.Now()}
	mux := newTestMux(svc, noopAuth)

	body := `{"name":"Acme Corp"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var p map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if p["name"] != "Acme Corp" {
		t.Errorf("name: got %v, want Acme Corp", p["name"])
	}
	if p["slug"] != "acme" {
		t.Errorf("slug changed unexpectedly: %v", p["slug"])
	}
}

func TestProjectHandler_Patch_AbsentName_IsNoOp(t *testing.T) {
	svc := newFakeProjectService()
	svc.projects["acme"] = &domain.Project{ID: "uid-1", Name: "Acme", Slug: "acme", CreatedAt: time.Now()}
	mux := newTestMux(svc, noopAuth)

	body := `{"slug":"ignored"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var p map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if p["name"] != "Acme" {
		t.Errorf("name changed unexpectedly: %v", p["name"])
	}
}

func TestProjectHandler_Patch_EmptyName_Returns400(t *testing.T) {
	svc := newFakeProjectService()
	svc.projects["acme"] = &domain.Project{ID: "uid-1", Name: "Acme", Slug: "acme", CreatedAt: time.Now()}
	mux := newTestMux(svc, noopAuth)

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestProjectHandler_Patch_MalformedJSON_Returns400(t *testing.T) {
	svc := newFakeProjectService()
	svc.projects["acme"] = &domain.Project{ID: "uid-1", Name: "Acme", Slug: "acme", CreatedAt: time.Now()}
	mux := newTestMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme", strings.NewReader(`{ not valid`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestProjectHandler_Patch_UnknownSlug_Returns404(t *testing.T) {
	mux := newTestMux(newFakeProjectService(), noopAuth)
	body := `{"name":"Whatever"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/ghost", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestProjectHandler_Delete_Returns204(t *testing.T) {
	svc := newFakeProjectService()
	svc.projects["acme"] = &domain.Project{ID: "uid-1", Name: "Acme", Slug: "acme", CreatedAt: time.Now()}
	mux := newTestMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/acme", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", rec.Code)
	}
}

func TestProjectHandler_Delete_UnknownSlug_Returns404(t *testing.T) {
	mux := newTestMux(newFakeProjectService(), noopAuth)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/ghost", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ── Auth ──────────────────────────────────────────────────────────────────────

func TestProjectHandler_Unauthenticated_Returns401(t *testing.T) {
	routes := []struct {
		method, path, body string
	}{
		{http.MethodGet, "/api/v1/projects", ""},
		{http.MethodPost, "/api/v1/projects", `{"name":"X","slug":"x"}`},
		{http.MethodGet, "/api/v1/projects/acme", ""},
		{http.MethodPatch, "/api/v1/projects/acme", `{"name":"X"}`},
		{http.MethodDelete, "/api/v1/projects/acme", ""},
	}
	mux := newTestMux(newFakeProjectService(), requireAuth401)

	for _, tc := range routes {
		var bodyReader *strings.Reader
		if tc.body != "" {
			bodyReader = strings.NewReader(tc.body)
		} else {
			bodyReader = strings.NewReader("")
		}
		req := httptest.NewRequest(tc.method, tc.path, bodyReader)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: got %d, want 401", tc.method, tc.path, rec.Code)
		}
	}
}

// ── Response shape ────────────────────────────────────────────────────────────

func TestProjectHandler_Get_IDIsString(t *testing.T) {
	svc := newFakeProjectService()
	svc.projects["acme"] = &domain.Project{ID: "uid-1", Name: "Acme", Slug: "acme", CreatedAt: time.Now()}
	mux := newTestMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var p map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := p["id"].(string); !ok {
		t.Errorf("'id' is not a string: %T %v", p["id"], p["id"])
	}
}

// ── RBAC ──────────────────────────────────────────────────────────────────────

func TestProjectHandler_Create_Forbidden_Returns403(t *testing.T) {
	svc := newFakeProjectService()
	svc.mutateErr = domain.ErrForbidden
	mux := newTestMux(svc, noopAuth)

	body := `{"name":"Acme","slug":"acme"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(body))
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

func TestProjectHandler_Patch_Forbidden_Returns403(t *testing.T) {
	svc := newFakeProjectService()
	svc.projects["acme"] = &domain.Project{ID: "uid-1", Name: "Acme", Slug: "acme", CreatedAt: time.Now()}
	svc.mutateErr = domain.ErrForbidden
	mux := newTestMux(svc, noopAuth)

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

func TestProjectHandler_Delete_Forbidden_Returns403(t *testing.T) {
	svc := newFakeProjectService()
	svc.projects["acme"] = &domain.Project{ID: "uid-1", Name: "Acme", Slug: "acme", CreatedAt: time.Now()}
	svc.mutateErr = domain.ErrForbidden
	mux := newTestMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/acme", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

func TestProjectHandler_Get_CreatedAtIsISO8601UTC(t *testing.T) {
	ts := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	svc := newFakeProjectService()
	svc.projects["acme"] = &domain.Project{ID: "uid-1", Name: "Acme", Slug: "acme", CreatedAt: ts}
	mux := newTestMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var p map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	createdAt, ok := p["created_at"].(string)
	if !ok {
		t.Fatalf("'created_at' is not a string: %T", p["created_at"])
	}
	parsed, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		t.Fatalf("'created_at' not ISO 8601: %q — %v", createdAt, err)
	}
	if parsed.Location() != time.UTC && parsed.Location().String() != "" {
		t.Errorf("'created_at' not UTC: %q", createdAt)
	}
}
