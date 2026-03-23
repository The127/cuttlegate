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

// fakeAPIKeyService is a test double for the apiKeyService interface.
type fakeAPIKeyService struct {
	keys      []app.APIKeyView
	createErr error
	listErr   error
}

func (f *fakeAPIKeyService) Create(_ context.Context, projectID, environmentID, name string, tier domain.ToolCapabilityTier) (*app.APIKeyCreateResult, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	v := app.APIKeyView{
		ID:             "key-uuid-1",
		ProjectID:      projectID,
		EnvironmentID:  environmentID,
		Name:           name,
		DisplayPrefix:  "abcd1234",
		CapabilityTier: string(tier),
		CreatedAt:      time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
	}
	f.keys = append(f.keys, v)
	return &app.APIKeyCreateResult{
		APIKeyView: v,
		Plaintext:  "cg_abcd1234testplaintext",
	}, nil
}

func (f *fakeAPIKeyService) List(_ context.Context, projectID, environmentID string) ([]app.APIKeyView, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	result := make([]app.APIKeyView, 0, len(f.keys))
	for _, k := range f.keys {
		if k.ProjectID == projectID && k.EnvironmentID == environmentID {
			result = append(result, k)
		}
	}
	return result, nil
}

func (f *fakeAPIKeyService) Revoke(_ context.Context, id string) error { return nil }

func (f *fakeAPIKeyService) UpdateCapabilityTier(_ context.Context, id string, tier domain.ToolCapabilityTier) (*app.APIKeyView, error) {
	return &app.APIKeyView{
		ID:             id,
		Name:           "test-key",
		DisplayPrefix:  "abcd1234",
		CapabilityTier: string(tier),
		CreatedAt:      time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
	}, nil
}

func newAPIKeyMux(svc *fakeAPIKeyService, auth func(http.Handler) http.Handler) *http.ServeMux {
	resolver := newFakeResolver("acme")
	envs := &fakeEnvResolver{envs: map[string]*domain.Environment{
		"proj-acme/prod": {
			ID:        "env-prod",
			ProjectID: "proj-acme",
			Name:      "Production",
			Slug:      "prod",
			CreatedAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
		},
	}}
	mux := http.NewServeMux()
	httpadapter.NewAPIKeyHandler(svc, resolver, envs).RegisterRoutes(mux, auth)
	return mux
}

// ── Create ───────────────────────────────────────────────────────────────────

func TestAPIKeyHandler_Create_WithTier(t *testing.T) {
	svc := &fakeAPIKeyService{}
	mux := newAPIKeyMux(svc, noopAuth)

	body := `{"name":"SDK Key","capability_tier":"write"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments/prod/api-keys", strings.NewReader(body))
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
	if got := resp["capability_tier"]; got != "write" {
		t.Errorf("capability_tier = %v, want write", got)
	}
}

func TestAPIKeyHandler_Create_DefaultTier(t *testing.T) {
	svc := &fakeAPIKeyService{}
	mux := newAPIKeyMux(svc, noopAuth)

	body := `{"name":"SDK Key"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments/prod/api-keys", strings.NewReader(body))
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
	if got := resp["capability_tier"]; got != "read" {
		t.Errorf("capability_tier = %v, want read", got)
	}
}

func TestAPIKeyHandler_Create_InvalidTier(t *testing.T) {
	svc := &fakeAPIKeyService{}
	mux := newAPIKeyMux(svc, noopAuth)

	body := `{"name":"SDK Key","capability_tier":"superadmin"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/environments/prod/api-keys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
	var errResp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := errResp["error"]; got != "invalid_capability_tier" {
		t.Errorf("error code = %v, want invalid_capability_tier", got)
	}
	if got := errResp["message"]; got != "capability_tier must be one of: read, write, destructive" {
		t.Errorf("message = %v, want exact spec message", got)
	}
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestAPIKeyHandler_List_IncludesTier(t *testing.T) {
	svc := &fakeAPIKeyService{
		keys: []app.APIKeyView{
			{
				ID:             "key-1",
				ProjectID:      "proj-acme",
				EnvironmentID:  "env-prod",
				Name:           "CI Key",
				DisplayPrefix:  "abcd1234",
				CapabilityTier: "write",
				CreatedAt:      time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	mux := newAPIKeyMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme/environments/prod/api-keys", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	items, ok := body["api_keys"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected 1 api_key, got %v", body["api_keys"])
	}
	item := items[0].(map[string]any)
	if got := item["capability_tier"]; got != "write" {
		t.Errorf("capability_tier = %v, want write", got)
	}
}
