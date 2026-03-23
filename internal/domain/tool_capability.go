package domain

// ToolCapabilityTier governs which MCP tools an API key credential may call.
// It is orthogonal to [Role]: Role controls what project operations a human user
// (identified by OIDC JWT) may perform; ToolCapabilityTier controls which MCP
// tools an API key may invoke.
//
// Tiers are ordered by risk: Read < Write < Destructive. A credential at a
// given tier may call all tools at that tier and all lower tiers.
//
// The string values are part of the MCP API contract — they appear in error
// response bodies and must not change once the MCP server ships.
type ToolCapabilityTier string

const (
	// TierRead permits list, get, and evaluate tools — no state mutations.
	TierRead ToolCapabilityTier = "read"

	// TierWrite permits all read-tier tools plus create and update tools.
	TierWrite ToolCapabilityTier = "write"

	// TierDestructive permits all write-tier tools plus delete tools.
	TierDestructive ToolCapabilityTier = "destructive"
)

// Permits reports whether t grants access to a tool requiring required.
// The ordering is Read < Write < Destructive. An unrecognised tier value
// is treated as below all valid tiers and always returns false.
func (t ToolCapabilityTier) Permits(required ToolCapabilityTier) bool {
	return tierRank(t) >= tierRank(required)
}

// Valid reports whether t is a recognised capability tier.
func (t ToolCapabilityTier) Valid() bool {
	switch t {
	case TierRead, TierWrite, TierDestructive:
		return true
	}
	return false
}

func tierRank(t ToolCapabilityTier) int {
	switch t {
	case TierRead:
		return 0
	case TierWrite:
		return 1
	case TierDestructive:
		return 2
	}
	return -1
}
