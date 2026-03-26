//go:build integration

package dbadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	dbadapter "github.com/The127/cuttlegate/internal/adapters/db"
	"github.com/The127/cuttlegate/internal/domain"
)

func TestPostgresEnvironmentRepository(t *testing.T) {
	db := newTestDB(t)
	projRepo := dbadapter.NewPostgresProjectRepository(db)
	envRepo := dbadapter.NewPostgresEnvironmentRepository(db)
	ctx := context.Background()

	// Seed two projects
	proj1 := domain.Project{
		ID:        "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		Name:      "Env Test Project 1",
		Slug:      "env-test-proj-1",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	proj2 := domain.Project{
		ID:        "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
		Name:      "Env Test Project 2",
		Slug:      "env-test-proj-2",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}

	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM environments WHERE project_id IN ($1, $2)`, proj1.ID, proj2.ID)
		db.ExecContext(ctx, `DELETE FROM projects WHERE id IN ($1, $2)`, proj1.ID, proj2.ID)
	})

	if err := projRepo.Create(ctx, proj1); err != nil {
		t.Fatalf("seed proj1: %v", err)
	}
	if err := projRepo.Create(ctx, proj2); err != nil {
		t.Fatalf("seed proj2: %v", err)
	}

	t.Run("Create and GetBySlug", func(t *testing.T) {
		env := domain.Environment{
			ID:        "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
			ProjectID: proj1.ID,
			Name:      "Staging",
			Slug:      "staging",
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
		if err := envRepo.Create(ctx, env); err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := envRepo.GetBySlug(ctx, proj1.ID, "staging")
		if err != nil {
			t.Fatalf("GetBySlug: %v", err)
		}
		if got.ID != env.ID || got.ProjectID != env.ProjectID || got.Slug != env.Slug {
			t.Errorf("environment mismatch: got %+v, want %+v", got, env)
		}
	})

	t.Run("Duplicate slug within project returns ErrConflict", func(t *testing.T) {
		err := envRepo.Create(ctx, domain.Environment{
			ID:        "dddddddd-dddd-4ddd-8ddd-dddddddddddd",
			ProjectID: proj1.ID,
			Name:      "Staging Dup",
			Slug:      "staging",
			CreatedAt: time.Now().UTC(),
		})
		if !errors.Is(err, domain.ErrConflict) {
			t.Errorf("expected ErrConflict, got %v", err)
		}
	})

	t.Run("Same slug under different project succeeds", func(t *testing.T) {
		err := envRepo.Create(ctx, domain.Environment{
			ID:        "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee",
			ProjectID: proj2.ID,
			Name:      "Staging",
			Slug:      "staging",
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			t.Errorf("expected success, got %v", err)
		}
	})

	t.Run("GetBySlug absent returns ErrNotFound", func(t *testing.T) {
		_, err := envRepo.GetBySlug(ctx, proj1.ID, "ghost")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("ListByProject returns environments ordered by created_at ASC", func(t *testing.T) {
		// Use proj2 which has only "staging" so far; add two more with explicit timestamps
		t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
		t3 := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

		// Clear proj2 environments for a clean ordering test
		db.ExecContext(ctx, `DELETE FROM environments WHERE project_id = $1`, proj2.ID)

		for _, e := range []domain.Environment{
			{ID: "ff000000-0000-4000-8000-000000000001", ProjectID: proj2.ID, Name: "A", Slug: "env-a", CreatedAt: t1},
			{ID: "ff000000-0000-4000-8000-000000000002", ProjectID: proj2.ID, Name: "B", Slug: "env-b", CreatedAt: t2},
			{ID: "ff000000-0000-4000-8000-000000000003", ProjectID: proj2.ID, Name: "C", Slug: "env-c", CreatedAt: t3},
		} {
			if err := envRepo.Create(ctx, e); err != nil {
				t.Fatalf("seed env %s: %v", e.Slug, err)
			}
		}

		list, err := envRepo.ListByProject(ctx, proj2.ID)
		if err != nil {
			t.Fatalf("ListByProject: %v", err)
		}
		if len(list) != 3 {
			t.Fatalf("expected 3 environments, got %d", len(list))
		}
		if list[0].Slug != "env-a" || list[1].Slug != "env-b" || list[2].Slug != "env-c" {
			t.Errorf("wrong order: got %v, %v, %v", list[0].Slug, list[1].Slug, list[2].Slug)
		}
	})

	t.Run("ListByProject returns empty slice never nil", func(t *testing.T) {
		list, err := envRepo.ListByProject(ctx, "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("ListByProject: %v", err)
		}
		if list == nil {
			t.Error("expected empty slice, got nil")
		}
	})

	t.Run("Delete and confirm gone", func(t *testing.T) {
		env := domain.Environment{
			ID:        "11111111-2222-4333-8444-555555555555",
			ProjectID: proj1.ID,
			Name:      "To Delete",
			Slug:      "to-delete",
			CreatedAt: time.Now().UTC(),
		}
		if err := envRepo.Create(ctx, env); err != nil {
			t.Fatalf("Create before delete: %v", err)
		}
		if err := envRepo.Delete(ctx, env.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		_, err := envRepo.GetBySlug(ctx, proj1.ID, "to-delete")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound after delete, got %v", err)
		}
	})

	t.Run("Delete absent ID returns ErrNotFound", func(t *testing.T) {
		err := envRepo.Delete(ctx, "00000000-0000-0000-0000-000000000000")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}
