package httpadapter

import (
	"context"
	"net/http"

	"github.com/The127/cuttlegate/internal/domain"
)

// projectResolver resolves a project slug to a domain.Project.
type projectResolver interface {
	GetBySlug(ctx context.Context, slug string) (*domain.Project, error)
}

// environmentResolver resolves an environment slug within a project to a domain.Environment.
type environmentResolver interface {
	GetBySlug(ctx context.Context, projectID, slug string) (*domain.Environment, error)
}

// resolveProjectAndEnv looks up a project and environment by their URL slugs.
// On failure it writes an error response and returns false.
func resolveProjectAndEnv(ctx context.Context, w http.ResponseWriter, projects projectResolver, envs environmentResolver, projSlug, envSlug string) (*domain.Project, *domain.Environment, bool) {
	proj, err := projects.GetBySlug(ctx, projSlug)
	if err != nil {
		WriteError(w, err)
		return nil, nil, false
	}
	env, err := envs.GetBySlug(ctx, proj.ID, envSlug)
	if err != nil {
		WriteError(w, err)
		return nil, nil, false
	}
	return proj, env, true
}
