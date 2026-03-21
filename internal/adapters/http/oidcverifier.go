package httpadapter

import (
	"context"
	"errors"
	"fmt"

	gooidc "github.com/coreos/go-oidc/v3/oidc"

	"github.com/karo/cuttlegate/internal/domain"
)

// claimsVerifier is an internal seam that verifies a raw token and returns its claims.
// The production implementation wraps *gooidc.IDTokenVerifier; tests inject a stub.
type claimsVerifier interface {
	verifyClaims(ctx context.Context, rawToken string) (map[string]any, error)
}

// goOIDCAdapter wraps *gooidc.IDTokenVerifier to implement claimsVerifier.
type goOIDCAdapter struct {
	v *gooidc.IDTokenVerifier
}

func (a *goOIDCAdapter) verifyClaims(ctx context.Context, rawToken string) (map[string]any, error) {
	idToken, err := a.v.Verify(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return nil, errors.New("oidc: failed to extract claims")
	}
	return claims, nil
}

// OIDCVerifier implements ports.TokenVerifier by validating Bearer tokens
// against the OIDC provider's JWKS. JWKS keys are fetched on first use and
// cached automatically by the go-oidc library.
type OIDCVerifier struct {
	verifier  claimsVerifier
	roleClaim string
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
		verifier:  &goOIDCAdapter{v: provider.Verifier(cfg)},
		roleClaim: roleClaim,
	}, nil
}

// Verify validates the token signature, expiry, and audience, then extracts
// domain claims. Returns an error if the token is invalid or if the role claim
// configured via roleClaim is absent or does not map to a recognised role.
//
// Fail-closed: an absent or unrecognised role claim is an error, not a
// silent downgrade to RoleViewer. A token without an explicit role must not
// be granted any access.
func (v *OIDCVerifier) Verify(ctx context.Context, token string) (domain.User, error) {
	claims, err := v.verifier.verifyClaims(ctx, token)
	if err != nil {
		return domain.User{}, err
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return domain.User{}, errors.New("oidc: missing sub claim")
	}
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)

	roleStr, ok := claims[v.roleClaim].(string)
	if !ok || !domain.Role(roleStr).Valid() {
		return domain.User{}, fmt.Errorf("oidc: missing or invalid %q claim", v.roleClaim)
	}

	return domain.User{Sub: sub, Email: email, Name: name, Role: domain.Role(roleStr)}, nil
}
