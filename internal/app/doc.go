// Package app contains the application use-case layer.
//
// This package orchestrates domain objects and calls ports (interfaces) to fulfill
// use cases. It does not contain business rules (those live in domain) nor
// infrastructure details (those live in adapters).
//
// Allowed imports: domain, domain/ports, stdlib. No adapter packages.
package app
