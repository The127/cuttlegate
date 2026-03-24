# MCP Tool Reference

Cuttlegate exposes four MCP tools for AI agent integration. See [ADR 0028](../adr/0028-mcp-tool-capability-tiers.md) for the capability tier system.

## Capability Tiers

| Tier | Allowed Operations | Tools |
|---|---|---|
| `read` | List, get, evaluate | `list_flags`, `evaluate_flag` |
| `write` | All read + create, update | `list_flags`, `evaluate_flag`, `enable_flag`, `disable_flag` |
| `destructive` | All write + delete | All tools (no delete tools exist yet) |

Tier is enforced twice: tools are filtered from the tool list at connection time, and each call verifies the capability at invocation time (live check, not cached).

## Authentication

All tools require a valid API key passed via the MCP connection. The API key determines the project, environment, and capability tier.

## Error Responses

| Error | When |
|---|---|
| `{"error": "unauthenticated"}` | No valid API key |
| `{"error": "insufficient_capability", "required": "<tier>", "provided": "<tier>"}` | API key tier too low |
| `{"error": "internal_error"}` | Server error |

---

## list_flags

**Tier:** read

Lists all feature flags and their current state for a given project and environment.

### Input Schema

| Field | Type | Required | Description |
|---|---|---|---|
| `project_slug` | string | yes | Project slug |
| `environment_slug` | string | yes | Environment slug |

### Example

```json
// Request
{
  "method": "tools/call",
  "params": {
    "name": "list_flags",
    "arguments": {
      "project_slug": "my-project",
      "environment_slug": "production"
    }
  }
}

// Response
{
  "content": [
    {
      "type": "text",
      "text": "[{\"key\":\"dark-mode\",\"enabled\":true,\"type\":\"bool\"}, ...]"
    }
  ]
}
```

---

## evaluate_flag

**Tier:** read

Evaluates a single feature flag for a given user context and returns the current value and reason.

### Input Schema

| Field | Type | Required | Description |
|---|---|---|---|
| `project_slug` | string | yes | Project slug |
| `environment_slug` | string | yes | Environment slug |
| `key` | string | yes | Flag key |
| `eval_context` | object | yes | Evaluation context |
| `eval_context.user_id` | string | no | User identifier for targeting rules |
| `eval_context.attributes` | object | no | Key-value pairs for targeting rules |

### Example

```json
// Request
{
  "method": "tools/call",
  "params": {
    "name": "evaluate_flag",
    "arguments": {
      "project_slug": "my-project",
      "environment_slug": "production",
      "key": "dark-mode",
      "eval_context": {
        "user_id": "user-123",
        "attributes": {"plan": "pro"}
      }
    }
  }
}

// Response
{
  "content": [
    {
      "type": "text",
      "text": "{\"key\":\"dark-mode\",\"enabled\":true,\"variant\":\"true\",\"reason\":\"rule_match\"}"
    }
  ]
}
```

---

## enable_flag

**Tier:** write

Enables a feature flag in a specific environment. Mutates flag state and emits an audit event.

### Input Schema

| Field | Type | Required | Description |
|---|---|---|---|
| `project_slug` | string | yes | Project slug |
| `environment_slug` | string | yes | Environment slug |
| `key` | string | yes | Flag key |

### Example

```json
// Request
{
  "method": "tools/call",
  "params": {
    "name": "enable_flag",
    "arguments": {
      "project_slug": "my-project",
      "environment_slug": "production",
      "key": "dark-mode"
    }
  }
}

// Response
{
  "content": [
    {
      "type": "text",
      "text": "{\"key\":\"dark-mode\",\"enabled\":true}"
    }
  ]
}
```

---

## disable_flag

**Tier:** write

Disables a feature flag in a specific environment. Mutates flag state and emits an audit event.

### Input Schema

| Field | Type | Required | Description |
|---|---|---|---|
| `project_slug` | string | yes | Project slug |
| `environment_slug` | string | yes | Environment slug |
| `key` | string | yes | Flag key |

### Example

```json
// Request
{
  "method": "tools/call",
  "params": {
    "name": "disable_flag",
    "arguments": {
      "project_slug": "my-project",
      "environment_slug": "production",
      "key": "dark-mode"
    }
  }
}

// Response
{
  "content": [
    {
      "type": "text",
      "text": "{\"key\":\"dark-mode\",\"enabled\":false}"
    }
  ]
}
```
