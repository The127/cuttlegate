package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StoredToken is the persisted OIDC token set.
type StoredToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	Issuer       string    `json:"issuer"`
	ClientID     string    `json:"client_id"`
}

func tokenPath() string {
	return filepath.Join(configDir(), "token.json")
}

// LoadToken reads the stored token; returns nil if absent.
func LoadToken() (*StoredToken, error) {
	data, err := os.ReadFile(tokenPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var t StoredToken
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}
	return &t, nil
}

// Save writes the token to disk.
func (t *StoredToken) Save() error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(tokenPath(), data, 0o600)
}

// Expired reports whether the token has expired (with a 30-second buffer).
func (t *StoredToken) Expired() bool {
	return time.Now().After(t.ExpiresAt.Add(-30 * time.Second))
}
