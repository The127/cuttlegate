// Package main is the server entrypoint.
//
// This package contains wiring only — it instantiates adapters, injects dependencies,
// and starts the server. No business logic lives here.
//
// Allowed imports: adapters (http, db), app, domain/ports, stdlib, and configuration
// libraries. This is the only package permitted to import from multiple adapter packages.
package main
