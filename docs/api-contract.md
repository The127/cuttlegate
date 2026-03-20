# API Contract

> **Status:** Agreed
> **Validated against:** `GET /api/v1/projects` (Sprint 2, 2026-03-20)
> **Scope:** All JSON HTTP endpoints under `/api/v1/`

This is a working agreement, not an ADR. It governs the shape of every API response so that Zod schemas on the frontend and `WriteError`/`writeJSON` on the backend stay in sync. Changes require explicit sign-off from architecture and frontend.

---

## Error shape

All error responses use this structure, regardless of status code:

```json
{
  "error":   "snake_case_code",
  "message": "Human-readable description"
}
```

- `Content-Type: application/json` is always set on error responses.
- `error` is a machine-readable code for use in `switch` statements and Zod discriminated unions.
- `message` is for display or logging only — do not branch on it.

**Known error codes:**

| Code | HTTP status | Meaning |
|---|---|---|
| `unauthorized` | 401 | No valid Bearer token |
| `forbidden` | 403 | Authenticated but insufficient role |
| `not_found` | 404 | Resource does not exist |
| `conflict` | 409 | Uniqueness violation |
| `last_admin` | 409 | Cannot remove the last project admin |
| `bad_request` | 400 | Malformed request body or missing required field |
| `internal_error` | 500 | Unexpected server error |

**Implementation guarantee:** all error responses flow through `WriteError` in `internal/adapters/http/errors.go`. The 401 case flows through `writeUnauthorized` (same shape). There is no code path that writes a non-conforming error body.

---

## Empty collections

List endpoints always return an empty JSON array — never `null`, never a 404, never an absent field.

```json
{ "projects": [] }
```

**Implementation guarantee:** list handlers initialise their slice with `make([]T, 0, ...)` before marshalling, ensuring `[]` rather than `null` in the JSON output.

---

## List response wrapper

List endpoints wrap the collection in a named key matching the plural resource name:

```json
GET /api/v1/projects              → { "projects": [...] }
GET /api/v1/projects/{slug}/environments → { "environments": [...] }
GET /api/v1/projects/{slug}/members      → { "members": [...] }
GET /api/v1/projects/{slug}/flags        → { "flags": [...] }
```

The Zod schema shape is `z.object({ <resource_plural>: z.array(<ItemSchema>) })`.

---

## Pagination

No endpoints are paginated in Sprint 2. All list endpoints return the full collection.

When pagination is introduced (Sprint 3 or later), the agreed style is **cursor-based**:

```json
{
  "flags": [...],
  "next_cursor": "opaque-string-or-null"
}
```

- No offset/limit pagination.
- `next_cursor: null` means the last page.
- Total count is not returned by default.

Frontend code must not assume a list response contains all items once pagination lands.

---

## Timestamps

All timestamp fields are JSON strings in **RFC 3339 / ISO 8601 format, UTC timezone**:

```json
"created_at": "2026-03-20T10:00:00Z"
```

- Always a string — never a Unix epoch integer.
- Always UTC — the `Z` suffix is guaranteed.
- Field names use `_at` suffix (e.g. `created_at`, `updated_at`).

**Implementation guarantee:** every `to*Response` mapper calls `.UTC()` on `time.Time` before marshalling. `encoding/json` serialises `time.Time` as RFC 3339.

---

## IDs

All ID fields are **JSON strings in UUID v4 format**:

```json
"id": "3d966553-d386-42f1-9d4c-5fccce1d81ed"
```

- Always a string — never a JSON number, even if the underlying database uses an integer type.
- Field name is always `id` on the primary identifier.
- Foreign key references use `<resource>_id` (e.g. `project_id`, `environment_id`).

---

## Booleans

Boolean fields use JSON `true`/`false` — never `0`/`1`, never a string `"true"`, never absent-means-false.

```json
"enabled": true
```

No boolean fields exist in Sprint 2 responses. This convention applies from Sprint 3 onward.

---

## Validation record

| Date | Endpoint | Validator | Outcome |
|---|---|---|---|
| 2026-03-20 | `GET /api/v1/projects` |  | Conforms — error shape, empty array, timestamps, IDs all verified against `project_handler_test.go` |
