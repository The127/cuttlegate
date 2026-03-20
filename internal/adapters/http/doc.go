// Package httpadapter implements the HTTP entry points for the application.
//
// This package contains HTTP handlers and middleware. It translates HTTP requests
// into app-layer calls and maps responses back to HTTP.
//
// Allowed imports: app, domain/ports, domain, stdlib, HTTP and OIDC/OAuth2 client libraries.
// Must not import sibling adapter packages (e.g. dbadapter).
package httpadapter
