package httpadapter

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// RequireBearer returns middleware that authenticates requests using a Bearer
// token in the Authorization header, validated via the provided TokenVerifier.
//
// On success the authenticated domain.User and domain.AuthContext are injected
// into the request context, and the user's profile (name, email) is upserted
// into the local cache via userRepo. The upsert is best-effort: if it fails,
// the error is logged and the request proceeds normally.
//
// On failure the request is rejected with 401.
func RequireBearer(verifier ports.TokenVerifier, userRepo ports.UserRepository) func(http.Handler) http.Handler {
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

			if err := userRepo.Upsert(ctx, &user); err != nil {
				slog.WarnContext(ctx, "user profile cache upsert failed", "sub", user.Sub, "err", err)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
