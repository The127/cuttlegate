// Package mcp implements the Cuttlegate MCP server adapter.
//
// This package owns: MCP server initialisation, HTTP+SSE transport (2024-11-05),
// tool registration, connection-time tool list filtering by ToolCapabilityTier,
// per-call capability enforcement, and tool handlers for list_flags, evaluate_flag,
// enable_flag, and disable_flag.
//
// Write-tier tool calls (enable_flag, disable_flag) emit audit events via
// FlagService.SetEnabled with Source="mcp". Audit failure on write tools is
// a hard error — the mutation is not reported as successful if audit fails.
//
// This package does not own: domain entities, capability tier definitions
// (internal/domain/tool_capability.go), RBAC rules (internal/app/rbac.go),
// or API key storage (internal/adapters/db).
//
// Start here: Server in server.go. Capability tier enforcement follows a
// dual-check pattern — connection-time tool list filter (discovery control)
// in tools.go, plus per-call live lookup from domain/ports.APIKeyRepository
// (security gate) in server.go:handleMessage. The per-call check is mandatory;
// see ADR 0028 for the full rationale.
package mcp
