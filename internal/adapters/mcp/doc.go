// Package mcp implements the Cuttlegate MCP server adapter.
//
// This package owns: MCP server initialisation, tool registration, connection-time
// tool list filtering by ToolCapabilityTier, and per-call capability enforcement.
// Every write-tier and destructive-tier tool call emits an audit event via
// app.AuditService — tool name must be included in the event payload.
//
// This package does not own: domain entities, capability tier definitions
// (internal/domain/tool_capability.go), RBAC rules (internal/app/rbac.go),
// or API key storage (internal/adapters/db).
//
// Start here: capability tier enforcement follows a dual-check pattern —
// connection-time tool list filter (discovery control) plus per-call live
// lookup from domain/ports.APIKeyRepository (security gate). The per-call
// check is mandatory; see ADR 0028 for the full rationale.
package mcp
