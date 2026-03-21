package domain

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	key, plaintext, err := GenerateAPIKey("id-1", "proj-1", "env-1", "test key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(plaintext, "cg_") {
		t.Errorf("plaintext should start with cg_, got %q", plaintext)
	}

	if len(key.DisplayPrefix) != displayPrefixLen {
		t.Errorf("display prefix length = %d, want %d", len(key.DisplayPrefix), displayPrefixLen)
	}

	// Verify hash matches
	hash := HashAPIKey(plaintext)
	if hash != key.KeyHash {
		t.Error("HashAPIKey(plaintext) does not match key.KeyHash")
	}

	if key.Revoked() {
		t.Error("new key should not be revoked")
	}
}

func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	_, p1, _ := GenerateAPIKey("id-1", "proj-1", "env-1", "key 1")
	_, p2, _ := GenerateAPIKey("id-2", "proj-1", "env-1", "key 2")
	if p1 == p2 {
		t.Error("two generated keys should not be identical")
	}
}
