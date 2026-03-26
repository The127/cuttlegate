package app

import (
	"context"

	"github.com/The127/cuttlegate/internal/domain"
	"github.com/The127/cuttlegate/internal/domain/ports"
)

// AuditService orchestrates audit log queries.
type AuditService struct {
	repo ports.AuditRepository
}

// NewAuditService constructs an AuditService.
func NewAuditService(repo ports.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

// ListByProject returns audit events for a project, filtered and paginated.
// Requires at least viewer role (per ADR 0008).
func (s *AuditService) ListByProject(ctx context.Context, projectID string, filter domain.AuditFilter) ([]*domain.AuditEvent, error) {
	if _, err := requireRole(ctx, domain.RoleViewer); err != nil {
		return nil, err
	}
	filter = domain.NormalizeAuditFilter(filter)
	return s.repo.ListByProject(ctx, projectID, filter)
}
