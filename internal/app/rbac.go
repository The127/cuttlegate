package app

import (
	"context"

	"github.com/The127/cuttlegate/internal/domain"
)

// roleLevel maps each Role to a numeric level for comparison.
// Higher value = more permissive.
var roleLevel = map[domain.Role]int{
	domain.RoleViewer: 1,
	domain.RoleEditor: 2,
	domain.RoleAdmin:  3,
}

// requireRole extracts the AuthContext from ctx and returns ErrForbidden if
// the caller's role is below min, or if no AuthContext is present.
func requireRole(ctx context.Context, min domain.Role) (domain.AuthContext, error) {
	ac, ok := domain.AuthContextFrom(ctx)
	if !ok || roleLevel[ac.Role] < roleLevel[min] {
		return domain.AuthContext{}, domain.ErrForbidden
	}
	return ac, nil
}
