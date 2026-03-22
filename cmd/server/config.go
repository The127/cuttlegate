package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	httpadapter "github.com/karo/cuttlegate/internal/adapters/http"
)

// Config holds all runtime configuration for the server, read from env vars.
type Config struct {
	OIDCIssuer            string                        // OIDC provider base URL for discovery (required)
	OIDCAudience          string                        // expected aud claim in Bearer tokens; empty = skip check
	OIDCRoleClaim         string                        // JWT claim name carrying the Cuttlegate role (default: "role")
	OIDCMissingRolePolicy httpadapter.MissingRolePolicy // what to do when the role claim is absent (default: reject)
	OIDCClientID          string                        // OIDC client_id for the SPA (returned by /api/v1/config)
	OIDCRedirectURI       string                        // OIDC redirect_uri for the SPA (returned by /api/v1/config)
	OIDCSPAAuthority      string                        // OIDC authority URL for the SPA (browser-reachable); defaults to OIDCIssuer
	Addr                  string                        // listen address (default: :8080)
	DSN                   string                        // postgres DATABASE_URL; required when AutoMigrate is true
	AutoMigrate           bool                          // run migrations at startup — dev/test only; unsafe in production (rolling restarts can race between old pods and a migrated schema)
	EvalRateLimit         int                           // max evaluation requests per user per EvalRateLimitWindow (default: 600)
	EvalRateLimitWindow   time.Duration                 // window size for eval rate limiting (default: 1m)
	UIAppName             string                        // application name shown in the SPA (default: "Cuttlegate"); env: UI_APP_NAME
	UILogoURL             string                        // URL of the logo shown in the SPA nav bar; empty = text name (env: UI_LOGO_URL)
	UIAccentColour        string                        // CSS hex colour for the SPA accent (default: "#2563eb"); env: UI_ACCENT_COLOUR
}

// Load reads configuration from environment variables.
// Returns an error if any required variable is missing.
func Load() (Config, error) {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	roleClaim := os.Getenv("OIDC_ROLE_CLAIM")
	if roleClaim == "" {
		roleClaim = "role"
	}

	missingRolePolicy := httpadapter.MissingRolePolicyReject
	if rawPolicy, ok := os.LookupEnv("OIDC_MISSING_ROLE_POLICY"); ok {
		switch httpadapter.MissingRolePolicy(rawPolicy) {
		case httpadapter.MissingRolePolicyReject:
			// already the default
		case httpadapter.MissingRolePolicyViewer:
			missingRolePolicy = httpadapter.MissingRolePolicyViewer
		default:
			return Config{}, fmt.Errorf("invalid OIDC_MISSING_ROLE_POLICY %q: must be %q or %q",
				rawPolicy, httpadapter.MissingRolePolicyReject, httpadapter.MissingRolePolicyViewer)
		}
	}
	evalRateLimit := 600
	if v := os.Getenv("EVAL_RATE_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			evalRateLimit = n
		}
	}
	evalRateLimitWindow := time.Minute
	if v := os.Getenv("EVAL_RATE_LIMIT_WINDOW"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			evalRateLimitWindow = d
		}
	}

	spaAuthority := os.Getenv("OIDC_SPA_AUTHORITY")

	uiAppName := os.Getenv("UI_APP_NAME")
	if uiAppName == "" {
		uiAppName = "Cuttlegate"
	}
	uiAccentColour := os.Getenv("UI_ACCENT_COLOUR")
	if uiAccentColour == "" {
		uiAccentColour = "#2563eb"
	}

	cfg := Config{
		OIDCIssuer:            os.Getenv("OIDC_ISSUER"),
		OIDCAudience:          os.Getenv("OIDC_AUDIENCE"),
		OIDCRoleClaim:         roleClaim,
		OIDCMissingRolePolicy: missingRolePolicy,
		OIDCClientID:          os.Getenv("OIDC_CLIENT_ID"),
		OIDCRedirectURI:       os.Getenv("OIDC_REDIRECT_URI"),
		OIDCSPAAuthority:      spaAuthority,
		Addr:                  addr,
		DSN:                   os.Getenv("DATABASE_URL"),
		AutoMigrate:           os.Getenv("AUTO_MIGRATE") == "true",
		EvalRateLimit:         evalRateLimit,
		EvalRateLimitWindow:   evalRateLimitWindow,
		UIAppName:             uiAppName,
		UILogoURL:             os.Getenv("UI_LOGO_URL"),
		UIAccentColour:        uiAccentColour,
	}

	if cfg.OIDCIssuer == "" {
		return Config{}, errors.New("missing required env var: OIDC_ISSUER")
	}

	return cfg, nil
}
