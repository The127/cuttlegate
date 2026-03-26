package httpadapter

import (
	"testing"

	"github.com/The127/cuttlegate/internal/domain"
)

func TestIdentityRoleMapper(t *testing.T) {
	m := IdentityRoleMapper{}

	tests := []struct {
		input string
		want  domain.Role
		ok    bool
	}{
		{"admin", domain.RoleAdmin, true},
		{"editor", domain.RoleEditor, true},
		{"viewer", domain.RoleViewer, true},
		{"cuttlegate:admin", domain.Role(""), false},
		{"ADMIN", domain.Role(""), false},
		{"", domain.Role(""), false},
	}
	for _, tt := range tests {
		got, ok := m.MapRole(tt.input)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("IdentityRoleMapper.MapRole(%q) = (%q, %v), want (%q, %v)",
				tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestStaticRoleMapper(t *testing.T) {
	m := NewStaticRoleMapper(map[string]domain.Role{
		"cuttlegate:admin":  domain.RoleAdmin,
		"cuttlegate:editor": domain.RoleEditor,
		"cuttlegate:viewer": domain.RoleViewer,
	})

	tests := []struct {
		input string
		want  domain.Role
		ok    bool
	}{
		// Mapped values
		{"cuttlegate:admin", domain.RoleAdmin, true},
		{"cuttlegate:editor", domain.RoleEditor, true},
		{"cuttlegate:viewer", domain.RoleViewer, true},
		// Identity fallback — bare role names still work
		{"admin", domain.RoleAdmin, true},
		{"editor", domain.RoleEditor, true},
		{"viewer", domain.RoleViewer, true},
		// Unrecognised
		{"superuser", domain.Role(""), false},
		{"", domain.Role(""), false},
	}
	for _, tt := range tests {
		got, ok := m.MapRole(tt.input)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("StaticRoleMapper.MapRole(%q) = (%q, %v), want (%q, %v)",
				tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestStaticRoleMapper_EmptyMap(t *testing.T) {
	m := NewStaticRoleMapper(map[string]domain.Role{})

	// With no mappings, behaves like identity mapper
	if got, ok := m.MapRole("admin"); !ok || got != domain.RoleAdmin {
		t.Errorf("empty StaticRoleMapper should fall back to identity for valid roles")
	}
	if _, ok := m.MapRole("cuttlegate:admin"); ok {
		t.Errorf("empty StaticRoleMapper should not recognise unmapped values")
	}
}
