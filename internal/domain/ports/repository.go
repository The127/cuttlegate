package ports

import (
	"context"

	"github.com/karo/cuttlegate/internal/domain"
)

// FlagRepository is the port for persisting and retrieving feature flag entities.
type FlagRepository interface {
	Create(ctx context.Context, flag *domain.Flag) error
	GetByKey(ctx context.Context, projectID, key string) (*domain.Flag, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.Flag, error)
	Update(ctx context.Context, flag *domain.Flag) error
	Delete(ctx context.Context, id string) error
}

// ProjectRepository is the port for persisting and retrieving project entities.
type ProjectRepository interface {
	Create(ctx context.Context, project domain.Project) error
	GetBySlug(ctx context.Context, slug string) (*domain.Project, error)
	List(ctx context.Context) ([]*domain.Project, error)
	UpdateName(ctx context.Context, id, name string) error
	Delete(ctx context.Context, id string) error
}

// EnvironmentRepository is the port for persisting and retrieving environment entities.
type EnvironmentRepository interface {
	Create(ctx context.Context, env domain.Environment) error
	GetBySlug(ctx context.Context, projectID, slug string) (*domain.Environment, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.Environment, error)
	Delete(ctx context.Context, id string) error
}

// RuleRepository is the port for persisting and retrieving targeting rules.
type RuleRepository interface {
	ListByFlagEnvironment(ctx context.Context, flagID, environmentID string) ([]*domain.Rule, error)
	Upsert(ctx context.Context, rule *domain.Rule) error
	Delete(ctx context.Context, id string) error
}
