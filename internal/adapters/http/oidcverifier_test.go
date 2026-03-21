package httpadapter

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/karo/cuttlegate/internal/domain"
)

func TestResolveRole(t *testing.T) {
	tests := []struct {
		name        string
		roleStr     string
		wantRole    domain.Role
		wantWarning bool
	}{
		{
			name:        "valid admin role",
			roleStr:     "admin",
			wantRole:    domain.RoleAdmin,
			wantWarning: false,
		},
		{
			name:        "valid editor role",
			roleStr:     "editor",
			wantRole:    domain.RoleEditor,
			wantWarning: false,
		},
		{
			name:        "valid viewer role",
			roleStr:     "viewer",
			wantRole:    domain.RoleViewer,
			wantWarning: false,
		},
		{
			name:        "unrecognised non-empty role warns and defaults to viewer",
			roleStr:     "Admin",
			wantRole:    domain.RoleViewer,
			wantWarning: true,
		},
		{
			name:        "empty role defaults silently to viewer",
			roleStr:     "",
			wantRole:    domain.RoleViewer,
			wantWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))

			got := resolveRole(context.Background(), logger, tt.roleStr, "test-sub")

			if got != tt.wantRole {
				t.Errorf("resolveRole(%q) = %q, want %q", tt.roleStr, got, tt.wantRole)
			}

			logged := buf.String()
			hasWarning := len(logged) > 0
			if hasWarning != tt.wantWarning {
				t.Errorf("resolveRole(%q) warning logged = %v, want %v; log output: %q",
					tt.roleStr, hasWarning, tt.wantWarning, logged)
			}

			if tt.wantWarning {
				if !bytes.Contains(buf.Bytes(), []byte(tt.roleStr)) {
					t.Errorf("warning should contain role value %q, got: %q", tt.roleStr, logged)
				}
				if !bytes.Contains(buf.Bytes(), []byte("test-sub")) {
					t.Errorf("warning should contain sub value, got: %q", logged)
				}
			}
		})
	}
}
