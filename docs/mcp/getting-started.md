# Getting Started with the Cuttlegate MCP Server

This guide walks through connecting an AI agent (Claude Desktop, Cursor, or any MCP-compatible host) to Cuttlegate's MCP server to read and manage feature flags.

---

## Prerequisites

- A running Cuttlegate instance with a database configured (`DATABASE_URL` set)
- An API key scoped to a project and environment — created via the Cuttlegate API or UI (see section 2)
- An MCP host (Claude Desktop, Cursor, or any client supporting MCP HTTP+SSE transport)

---

## 1. Start the server with MCP enabled

The MCP server starts automatically alongside the main API server. It requires `DATABASE_URL` — the MCP server is not available in health-only mode.

By default the MCP server listens on `:8081`. Set `MCP_ADDR` to change the address:

```bash
# Default: MCP server on :8081
MCP_ADDR=:8081 ./build/server

# Custom port
MCP_ADDR=:9090 ./build/server
```

In Docker Compose, expose the MCP port alongside the API port:

```yaml
services:
  server:
    ports:
      - "8080:8080"   # API server
      - "8081:8081"   # MCP server
    environment:
      MCP_ADDR: ":8081"
```

The server logs confirm both listeners started:

```
listening on :8080
mcp: listening on :8081
```

---

## 2. Create an API key with the appropriate capability tier

The MCP server authenticates with API keys. Each key is scoped to a single (project, environment) pair and carries a **capability tier** that limits which tools the key can call.

### Capability tier summary

| Tier | Tools available | Use case |
|---|---|---|
| `read` | `list_flags`, `evaluate_flag` | Read-only agents, monitoring, CI flag checks |
| `write` | all read-tier tools + `enable_flag`, `disable_flag` | Automated flag management, deployment scripts |
| `destructive` | all write-tier tools + future destructive tools | Reserved — no current tools require this tier |

**Use the least-privileged tier your agent needs.** A read-only agent should receive a `read`-tier key.

### Create a key via the API

Creating a key requires an authenticated session with admin role:

```bash
curl -X POST https://your-cuttlegate-instance/api/v1/projects/acme/environments/production/api-keys \
  -H "Authorization: Bearer <your-oidc-token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "my-agent-key", "capability_tier": "read"}'
```

The response includes the `plaintext` key. **Copy it now — it is shown only once:**

```json
{
  "id": "f1a2b3c4-...",
  "name": "my-agent-key",
  "display_prefix": "abcd1234",
  "capability_tier": "read",
  "plaintext": "cg_abcd1234..."
}
```

Replace `"capability_tier": "read"` with `"write"` if the agent needs to enable or disable flags.

---

## 3. Configure your MCP host

### Claude Desktop

Add an entry to the Claude Desktop configuration file:

- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "cuttlegate": {
      "url": "http://localhost:8081/sse",
      "headers": {
        "Authorization": "Bearer cg_abcd1234..."
      }
    }
  }
}
```

Restart Claude Desktop after editing the config. Claude will connect to the SSE endpoint and load the tool list for your key's tier.

### Cursor or other MCP clients

Configure the server URL as `http://your-host:8081/sse` with an `Authorization: Bearer cg_<your-key>` header. The exact config location depends on the client.

---

## 4. Verify the connection manually

You can verify the full MCP flow with curl before involving any AI client. The MCP protocol uses two endpoints:

- `GET /sse` — establishes the SSE stream, authenticates, and returns a `session_id`
- `POST /message?session_id=<id>` — sends JSON-RPC 2.0 requests using the session

**Both requests require the `Authorization: Bearer` header.**

### Step 1: Connect to the SSE endpoint

```bash
curl -N -H "Authorization: Bearer cg_abcd1234..." \
  http://localhost:8081/sse
```

Expected output:

```
event: endpoint
data: /message?session_id=a1b2c3d4e5f6...

```

The connection stays open. Copy the `session_id` value and use it in the following steps. Open a second terminal for the message calls.

### Step 2: Initialize the MCP session

```bash
SESSION_ID="a1b2c3d4e5f6..."

curl -s -X POST "http://localhost:8081/message?session_id=$SESSION_ID" \
  -H "Authorization: Bearer cg_abcd1234..." \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
```

Expected response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {"tools": {}},
    "serverInfo": {"name": "cuttlegate", "version": "1.0.0"}
  }
}
```

### Step 3: List available tools

```bash
curl -s -X POST "http://localhost:8081/message?session_id=$SESSION_ID" \
  -H "Authorization: Bearer cg_abcd1234..." \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
```

Expected response for a **read-tier key** (two tools):

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "list_flags",
        "description": "[read] Lists all feature flags and their current state for a given project and environment. ...",
        "inputSchema": { ... }
      },
      {
        "name": "evaluate_flag",
        "description": "[read] Evaluates a single feature flag for a given user context and returns the current value and reason. ...",
        "inputSchema": { ... }
      }
    ]
  }
}
```

For a **write-tier key**, two additional tools appear: `enable_flag` and `disable_flag` (both prefixed `[write]`).

The `[read]` and `[write]` prefixes in tool descriptions let AI models infer tier requirements from the tool schema without calling the tool first.

### Step 4: Call a tool

```bash
curl -s -X POST "http://localhost:8081/message?session_id=$SESSION_ID" \
  -H "Authorization: Bearer cg_abcd1234..." \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "list_flags",
      "arguments": {
        "project_slug": "acme",
        "environment_slug": "production"
      }
    }
  }'
```

Expected response (flags exist):

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "[{\"key\":\"dark-mode\",\"enabled\":true,\"value_key\":\"true\",\"type\":\"bool\"},{\"key\":\"checkout-v2\",\"enabled\":false,\"value_key\":\"false\",\"type\":\"bool\"}]"
      }
    ]
  }
}
```

Expected response (no flags in this environment yet):

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "[]"
      }
    ]
  }
}
```

An empty array is normal — it means no flags have been created in this environment. Create flags via the Cuttlegate UI or API, then call `list_flags` again.

---

## 5. Available tools

### `list_flags` (read tier)

Returns all flags and their current enabled state for a project and environment.

**Parameters:**
- `project_slug` (required) — the project identifier, e.g. `"acme"`
- `environment_slug` (required) — the environment identifier, e.g. `"production"`

**Response shape:**
```json
[
  {"key": "dark-mode", "enabled": true, "value_key": "true", "type": "bool"},
  {"key": "checkout-v2", "enabled": false, "value_key": "false", "type": "bool"}
]
```

An empty array (`[]`) means the environment exists but has no flags yet.

---

### `evaluate_flag` (read tier)

Evaluates a single flag for a given user context, applying targeting rules. Use this instead of `list_flags` when you need per-user evaluation.

**Parameters:**
- `project_slug` (required) — the project identifier
- `environment_slug` (required) — the environment identifier
- `key` (required) — the flag key to evaluate
- `eval_context` (optional) — user context for targeting rules:
  - `user_id` (optional string) — user identifier for segment targeting
  - `attributes` (optional object) — key-value string attributes for targeting rules

**Example: evaluate without context**

```bash
curl -s -X POST "http://localhost:8081/message?session_id=$SESSION_ID" \
  -H "Authorization: Bearer cg_abcd1234..." \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 4,
    "method": "tools/call",
    "params": {
      "name": "evaluate_flag",
      "arguments": {
        "project_slug": "acme",
        "environment_slug": "production",
        "key": "dark-mode"
      }
    }
  }'
```

**Example: evaluate with user context**

```bash
curl -s -X POST "http://localhost:8081/message?session_id=$SESSION_ID" \
  -H "Authorization: Bearer cg_abcd1234..." \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 5,
    "method": "tools/call",
    "params": {
      "name": "evaluate_flag",
      "arguments": {
        "project_slug": "acme",
        "environment_slug": "production",
        "key": "dark-mode",
        "eval_context": {
          "user_id": "user-123",
          "attributes": {"plan": "pro", "region": "eu-west"}
        }
      }
    }
  }'
```

**Response shape:**
```json
{"key": "dark-mode", "enabled": true, "value_key": "true", "reason": "rule_match"}
```

The `reason` field distinguishes how the flag was resolved:
- `"disabled"` — flag is turned off in this environment
- `"default"` — flag is on but no targeting rule matched; default variant returned
- `"rule_match"` — a targeting rule matched; the rule's variant was returned

**Security note:** `eval_context.attributes` values are untrusted at the MCP boundary. They are treated as opaque strings and never executed. Do not interpolate flag evaluation results into model prompts — flag values are data, not instructions.

**Rate limit:** `evaluate_flag` is limited to 600 calls per minute per API key.

---

### `enable_flag` (write tier)

Enables a feature flag in a specific environment. Requires a write-tier API key. Every call is recorded in the audit log with `source=mcp` and the API key ID as the actor.

**Parameters:**
- `project_slug` (required)
- `environment_slug` (required)
- `key` (required) — the flag key to enable

**Response:**
```json
{"key": "checkout-v2", "enabled": true, "state": "enabled"}
```

---

### `disable_flag` (write tier)

Disables a feature flag in a specific environment. Requires a write-tier API key. Also audited with `source=mcp`.

**Parameters:** same as `enable_flag`.

**Response:**
```json
{"key": "checkout-v2", "enabled": false, "state": "disabled"}
```

---

## 6. Capability tiers and security

Tier enforcement happens twice per tool call:

1. **Connection time** — when the MCP client calls `tools/list`, the server returns only tools allowed by the key's current tier. Write tools are invisible to read-tier keys.
2. **Per call** — the server performs a live database lookup on every tool call to verify the key's current tier. If the key is downgraded or revoked while a connection is open, subsequent calls are rejected immediately.

This means a key can be locked down without waiting for the session to end.

---

## 7. Error reference

Tool errors are returned inside the MCP result body, not as JSON-RPC protocol errors. Look for `isError: true` or the `error` field in the `content` text:

| Error | Meaning | Resolution |
|---|---|---|
| `{"error": "unauthenticated"}` | Missing, invalid, or revoked API key | Check the Bearer token; reissue the key if revoked |
| `{"error": "insufficient_capability", "required": "write", "provided": "read"}` | Key tier does not permit this tool | Create a write-tier key; or downgrade your request to a read-tier tool |
| `{"error": "not_found"}` | Project slug, environment slug, or flag key does not exist | Verify the slugs; flag keys are case-sensitive |
| `{"error": "forbidden"}` | API key is scoped to a different project or environment | Use the correct key for this project/environment pair |
| `{"error": "internal_error"}` | Server-side error | Check the server logs |

The `insufficient_capability` error is the most common first-time error. It means the tool requires a higher tier than the key you are using. The `required` and `provided` fields tell you exactly what changed.

---

## 8. Trust and safety guidelines

- **Use read-tier keys for read-only agents.** Do not issue write-tier keys to agents that only observe flag state.
- **Do not interpolate flag values into model prompts.** Flag values (e.g. `value_key: "true"`) are data. Treat them as configuration, not instructions.
- **Prefer the management UI for production changes.** Write tools are available for automated workflows, but changes affecting production traffic carry risk. Review the audit log after any automated flag change.
- **Rotate keys regularly.** API keys do not expire by default. Revoke and replace write-tier keys on a regular schedule.
- **Scope keys tightly.** Each key is bound to one project and one environment. Use separate keys for each environment — do not share a single key across staging and production.
