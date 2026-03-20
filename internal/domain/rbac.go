package domain

import (
	"context"
	"errors"
)

// Role represents the access level of an authenticated user.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

// ErrForbidden is returned when an authenticated user lacks the required role.
var ErrForbidden = errors.New("forbidden")

// Valid reports whether r is a recognised role.
func (r Role) Valid() bool {
	switch r {
	case RoleAdmin, RoleEditor, RoleViewer:
		return true
	}
	return false
}

// AuthContext carries the identity and role of the authenticated caller.
type AuthContext struct {
	UserID string
	Role   Role
}

type authContextKey struct{}

// NewAuthContext returns ctx with ac stored as the auth context.
func NewAuthContext(ctx context.Context, ac AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey{}, ac)
}

// AuthContextFrom retrieves the AuthContext stored by NewAuthContext. ok is false if absent.
func AuthContextFrom(ctx context.Context) (AuthContext, bool) {
	ac, ok := ctx.Value(authContextKey{}).(AuthContext)
	return ac, ok
}
