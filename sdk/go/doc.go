// Package cuttlegate is the Go SDK for the Cuttlegate feature-flag service.
//
// What this package owns:
//   - Client interface and NewClient constructor
//   - Config struct with validation
//   - Typed error types: AuthError, NotFoundError, ServerError
//   - EvalContext, EvalResult, FlagUpdate types
//
// What it does not own:
//   - HTTP transport internals (injected via Config.HTTPClient / Config.StreamHTTPClient)
//   - Test helpers (see subpackage sdk/go/testing)
//
// Start here:
//   - NewClient — construct an authenticated client
//   - Client.EvaluateAll — evaluate all flags for a context (bulk, one HTTP call)
//   - Client.Evaluate — evaluate a single flag by key (NotFoundError if missing)
//   - Client.Bool / Client.String — typed convenience helpers
//   - Client.Subscribe — real-time SSE stream of flag state changes
package cuttlegate
