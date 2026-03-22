package httpadapter

import (
	"net/http"
	"strings"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// RequireBearer returns middleware that authenticates requests using a Bearer
// token in the Authorization header, validated via the provided TokenVerifier.
//
// On success the authenticated domain.User and domain.AuthContext are injected
// into the request context. On failure the request is rejected with 401.
func RequireBearer(verifier ports.TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeUnauthorized(w)
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")

			user, err := verifier.Verify(r.Context(), token)
			if err != nil {
				writeVerifyError(w, err)
				return
			}

			ac := domain.AuthContext{UserID: user.Sub, Role: user.Role}
			ctx := domain.NewAuthContext(domain.NewContext(r.Context(), user), ac)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
