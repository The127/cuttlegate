// Package main is the standalone database migration runner.
//
// This binary applies or rolls back SQL migrations in db/migrations. It is separate
// from the server binary so migrations can be run independently in CI and production.
//
// Allowed imports: dbadapter, stdlib, and migration libraries. No app or HTTP packages.
package main
