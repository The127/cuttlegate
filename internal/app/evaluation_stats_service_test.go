package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// fakeStatsRepository is an in-memory FlagEvaluationStatsRepository.
type fakeStatsRepository struct {
	stats map[string]*domain.FlagEvaluationStats // key: flagID+"/"+envID
	err   error
}

func newFakeStatsRepository() *fakeStatsRepository {
	return &fakeStatsRepository{stats: make(map[string]*domain.FlagEvaluationStats)}
}

var _ ports.FlagEvaluationStatsRepository = (*fakeStatsRepository)(nil)

func (r *fakeStatsRepository) Upsert(_ context.Context, flagID, environmentID, flagKey string, evaluatedAt time.Time) error {
	if r.err != nil {
		return r.err
	}
	key := flagID + "/" + environmentID
	if existing, ok := r.stats[key]; ok {
		existing.EvaluationCount++
		existing.LastEvaluatedAt = evaluatedAt
	} else {
		r.stats[key] = &domain.FlagEvaluationStats{
			FlagID:          flagID,
			EnvironmentID:   environmentID,
			FlagKey:         flagKey,
			EvaluationCount: 1,
			LastEvaluatedAt: evaluatedAt,
		}
	}
	return nil
}

func (r *fakeStatsRepository) GetByFlagEnvironment(_ context.Context, flagID, environmentID string) (*domain.FlagEvaluationStats, error) {
	if r.err != nil {
		return nil, r.err
	}
	key := flagID + "/" + environmentID
	if s, ok := r.stats[key]; ok {
		return s, nil
	}
	return &domain.FlagEvaluationStats{FlagID: flagID, EnvironmentID: environmentID}, nil
}

// bucketsResult holds the canned result for GetBuckets in tests.
type fakeStatsRepositoryWithBuckets struct {
	*fakeStatsRepository
	buckets   []domain.EvaluationBucket
	bucketErr error
}

func (r *fakeStatsRepositoryWithBuckets) GetBuckets(_ context.Context, _, _, _ string, _ time.Time, _ string) ([]domain.EvaluationBucket, error) {
	return r.buckets, r.bucketErr
}

func (r *fakeStatsRepository) GetBuckets(_ context.Context, _, _, _ string, _ time.Time, _ string) ([]domain.EvaluationBucket, error) {
	return nil, nil
}

func newStatsSvc(flagRepo *fakeFlagRepository, statsRepo *fakeStatsRepository) *app.EvaluationStatsService {
	return app.NewEvaluationStatsService(statsRepo, flagRepo)
}

// @happy: flag has been evaluated — GetStats returns count and last timestamp.
func TestEvaluationStatsService_GetStats_WithEvaluations(t *testing.T) {
	ctx := authCtx("viewer-1", domain.RoleViewer)
	flagRepo := newFakeFlagRepository()
	statsRepo := newFakeStatsRepository()

	flag := seedFlag(t, flagRepo, "flag-id-1", "proj-1", "my-flag")
	ts := time.Date(2026, 3, 21, 14, 0, 0, 0, time.UTC)
	if err := statsRepo.Upsert(ctx, flag.ID, "env-1", flag.Key, ts); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	svc := newStatsSvc(flagRepo, statsRepo)
	view, err := svc.GetStats(ctx, "proj-1", "env-1", "my-flag")
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if view.EvaluationCount != 1 {
		t.Errorf("EvaluationCount: want 1, got %d", view.EvaluationCount)
	}
	if view.LastEvaluatedAt == nil {
		t.Fatal("LastEvaluatedAt: want non-nil")
	}
	if !view.LastEvaluatedAt.Equal(ts) {
		t.Errorf("LastEvaluatedAt: want %v, got %v", ts, *view.LastEvaluatedAt)
	}
}

// @edge: flag never evaluated — GetStats returns count 0 and nil timestamp.
func TestEvaluationStatsService_GetStats_NeverEvaluated(t *testing.T) {
	ctx := authCtx("viewer-1", domain.RoleViewer)
	flagRepo := newFakeFlagRepository()
	statsRepo := newFakeStatsRepository()

	seedFlag(t, flagRepo, "flag-id-1", "proj-1", "new-flag")

	svc := newStatsSvc(flagRepo, statsRepo)
	view, err := svc.GetStats(ctx, "proj-1", "env-1", "new-flag")
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if view.EvaluationCount != 0 {
		t.Errorf("EvaluationCount: want 0, got %d", view.EvaluationCount)
	}
	if view.LastEvaluatedAt != nil {
		t.Errorf("LastEvaluatedAt: want nil, got %v", view.LastEvaluatedAt)
	}
}

// @error-path: flag does not exist — GetStats returns ErrNotFound.
func TestEvaluationStatsService_GetStats_FlagNotFound(t *testing.T) {
	ctx := authCtx("viewer-1", domain.RoleViewer)
	flagRepo := newFakeFlagRepository()
	statsRepo := newFakeStatsRepository()

	svc := newStatsSvc(flagRepo, statsRepo)
	_, err := svc.GetStats(ctx, "proj-1", "env-1", "no-such-flag")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// @happy: GetBuckets returns populated view for valid parameters.
func TestEvaluationStatsService_GetBuckets_Happy(t *testing.T) {
	ctx := authCtx("viewer-1", domain.RoleViewer)
	flagRepo := newFakeFlagRepository()
	seedFlag(t, flagRepo, "flag-id-1", "proj-1", "my-flag")

	now := time.Now().UTC().Truncate(24 * time.Hour)
	fakeBuckets := []domain.EvaluationBucket{
		{Timestamp: now.Add(-6 * 24 * time.Hour), Total: 0, Variants: map[string]int64{}},
		{Timestamp: now.Add(-5 * 24 * time.Hour), Total: 142, Variants: map[string]int64{"enabled": 98, "disabled": 44}},
		{Timestamp: now.Add(-4 * 24 * time.Hour), Total: 0, Variants: map[string]int64{}},
		{Timestamp: now.Add(-3 * 24 * time.Hour), Total: 0, Variants: map[string]int64{}},
		{Timestamp: now.Add(-2 * 24 * time.Hour), Total: 0, Variants: map[string]int64{}},
		{Timestamp: now.Add(-1 * 24 * time.Hour), Total: 0, Variants: map[string]int64{}},
		{Timestamp: now, Total: 0, Variants: map[string]int64{}},
	}
	statsRepo := &fakeStatsRepositoryWithBuckets{
		fakeStatsRepository: newFakeStatsRepository(),
		buckets:             fakeBuckets,
	}

	svc := app.NewEvaluationStatsService(statsRepo, flagRepo)
	view, err := svc.GetBuckets(ctx, "proj-1", "env-1", "production", "my-flag", "7d", "day")
	if err != nil {
		t.Fatalf("GetBuckets: %v", err)
	}
	if view.FlagKey != "my-flag" {
		t.Errorf("FlagKey: want my-flag, got %q", view.FlagKey)
	}
	if view.Window != "7d" {
		t.Errorf("Window: want 7d, got %q", view.Window)
	}
	if view.BucketSize != "day" {
		t.Errorf("BucketSize: want day, got %q", view.BucketSize)
	}
	if len(view.Buckets) != 7 {
		t.Errorf("Buckets: want 7, got %d", len(view.Buckets))
	}
}

// @error-path: invalid window — GetBuckets returns ErrInvalidParameter.
func TestEvaluationStatsService_GetBuckets_InvalidWindow(t *testing.T) {
	ctx := authCtx("viewer-1", domain.RoleViewer)
	flagRepo := newFakeFlagRepository()
	seedFlag(t, flagRepo, "flag-id-1", "proj-1", "my-flag")
	statsRepo := newFakeStatsRepository()

	svc := app.NewEvaluationStatsService(statsRepo, flagRepo)
	_, err := svc.GetBuckets(ctx, "proj-1", "env-1", "production", "my-flag", "45d", "day")
	if !errors.Is(err, app.ErrInvalidParameter) {
		t.Errorf("want ErrInvalidParameter, got %v", err)
	}
}

// @error-path: hour bucket with window > 7d — returns ErrInvalidParameter.
func TestEvaluationStatsService_GetBuckets_HourBucketInvalidWindow(t *testing.T) {
	ctx := authCtx("viewer-1", domain.RoleViewer)
	flagRepo := newFakeFlagRepository()
	seedFlag(t, flagRepo, "flag-id-1", "proj-1", "my-flag")
	statsRepo := newFakeStatsRepository()

	svc := app.NewEvaluationStatsService(statsRepo, flagRepo)
	_, err := svc.GetBuckets(ctx, "proj-1", "env-1", "production", "my-flag", "30d", "hour")
	if !errors.Is(err, app.ErrInvalidParameter) {
		t.Errorf("want ErrInvalidParameter, got %v", err)
	}
}

// @error-path: invalid bucket size — returns ErrInvalidParameter.
func TestEvaluationStatsService_GetBuckets_InvalidBucketSize(t *testing.T) {
	ctx := authCtx("viewer-1", domain.RoleViewer)
	flagRepo := newFakeFlagRepository()
	seedFlag(t, flagRepo, "flag-id-1", "proj-1", "my-flag")
	statsRepo := newFakeStatsRepository()

	svc := app.NewEvaluationStatsService(statsRepo, flagRepo)
	_, err := svc.GetBuckets(ctx, "proj-1", "env-1", "production", "my-flag", "7d", "week")
	if !errors.Is(err, app.ErrInvalidParameter) {
		t.Errorf("want ErrInvalidParameter, got %v", err)
	}
}

// @error-path: unknown flag — GetBuckets returns ErrNotFound.
func TestEvaluationStatsService_GetBuckets_FlagNotFound(t *testing.T) {
	ctx := authCtx("viewer-1", domain.RoleViewer)
	flagRepo := newFakeFlagRepository()
	statsRepo := newFakeStatsRepository()

	svc := app.NewEvaluationStatsService(statsRepo, flagRepo)
	_, err := svc.GetBuckets(ctx, "proj-1", "env-1", "production", "no-such-flag", "7d", "day")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}
