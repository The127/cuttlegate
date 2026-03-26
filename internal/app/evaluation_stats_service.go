package app

import (
	"context"
	"errors"
	"time"

	"github.com/The127/cuttlegate/internal/domain"
	"github.com/The127/cuttlegate/internal/domain/ports"
)

// FlagStatsView is the result of querying evaluation statistics for a flag.
type FlagStatsView struct {
	LastEvaluatedAt *time.Time
	EvaluationCount int64
}

// BucketView is a single time slot in a bucketed evaluation count result.
type BucketView struct {
	Timestamp time.Time
	Total     int64
	Variants  map[string]int64
}

// EvaluationBucketsView is the result of a bucketed evaluation count query.
type EvaluationBucketsView struct {
	FlagKey     string
	Environment string
	Window      string
	BucketSize  string
	Buckets     []BucketView
}

// validWindows maps window string values to their duration.
var validWindows = map[string]time.Duration{
	"1d":  24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"14d": 14 * 24 * time.Hour,
	"30d": 30 * 24 * time.Hour,
	"90d": 90 * 24 * time.Hour,
}

// windowAllowsHour lists window values that permit bucket=hour.
var windowAllowsHour = map[string]bool{
	"1d": true,
	"7d": true,
}

// ErrInvalidParameter is returned when a request parameter is invalid or missing.
var ErrInvalidParameter = errors.New("invalid_parameter")

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

// GetBuckets returns time-bucketed evaluation counts for a flag in a specific
// environment. window must be one of: 1d, 7d, 14d, 30d, 90d. bucketSize must
// be "day" or "hour"; "hour" is only valid when window <= 7d.
// The response always contains a full set of zero-filled buckets for the window.
// Requires at least viewer role.
func (s *EvaluationStatsService) GetBuckets(ctx context.Context, projectID, environmentID, envSlug, flagKey, window, bucketSize string) (*EvaluationBucketsView, error) {
	if _, err := requireRole(ctx, domain.RoleViewer); err != nil {
		return nil, err
	}

	windowDur, ok := validWindows[window]
	if !ok {
		return nil, ErrInvalidParameter
	}
	if bucketSize != "day" && bucketSize != "hour" {
		return nil, ErrInvalidParameter
	}
	if bucketSize == "hour" && !windowAllowsHour[window] {
		return nil, ErrInvalidParameter
	}

	// Verify the flag exists in this project — return ErrNotFound if absent.
	if _, err := s.flagRepo.GetByKey(ctx, projectID, flagKey); err != nil {
		return nil, err
	}

	var bucketDur time.Duration
	if bucketSize == "hour" {
		bucketDur = time.Hour
	} else {
		bucketDur = 24 * time.Hour
	}

	// since is the start of the oldest bucket in the window.
	// generate_series is inclusive on both ends, so with (windowDur/bucketDur) steps
	// we need since = now - windowDur + bucketDur to produce exactly windowDur/bucketDur buckets.
	// Example: 7d/day → since = now - 6d → slots [now-6d, now-5d, …, now] = 7 buckets.
	now := time.Now().UTC().Truncate(bucketDur)
	since := now.Add(-windowDur).Add(bucketDur)

	raw, err := s.statsRepo.GetBuckets(ctx, projectID, environmentID, flagKey, since, bucketSize)
	if err != nil {
		return nil, err
	}

	buckets := make([]BucketView, len(raw))
	for i, b := range raw {
		bv := BucketView{
			Timestamp: b.Timestamp,
			Total:     b.Total,
			Variants:  b.Variants,
		}
		if bv.Variants == nil {
			bv.Variants = map[string]int64{}
		}
		buckets[i] = bv
	}

	return &EvaluationBucketsView{
		FlagKey:     flagKey,
		Environment: envSlug,
		Window:      window,
		BucketSize:  bucketSize,
		Buckets:     buckets,
	}, nil
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
