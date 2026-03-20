package domain

import "context"

// User represents an authenticated identity derived from OIDC claims.
type User struct {
	Sub   string
	Email string
	Name  string
	Role  Role
}

type contextKey struct{}

// NewContext returns ctx with u stored as the authenticated user.
func NewContext(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, contextKey{}, u)
}

// FromContext retrieves the User stored by NewContext. ok is false if absent.
func FromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(contextKey{}).(User)
	return u, ok
}
