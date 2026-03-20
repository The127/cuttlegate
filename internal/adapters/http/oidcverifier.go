package httpadapter

import (
	"context"
	"errors"

	gooidc "github.com/coreos/go-oidc/v3/oidc"

	"github.com/karo/cuttlegate/internal/domain"
)

// OIDCVerifier implements ports.TokenVerifier by validating Bearer tokens
// against the OIDC provider's JWKS. JWKS keys are fetched on first use and
// cached automatically by the go-oidc library.
type OIDCVerifier struct {
	verifier   *gooidc.IDTokenVerifier
	roleClaim  string
}

// NewOIDCVerifier discovers the OIDC provider at issuer and returns a verifier.
// audience is the expected aud claim — pass an empty string to skip the check.
// roleClaim is the JWT claim name carrying the Cuttlegate role (e.g. "role").
func NewOIDCVerifier(ctx context.Context, issuer, audience, roleClaim string) (*OIDCVerifier, error) {
	provider, err := gooidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	cfg := &gooidc.Config{}
	if audience != "" {
		cfg.ClientID = audience
	} else {
		cfg.SkipClientIDCheck = true
	}
	return &OIDCVerifier{
		verifier:  provider.Verifier(cfg),
		roleClaim: roleClaim,
	}, nil
}

// Verify validates the token signature, expiry, and audience, then extracts
// domain claims. Returns an error if the token is invalid for any reason.
func (v *OIDCVerifier) Verify(ctx context.Context, token string) (domain.User, error) {
	idToken, err := v.verifier.Verify(ctx, token)
	if err != nil {
		return domain.User{}, err
	}

	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return domain.User{}, errors.New("oidc: failed to extract claims")
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return domain.User{}, errors.New("oidc: missing sub claim")
	}
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)

	roleStr, _ := claims[v.roleClaim].(string)
	role := domain.Role(roleStr)
	if !role.Valid() {
		role = domain.RoleViewer
	}

	return domain.User{Sub: sub, Email: email, Name: name, Role: role}, nil
}
