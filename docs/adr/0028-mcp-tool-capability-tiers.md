# ADR 0028: MCP tool capability tiers

**Date:** 2026-03-23
**Status:** Accepted
**Issue:** #145

## Context

Cuttlegate will expose an MCP server (#107) so AI agents can interact with feature flag management. The MCP specification has no built-in permission layer — every tool is equally callable by any connected client. Without application-level controls, an autonomous agent with a single credential could call `delete_flag` or `delete_environment` with the same ease as `list_flags`.

The existing API key auth path (ADR 0012) scopes keys to a (project, environment) pair but does not differentiate between read and mutating operations. RBAC (ADR 0008) governs which project operations a human user (identified by OIDC role) may perform — it is orthogonal to the MCP tool access question.

Hivetrack's flat MCP tool list — where `delete_issue` sits alongside `list_issues` with no differentiation — is the cautionary example that motivated this decision.

**Alternatives considered:**

- **Runtime confirmation mechanism:** agent receives a prompt asking to confirm destructive calls. Rejected — confirmation is a client/agent concern, not a server concern. A server-side confirmation mechanism is not specified by MCP and could be bypassed.
- **Single permission tier (all-or-nothing):** issue one key with full access or none. Rejected — coarse-grained; operators cannot grant read-only agents without also granting write access.
- **Three tiers (chosen):** tier is a property of the API key credential. The server filters the advertised tool list and enforces per-call.

## Decision

MCP tool access is governed by a **ToolCapabilityTier** assigned to each API key credential. Three tiers exist:

| Tier | String value | Tools included |
|---|---|---|
| Read | `"read"` | list, get, evaluate — no state mutations |
| Write | `"write"` | all read-tier tools, plus create and update |
| Destructive | `"destructive"` | all write-tier tools, plus delete |

The string values `"read"`, `"write"`, `"destructive"` are part of the MCP API contract. They appear in error response bodies and must not change once the MCP server ships.

**Dual enforcement** is mandatory:

1. **Connection-time tool list filter:** when the MCP client calls `tools/list`, the server returns only tools permitted by the credential's current tier. This limits tool discovery — a prompt injection cannot instruct an agent to call a tool that does not appear in the tool list.

2. **Per-call capability check:** on every tool call, the server resolves the credential's current tier from `APIKeyRepository` and rejects calls that exceed it. This is the security gate. It is not optional and is not made redundant by connection-time filtering.

**Per-call lookup must be live.** The credential's tier is read from `APIKeyRepository` on each tool call — not from a value cached at connection time. This is required for the tier-downgrade scenario: if an operator downgrades a key from `write` to `read` while a connection is active, subsequent write-tier calls on that connection must be rejected. A connection-time-cached tier would leave the credential exploitable for the connection lifetime after downgrade.

**Known tradeoff:** live per-call lookup adds one database read per MCP tool call. On high-frequency evaluation paths this may matter. Caching strategies (e.g. short TTL in-memory cache keyed on key hash) are a known option for #107 to evaluate — the security invariant (tier cannot be stale by more than the cache TTL) must be stated explicitly if caching is introduced. This ADR does not prohibit caching; it requires that any cache invalidation strategy be documented when implemented.

## Rationale

**Tiers over scopes.** A scope-per-tool model (e.g. OAuth-style scope strings) would be more expressive but would require operators to manage a scope list per key. Three tiers map cleanly to the three risk levels that motivated this decision: safe reads, reversible writes, irreversible deletes.

**No runtime confirmation.** Requiring the server to prompt for confirmation on destructive calls adds protocol complexity, is not specified by MCP, and places a burden on agent implementers. The simpler and more reliable guard is structural: destructive tools are not advertised to credentials that do not carry `destructive` tier. An agent cannot call what it cannot see.

**Tier lives on the API key, not on a separate credential type.** Adding a new credential type would be a new auth surface (contradicting ADR 0006). Extending `APIKey` with a tier field reuses the existing authentication path, the existing `APIKeyRepository`, and the existing key management endpoints.

**RBAC and capability tier are orthogonal.** `Role` (admin/editor/viewer) answers "can this human user perform this project operation?" `ToolCapabilityTier` answers "can this API key credential call this MCP tool?" Both checks may apply on a given request; neither subsumes the other.

## Consequences

**Type placement:**
- `ToolCapabilityTier` type and `TierRead`, `TierWrite`, `TierDestructive` constants live in `internal/domain/tool_capability.go`, alongside the existing `APIKey` entity.
- `APIKey` gains a `CapabilityTier ToolCapabilityTier` field.

**Enforcement placement:**
- Connection-time tool list filtering and per-call capability check both live in `internal/adapters/mcp/`.
- No enforcement logic belongs in `internal/domain/` or `internal/app/`.
- No new port interface is required — enforcement uses the existing `APIKeyRepository` port.

**Audit trail:**
- Every write-tier and destructive-tier tool call must emit an audit event via the existing `AuditService`.
- The audit event payload must include the MCP tool name. A generic "mutated via MCP" entry is insufficient — operators need tool-level granularity for incident investigation.
- Read-tier tool calls do not emit audit events (consistent with existing HTTP read endpoints).

**Error responses:**
- Unauthenticated: `{"error": "unauthenticated"}`
- Insufficient capability: `{"error": "insufficient_capability", "required": "<tier>", "provided": "<tier>"}`
- Internal error (e.g. audit service unavailable on a write call): `{"error": "internal_error"}`
- These response shapes are locked as the MCP API contract.

**Tool description convention:**
- MCP tool descriptions should carry a tier prefix so AI models can infer capability requirements from the schema without calling the tool first. Convention: `[read]`, `[write]`, or `[destructive]` at the start of the tool description string. This is an implementation convention for #107, not an enforced contract.

**Migration:**
- The `api_keys` table gains a `capability_tier` column (not null, default `'read'`). Existing keys default to `read` tier on migration — the safest default.

**What #107 inherits from this ADR:**
- `ToolCapabilityTier` domain type and constants are ready.
- `internal/adapters/mcp/` package stub exists (see `doc.go`).
- Error response bodies are specified.
- Audit requirements (what to log, where to emit) are specified.
- Caching tradeoff is named and available for #107 to resolve.
