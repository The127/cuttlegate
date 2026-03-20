// Package dbadapter provides SQL implementations of the repository ports defined in
// domain/ports.
//
// This package translates between the port interfaces and the underlying database.
// It knows about SQL and database drivers; it does not know about HTTP or business rules.
//
// Allowed imports: domain/ports, domain, stdlib, and database libraries.
// Must not import sibling adapter packages (e.g. httpadapter).
package dbadapter
