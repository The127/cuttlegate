package ports

import (
	"context"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
)

// FlagEvaluationStatsRepository is the port for persisting and querying
// flag evaluation statistics. Implementations must be safe for concurrent use.
type FlagEvaluationStatsRepository interface {
	// Upsert increments the evaluation count and updates last_evaluated_at for
	// the given flag+environment combination. If no row exists it is created.
	Upsert(ctx context.Context, flagID, environmentID, flagKey string, evaluatedAt time.Time) error

	// GetByFlagEnvironment returns the evaluation stats for a flag in an
	// environment. Returns a zero-count stats struct if no evaluations have
	// been recorded — never returns ErrNotFound.
	GetByFlagEnvironment(ctx context.Context, flagID, environmentID string) (*domain.FlagEvaluationStats, error)
}
