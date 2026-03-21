// Package domain contains the core business entities, value objects, and rules for Cuttlegate.
//
// This is where projects, flags, environments, rules, and evaluation logic live as
// plain Go structs and functions. Everything here is pure — no database, no HTTP, no
// external dependencies. Only the standard library is allowed.
//
// This package does not define how entities are stored or served. Persistence interfaces
// live in domain/ports; HTTP and SQL live in adapters. If you need to add a new business
// concept, it belongs here. If you need to add a new way to store or fetch one, it does not.
//
// Start here: [Flag] is the central entity. Most other types exist to support flag
// evaluation — see [EvalContext] and [EvalResult] for the evaluation contract.
package domain
