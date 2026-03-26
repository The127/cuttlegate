package ports

import (
	"context"

	"github.com/The127/cuttlegate/internal/domain"
)

// FlagEnvironmentStateRepository is the port for persisting per-environment flag state.
type FlagEnvironmentStateRepository interface {
	// CreateBatch inserts disabled state rows for a slice of flags across environments.
	// Used at flag creation time to ensure every flag has a state row for each existing environment.
	CreateBatch(ctx context.Context, states []*domain.FlagEnvironmentState) error
	// ListByEnvironment returns all state rows for a given environment. Returns empty slice, never nil.
	ListByEnvironment(ctx context.Context, environmentID string) ([]*domain.FlagEnvironmentState, error)
	// GetByFlagAndEnvironment returns the state row for a specific flag+environment pair.
	// Returns nil, nil when no state row exists (treated as disabled by the evaluation engine).
	GetByFlagAndEnvironment(ctx context.Context, flagID, environmentID string) (*domain.FlagEnvironmentState, error)
	// SetEnabled updates the enabled field for a specific flag+environment pair.
	// Returns ErrNotFound if no state row exists for that combination.
	SetEnabled(ctx context.Context, flagID, environmentID string, enabled bool) error
	// Upsert inserts a state row or updates the enabled field if one already exists.
	Upsert(ctx context.Context, state *domain.FlagEnvironmentState) error
}
