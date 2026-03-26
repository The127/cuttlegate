package mcp

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/The127/cuttlegate/internal/domain"
)

// session holds per-connection state for a connected MCP client.
type session struct {
	id    string
	keyID string
	tier  domain.ToolCapabilityTier
}

// newSessionID generates a cryptographically random session identifier.
func newSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("mcp: failed to generate session ID: " + err.Error())
	}
	return hex.EncodeToString(b)
}
