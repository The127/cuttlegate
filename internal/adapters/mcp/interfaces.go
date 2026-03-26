package mcp

import (
	"context"

	"github.com/The127/cuttlegate/internal/app"
	"github.com/The127/cuttlegate/internal/domain"
)

// apiKeyAuthenticator is the subset of app.APIKeyService used by the MCP server.
type apiKeyAuthenticator interface {
	AuthenticateMCP(ctx context.Context, plaintext string) (*domain.APIKey, error)
}

// flagService is the subset of app.FlagService used by the MCP server.
type flagService interface {
	ListByEnvironment(ctx context.Context, projectID, environmentID string) ([]*app.FlagEnvironmentView, error)
	SetEnabled(ctx context.Context, params app.SetEnabledParams) error
}

// evaluationService is the subset of app.EvaluationService used by the MCP server.
type evaluationService interface {
	Evaluate(ctx context.Context, projectID, environmentID, flagKey string, evalCtx domain.EvalContext) (*app.EvalView, error)
}

// projectResolver resolves project slugs to domain.Project.
type projectResolver interface {
	GetBySlug(ctx context.Context, slug string) (*domain.Project, error)
}

// environmentResolver resolves environment slugs to domain.Environment.
type environmentResolver interface {
	GetBySlug(ctx context.Context, projectID, slug string) (*domain.Environment, error)
}
