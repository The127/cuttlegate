# ADR 0008: RBAC enforced in the app layer, not the HTTP adapter

**Date:** 2026-03-21
**Status:** Accepted
**Issue:** #123

## Context

Cuttlegate enforces authentication at the HTTP adapter layer via OIDC middleware: incoming requests must carry a valid JWT, and the middleware extracts the caller's identity and project role into an `AuthContext` stored on the request `context.Context`.

Through Sprint 2, this was the only enforcement gate. No authorization check existed in the app layer — any authenticated user could call any endpoint regardless of project membership or role. This included cross-project mutations (flag create/update/delete, environment create/delete, project member management, project rename/delete) and read operations (flag evaluation). The exposure window closed when #92 shipped in Sprint 3, which introduced `requireRole` in `internal/app/rbac.go` and applied it at the top of every app-layer service method.

The decision was whether to enforce RBAC in the HTTP adapter layer (middleware or handler) or in the app layer (service methods). Both are technically viable.

**Alternative A — HTTP adapter enforcement:** Add role checks in HTTP middleware or handlers, alongside authentication checks.

**Alternative B — App layer enforcement:** Add role checks inside each app-layer service method, using `requireRole(ctx, domain.RoleXxx)` before any data access.

## Decision

RBAC is enforced in the **app layer** (`internal/app/`), not in the HTTP adapter (`internal/adapters/http/`).

The mechanism is `requireRole(ctx, min domain.Role)` in `internal/app/rbac.go`, called at the top of every service method that requires authorization. It reads the `AuthContext` from `context.Context` and returns `domain.ErrForbidden` if the caller's role is below the required minimum.

## Rationale

**HTTP middleware handles authentication only.** Its job is to answer "is the caller who they claim to be?" — validating the JWT, resolving the project from the request path, and placing the resulting `AuthContext` on the context. This is an adapter concern.

**Authorization is a domain/app concern.** "Is this caller allowed to do this thing?" is a question about business rules, not transport. It belongs in the layer that owns the operation.

**Non-HTTP callers would be unprotected otherwise.** The app layer is the single entry point for all current and future callers: HTTP, a future gRPC server, an MCP tool server, or direct calls in integration tests. Enforcing RBAC only in the HTTP adapter means every new transport must remember to replicate the checks — a silent gap waiting to happen.

**Testability.** App-layer authorization can be verified in unit tests without spinning up an HTTP server. Tests call service methods directly with a crafted `context.Context`; no adapter plumbing required.

**Uniform treatment of read and write operations.** Read-only endpoints (e.g. flag evaluation) require `RoleViewer`; mutating operations require `RoleEditor` or `RoleAdmin`. The distinction is made consistently at the service boundary. There are no "low-risk" paths exempted from the check.

## Consequences

**What is now true:**

- Every public method on every app-layer service calls `requireRole` before any data access. This is a load-bearing invariant — not a suggestion.
- Project isolation is enforced: a caller can only act on resources they have a role for in that project.
- New transports (gRPC, MCP, CLI) inherit correct authorization automatically by calling the same app-layer services.

**What is harder:**

- `arch_test.go` does not currently verify that every service method calls `requireRole`. The gate today is code review. A future improvement would be a lint rule or test that enforces the pattern — this ADR names that gap explicitly so it is not forgotten.

**What a correct implementation looks like:**

Every new app-layer method that accesses project-scoped data must begin with:

```go
if _, err := requireRole(ctx, domain.RoleXxx); err != nil {
    return ..., err
}
```

Read operations use `domain.RoleViewer` as the minimum. Write/delete operations use `domain.RoleEditor` or `domain.RoleAdmin` as appropriate. There are no exceptions for operations deemed "low severity" — uniform enforcement is the point.
