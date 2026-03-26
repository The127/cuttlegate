package httpadapter

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/The127/cuttlegate/internal/domain"
)

func TestResolveRole(t *testing.T) {
	tests := []struct {
		name        string
		roleStr     string
		policy      MissingRolePolicy
		wantRole    domain.Role
		wantErr     error
		wantWarning bool
	}{
		{
			name:     "valid admin role",
			roleStr:  "admin",
			policy:   MissingRolePolicyReject,
			wantRole: domain.RoleAdmin,
		},
		{
			name:     "valid editor role",
			roleStr:  "editor",
			policy:   MissingRolePolicyReject,
			wantRole: domain.RoleEditor,
		},
		{
			name:     "valid viewer role",
			roleStr:  "viewer",
			policy:   MissingRolePolicyReject,
			wantRole: domain.RoleViewer,
		},
		{
			name:        "unrecognised non-empty role warns and defaults to viewer",
			roleStr:     "Admin",
			policy:      MissingRolePolicyReject,
			wantRole:    domain.RoleViewer,
			wantWarning: true,
		},
		{
			name:    "missing claim with reject policy returns errMissingRoleClaim",
			roleStr: "",
			policy:  MissingRolePolicyReject,
			wantErr: errMissingRoleClaim,
		},
		{
			name:        "missing claim with viewer policy grants viewer and warns",
			roleStr:     "",
			policy:      MissingRolePolicyViewer,
			wantRole:    domain.RoleViewer,
			wantWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))

			got, err := resolveRole(context.Background(), logger, tt.roleStr, "test-sub", tt.policy, IdentityRoleMapper{})

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("resolveRole(%q, %q) error = %v, want %v", tt.roleStr, tt.policy, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("resolveRole(%q, %q) unexpected error: %v", tt.roleStr, tt.policy, err)
			}

			if got != tt.wantRole {
				t.Errorf("resolveRole(%q, %q) = %q, want %q", tt.roleStr, tt.policy, got, tt.wantRole)
			}

			logged := buf.String()
			hasWarning := len(logged) > 0
			if hasWarning != tt.wantWarning {
				t.Errorf("resolveRole(%q, %q) warning logged = %v, want %v; log: %q",
					tt.roleStr, tt.policy, hasWarning, tt.wantWarning, logged)
			}

			if tt.wantWarning && tt.roleStr != "" {
				if !bytes.Contains(buf.Bytes(), []byte(tt.roleStr)) {
					t.Errorf("warning should contain role value %q, got: %q", tt.roleStr, logged)
				}
			}
			if tt.wantWarning {
				if !bytes.Contains(buf.Bytes(), []byte("test-sub")) {
					t.Errorf("warning should contain sub value, got: %q", logged)
				}
			}
		})
	}
}

func TestResolveRole_RejectLogsSubject(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	_, err := resolveRole(context.Background(), logger, "", "user-abc", MissingRolePolicyReject, IdentityRoleMapper{})

	if !errors.Is(err, errMissingRoleClaim) {
		t.Fatalf("expected errMissingRoleClaim, got %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("user-abc")) {
		t.Errorf("reject log should contain subject, got: %q", buf.String())
	}
}
