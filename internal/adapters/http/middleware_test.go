package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karo/cuttlegate/internal/domain"
)

// stubVerifier is a test double for ports.TokenVerifier.
type stubVerifier struct {
	user domain.User
	err  error
}

func (s *stubVerifier) Verify(_ context.Context, _ string) (domain.User, error) {
	return s.user, s.err
}

func assert401JSON(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", ct, "application/json")
	}
}

func TestRequireBearer_MissingHeader_Returns401(t *testing.T) {
	mw := RequireBearer(&stubVerifier{err: errors.New("should not be called")})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert401JSON(t, rec)
}

func TestRequireBearer_NotBearerScheme_Returns401(t *testing.T) {
	mw := RequireBearer(&stubVerifier{err: errors.New("should not be called")})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert401JSON(t, rec)
}

func TestRequireBearer_InvalidToken_Returns401(t *testing.T) {
	mw := RequireBearer(&stubVerifier{err: errors.New("invalid token")})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bad.token.here")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert401JSON(t, rec)
}

func TestRequireBearer_ValidToken_InjectsUserAndAuthContext(t *testing.T) {
	want := domain.User{Sub: "sub42", Email: "alice@example.com", Name: "Alice", Role: domain.RoleAdmin}
	mw := RequireBearer(&stubVerifier{user: want})

	var gotUser domain.User
	var gotAC domain.AuthContext
	var gotACOK bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, _ = domain.FromContext(r.Context())
		gotAC, gotACOK = domain.AuthContextFrom(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer any.valid.token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if gotUser != want {
		t.Errorf("user: got %+v, want %+v", gotUser, want)
	}
	if !gotACOK {
		t.Fatal("AuthContext not present in context")
	}
	wantAC := domain.AuthContext{UserID: "sub42", Role: domain.RoleAdmin}
	if gotAC != wantAC {
		t.Errorf("AuthContext: got %+v, want %+v", gotAC, wantAC)
	}
}
