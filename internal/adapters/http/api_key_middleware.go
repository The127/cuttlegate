package httpadapter

import (
	"context"
	"net/http"
	"strings"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// apiKeyAuthenticator verifies API key plaintexts and returns the scoped
// project and environment IDs.
type apiKeyAuthenticator interface {
	Authenticate(ctx context.Context, plaintext string) (projectID, environmentID string, err error)
}

// APIKeyScope carries the project and environment IDs that an API key is scoped to.
type APIKeyScope struct {
	ProjectID     string
	EnvironmentID string
}

type apiKeyScopeKey struct{}

// APIKeyScopeFrom retrieves the APIKeyScope stored by RequireBearerOrAPIKey.
// ok is false if the request was not authenticated via API key.
func APIKeyScopeFrom(ctx context.Context) (APIKeyScope, bool) {
	s, ok := ctx.Value(apiKeyScopeKey{}).(APIKeyScope)
	return s, ok
}

// apiKeyScopeAllows checks whether an API-key-authenticated request is
// permitted to access the given project and environment. If the request was
// authenticated via OIDC (no APIKeyScope in context), it always returns true.
func apiKeyScopeAllows(ctx context.Context, projectID, environmentID string) bool {
	scope, ok := APIKeyScopeFrom(ctx)
	if !ok {
		return true // OIDC auth — no scope restriction
	}
	return scope.ProjectID == projectID && scope.EnvironmentID == environmentID
}

// RequireBearerOrAPIKey returns middleware that authenticates requests using
// either an API key (Bearer cg_...) or an OIDC token (Bearer eyJ...).
//
// API key path: verifies key via apiKeyAuthenticator, injects APIKeyScope and
// a synthetic AuthContext with RoleViewer (API keys have read-only evaluation access).
//
// OIDC path: delegates to the provided TokenVerifier (same as RequireBearer).
func RequireBearerOrAPIKey(verifier ports.TokenVerifier, apiKeys apiKeyAuthenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeUnauthorized(w)
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// API key path: keys start with "cg_"
			if strings.HasPrefix(token, "cg_") {
				projectID, environmentID, err := apiKeys.Authenticate(r.Context(), token)
				if err != nil {
					writeUnauthorized(w)
					return
				}

				scope := APIKeyScope{ProjectID: projectID, EnvironmentID: environmentID}
				ctx := context.WithValue(r.Context(), apiKeyScopeKey{}, scope)
				// Inject a synthetic auth context — API keys get viewer-level
				// access to the evaluation endpoint only.
				ac := domain.AuthContext{UserID: "api-key", Role: domain.RoleViewer}
				ctx = domain.NewAuthContext(ctx, ac)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// OIDC path: standard Bearer token verification
			user, err := verifier.Verify(r.Context(), token)
			if err != nil {
				writeUnauthorized(w)
				return
			}

			ac := domain.AuthContext{UserID: user.Sub, Role: user.Role}
			ctx := domain.NewAuthContext(domain.NewContext(r.Context(), user), ac)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
