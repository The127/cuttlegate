package httpadapter

import (
	"context"

	"github.com/karo/cuttlegate/internal/domain"
)

// projectResolver resolves a project slug to a domain.Project.
type projectResolver interface {
	GetBySlug(ctx context.Context, slug string) (*domain.Project, error)
}
