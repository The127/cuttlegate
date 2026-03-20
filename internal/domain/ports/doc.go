// Package ports defines the interfaces (ports) through which the domain and application
// layers communicate with the outside world.
//
// This package contains Go interfaces only — no implementations. Adapters implement these
// interfaces; the app layer depends on them.
//
// Allowed imports: domain, stdlib. No adapter or infrastructure packages.
package ports
