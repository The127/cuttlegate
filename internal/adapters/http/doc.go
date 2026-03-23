// Package httpadapter implements the HTTP handlers and middleware that expose the
// application layer over HTTP.
//
// Handlers translate incoming HTTP requests into app-service calls and map the results
// back to JSON responses. Middleware handles cross-cutting concerns like authentication
// (OIDC token verification) and rate limiting.
//
// CORS is intentionally absent: all current API consumers are either the SPA (served
// from the same origin as the API in the reference deployment, so no cross-origin
// requests are made) or server-side SDK clients (Go, Python, JS in Node) that do not
// originate browser requests. If a browser SDK is introduced in future, CORS middleware
// must be added here.
//
// This package does not contain business logic or database queries. Business rules live
// in the domain package; persistence lives in adapters/db. Handlers must not import
// sibling adapter packages — they talk to the app layer only.
//
// Start here: [FlagHandler] for the flag management API, or [EvaluationHandler] for
// the SDK-facing flag evaluation endpoint.
package httpadapter
