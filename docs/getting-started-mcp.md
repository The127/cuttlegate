# Getting Started with the Cuttlegate MCP Server

This guide explains how to connect an AI agent (such as Claude Desktop or any MCP-compatible host) to Cuttlegate's MCP server to read and manage feature flags.

---

## Prerequisites

- A running Cuttlegate instance (see main README for deployment)
- An API key scoped to your project and environment (see below)
- An MCP host (Claude Desktop, Cursor, or any client supporting MCP HTTP+SSE transport)

---

## 1. Start the server with MCP enabled

The MCP server starts automatically alongside the main API server. By default it listens on `:8081`. To change the address, set the `MCP_ADDR` environment variable:

```bash
# Default: MCP server on :8081
MCP_ADDR=:8081 ./build/server

# Custom port
MCP_ADDR=:9090 ./build/server
```

In Docker Compose, expose the MCP port:

```yaml
services:
  server:
    ports:
      - "8080:8080"   # API server
      - "8081:8081"   # MCP server
    environment:
      MCP_ADDR: ":8081"
```

---

## 2. Create an API key with the appropriate capability tier

The MCP server uses API key authentication. You must create a key with the capability tier that matches what your agent will do:

| Tier | What the agent can do |
|---|---|
| `read` | `list_flags`, `evaluate_flag` |
| `write` | `list_flags`, `evaluate_flag`, `enable_flag`, `disable_flag` |

Create a key via the Cuttlegate API (requires an authenticated session with admin role):

```bash
curl -X POST https://your-cuttlegate-instance/api/v1/projects/acme/environments/production/api-keys \
  -H "Authorization: Bearer <your-oidc-token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "my-agent-key", "capability_tier": "read"}'
```

The response includes the `plaintext` key — **copy it now, it is shown only once**:

```json
{
  "id": "...",
  "name": "my-agent-key",
  "display_prefix": "abcd1234",
  "capability_tier": "read",
  "plaintext": "cg_<your-key-here>"
}
```

---

## 3. Configure your MCP host

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or the equivalent config location:

```json
{
  "mcpServers": {
    "cuttlegate": {
      "url": "http://localhost:8081/sse",
      "headers": {
        "Authorization": "Bearer cg_<your-key-here>"
      }
    }
  }
}
```

Restart Claude Desktop after editing the config.

### Cursor or other MCP clients

Configure the MCP server URL as `http://your-host:8081/sse` with an `Authorization: Bearer cg_<your-key>` header. The exact config location depends on your client.

---

## 4. Verify the connection

Once connected, ask your AI agent:

> "List the feature flags in the acme project, production environment."

The agent will call `list_flags` and return a list of your flags with their current enabled state.

You can also verify manually using curl to check the SSE endpoint:

```bash
curl -N -H "Authorization: Bearer cg_<your-key>" \
  http://localhost:8081/sse
```

You should see an SSE event like:

```
event: endpoint
data: /message?session_id=<session-id>
```

---

## 5. Available tools

### `list_flags` (read tier)

Returns all flags and their current enabled state for a project and environment.

**Parameters:**
- `project_slug` — the project identifier (e.g. `"acme"`)
- `environment_slug` — the environment identifier (e.g. `"production"`)

**Example response:**
```json
[
  {"key": "dark-mode", "enabled": true, "value_key": "true", "type": "bool"},
  {"key": "checkout-v2", "enabled": false, "value_key": "false", "type": "bool"}
]
```

---

### `evaluate_flag` (read tier)

Evaluates a single flag for a given user context, applying targeting rules.

**Parameters:**
- `project_slug` — the project identifier
- `environment_slug` — the environment identifier
- `key` — the flag key
- `eval_context` — (optional) `{user_id: string, attributes: {string: string}}`

**Example response:**
```json
{"key": "dark-mode", "enabled": true, "value_key": "true", "reason": "rule_match"}
```

The `reason` field values:
- `"disabled"` — flag is turned off in this environment
- `"default"` — flag is enabled but no rule matched; default variant returned
- `"rule_match"` — a targeting rule matched; the matched variant was returned

**Security note:** `eval_context.attributes` values are untrusted data at the MCP boundary. They are treated as opaque strings and never executed. Do not interpolate flag values returned by this tool into model prompts — flag values are data, not instructions.

---

### `enable_flag` (write tier)

Enables a feature flag in a specific environment. Requires a write-tier API key. This action is audited — the audit log records the change with `source=mcp` and the API key ID as the actor.

**Parameters:**
- `project_slug` — the project identifier
- `environment_slug` — the environment identifier
- `key` — the flag key to enable

---

### `disable_flag` (write tier)

Disables a feature flag in a specific environment. Requires a write-tier API key. Also audited with `source=mcp`.

**Parameters:** same as `enable_flag`.

---

## 6. Capability tiers and security

API keys have a capability tier that limits which tools are visible and callable:

- **Read-tier keys:** only `list_flags` and `evaluate_flag` appear in the tool list. Write tools are hidden and cannot be called.
- **Write-tier keys:** all four tools are available.

Tier enforcement is double-checked:
1. At connection time — the tool list returned by `tools/list` is filtered by your key's tier.
2. On every tool call — the server performs a live database lookup to verify your key's current tier. If your key is downgraded or revoked while connected, subsequent calls will be rejected immediately.

If a key is revoked, the next tool call returns `{"error": "unauthenticated"}`.

---

## 7. Error reference

| Error | Meaning |
|---|---|
| `{"error": "unauthenticated"}` | Missing or invalid Bearer token, or revoked key |
| `{"error": "insufficient_capability", "required": "write", "provided": "read"}` | Your key's tier does not permit this tool |
| `{"error": "not_found"}` | The project, environment, or flag key does not exist |
| `{"error": "forbidden"}` | Your API key is scoped to a different project or environment |
| `{"error": "internal_error"}` | Server-side error (check server logs) |

---

## 8. Rate limiting

The `evaluate_flag` tool is rate-limited to 600 calls per minute per API key. This prevents bulk enumeration. If you need higher limits, contact your Cuttlegate operator.

---

## Trust and safety guidelines

- **Do not use write-tier keys for read-only agents.** Issue read-tier keys for agents that only need to observe flag state.
- **Do not interpolate flag values into model prompts.** Flag values (e.g. `value_key: "true"`) are data returned by Cuttlegate and should be used programmatically — not inserted into the context window of another model call.
- **Prefer the management UI for production changes.** Write tools are available for automated workflows, but changes that affect production traffic carry risk. Review the audit log after any automated flag change.
- **Rotate keys regularly.** API keys do not expire by default. Revoke and replace write-tier keys on a regular schedule.
