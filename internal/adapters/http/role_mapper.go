package httpadapter

import "github.com/karo/cuttlegate/internal/domain"

// RoleMapper translates an external OIDC claim value to a domain.Role.
type RoleMapper interface {
	// MapRole returns the domain role for the given claim value and true,
	// or zero-value and false if the value is not recognised.
	MapRole(claimValue string) (domain.Role, bool)
}

// IdentityRoleMapper treats the claim value as a literal domain.Role name.
// Use this when the OIDC provider emits "admin", "editor", or "viewer" directly.
type IdentityRoleMapper struct{}

func (IdentityRoleMapper) MapRole(claimValue string) (domain.Role, bool) {
	r := domain.Role(claimValue)
	return r, r.Valid()
}

// StaticRoleMapper maps external claim values to domain roles via a fixed lookup table.
// Use this when the OIDC provider emits provider-specific role names
// (e.g. "cuttlegate:admin") that need translating.
type StaticRoleMapper struct {
	m map[string]domain.Role
}

// NewStaticRoleMapper builds a mapper from a claim-value → domain.Role map.
// It falls back to identity matching for values not in the map.
func NewStaticRoleMapper(m map[string]domain.Role) StaticRoleMapper {
	return StaticRoleMapper{m: m}
}

func (s StaticRoleMapper) MapRole(claimValue string) (domain.Role, bool) {
	if r, ok := s.m[claimValue]; ok {
		return r, true
	}
	// Fall back to identity: if the claim value is already a valid role name, accept it.
	r := domain.Role(claimValue)
	return r, r.Valid()
}
