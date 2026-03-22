package main

import (
	"testing"

	httpadapter "github.com/karo/cuttlegate/internal/adapters/http"
)

func TestLoad_AllVarsSet(t *testing.T) {
	t.Setenv("OIDC_ISSUER", "https://auth.example.com")
	t.Setenv("OIDC_AUDIENCE", "cuttlegate")
	t.Setenv("OIDC_ROLE_CLAIM", "cuttlegate_role")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	checks := []struct {
		field string
		got   string
		want  string
	}{
		{"OIDCIssuer", cfg.OIDCIssuer, "https://auth.example.com"},
		{"OIDCAudience", cfg.OIDCAudience, "cuttlegate"},
		{"OIDCRoleClaim", cfg.OIDCRoleClaim, "cuttlegate_role"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %q, want %q", c.field, c.got, c.want)
		}
	}
}

func TestLoad_DefaultRoleClaim(t *testing.T) {
	t.Setenv("OIDC_ISSUER", "https://auth.example.com")
	t.Setenv("OIDC_ROLE_CLAIM", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OIDCRoleClaim != "role" {
		t.Errorf("OIDCRoleClaim default: got %q, want %q", cfg.OIDCRoleClaim, "role")
	}
}

func TestLoad_MissingIssuer(t *testing.T) {
	t.Setenv("OIDC_ISSUER", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when OIDC_ISSUER is missing")
	}
}

func TestLoad_MissingRolePolicy_DefaultIsReject(t *testing.T) {
	t.Setenv("OIDC_ISSUER", "https://auth.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OIDCMissingRolePolicy != httpadapter.MissingRolePolicyReject {
		t.Errorf("default OIDCMissingRolePolicy = %q, want %q", cfg.OIDCMissingRolePolicy, httpadapter.MissingRolePolicyReject)
	}
}

func TestLoad_MissingRolePolicy_ExplicitReject(t *testing.T) {
	t.Setenv("OIDC_ISSUER", "https://auth.example.com")
	t.Setenv("OIDC_MISSING_ROLE_POLICY", "reject")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OIDCMissingRolePolicy != httpadapter.MissingRolePolicyReject {
		t.Errorf("OIDCMissingRolePolicy = %q, want %q", cfg.OIDCMissingRolePolicy, httpadapter.MissingRolePolicyReject)
	}
}

func TestLoad_MissingRolePolicy_Viewer(t *testing.T) {
	t.Setenv("OIDC_ISSUER", "https://auth.example.com")
	t.Setenv("OIDC_MISSING_ROLE_POLICY", "viewer")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OIDCMissingRolePolicy != httpadapter.MissingRolePolicyViewer {
		t.Errorf("OIDCMissingRolePolicy = %q, want %q", cfg.OIDCMissingRolePolicy, httpadapter.MissingRolePolicyViewer)
	}
}

func TestLoad_MissingRolePolicy_InvalidValue(t *testing.T) {
	t.Setenv("OIDC_ISSUER", "https://auth.example.com")
	t.Setenv("OIDC_MISSING_ROLE_POLICY", "permissive")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid OIDC_MISSING_ROLE_POLICY value")
	}
}

func TestLoad_MissingRolePolicy_EmptyStringIsInvalid(t *testing.T) {
	t.Setenv("OIDC_ISSUER", "https://auth.example.com")
	t.Setenv("OIDC_MISSING_ROLE_POLICY", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when OIDC_MISSING_ROLE_POLICY is set to empty string")
	}
}
