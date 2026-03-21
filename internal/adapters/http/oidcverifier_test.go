package httpadapter

import (
	"context"
	"testing"

	"github.com/karo/cuttlegate/internal/domain"
)

// stubClaimsVerifier implements claimsVerifier for unit tests, returning
// a fixed claim map or error without requiring a real OIDC provider.
type stubClaimsVerifier struct {
	claims map[string]any
	err    error
}

func (s *stubClaimsVerifier) verifyClaims(_ context.Context, _ string) (map[string]any, error) {
	return s.claims, s.err
}

func newTestVerifier(claims map[string]any, roleClaim string) *OIDCVerifier {
	return &OIDCVerifier{verifier: &stubClaimsVerifier{claims: claims}, roleClaim: roleClaim}
}

func TestOIDCVerifier_MissingRoleClaim_ReturnsError(t *testing.T) {
	// Default role claim ("role") is absent — must fail closed, not default to RoleViewer.
	v := newTestVerifier(map[string]any{
		"sub":   "user-1",
		"email": "alice@example.com",
	}, "role")

	_, err := v.Verify(context.Background(), "any-token")
	if err == nil {
		t.Fatal("expected error for missing role claim, got nil")
	}
}

func TestOIDCVerifier_MissingCustomRoleClaim_ReturnsError(t *testing.T) {
	// Custom OIDC_ROLE_CLAIM ("user_role") is absent — must also fail closed.
	v := newTestVerifier(map[string]any{
		"sub":  "user-2",
		"role": "admin", // default claim present but not the configured one
	}, "user_role")

	_, err := v.Verify(context.Background(), "any-token")
	if err == nil {
		t.Fatal("expected error for missing custom role claim, got nil")
	}
}

func TestOIDCVerifier_InvalidRoleValue_ReturnsError(t *testing.T) {
	// Role claim present but value is not a recognised role — must fail closed.
	v := newTestVerifier(map[string]any{
		"sub":  "user-3",
		"role": "superuser", // not admin/editor/viewer
	}, "role")

	_, err := v.Verify(context.Background(), "any-token")
	if err == nil {
		t.Fatal("expected error for unrecognised role value, got nil")
	}
}

func TestOIDCVerifier_ValidRoleClaim_ReturnsUser(t *testing.T) {
	// Happy path: role claim present under the configured claim name.
	v := newTestVerifier(map[string]any{
		"sub":       "user-4",
		"email":     "bob@example.com",
		"name":      "Bob",
		"user_role": "editor",
	}, "user_role")

	user, err := v.Verify(context.Background(), "any-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := domain.User{Sub: "user-4", Email: "bob@example.com", Name: "Bob", Role: domain.RoleEditor}
	if user != want {
		t.Errorf("got %+v, want %+v", user, want)
	}
}
