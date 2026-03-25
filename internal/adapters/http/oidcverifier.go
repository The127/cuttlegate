package httpadapter

import (
	"context"
	"errors"
	"log/slog"

	gooidc "github.com/coreos/go-oidc/v3/oidc"

	"github.com/karo/cuttlegate/internal/domain"
)

// MissingRolePolicy controls behaviour when a verified OIDC token carries no
// role claim. reject is the secure default; viewer enables a permissive
// fallback for operators who need it.
type MissingRolePolicy string

const (
	// MissingRolePolicyReject returns 401 when the role claim is absent.
	// This is the default — a missing claim is treated as a misconfiguration.
	MissingRolePolicyReject MissingRolePolicy = "reject"

	// MissingRolePolicyViewer grants the viewer role when the role claim is
	// absent and emits a warning log. Opt-in via OIDC_MISSING_ROLE_POLICY=viewer.
	MissingRolePolicyViewer MissingRolePolicy = "viewer"
)

// errMissingRoleClaim is returned by Verify when the token has no role claim
// and the policy is MissingRolePolicyReject.
//
// Constraint: this sentinel lives in the adapter package rather than
// domain/ports. It works today because all consumers (RequireBearer,
// RequireBearerOrAPIKey, writeVerifyError) are in the same package as
// OIDCVerifier. A second TokenVerifier implementation that needs to signal the
// same condition could not import this sentinel without a layer violation
// (adapter → adapter). If a second verifier is ever added, move this sentinel
// to domain/ports and update all consumers. See #185 retro.
var errMissingRoleClaim = errors.New("oidc: missing role claim")

// OIDCVerifier implements ports.TokenVerifier by validating Bearer tokens
// against the OIDC provider's JWKS. JWKS keys are fetched on first use and
// cached automatically by the go-oidc library.
type OIDCVerifier struct {
	verifier          *gooidc.IDTokenVerifier
	roleClaim         string
	missingRolePolicy MissingRolePolicy
	roleMapper        RoleMapper
	logger            *slog.Logger
}

// NewOIDCVerifier discovers the OIDC provider at issuer and returns a verifier.
// audience is the expected aud claim — pass an empty string to skip the check.
// roleClaim is the JWT claim name carrying the Cuttlegate role (e.g. "role").
// policy controls behaviour when the role claim is absent from a verified token.
func NewOIDCVerifier(ctx context.Context, issuer, audience, roleClaim string, policy MissingRolePolicy, mapper RoleMapper, logger *slog.Logger) (*OIDCVerifier, error) {
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
	if mapper == nil {
		mapper = IdentityRoleMapper{}
	}
	return &OIDCVerifier{
		verifier:          provider.Verifier(cfg),
		roleClaim:         roleClaim,
		missingRolePolicy: policy,
		roleMapper:        mapper,
		logger:            logger,
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

	roleStr := extractRoleClaim(claims[v.roleClaim])
	role, err := resolveRole(ctx, v.logger, roleStr, sub, v.missingRolePolicy, v.roleMapper)
	if err != nil {
		return domain.User{}, err
	}

	return domain.User{Sub: sub, Email: email, Name: name, Role: role}, nil
}

// extractRoleClaim returns the role string from a claim value that may be
// either a plain string or a JSON array of strings. For arrays, the first
// element that is a recognised domain.Role wins; otherwise the first string
// element is returned so resolveRole can log an appropriate warning.
func extractRoleClaim(raw any) string {
	if s, ok := raw.(string); ok {
		return s
	}
	if arr, ok := raw.([]any); ok {
		first := ""
		for _, v := range arr {
			s, ok := v.(string)
			if !ok {
				continue
			}
			if domain.Role(s).Valid() {
				return s
			}
			if first == "" {
				first = s
			}
		}
		return first
	}
	return ""
}

// resolveRole maps a raw role claim string to a domain.Role.
//
// If roleStr is empty (claim absent), behaviour depends on policy:
//   - MissingRolePolicyReject (default): returns errMissingRoleClaim; subject is logged
//   - MissingRolePolicyViewer: grants viewer and emits a WARN log with subject and policy
//
// If roleStr is non-empty but unrecognised, logs a warning and defaults to viewer.
func resolveRole(ctx context.Context, logger *slog.Logger, roleStr, sub string, policy MissingRolePolicy, mapper RoleMapper) (domain.Role, error) {
	if roleStr == "" {
		if policy == MissingRolePolicyViewer {
			logger.WarnContext(ctx, "oidc: missing role claim, defaulting to viewer",
				"sub", sub,
				"policy", string(policy),
			)
			return domain.RoleViewer, nil
		}
		// Default: MissingRolePolicyReject
		logger.WarnContext(ctx, "oidc: missing role claim, rejecting token",
			"sub", sub,
			"policy", string(policy),
		)
		return domain.Role(""), errMissingRoleClaim
	}

	if mapper != nil {
		if role, ok := mapper.MapRole(roleStr); ok {
			return role, nil
		}
	}

	role := domain.Role(roleStr)
	if role.Valid() {
		return role, nil
	}
	logger.WarnContext(ctx, "unrecognised OIDC role claim, defaulting to viewer",
		"role", roleStr,
		"sub", sub,
	)
	return domain.RoleViewer, nil
}
