package ports

import (
	"context"

	"github.com/The127/cuttlegate/internal/domain"
)

// TokenVerifier validates a Bearer token and returns the authenticated User.
// The implementation is responsible for signature verification, expiry checks,
// and extracting domain claims (sub, email, name, role) from the token.
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (domain.User, error)
}
