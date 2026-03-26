package httpadapter

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/The127/cuttlegate/internal/domain"
	"github.com/The127/cuttlegate/internal/domain/ports"
)

// fakeAPIKeyAuthenticator implements apiKeyAuthenticator for tests.
type fakeAPIKeyAuthenticator struct {
	keys map[string]struct{ projectID, environmentID string }
}

func (f *fakeAPIKeyAuthenticator) Authenticate(_ context.Context, plaintext string) (string, string, error) {
	k, ok := f.keys[plaintext]
	if !ok {
		return "", "", domain.ErrForbidden
	}
	return k.projectID, k.environmentID, nil
}

// fakeTokenVerifier implements ports.TokenVerifier for tests.
type fakeTokenVerifier struct{}

func (f *fakeTokenVerifier) Verify(_ context.Context, token string) (domain.User, error) {
	if token == "valid-oidc-token" {
		return domain.User{Sub: "user-1", Email: "u@test.com", Role: domain.RoleAdmin}, nil
	}
	return domain.User{}, fmt.Errorf("invalid token")
}

var _ ports.TokenVerifier = (*fakeTokenVerifier)(nil)

func TestRequireBearerOrAPIKey_APIKey(t *testing.T) {
	auth := &fakeAPIKeyAuthenticator{
		keys: map[string]struct{ projectID, environmentID string }{
			"cg_testkey123": {"proj-1", "env-1"},
		},
	}
	middleware := RequireBearerOrAPIKey(&fakeTokenVerifier{}, auth)

	var gotScope APIKeyScope
	var gotScopeOK bool
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotScope, gotScopeOK = APIKeyScopeFrom(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer cg_testkey123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !gotScopeOK {
		t.Fatal("expected APIKeyScope in context")
	}
	if gotScope.ProjectID != "proj-1" || gotScope.EnvironmentID != "env-1" {
		t.Errorf("scope = %+v, want proj-1/env-1", gotScope)
	}
}

func TestRequireBearerOrAPIKey_OIDC(t *testing.T) {
	middleware := RequireBearerOrAPIKey(&fakeTokenVerifier{}, &fakeAPIKeyAuthenticator{
		keys: map[string]struct{ projectID, environmentID string }{},
	})

	var gotUser domain.User
	var gotUserOK bool
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotUserOK = domain.FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer valid-oidc-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !gotUserOK {
		t.Fatal("expected User in context")
	}
	if gotUser.Sub != "user-1" {
		t.Errorf("user.Sub = %q, want user-1", gotUser.Sub)
	}
}

func TestRequireBearerOrAPIKey_InvalidKey(t *testing.T) {
	middleware := RequireBearerOrAPIKey(&fakeTokenVerifier{}, &fakeAPIKeyAuthenticator{
		keys: map[string]struct{ projectID, environmentID string }{},
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer cg_invalidkey")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestRequireBearerOrAPIKey_NoHeader(t *testing.T) {
	middleware := RequireBearerOrAPIKey(&fakeTokenVerifier{}, &fakeAPIKeyAuthenticator{
		keys: map[string]struct{ projectID, environmentID string }{},
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAPIKeyScopeAllows(t *testing.T) {
	tests := []struct {
		name   string
		ctx    context.Context
		projID string
		envID  string
		want   bool
	}{
		{
			name:   "no scope (OIDC) always allowed",
			ctx:    context.Background(),
			projID: "any", envID: "any",
			want: true,
		},
		{
			name:   "matching scope",
			ctx:    context.WithValue(context.Background(), apiKeyScopeKey{}, APIKeyScope{ProjectID: "p1", EnvironmentID: "e1"}),
			projID: "p1", envID: "e1",
			want: true,
		},
		{
			name:   "wrong project",
			ctx:    context.WithValue(context.Background(), apiKeyScopeKey{}, APIKeyScope{ProjectID: "p1", EnvironmentID: "e1"}),
			projID: "p2", envID: "e1",
			want: false,
		},
		{
			name:   "wrong environment",
			ctx:    context.WithValue(context.Background(), apiKeyScopeKey{}, APIKeyScope{ProjectID: "p1", EnvironmentID: "e1"}),
			projID: "p1", envID: "e2",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apiKeyScopeAllows(tt.ctx, tt.projID, tt.envID); got != tt.want {
				t.Errorf("apiKeyScopeAllows() = %v, want %v", got, tt.want)
			}
		})
	}
}
