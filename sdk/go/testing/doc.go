// Package cuttlegatetesting provides an in-process mock client for testing
// flag integrations without a running Cuttlegate instance.
//
// What this package owns:
//   - MockClient — implements cuttlegate.Client with no network access
//   - Helpers: Enable, Disable, SetVariant, AssertEvaluated, AssertNotEvaluated
//
// What it does not own:
//   - Real HTTP communication (that is the parent package's responsibility)
//
// Start here:
//   - NewMockClient — construct a mock; set flag state directly before calling your code under test
package cuttlegatetesting
