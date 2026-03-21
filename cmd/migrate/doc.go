// Package main is the standalone database migration runner for Cuttlegate.
//
// This binary applies or rolls back SQL migrations from db/migrations/ against a
// PostgreSQL database. It runs independently of the server binary so that migrations
// can be executed separately in CI and production deployments.
//
// This package does not serve HTTP, run business logic, or interact with any application
// service. If you need to change a database schema, add a new migration file in
// db/migrations/ — do not modify this package.
//
// Start here: main.go for the migration execution logic.
package main
