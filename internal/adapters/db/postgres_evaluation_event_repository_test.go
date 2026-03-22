//go:build integration

package dbadapter_test

import (
	"context"
	"testing"
	"time"

	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// evalEvtID builds a UUID-shaped string for evaluation event tests.
func evalEvtID(n int) string {
	return "cccccccc-cccc-4ccc-9ccc-" + padID(n)
}

func TestPostgresEvaluationEventRepository(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	projRepo := dbadapter.NewPostgresProjectRepository(db)
	proj := domain.Project{
		ID:        "dddddddd-dddd-4ddd-8ddd-dddddddddddd",
		Name:      "Eval Event Test Project",
		Slug:      "eval-event-test",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	if err := projRepo.Create(ctx, proj); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	repo := dbadapter.NewPostgresEvaluationEventRepository(db)

	base := time.Now().UTC().Truncate(time.Second).Add(-120 * time.Second)

	// Seed 15 events for flag-alpha in env-prod.
	for i := 0; i < 15; i++ {
		e := &domain.EvaluationEvent{
			ID:            evalEvtID(i),
			FlagKey:       "flag-alpha",
			ProjectID:     proj.ID,
			EnvironmentID: "env-prod",
			UserID:        "user-1",
			InputContext:  `{"plan":"pro"}`,
			VariantKey:    "variant-a",
			Reason:        domain.ReasonRuleMatch,
			OccurredAt:    base.Add(time.Duration(i) * time.Second),
		}
		if i%3 == 0 {
			e.MatchedRuleID = "rule-001"
		}
		if err := repo.Publish(ctx, e); err != nil {
			t.Fatalf("publish event %d: %v", i, err)
		}
	}

	// Seed 5 events for flag-beta.
	for i := 0; i < 5; i++ {
		e := &domain.EvaluationEvent{
			ID:            evalEvtID(20 + i),
			FlagKey:       "flag-beta",
			ProjectID:     proj.ID,
			EnvironmentID: "env-prod",
			UserID:        "user-2",
			InputContext:  `{}`,
			VariantKey:    "off",
			Reason:        domain.ReasonDisabled,
			OccurredAt:    base.Add(time.Duration(30+i) * time.Second),
		}
		if err := repo.Publish(ctx, e); err != nil {
			t.Fatalf("publish flag-beta event %d: %v", i, err)
		}
	}

	// @happy: list returns all 15 flag-alpha events newest first.
	t.Run("list all flag-alpha events newest first", func(t *testing.T) {
		events, err := repo.ListByFlagEnvironment(ctx, proj.ID, "env-prod", "flag-alpha", ports.EvaluationFilter{Limit: 50})
		if err != nil {
			t.Fatalf("ListByFlagEnvironment: %v", err)
		}
		if got := len(events); got != 15 {
			t.Fatalf("want 15 events, got %d", got)
		}
		for i := 1; i < len(events); i++ {
			if events[i].OccurredAt.After(events[i-1].OccurredAt) {
				t.Errorf("events not in reverse-chronological order at index %d", i)
			}
		}
	})

	// @happy: flag isolation — flag-beta events do not appear in flag-alpha results.
	t.Run("flag isolation", func(t *testing.T) {
		events, err := repo.ListByFlagEnvironment(ctx, proj.ID, "env-prod", "flag-beta", ports.EvaluationFilter{Limit: 50})
		if err != nil {
			t.Fatalf("ListByFlagEnvironment: %v", err)
		}
		if got := len(events); got != 5 {
			t.Fatalf("want 5 flag-beta events, got %d", got)
		}
		for _, e := range events {
			if e.FlagKey != "flag-beta" {
				t.Errorf("unexpected flag_key %q", e.FlagKey)
			}
		}
	})

	// @edge: cursor pagination — page through all 15 events in pages of 5.
	t.Run("cursor pagination", func(t *testing.T) {
		var all []*domain.EvaluationEvent
		var before time.Time
		for page := 0; page < 4; page++ {
			f := ports.EvaluationFilter{Limit: 5}
			if !before.IsZero() {
				f.Before = before
			}
			events, err := repo.ListByFlagEnvironment(ctx, proj.ID, "env-prod", "flag-alpha", f)
			if err != nil {
				t.Fatalf("page %d: %v", page, err)
			}
			all = append(all, events...)
			if len(events) < 5 {
				break
			}
			before = events[len(events)-1].OccurredAt
		}
		if got := len(all); got != 15 {
			t.Fatalf("pagination: want 15 total, got %d", got)
		}
		seen := make(map[string]bool, len(all))
		for _, e := range all {
			if seen[e.ID] {
				t.Errorf("duplicate event ID %q across pages", e.ID)
			}
			seen[e.ID] = true
		}
	})

	// @happy: unknown flag returns empty slice.
	t.Run("no events for unknown flag returns empty slice", func(t *testing.T) {
		events, err := repo.ListByFlagEnvironment(ctx, proj.ID, "env-prod", "nonexistent", ports.EvaluationFilter{Limit: 50})
		if err != nil {
			t.Fatalf("ListByFlagEnvironment: %v", err)
		}
		if len(events) != 0 {
			t.Errorf("want 0 events, got %d", len(events))
		}
	})

	// @edge: DeleteOlderThan removes old events and leaves newer ones untouched.
	t.Run("DeleteOlderThan removes events before cutoff, leaves newer", func(t *testing.T) {
		// Seed two fresh events: one old (200s ago), one recent (10s ago).
		oldEvt := &domain.EvaluationEvent{
			ID:            evalEvtID(100),
			FlagKey:       "flag-retention",
			ProjectID:     proj.ID,
			EnvironmentID: "env-prod",
			UserID:        "u",
			InputContext:  `{}`,
			VariantKey:    "off",
			Reason:        domain.ReasonDisabled,
			OccurredAt:    time.Now().UTC().Add(-200 * time.Second),
		}
		recentEvt := &domain.EvaluationEvent{
			ID:            evalEvtID(101),
			FlagKey:       "flag-retention",
			ProjectID:     proj.ID,
			EnvironmentID: "env-prod",
			UserID:        "u",
			InputContext:  `{}`,
			VariantKey:    "off",
			Reason:        domain.ReasonDisabled,
			OccurredAt:    time.Now().UTC().Add(-10 * time.Second),
		}
		for _, e := range []*domain.EvaluationEvent{oldEvt, recentEvt} {
			if err := repo.Publish(ctx, e); err != nil {
				t.Fatalf("publish: %v", err)
			}
		}

		// Cutoff: 60s ago — old event (200s ago) should be deleted, recent (10s ago) kept.
		cutoff := time.Now().UTC().Add(-60 * time.Second)
		if err := repo.DeleteOlderThan(ctx, cutoff); err != nil {
			t.Fatalf("DeleteOlderThan: %v", err)
		}

		remaining, err := repo.ListByFlagEnvironment(ctx, proj.ID, "env-prod", "flag-retention", ports.EvaluationFilter{Limit: 50})
		if err != nil {
			t.Fatalf("list after delete: %v", err)
		}
		if got := len(remaining); got != 1 {
			t.Fatalf("want 1 event after retention (the recent one), got %d", got)
		}
		if remaining[0].ID != recentEvt.ID {
			t.Errorf("want recent event %s to survive, got %s", recentEvt.ID, remaining[0].ID)
		}
	})
}
