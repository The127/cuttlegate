// Package dbadapter provides PostgreSQL implementations of the repository interfaces
// defined in domain/ports.
//
// Each repository struct (e.g. PostgresFlagRepository) implements one port interface
// using SQL queries against a *sql.DB connection. Integration tests in this package
// run against a real Postgres instance via testcontainers.
//
// This package does not contain business logic or HTTP handling. It translates between
// port interfaces and SQL — nothing more. Adapters here must not import sibling adapter
// packages (e.g. httpadapter).
//
// Start here: [PostgresFlagRepository] for the most-used repository, or [postgres.go]
// for the shared database connection helper.
package dbadapter
