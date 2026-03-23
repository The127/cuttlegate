//go:build integration

package dbadapter_test

import (
	"context"
	"testing"
	"time"

	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
	"github.com/karo/cuttlegate/internal/domain"
)

// TestGetBuckets_Integration verifies the generate_series + LEFT JOIN zero-fill
// query against a real Postgres container.
func TestGetBuckets_Integration(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// Seed a project (required for FK-free inserts on evaluation_events).
	projRepo := dbadapter.NewPostgresProjectRepository(db)
	proj := domain.Project{
		ID:        "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee",
		Name:      "Buckets Integration Test",
		Slug:      "buckets-integration-test",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	if err := projRepo.Create(ctx, proj); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	eventRepo := dbadapter.NewPostgresEvaluationEventRepository(db)
	statsRepo := dbadapter.NewPostgresFlagEvaluationStatsRepository(db)

	// Seed events: 98 "enabled" + 44 "disabled" on the oldest day of the 7-day window,
	// nothing on the other 6 days.
	now := time.Now().UTC().Truncate(24 * time.Hour)
	day6 := now.Add(-6 * 24 * time.Hour).Add(time.Hour) // one hour into the oldest day

	evtID := func(n int) string {
		return "ffffffff-ffff-4fff-9fff-" + padID(n)
	}

	// Seed 98 "enabled" events.
	for i := 0; i < 98; i++ {
		e := &domain.EvaluationEvent{
			ID:            evtID(i),
			FlagKey:       "sparse-flag",
			ProjectID:     proj.ID,
			EnvironmentID: "env-buckets",
			UserID:        "u",
			InputContext:  `{}`,
			VariantKey:    "enabled",
			Reason:        domain.ReasonRuleMatch,
			OccurredAt:    day6.Add(time.Duration(i) * time.Second),
		}
		if err := eventRepo.Publish(ctx, e); err != nil {
			t.Fatalf("publish enabled event %d: %v", i, err)
		}
	}

	// Seed 44 "disabled" events.
	for i := 0; i < 44; i++ {
		e := &domain.EvaluationEvent{
			ID:            evtID(200 + i),
			FlagKey:       "sparse-flag",
			ProjectID:     proj.ID,
			EnvironmentID: "env-buckets",
			UserID:        "u",
			InputContext:  `{}`,
			VariantKey:    "disabled",
			Reason:        domain.ReasonDisabled,
			OccurredAt:    day6.Add(time.Duration(100+i) * time.Second),
		}
		if err := eventRepo.Publish(ctx, e); err != nil {
			t.Fatalf("publish disabled event %d: %v", i, err)
		}
	}

	// since is the start of the oldest slot: now - 6d gives exactly 7 daily buckets
	// because generate_series is inclusive on both ends (now-6d to now = 7 slots).
	since := now.Add(-6 * 24 * time.Hour)

	// @happy: 7-day window, daily buckets — exactly 7 buckets, one with events.
	t.Run("7d daily buckets zero-filled", func(t *testing.T) {
		buckets, err := statsRepo.GetBuckets(ctx, proj.ID, "env-buckets", "sparse-flag", since, "day")
		if err != nil {
			t.Fatalf("GetBuckets: %v", err)
		}
		if len(buckets) != 7 {
			t.Fatalf("want 7 buckets, got %d", len(buckets))
		}

		// Find the bucket containing day6 events.
		day6Bucket := day6.Truncate(24 * time.Hour)
		var found bool
		for _, b := range buckets {
			if b.Timestamp.Equal(day6Bucket) {
				found = true
				if b.Total != 142 {
					t.Errorf("day6 bucket total: want 142, got %d", b.Total)
				}
				if b.Variants["enabled"] != 98 {
					t.Errorf("enabled variant: want 98, got %d", b.Variants["enabled"])
				}
				if b.Variants["disabled"] != 44 {
					t.Errorf("disabled variant: want 44, got %d", b.Variants["disabled"])
				}
			} else {
				// All other days must be zero-filled.
				if b.Total != 0 {
					t.Errorf("bucket %v: want total=0 (zero-fill), got %d", b.Timestamp, b.Total)
				}
			}
		}
		if !found {
			t.Errorf("did not find expected bucket at %v in response", day6Bucket)
		}
	})

	// @happy: flag with no events in window — all buckets zero-filled.
	t.Run("no events — all buckets zero-filled", func(t *testing.T) {
		buckets, err := statsRepo.GetBuckets(ctx, proj.ID, "env-buckets", "new-flag-no-events", since, "day")
		if err != nil {
			t.Fatalf("GetBuckets: %v", err)
		}
		if len(buckets) != 7 {
			t.Fatalf("want 7 buckets, got %d", len(buckets))
		}
		for _, b := range buckets {
			if b.Total != 0 {
				t.Errorf("bucket %v: want total=0, got %d", b.Timestamp, b.Total)
			}
		}
	})

	// @happy: hourly buckets for 1d window — 24 buckets.
	// Truncate to hour (not day) so since = nowHour - 23h gives exactly 24 hourly
	// slots (nowHour-23h to nowHour inclusive) regardless of when the test runs.
	t.Run("1d hourly buckets", func(t *testing.T) {
		nowHour := time.Now().UTC().Truncate(time.Hour)
		hourSince := nowHour.Add(-23 * time.Hour)
		buckets, err := statsRepo.GetBuckets(ctx, proj.ID, "env-buckets", "sparse-flag", hourSince, "hour")
		if err != nil {
			t.Fatalf("GetBuckets hour: %v", err)
		}
		if len(buckets) != 24 {
			t.Errorf("want 24 hourly buckets, got %d", len(buckets))
		}
		for _, b := range buckets {
			if b.Total < 0 {
				t.Errorf("bucket total should be >= 0, got %d", b.Total)
			}
		}
	})

	// @happy: variants map is keyed by actual variant strings (multi-variant).
	t.Run("variant keys are strings not booleans", func(t *testing.T) {
		buckets, err := statsRepo.GetBuckets(ctx, proj.ID, "env-buckets", "sparse-flag", since, "day")
		if err != nil {
			t.Fatalf("GetBuckets: %v", err)
		}
		for _, b := range buckets {
			if b.Total > 0 {
				if _, ok := b.Variants["enabled"]; !ok {
					t.Errorf("expected 'enabled' key in variants map, got %v", b.Variants)
				}
				if _, ok := b.Variants["disabled"]; !ok {
					t.Errorf("expected 'disabled' key in variants map, got %v", b.Variants)
				}
			}
		}
	})
}
