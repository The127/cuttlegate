// Package main is the server entrypoint for Cuttlegate.
//
// This binary constructs all adapters (HTTP handlers, database repositories), injects
// them into application services, and starts the HTTP server. It reads configuration
// from environment variables and embeds the SPA static assets for serving.
//
// This package does not contain business logic, handler implementations, or SQL queries.
// Those live in app, adapters/http, and adapters/db respectively. This is wiring code
// only — if you are adding a new feature, you probably need a different package.
//
// Start here: main.go for the startup sequence, or [config.go] for the environment
// variable bindings.
package main
