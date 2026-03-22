//go:build integration

package dbadapter_test

import (
	"context"
	"testing"
	"time"

	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
	"github.com/karo/cuttlegate/internal/domain"
)

func TestPostgresAuditRepository_FilterAndPagination(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// Seed a project row (FK required by audit_events.project_id).
	projRepo := dbadapter.NewPostgresProjectRepository(db)
	proj := domain.Project{
		ID:        "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		Name:      "Audit Test Project",
		Slug:      "audit-test",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	if err := projRepo.Create(ctx, proj); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	// Seed a user so the JOIN can resolve actor_email.
	userRepo := dbadapter.NewPostgresUserRepository(db)
	actor := &domain.User{Sub: "actor-001", Name: "Alice", Email: "alice@example.com"}
	if err := userRepo.Upsert(ctx, actor); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	auditRepo := dbadapter.NewPostgresAuditRepository(db)

	// Insert 10 events for flag-a and 5 events for flag-b, spaced 1s apart.
	base := time.Now().UTC().Truncate(time.Second).Add(-60 * time.Second)
	eventID := func(n int) string {
		return "bbbbbbbb-bbbb-4bbb-8bbb-" + padID(n)
	}
	for i := 0; i < 10; i++ {
		e := &domain.AuditEvent{
			ID:              eventID(i),
			ProjectID:       proj.ID,
			ActorID:         actor.Sub,
			Action:          "flag.enabled",
			EntityType:      "flag",
			EntityID:        "flag-a-id",
			EntityKey:       "flag-a",
			EnvironmentSlug: "production",
			OccurredAt:      base.Add(time.Duration(i) * time.Second),
		}
		if err := auditRepo.Record(ctx, e); err != nil {
			t.Fatalf("record flag-a event %d: %v", i, err)
		}
	}
	for i := 0; i < 5; i++ {
		e := &domain.AuditEvent{
			ID:              eventID(10 + i),
			ProjectID:       proj.ID,
			ActorID:         actor.Sub,
			Action:          "flag.disabled",
			EntityType:      "flag",
			EntityID:        "flag-b-id",
			EntityKey:       "flag-b",
			EnvironmentSlug: "staging",
			OccurredAt:      base.Add(time.Duration(20+i) * time.Second),
		}
		if err := auditRepo.Record(ctx, e); err != nil {
			t.Fatalf("record flag-b event %d: %v", i, err)
		}
	}

	t.Run("list all — returns 15 events in reverse-chronological order", func(t *testing.T) {
		events, err := auditRepo.ListByProject(ctx, proj.ID, domain.AuditFilter{Limit: 50})
		if err != nil {
			t.Fatalf("ListByProject: %v", err)
		}
		if got := len(events); got != 15 {
			t.Fatalf("want 15 events, got %d", got)
		}
		// Must be reverse-chronological.
		for i := 1; i < len(events); i++ {
			if events[i].OccurredAt.After(events[i-1].OccurredAt) {
				t.Errorf("events not in reverse order at index %d", i)
			}
		}
	})

	t.Run("actor_email populated via JOIN", func(t *testing.T) {
		events, err := auditRepo.ListByProject(ctx, proj.ID, domain.AuditFilter{Limit: 1})
		if err != nil {
			t.Fatalf("ListByProject: %v", err)
		}
		if len(events) == 0 {
			t.Fatal("expected at least one event")
		}
		if got := events[0].ActorEmail; got != actor.Email {
			t.Errorf("actor_email: want %q, got %q", actor.Email, got)
		}
	})

	t.Run("environment_slug stored and retrieved", func(t *testing.T) {
		events, err := auditRepo.ListByProject(ctx, proj.ID, domain.AuditFilter{FlagKey: "flag-b", Limit: 10})
		if err != nil {
			t.Fatalf("ListByProject: %v", err)
		}
		for _, e := range events {
			if e.EnvironmentSlug != "staging" {
				t.Errorf("want environment_slug=staging, got %q", e.EnvironmentSlug)
			}
		}
	})

	t.Run("filter by flag_key — only flag-a events returned", func(t *testing.T) {
		events, err := auditRepo.ListByProject(ctx, proj.ID, domain.AuditFilter{FlagKey: "flag-a", Limit: 50})
		if err != nil {
			t.Fatalf("ListByProject: %v", err)
		}
		if got := len(events); got != 10 {
			t.Fatalf("want 10 flag-a events, got %d", got)
		}
		for _, e := range events {
			if e.EntityKey != "flag-a" {
				t.Errorf("unexpected entity_key %q in filtered result", e.EntityKey)
			}
		}
	})

	t.Run("cursor pagination — page through all 15 events in pages of 5", func(t *testing.T) {
		var all []*domain.AuditEvent
		var before time.Time
		for page := 0; page < 4; page++ {
			f := domain.AuditFilter{Limit: 5}
			if !before.IsZero() {
				f.Before = before
			}
			events, err := auditRepo.ListByProject(ctx, proj.ID, f)
			if err != nil {
				t.Fatalf("page %d ListByProject: %v", page, err)
			}
			all = append(all, events...)
			if len(events) < 5 {
				break
			}
			before = events[len(events)-1].OccurredAt
		}
		if got := len(all); got != 15 {
			t.Fatalf("pagination: want 15 total events, got %d", got)
		}
		// Verify no duplicates — each ID must be unique.
		seen := make(map[string]bool, len(all))
		for _, e := range all {
			if seen[e.ID] {
				t.Errorf("duplicate event ID %q across pages", e.ID)
			}
			seen[e.ID] = true
		}
	})

	t.Run("empty project returns empty slice", func(t *testing.T) {
		emptyProj := domain.Project{
			ID:        "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
			Name:      "Empty",
			Slug:      "empty",
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
		if err := projRepo.Create(ctx, emptyProj); err != nil {
			t.Fatalf("seed empty project: %v", err)
		}
		events, err := auditRepo.ListByProject(ctx, emptyProj.ID, domain.AuditFilter{Limit: 50})
		if err != nil {
			t.Fatalf("ListByProject empty: %v", err)
		}
		if len(events) != 0 {
			t.Errorf("want 0 events for empty project, got %d", len(events))
		}
	})
}

// padID pads n to a 12-char hex string for constructing UUIDs.
func padID(n int) string {
	s := "000000000000"
	ns := itoa12(n)
	return s[:len(s)-len(ns)] + ns
}

func itoa12(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := 12
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
