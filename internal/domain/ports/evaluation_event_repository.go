package ports

import (
	"context"
	"time"

	"github.com/The127/cuttlegate/internal/domain"
)

// EvaluationFilter constrains evaluation event queries.
type EvaluationFilter struct {
	Before time.Time // cursor: return events older than this timestamp (exclusive)
	Limit  int
}

// DefaultEvaluationLimit is the default page size for evaluation event queries.
const DefaultEvaluationLimit = 50

// MaxEvaluationLimit caps the number of events returned in a single query.
const MaxEvaluationLimit = 200

// NormalizeEvaluationFilter applies defaults and caps to f.
func NormalizeEvaluationFilter(f EvaluationFilter) EvaluationFilter {
	if f.Limit <= 0 {
		f.Limit = DefaultEvaluationLimit
	}
	if f.Limit > MaxEvaluationLimit {
		f.Limit = MaxEvaluationLimit
	}
	return f
}

// EvaluationEventRepository is the port for reading and writing evaluation events.
// The underlying store is append-only — records are never updated.
type EvaluationEventRepository interface {
	EvaluationEventPublisher
	ListByFlagEnvironment(ctx context.Context, projectID, environmentID, flagKey string, filter EvaluationFilter) ([]*domain.EvaluationEvent, error)
	// DeleteOlderThan removes events older than cutoff. Used by the retention worker.
	DeleteOlderThan(ctx context.Context, cutoff time.Time) error
}
