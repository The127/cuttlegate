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

// fakeFlagVariantService is a test double for the flagVariantService interface.
type fakeFlagVariantService struct {
	flags map[string]*domain.Flag // key: projectID+"/"+flagKey
	err   error
}

func newFakeVariantService() *fakeFlagVariantService {
	svc := &fakeFlagVariantService{flags: make(map[string]*domain.Flag)}
	// seed a string flag with two variants
	svc.flags["proj-acme/release"] = &domain.Flag{
		ID:                "flag-release",
		ProjectID:         "proj-acme",
		Key:               "release",
		Name:              "Release",
		Type:              domain.FlagTypeString,
		Variants:          []domain.Variant{{Key: "v1", Name: "V1"}, {Key: "v2", Name: "V2"}},
		DefaultVariantKey: "v1",
		CreatedAt:         time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
	}
	// seed a bool flag
	svc.flags["proj-acme/dark-mode"] = &domain.Flag{
		ID:                "flag-dark-mode",
		ProjectID:         "proj-acme",
		Key:               "dark-mode",
		Name:              "Dark Mode",
		Type:              domain.FlagTypeBool,
		Variants:          []domain.Variant{{Key: "true", Name: "On"}, {Key: "false", Name: "Off"}},
		DefaultVariantKey: "false",
		CreatedAt:         time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
	}
	return svc
}

func (f *fakeFlagVariantService) AddVariant(_ context.Context, projectID, flagKey string, v domain.Variant) (*domain.Flag, error) {
	if f.err != nil {
		return nil, f.err
	}
	flag, ok := f.flags[projectID+"/"+flagKey]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if flag.Type == domain.FlagTypeBool {
		return nil, domain.ErrImmutableVariants
	}
	for _, existing := range flag.Variants {
		if existing.Key == v.Key {
			return nil, domain.ErrConflict
		}
	}
	flag.Variants = append(flag.Variants, v)
	cp := *flag
	return &cp, nil
}

func (f *fakeFlagVariantService) RenameVariant(_ context.Context, projectID, flagKey, variantKey, newName string) (*domain.Flag, error) {
	if f.err != nil {
		return nil, f.err
	}
	flag, ok := f.flags[projectID+"/"+flagKey]
	if !ok {
		return nil, domain.ErrNotFound
	}
	for i, v := range flag.Variants {
		if v.Key == variantKey {
			flag.Variants[i].Name = newName
			cp := *flag
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeFlagVariantService) DeleteVariant(_ context.Context, projectID, flagKey, variantKey string) (*domain.Flag, error) {
	if f.err != nil {
		return nil, f.err
	}
	flag, ok := f.flags[projectID+"/"+flagKey]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if flag.Type == domain.FlagTypeBool {
		return nil, domain.ErrImmutableVariants
	}
	if variantKey == flag.DefaultVariantKey {
		return nil, domain.ErrDefaultVariant
	}
	if len(flag.Variants) == 1 {
		return nil, domain.ErrLastVariant
	}
	for i, v := range flag.Variants {
		if v.Key == variantKey {
			flag.Variants = append(flag.Variants[:i], flag.Variants[i+1:]...)
			cp := *flag
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func newVariantMux(svc *fakeFlagVariantService, auth func(http.Handler) http.Handler) *http.ServeMux {
	resolver := newFakeResolver("acme")
	mux := http.NewServeMux()
	httpadapter.NewFlagVariantHandler(svc, resolver).RegisterRoutes(mux, auth)
	return mux
}

// ── AddVariant ────────────────────────────────────────────────────────────────

func TestFlagVariantHandler_Add_Succeeds(t *testing.T) {
	mux := newVariantMux(newFakeVariantService(), noopAuth)
	body := `{"key":"v3","name":"V3"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/flags/release/variants", strings.NewReader(body))
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
	variants, ok := resp["variants"].([]any)
	if !ok {
		t.Fatal("missing variants")
	}
	if len(variants) != 3 {
		t.Errorf("variant count: got %d, want 3", len(variants))
	}
}

func TestFlagVariantHandler_Add_BoolFlag_Returns400(t *testing.T) {
	mux := newVariantMux(newFakeVariantService(), noopAuth)
	body := `{"key":"maybe","name":"Maybe"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/flags/dark-mode/variants", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestFlagVariantHandler_Add_DuplicateKey_Returns409(t *testing.T) {
	mux := newVariantMux(newFakeVariantService(), noopAuth)
	body := `{"key":"v1","name":"Duplicate"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/flags/release/variants", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status: got %d, want 409", rec.Code)
	}
}

func TestFlagVariantHandler_Add_MissingKey_Returns400(t *testing.T) {
	mux := newVariantMux(newFakeVariantService(), noopAuth)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/flags/release/variants", strings.NewReader(`{"name":"No Key"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

// ── RenameVariant ─────────────────────────────────────────────────────────────

func TestFlagVariantHandler_Rename_Succeeds(t *testing.T) {
	mux := newVariantMux(newFakeVariantService(), noopAuth)
	body := `{"name":"Version One"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/flags/release/variants/v1", strings.NewReader(body))
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
	variants := resp["variants"].([]any)
	v1 := variants[0].(map[string]any)
	if v1["name"] != "Version One" {
		t.Errorf("name: got %v, want Version One", v1["name"])
	}
}

func TestFlagVariantHandler_Rename_MissingName_Returns400(t *testing.T) {
	mux := newVariantMux(newFakeVariantService(), noopAuth)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/flags/release/variants/v1", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestFlagVariantHandler_Rename_UnknownVariant_Returns404(t *testing.T) {
	mux := newVariantMux(newFakeVariantService(), noopAuth)
	body := `{"name":"Ghost"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/flags/release/variants/ghost", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ── DeleteVariant ─────────────────────────────────────────────────────────────

func TestFlagVariantHandler_Delete_Succeeds(t *testing.T) {
	mux := newVariantMux(newFakeVariantService(), noopAuth)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/acme/flags/release/variants/v2", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	variants := resp["variants"].([]any)
	if len(variants) != 1 {
		t.Errorf("variant count: got %d, want 1", len(variants))
	}
}

func TestFlagVariantHandler_Delete_DefaultVariant_Returns409(t *testing.T) {
	mux := newVariantMux(newFakeVariantService(), noopAuth)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/acme/flags/release/variants/v1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status: got %d, want 409", rec.Code)
	}
}

func TestFlagVariantHandler_Delete_BoolFlag_Returns400(t *testing.T) {
	mux := newVariantMux(newFakeVariantService(), noopAuth)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/acme/flags/dark-mode/variants/true", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

// ── Auth ──────────────────────────────────────────────────────────────────────

func TestFlagVariantHandler_Unauthenticated_Returns401(t *testing.T) {
	routes := []struct {
		method, path, body string
	}{
		{http.MethodPost, "/api/v1/projects/acme/flags/release/variants", `{"key":"v3","name":"V3"}`},
		{http.MethodPatch, "/api/v1/projects/acme/flags/release/variants/v1", `{"name":"New"}`},
		{http.MethodDelete, "/api/v1/projects/acme/flags/release/variants/v2", ""},
	}
	mux := newVariantMux(newFakeVariantService(), requireAuth401)
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

func TestFlagVariantHandler_Add_Forbidden_Returns403(t *testing.T) {
	svc := newFakeVariantService()
	svc.err = domain.ErrForbidden
	mux := newVariantMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/flags/release/variants",
		strings.NewReader(`{"key":"v3","name":"V3"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

func TestFlagVariantHandler_Rename_Forbidden_Returns403(t *testing.T) {
	svc := newFakeVariantService()
	svc.err = domain.ErrForbidden
	mux := newVariantMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/flags/release/variants/v1",
		strings.NewReader(`{"name":"New"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

func TestFlagVariantHandler_Delete_Forbidden_Returns403(t *testing.T) {
	svc := newFakeVariantService()
	svc.err = domain.ErrForbidden
	mux := newVariantMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/acme/flags/release/variants/v2", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}
