package ports

import (
	"context"

	"github.com/karo/cuttlegate/internal/domain"
)

// AuditRepository is the port for persisting and querying audit events.
// Implementations must treat the underlying store as append-only.
type AuditRepository interface {
	Record(ctx context.Context, event *domain.AuditEvent) error
	ListByProject(ctx context.Context, projectID string, filter domain.AuditFilter) ([]*domain.AuditEvent, error)
}
