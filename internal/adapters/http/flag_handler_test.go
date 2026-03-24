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

// fakeFlagService is a test double for the flagService interface.
type fakeFlagService struct {
	flags     map[string]*domain.Flag // key: projectID+"/"+key
	byID      map[string]*domain.Flag
	mutateErr error // returned by Create, Update, DeleteByKey
}

func newFakeFlagService() *fakeFlagService {
	return &fakeFlagService{
		flags: make(map[string]*domain.Flag),
		byID:  make(map[string]*domain.Flag),
	}
}

func fk(projectID, key string) string { return projectID + "/" + key }

func (f *fakeFlagService) Create(_ context.Context, flag *domain.Flag) error {
	if f.mutateErr != nil {
		return f.mutateErr
	}
	k := fk(flag.ProjectID, flag.Key)
	if _, exists := f.flags[k]; exists {
		return domain.ErrConflict
	}
	flag.ID = "flag-uuid-" + flag.Key
	flag.CreatedAt = time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	cp := *flag
	f.flags[k] = &cp
	f.byID[cp.ID] = &cp
	return nil
}

func (f *fakeFlagService) GetByKey(_ context.Context, projectID, key string) (*domain.Flag, error) {
	flag, ok := f.flags[fk(projectID, key)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *flag
	return &cp, nil
}

func (f *fakeFlagService) ListByProject(_ context.Context, projectID string) ([]*domain.Flag, error) {
	result := make([]*domain.Flag, 0)
	for _, flag := range f.flags {
		if flag.ProjectID == projectID {
			cp := *flag
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (f *fakeFlagService) ListByProjectPaginated(ctx context.Context, projectID string, filter domain.FlagListFilter) ([]*domain.Flag, int, error) {
	filter.Normalize()
	all, _ := f.ListByProject(ctx, projectID)
	// Simple in-memory pagination for tests
	total := len(all)
	start := (filter.Page - 1) * filter.PerPage
	if start > total {
		return []*domain.Flag{}, total, nil
	}
	end := start + filter.PerPage
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

func (f *fakeFlagService) Update(_ context.Context, flag *domain.Flag) error {
	if f.mutateErr != nil {
		return f.mutateErr
	}
	existing, ok := f.byID[flag.ID]
	if !ok {
		return domain.ErrNotFound
	}
	existing.Name = flag.Name
	existing.Variants = flag.Variants
	existing.DefaultVariantKey = flag.DefaultVariantKey
	return nil
}

func (f *fakeFlagService) DeleteByKey(_ context.Context, projectID, key string) error {
	if f.mutateErr != nil {
		return f.mutateErr
	}
	k := fk(projectID, key)
	flag, ok := f.flags[k]
	if !ok {
		return domain.ErrNotFound
	}
	delete(f.byID, flag.ID)
	delete(f.flags, k)
	return nil
}

// fakeFlagProjectResolver is a minimal projectResolver for flag handler tests.
type fakeFlagProjectResolver struct {
	projects map[string]*domain.Project
}

func newFakeResolver(slugs ...string) *fakeFlagProjectResolver {
	r := &fakeFlagProjectResolver{projects: make(map[string]*domain.Project)}
	for _, s := range slugs {
		r.projects[s] = &domain.Project{ID: "proj-" + s, Slug: s, Name: s, CreatedAt: time.Now()}
	}
	return r
}

func (r *fakeFlagProjectResolver) GetBySlug(_ context.Context, slug string) (*domain.Project, error) {
	p, ok := r.projects[slug]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func newFlagMux(svc *fakeFlagService, resolver *fakeFlagProjectResolver, auth func(http.Handler) http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	httpadapter.NewFlagHandler(svc, resolver).RegisterRoutes(mux, auth)
	return mux
}

var boolFlagBody = `{"key":"dark-mode","name":"Dark Mode","type":"bool","variants":[{"key":"true","name":"On"},{"key":"false","name":"Off"}],"default_variant_key":"false"}`

// ── List ─────────────────────────────────────────────────────────────────────

func TestFlagHandler_List_EmptyReturnsWrappedArray(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme/flags", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	flags, ok := body["flags"]
	if !ok {
		t.Fatal("missing 'flags' key")
	}
	arr, ok := flags.([]any)
	if !ok || len(arr) != 0 {
		t.Errorf("expected empty array, got %v", flags)
	}
	// Verify pagination envelope fields
	if body["total"] != float64(0) {
		t.Errorf("total: got %v, want 0", body["total"])
	}
	if body["page"] != float64(1) {
		t.Errorf("page: got %v, want 1", body["page"])
	}
	if body["per_page"] != float64(50) {
		t.Errorf("per_page: got %v, want 50", body["per_page"])
	}
}

func TestFlagHandler_List_UnknownProject_Returns404(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver(), noopAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/ghost/flags", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestFlagHandler_Create_Succeeds(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/flags", strings.NewReader(boolFlagBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201", rec.Code)
	}
	var f map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&f); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, field := range []string{"id", "project_id", "key", "name", "type", "variants", "default_variant_key", "created_at"} {
		if _, ok := f[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}
	if f["key"] != "dark-mode" {
		t.Errorf("key: got %v", f["key"])
	}
}

func TestFlagHandler_Create_DuplicateKey_Returns409(t *testing.T) {
	svc := newFakeFlagService()
	resolver := newFakeResolver("acme")
	mux := newFlagMux(svc, resolver, noopAuth)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/flags", strings.NewReader(boolFlagBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if i == 1 && rec.Code != http.StatusConflict {
			t.Fatalf("second create: got %d, want 409", rec.Code)
		}
	}
}

func TestFlagHandler_Create_InvalidJSON_Returns400(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/flags", strings.NewReader(`{ not valid`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestFlagHandler_Get_Succeeds(t *testing.T) {
	svc := newFakeFlagService()
	resolver := newFakeResolver("acme")
	mux := newFlagMux(svc, resolver, noopAuth)

	// seed
	httptest.NewRecorder() // discard
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/flags", strings.NewReader(boolFlagBody))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(httptest.NewRecorder(), req)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme/flags/dark-mode", nil)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec2.Code)
	}
}

func TestFlagHandler_Get_UnknownKey_Returns404(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme/flags/ghost", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ── PATCH ─────────────────────────────────────────────────────────────────────

func seedFlag(t *testing.T, mux *http.ServeMux) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/flags", strings.NewReader(boolFlagBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed flag: got %d", rec.Code)
	}
}

func TestFlagHandler_Patch_UpdatesName(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	seedFlag(t, mux)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/flags/dark-mode",
		strings.NewReader(`{"name":"Dark Mode Beta"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var f map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&f); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if f["name"] != "Dark Mode Beta" {
		t.Errorf("name: got %v", f["name"])
	}
}

func TestFlagHandler_Patch_IgnoresKeyAndType(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	seedFlag(t, mux)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/flags/dark-mode",
		strings.NewReader(`{"key":"new-key","type":"string"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var f map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&f); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if f["key"] != "dark-mode" {
		t.Errorf("key changed: got %v", f["key"])
	}
	if f["type"] != "bool" {
		t.Errorf("type changed: got %v", f["type"])
	}
}

func TestFlagHandler_Patch_UnknownFlag_Returns404(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/flags/ghost",
		strings.NewReader(`{"name":"X"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

func TestFlagHandler_Patch_InvalidJSON_Returns400(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	seedFlag(t, mux)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/flags/dark-mode",
		strings.NewReader(`{ not valid`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestFlagHandler_Delete_Returns204(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	seedFlag(t, mux)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/acme/flags/dark-mode", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", rec.Code)
	}
}

func TestFlagHandler_Delete_UnknownFlag_Returns404(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/acme/flags/ghost", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ── Auth ──────────────────────────────────────────────────────────────────────

func TestFlagHandler_Unauthenticated_Returns401(t *testing.T) {
	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/projects/acme/flags"},
		{http.MethodPost, "/api/v1/projects/acme/flags"},
		{http.MethodGet, "/api/v1/projects/acme/flags/dark-mode"},
		{http.MethodPatch, "/api/v1/projects/acme/flags/dark-mode"},
		{http.MethodDelete, "/api/v1/projects/acme/flags/dark-mode"},
	}
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), requireAuth401)
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

func TestFlagHandler_Create_Forbidden_Returns403(t *testing.T) {
	svc := newFakeFlagService()
	svc.mutateErr = domain.ErrForbidden
	mux := newFlagMux(svc, newFakeResolver("acme"), noopAuth)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/flags", strings.NewReader(boolFlagBody))
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

func TestFlagHandler_Update_Forbidden_Returns403(t *testing.T) {
	svc := newFakeFlagService()
	// seed so GetByKey succeeds before Update is called
	svc.flags[fk("proj-acme", "dark-mode")] = &domain.Flag{
		ID: "flag-1", ProjectID: "proj-acme", Key: "dark-mode",
	}
	svc.byID["flag-1"] = svc.flags[fk("proj-acme", "dark-mode")]
	svc.mutateErr = domain.ErrForbidden
	mux := newFlagMux(svc, newFakeResolver("acme"), noopAuth)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/flags/dark-mode",
		strings.NewReader(`{"name":"New Name"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

func TestFlagHandler_Delete_Forbidden_Returns403(t *testing.T) {
	svc := newFakeFlagService()
	svc.mutateErr = domain.ErrForbidden
	mux := newFlagMux(svc, newFakeResolver("acme"), noopAuth)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/acme/flags/dark-mode", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

// ── Pagination validation ─────────────────────────────────────────────────────

func assertBadRequest(t *testing.T, mux *http.ServeMux, url, wantMsg string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400 for %s", rec.Code, url)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "bad_request" {
		t.Errorf("error: got %v, want bad_request", body["error"])
	}
	if body["message"] != wantMsg {
		t.Errorf("message: got %q, want %q", body["message"], wantMsg)
	}
}

func TestFlagHandler_List_NegativePage_Returns400(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	assertBadRequest(t, mux, "/api/v1/projects/acme/flags?page=-1", "page must be a positive integer")
}

func TestFlagHandler_List_ZeroPage_Returns400(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	assertBadRequest(t, mux, "/api/v1/projects/acme/flags?page=0", "page must be a positive integer")
}

func TestFlagHandler_List_NonIntegerPage_Returns400(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	assertBadRequest(t, mux, "/api/v1/projects/acme/flags?page=abc", "page must be a positive integer")
}

func TestFlagHandler_List_NegativePerPage_Returns400(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	assertBadRequest(t, mux, "/api/v1/projects/acme/flags?per_page=-10", "per_page must be a positive integer")
}

func TestFlagHandler_List_InvalidSortBy_Returns400(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	assertBadRequest(t, mux, "/api/v1/projects/acme/flags?sort_by=invalid", "sort_by must be one of: key, name, type, created_at")
}

func TestFlagHandler_List_InvalidSortDir_Returns400(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	assertBadRequest(t, mux, "/api/v1/projects/acme/flags?sort_dir=sideways", "sort_dir must be one of: asc, desc")
}

func TestFlagHandler_List_PerPageCappedAt100(t *testing.T) {
	mux := newFlagMux(newFakeFlagService(), newFakeResolver("acme"), noopAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme/flags?per_page=500", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["per_page"] != float64(100) {
		t.Errorf("per_page: got %v, want 100", body["per_page"])
	}
}

func TestFlagHandler_List_PaginationEnvelope(t *testing.T) {
	svc := newFakeFlagService()
	mux := newFlagMux(svc, newFakeResolver("acme"), noopAuth)
	seedFlag(t, mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme/flags?page=1&per_page=25&sort_by=key&sort_dir=asc", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, field := range []string{"flags", "total", "page", "per_page"} {
		if _, ok := body[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}
	if body["page"] != float64(1) {
		t.Errorf("page: got %v, want 1", body["page"])
	}
	if body["per_page"] != float64(25) {
		t.Errorf("per_page: got %v, want 25", body["per_page"])
	}
	if body["total"] != float64(1) {
		t.Errorf("total: got %v, want 1", body["total"])
	}
}
