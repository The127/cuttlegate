package app_test

import (
	"context"
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
