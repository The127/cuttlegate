package ports

import (
	"context"
	"time"
)

// DomainEvent is the common interface for all events produced by the domain layer.
// Concrete event types implement this interface and carry their own typed payload.
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// EventPublisher is the port for publishing domain events to interested consumers
// (e.g. SSE delivery, audit log). Implementations must be safe for concurrent use.
type EventPublisher interface {
	Publish(ctx context.Context, event DomainEvent) error
}
