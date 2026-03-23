# ADR 0030: MCP transport — HTTP+SSE over stdio

**Date:** 2026-03-23
**Status:** Accepted
**Issue:** #107

## Context

The MCP server for Cuttlegate needs a transport mechanism. Two options were on the table:

1. **stdio** — process-level pipes; the MCP host spawns a subprocess and communicates via stdin/stdout
2. **HTTP+SSE** — MCP server listens on a dedicated TCP port; host connects via HTTP with SSE for streaming

Cuttlegate is deployed as a server-side process (Docker Compose, container) accessed by remote clients. The deployment model is not a local tool invocation — it is a networked service.

## Decision

Use **HTTP+SSE** transport. The MCP server listens on a dedicated port controlled by the `MCP_ADDR` environment variable (default `:8081`).

## Rationale

- **stdio is incompatible with remote deployment.** stdio transport requires the MCP host to spawn the server as a local subprocess. Cuttlegate runs as a remote networked service — operators connect Claude Desktop or other MCP hosts from their workstation to a server that may be on a different machine or in a container. stdio cannot satisfy this topology.
- **Dedicated port enables independent firewall control.** Operators can expose `:8080` (API) without exposing `:8081` (MCP), or vice versa. Mixing MCP and API on the same port would require path-based ACLs, which are more complex and error-prone.
- **HTTP+SSE is the standard networked MCP transport.** The MCP specification defines HTTP+SSE as the transport for networked/remote server deployments. Using it keeps Cuttlegate aligned with tooling expectations.
- **`MCP_ADDR` follows existing `SERVER_ADDR` convention.** Operators already configure `SERVER_ADDR` for the API server; `MCP_ADDR` is consistent and discoverable.

## Consequences

- The MCP server starts as a second listener in `cmd/server/main.go`, bound to `MCP_ADDR`.
- Operators must expose `:8081` (or their configured port) in Docker Compose / firewall rules to enable MCP access.
- The getting-started guide (#272) must document `MCP_ADDR` and the port exposure requirement.
- stdio transport is not supported and will not be added without a new ADR.
