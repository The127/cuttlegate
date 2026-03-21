// Package ports defines the interfaces through which the application layer interacts
// with infrastructure — databases, event buses, token verifiers, and anything else that
// lives outside the domain.
//
// Every type here is a Go interface. There are no implementations. Adapters (in adapters/db,
// adapters/http) implement these interfaces; the app layer depends on them by type, never
// by concrete struct.
//
// This package does not contain business logic or infrastructure code. If you need to add
// a new way for the app layer to talk to the outside world, define the interface here.
// If you need to implement it, that goes in an adapter package.
//
// Start here: [FlagRepository] and [FlagEnvironmentStateRepository] are the most-used
// interfaces — they back flag CRUD and evaluation.
package ports
