package main

import (
	"errors"
	"os"
)

// Config holds all runtime configuration for the server, read from env vars.
type Config struct {
	OIDCIssuer      string // OIDC provider base URL for discovery (required)
	OIDCAudience    string // expected aud claim in Bearer tokens; empty = skip check
	OIDCRoleClaim   string // JWT claim name carrying the Cuttlegate role (default: "role")
	OIDCClientID    string // OIDC client_id for the SPA (returned by /api/v1/config)
	OIDCRedirectURI string // OIDC redirect_uri for the SPA (returned by /api/v1/config)
	Addr            string // listen address (default: :8080)
	DSN             string // postgres DATABASE_URL; required when AutoMigrate is true
	AutoMigrate     bool   // run migrations at startup — dev/test only; unsafe in production (rolling restarts can race between old pods and a migrated schema)
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

	cfg := Config{
		OIDCIssuer:      os.Getenv("OIDC_ISSUER"),
		OIDCAudience:    os.Getenv("OIDC_AUDIENCE"),
		OIDCRoleClaim:   roleClaim,
		OIDCClientID:    os.Getenv("OIDC_CLIENT_ID"),
		OIDCRedirectURI: os.Getenv("OIDC_REDIRECT_URI"),
		Addr:            addr,
		DSN:             os.Getenv("DATABASE_URL"),
		AutoMigrate:     os.Getenv("AUTO_MIGRATE") == "true",
	}

	if cfg.OIDCIssuer == "" {
		return Config{}, errors.New("missing required env var: OIDC_ISSUER")
	}

	return cfg, nil
}
