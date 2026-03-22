package app

import (
	"context"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// FlagStatsView is the result of querying evaluation statistics for a flag.
type FlagStatsView struct {
	LastEvaluatedAt *time.Time
	EvaluationCount int64
}

// EvaluationStatsService orchestrates flag evaluation statistics use cases.
type EvaluationStatsService struct {
	statsRepo ports.FlagEvaluationStatsRepository
	flagRepo  ports.FlagRepository
}

// NewEvaluationStatsService constructs an EvaluationStatsService.
func NewEvaluationStatsService(
	statsRepo ports.FlagEvaluationStatsRepository,
	flagRepo ports.FlagRepository,
) *EvaluationStatsService {
	return &EvaluationStatsService{
		statsRepo: statsRepo,
		flagRepo:  flagRepo,
	}
}

// GetStats returns evaluation statistics for a flag in a specific environment.
// Returns zero counts when no evaluations have been recorded — never ErrNotFound.
// Requires at least viewer role.
func (s *EvaluationStatsService) GetStats(ctx context.Context, projectID, environmentID, flagKey string) (*FlagStatsView, error) {
	if _, err := requireRole(ctx, domain.RoleViewer); err != nil {
		return nil, err
	}

	flag, err := s.flagRepo.GetByKey(ctx, projectID, flagKey)
	if err != nil {
		return nil, err
	}

	stats, err := s.statsRepo.GetByFlagEnvironment(ctx, flag.ID, environmentID)
	if err != nil {
		return nil, err
	}

	view := &FlagStatsView{
		EvaluationCount: stats.EvaluationCount,
	}
	if stats.EvaluationCount > 0 {
		t := stats.LastEvaluatedAt.UTC()
		view.LastEvaluatedAt = &t
	}
	return view, nil
}
