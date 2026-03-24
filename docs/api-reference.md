# REST API Reference

All endpoints are under `/api/v1/` unless noted. See [api-contract.md](api-contract.md) for wire conventions (error shape, timestamps, IDs, empty collections).

## Authentication

| Type | Header | Used by |
|---|---|---|
| OIDC Bearer | `Authorization: Bearer <token>` | All management endpoints |
| Bearer or API Key | `Authorization: Bearer <token>` or `Authorization: Bearer <api_key>` | Evaluation endpoints only |
| Public | None | Health, config |

## Error shape

All errors: `{"error": "<code>", "message": "<text>"}`. Codes: `unauthorized` (401), `forbidden` (403), `not_found` (404), `conflict` (409), `last_admin` (409), `bad_request` (400), `internal_error` (500).

---

## Health (public)

| Method | Path | Description |
|---|---|---|
| GET | `/healthz` | Liveness probe — always 200 |
| GET | `/readyz` | Readiness probe — 200 if DB connected |
| GET | `/health` | Detailed health with DB latency |
| GET | `/api/v1/config` | SPA runtime config (OIDC issuer, client ID) |

---

## Projects (Bearer)

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/projects` | List all projects |
| POST | `/api/v1/projects` | Create project |
| GET | `/api/v1/projects/{slug}` | Get project |
| PATCH | `/api/v1/projects/{slug}` | Update project |
| DELETE | `/api/v1/projects/{slug}` | Delete project |

**Response wrapper:** `{"projects": [...]}`

### POST /api/v1/projects

```json
// Request
{"name": "My Project", "slug": "my-project"}

// Response 201
{"id": "uuid", "name": "My Project", "slug": "my-project", "created_at": "2026-03-20T10:00:00Z"}
```

---

## Project Members (Bearer)

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/projects/{slug}/members` | List members |
| POST | `/api/v1/projects/{slug}/members` | Add member |
| PATCH | `/api/v1/projects/{slug}/members/{user_id}` | Update member role |
| DELETE | `/api/v1/projects/{slug}/members/{user_id}` | Remove member |

**Response wrapper:** `{"members": [...]}`

### POST /api/v1/projects/{slug}/members

```json
// Request
{"user_id": "uuid", "role": "editor"}

// Response 201
{"user_id": "uuid", "role": "editor", "created_at": "2026-03-20T10:00:00Z"}
```

Roles: `admin`, `editor`, `viewer`. Error `last_admin` (409) when removing the last admin.

---

## Environments (Bearer)

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/projects/{slug}/environments` | List environments |
| POST | `/api/v1/projects/{slug}/environments` | Create environment |
| GET | `/api/v1/projects/{slug}/environments/{env_slug}` | Get environment |
| PATCH | `/api/v1/projects/{slug}/environments/{env_slug}` | Update environment |
| DELETE | `/api/v1/projects/{slug}/environments/{env_slug}` | Delete environment |

**Response wrapper:** `{"environments": [...]}`

### POST /api/v1/projects/{slug}/environments

```json
// Request
{"name": "Production", "slug": "production"}

// Response 201
{"id": "uuid", "project_id": "uuid", "name": "Production", "slug": "production", "created_at": "2026-03-20T10:00:00Z"}
```

---

## Flags (Bearer)

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/projects/{slug}/flags` | List flags (paginated) |
| POST | `/api/v1/projects/{slug}/flags` | Create flag |
| GET | `/api/v1/projects/{slug}/flags/{key}` | Get flag |
| PATCH | `/api/v1/projects/{slug}/flags/{key}` | Update flag |
| DELETE | `/api/v1/projects/{slug}/flags/{key}` | Delete flag |

### GET /api/v1/projects/{slug}/flags

Paginated. Query params:

| Param | Type | Default | Description |
|---|---|---|---|
| `page` | int | 1 | Page number (1-based) |
| `per_page` | int | 50 | Items per page (max 100) |
| `search` | string | — | Filter by key or name (case-insensitive substring) |
| `sort_by` | string | `created_at` | Sort column: `key`, `name`, `type`, `created_at` |
| `sort_dir` | string | `asc` | Sort direction: `asc`, `desc` |

```json
// Response 200
{
  "flags": [
    {
      "id": "uuid",
      "project_id": "uuid",
      "key": "dark-mode",
      "name": "Dark Mode",
      "type": "bool",
      "variants": [{"key": "true", "name": "On"}, {"key": "false", "name": "Off"}],
      "default_variant_key": "false",
      "created_at": "2026-03-20T10:00:00Z"
    }
  ],
  "total": 75,
  "page": 1,
  "per_page": 50
}
```

### POST /api/v1/projects/{slug}/flags

```json
// Request
{"key": "dark-mode", "name": "Dark Mode", "type": "bool"}

// Response 201
{"id": "uuid", "project_id": "uuid", "key": "dark-mode", "name": "Dark Mode", "type": "bool", "variants": [...], "default_variant_key": "false", "created_at": "..."}
```

---

## Flag Variants (Bearer)

| Method | Path | Description |
|---|---|---|
| POST | `/api/v1/projects/{slug}/flags/{key}/variants` | Add variant |
| PATCH | `/api/v1/projects/{slug}/flags/{key}/variants/{variant_key}` | Rename variant |
| DELETE | `/api/v1/projects/{slug}/flags/{key}/variants/{variant_key}` | Delete variant |

---

## Flag Environment State (Bearer)

| Method | Path | Description |
|---|---|---|
| GET | `.../{slug}/environments/{env_slug}/flags` | List flag states for environment |
| GET | `.../{slug}/environments/{env_slug}/flags/{key}` | Get single flag state |
| PATCH | `.../{slug}/environments/{env_slug}/flags/{key}` | Set flag enabled/disabled |

**Response wrapper:** `{"flags": [...]}`

```json
// Flag state object
{
  "id": "uuid",
  "project_id": "uuid",
  "key": "dark-mode",
  "name": "Dark Mode",
  "type": "bool",
  "variants": [...],
  "default_variant_key": "false",
  "enabled": true
}
```

### PATCH .../{env_slug}/flags/{key}

```json
// Request
{"enabled": true}
```

---

## Rules / Targeting (Bearer)

| Method | Path | Description |
|---|---|---|
| POST | `.../{slug}/flags/{key}/environments/{env_slug}/rules` | Create rule |
| GET | `.../{slug}/flags/{key}/environments/{env_slug}/rules` | List rules |
| PATCH | `.../{slug}/flags/{key}/environments/{env_slug}/rules/{ruleID}` | Update rule |
| DELETE | `.../{slug}/flags/{key}/environments/{env_slug}/rules/{ruleID}` | Delete rule |

**Response wrapper:** `{"rules": [...]}`

---

## Segments (Bearer)

| Method | Path | Description |
|---|---|---|
| POST | `/api/v1/projects/{slug}/segments` | Create segment |
| GET | `/api/v1/projects/{slug}/segments` | List segments |
| GET | `/api/v1/projects/{slug}/segments/{segmentSlug}` | Get segment |
| PATCH | `/api/v1/projects/{slug}/segments/{segmentSlug}` | Update segment |
| DELETE | `/api/v1/projects/{slug}/segments/{segmentSlug}` | Delete segment |
| PUT | `.../{segmentSlug}/members` | Set segment members (replace all) |
| GET | `.../{segmentSlug}/members` | List segment members |

**Response wrapper:** `{"segments": [...]}`, `{"members": [...]}`

---

## API Keys (Bearer)

| Method | Path | Description |
|---|---|---|
| POST | `.../{slug}/environments/{env_slug}/api-keys` | Create API key |
| GET | `.../{slug}/environments/{env_slug}/api-keys` | List API keys |
| PATCH | `.../{slug}/environments/{env_slug}/api-keys/{key_id}` | Update API key tier |
| DELETE | `.../{slug}/environments/{env_slug}/api-keys/{key_id}` | Revoke API key |

API keys are scoped to a project + environment. The full key value is only returned on creation.

---

## Evaluation (Bearer or API Key)

These two endpoints accept both OIDC bearer tokens and API keys. Rate-limited when using API keys.

| Method | Path | Description |
|---|---|---|
| POST | `.../{slug}/environments/{env_slug}/flags/{key}/evaluate` | Evaluate single flag |
| POST | `.../{slug}/environments/{env_slug}/evaluate` | Evaluate all flags |

### POST .../evaluate (bulk)

```json
// Request
{"context": {"user_id": "user-123", "attributes": {"plan": "pro"}}}

// Response 200
{
  "flags": [
    {
      "key": "dark-mode",
      "enabled": true,
      "value": null,
      "value_key": "true",
      "reason": "rule_match",
      "type": "bool"
    }
  ],
  "evaluated_at": "2026-03-20T10:00:00Z"
}
```

> **SDK mapping:** `value_key` on the wire maps to `variant` in all SDKs. `value` is deprecated.

### POST .../flags/{key}/evaluate (single)

```json
// Request
{"context": {"user_id": "user-123", "attributes": {"plan": "pro"}}}

// Response 200
{
  "key": "dark-mode",
  "enabled": true,
  "value": null,
  "value_key": "true",
  "reason": "rule_match",
  "type": "bool",
  "evaluated_at": "2026-03-20T10:00:00Z"
}
```

---

## Evaluation Audit (Bearer)

| Method | Path | Description |
|---|---|---|
| GET | `.../{slug}/environments/{env_slug}/flags/{key}/evaluations` | List evaluation history |

Cursor-based pagination using `before` (RFC 3339 timestamp) and `limit` query params. Returns `next_cursor` (null on last page).

---

## Evaluation Stats (Bearer)

| Method | Path | Description |
|---|---|---|
| GET | `.../{slug}/environments/{env_slug}/flags/{key}/stats` | Get flag evaluation stats |
| GET | `.../{slug}/environments/{env_slug}/flags/{key}/stats/buckets` | Get evaluation time buckets |

### GET .../stats

```json
// Response 200
{"last_evaluated_at": "2026-03-21T14:00:00Z", "evaluation_count": 42}
```

---

## Audit Log (Bearer)

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/projects/{slug}/audit` | List audit entries |

Cursor-based pagination using `before` and `limit` query params.

---

## Promotion (Bearer)

| Method | Path | Description |
|---|---|---|
| POST | `.../{slug}/environments/{env_slug}/flags/{key}/promote` | Promote single flag to next environment |
| POST | `.../{slug}/environments/{env_slug}/promote` | Promote all flags to next environment |

---

## Server-Sent Events (Bearer)

| Method | Path | Description |
|---|---|---|
| GET | `.../{slug}/environments/{env_slug}/flags/stream` | SSE stream for flag state changes |

Event type: `flag.state_changed`

```json
{
  "type": "flag.state_changed",
  "project": "my-project",
  "environment": "production",
  "flag_key": "dark-mode",
  "enabled": true,
  "occurred_at": "2026-03-20T10:01:00Z"
}
```

> **SDK mapping:** Wire `flag_key` / `occurred_at` map to camelCase `flagKey` / `occurredAt` in JS SDK. This event shape is a locked SDK contract (Sprint 6).

---

## Endpoint count

| Resource | Endpoints |
|---|---|
| Health / Config | 4 (public) |
| Projects | 5 |
| Members | 4 |
| Environments | 5 |
| Flags | 5 |
| Variants | 3 |
| Flag-Env State | 3 |
| Rules | 4 |
| Segments | 7 |
| API Keys | 4 |
| Evaluation | 2 |
| Eval Audit | 1 |
| Eval Stats | 2 |
| Audit | 1 |
| Promotion | 2 |
| SSE | 1 |
| **Total** | **53** |
