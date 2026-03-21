// Package app contains the use-case services that orchestrate domain objects through
// port interfaces to fulfill business operations.
//
// Each service (e.g. FlagService, EvaluationService) owns one area of functionality.
// Services accept port interfaces via constructor injection and call them to read and
// write data. They enforce RBAC rules before delegating to the domain layer.
//
// This package does not know about HTTP, SQL, or any infrastructure technology. It also
// does not contain business rules — those live in the domain package. If your code needs
// an http.Request or a *sql.DB, it belongs in an adapter, not here.
//
// Start here: [FlagService] for flag CRUD operations, or [EvaluationService] for the
// flag evaluation use case that SDKs call.
package app
