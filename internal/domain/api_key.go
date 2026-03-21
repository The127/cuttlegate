package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"
)

const (
	apiKeyPrefix     = "cg_"
	apiKeyRandBytes  = 32
	displayPrefixLen = 8
)

// APIKey represents a hashed API key scoped to a project and environment.
type APIKey struct {
	ID            string
	ProjectID     string
	EnvironmentID string
	Name          string
	KeyHash       [32]byte
	DisplayPrefix string
	CreatedAt     time.Time
	RevokedAt     *time.Time
}

// Revoked reports whether the key has been revoked.
func (k *APIKey) Revoked() bool {
	return k.RevokedAt != nil
}

// GenerateAPIKey creates a new API key, returning the domain entity and the
// plaintext key. The plaintext is shown to the caller once and never stored.
func GenerateAPIKey(id, projectID, environmentID, name string) (*APIKey, string, error) {
	raw := make([]byte, apiKeyRandBytes)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", err
	}

	encoded := base64.RawURLEncoding.EncodeToString(raw)
	plaintext := apiKeyPrefix + encoded
	hash := sha256.Sum256([]byte(plaintext))

	return &APIKey{
		ID:            id,
		ProjectID:     projectID,
		EnvironmentID: environmentID,
		Name:          name,
		KeyHash:       hash,
		DisplayPrefix: encoded[:displayPrefixLen],
		CreatedAt:     time.Now(),
	}, plaintext, nil
}

// HashAPIKey returns the SHA-256 hash of a plaintext API key.
func HashAPIKey(plaintext string) [32]byte {
	return sha256.Sum256([]byte(plaintext))
}
