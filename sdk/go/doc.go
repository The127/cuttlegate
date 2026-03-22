// Package cuttlegate is the Go SDK for the Cuttlegate feature-flag service.
//
// What this package owns:
//   - Client interface and NewClient constructor
//   - Config struct with validation
//   - Typed error types: AuthError, NotFoundError, ServerError
//   - EvalContext and EvalResult types
//
// What it does not own:
//   - HTTP transport internals (injected via Config.HTTPClient)
//   - Test helpers (see subpackage sdk/go/testing)
//
// Start here:
//   - NewClient — construct an authenticated client
//   - Client.Evaluate — evaluate all flags for a context
//   - Client.EvaluateFlag — evaluate a single flag by key
package cuttlegate
