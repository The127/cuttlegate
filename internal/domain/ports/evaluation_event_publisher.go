package ports

import (
	"context"

	"github.com/karo/cuttlegate/internal/domain"
)

// EvaluationEventPublisher is the port for persisting flag evaluation events.
// Implementations must be safe for concurrent use.
// Publish is best-effort — callers must not fail the evaluation path on error.
type EvaluationEventPublisher interface {
	Publish(ctx context.Context, event *domain.EvaluationEvent) error
}
