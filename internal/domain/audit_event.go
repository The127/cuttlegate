package domain

import "time"

// Source constants identify the origin of an audit event.
//
//   - SourceHTTP: default — event originated from an HTTP API call (empty string for backward compat).
//   - SourceMCP: event originated from an MCP tool invocation.
//   - SourceSystem: event originated from a system process (e.g. retention cleanup, evaluation-time fixups)
//     with no human actor. ActorID is typically empty for system events.
const (
	SourceHTTP   = ""
	SourceMCP    = "mcp"
	SourceSystem = "system"
)

// AuditEvent records a mutation performed by an actor on a project resource.
// Instances are append-only — once persisted, they are never modified.
type AuditEvent struct {
	ID              string
	ProjectID       string
	ActorID         string // sub of the acting user; empty for system-generated events (Source == SourceSystem)
	ActorEmail      string // read-concern: populated by JOIN on users; not stored on the event row; empty when ActorID is empty
	Action          string
	EntityType      string
	EntityID        string
	EntityKey       string // human-readable key (e.g. flag key) — stored for queryability after deletion
	EnvironmentSlug string // slug of the environment affected; empty for project-scoped actions
	Source          string // origin of the mutation: SourceHTTP (default), SourceMCP, or SourceSystem
	BeforeState     string
	AfterState      string
	OccurredAt      time.Time
}

// AuditFilter constrains audit log queries.
type AuditFilter struct {
	FlagKey string
	Before  time.Time // cursor: return events older than this
	Limit   int
}

// DefaultAuditLimit is used when no limit is specified.
const DefaultAuditLimit = 50

// MaxAuditLimit caps the number of events returned in a single query.
const MaxAuditLimit = 200

// NormalizeAuditFilter applies defaults and caps to f.
func NormalizeAuditFilter(f AuditFilter) AuditFilter {
	if f.Limit <= 0 {
		f.Limit = DefaultAuditLimit
	}
	if f.Limit > MaxAuditLimit {
		f.Limit = MaxAuditLimit
	}
	return f
}
