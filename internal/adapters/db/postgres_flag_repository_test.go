//go:build integration

package dbadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
	"github.com/karo/cuttlegate/internal/domain"
)

var boolVariants = []domain.Variant{
	{Key: "true", Name: "On"},
	{Key: "false", Name: "Off"},
}

func seedFlagProject(t *testing.T, ctx context.Context, db interface {
	Create(context.Context, domain.Project) error
}, id, slug string) {
	t.Helper()
	p := domain.Project{
		ID:        id,
		Name:      slug,
		Slug:      slug,
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	if err := db.Create(ctx, p); err != nil {
		t.Fatalf("seed project %s: %v", slug, err)
	}
}

func TestPostgresFlagRepository(t *testing.T) {
	db := openTestDB(t)
	projRepo := dbadapter.NewPostgresProjectRepository(db)
	flagRepo := dbadapter.NewPostgresFlagRepository(db)
	ctx := context.Background()

	const projID = "cccccccc-1111-4111-8111-111111111111"
	const projID2 = "cccccccc-2222-4222-8222-222222222222"

	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM flags WHERE project_id IN ($1, $2)`, projID, projID2)
		db.ExecContext(ctx, `DELETE FROM projects WHERE id IN ($1, $2)`, projID, projID2)
	})

	seedFlagProject(t, ctx, projRepo, projID, "flag-test-proj-1")
	seedFlagProject(t, ctx, projRepo, projID2, "flag-test-proj-2")

	t.Run("Create and GetByKey roundtrip — variants survive JSON marshalling", func(t *testing.T) {
		f := &domain.Flag{
			ID:                "flag-aaaa-0001",
			ProjectID:         projID,
			Key:               "dark-mode",
			Name:              "Dark Mode",
			Type:              domain.FlagTypeBool,
			Variants:          boolVariants,
			DefaultVariantKey: "false",
			CreatedAt:         time.Now().UTC().Truncate(time.Microsecond),
		}
		if err := flagRepo.Create(ctx, f); err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := flagRepo.GetByKey(ctx, projID, "dark-mode")
		if err != nil {
			t.Fatalf("GetByKey: %v", err)
		}
		if got.ID != f.ID || got.Key != f.Key || got.Name != f.Name {
			t.Errorf("flag mismatch: got %+v", got)
		}
		if len(got.Variants) != 2 || got.Variants[0].Key != "true" || got.Variants[1].Key != "false" {
			t.Errorf("variants mismatch: %+v", got.Variants)
		}
		if got.DefaultVariantKey != "false" {
			t.Errorf("default_variant_key: got %q", got.DefaultVariantKey)
		}
	})

	t.Run("Duplicate (project_id, key) returns ErrConflict", func(t *testing.T) {
		err := flagRepo.Create(ctx, &domain.Flag{
			ID:                "flag-aaaa-dup1",
			ProjectID:         projID,
			Key:               "dark-mode",
			Name:              "Dup",
			Type:              domain.FlagTypeBool,
			Variants:          boolVariants,
			DefaultVariantKey: "false",
			CreatedAt:         time.Now().UTC(),
		})
		if !errors.Is(err, domain.ErrConflict) {
			t.Errorf("expected ErrConflict, got %v", err)
		}
	})

	t.Run("Update mutates name/variants/default but leaves key and type unchanged", func(t *testing.T) {
		updated := &domain.Flag{
			ID:                "flag-aaaa-0001",
			ProjectID:         projID,
			Key:               "should-not-change",
			Name:              "Dark Mode v2",
			Type:              domain.FlagTypeString,
			Variants:          []domain.Variant{{Key: "true", Name: "Enabled"}, {Key: "false", Name: "Disabled"}},
			DefaultVariantKey: "true",
		}
		if err := flagRepo.Update(ctx, updated); err != nil {
			t.Fatalf("Update: %v", err)
		}
		got, err := flagRepo.GetByKey(ctx, projID, "dark-mode")
		if err != nil {
			t.Fatalf("GetByKey after update: %v", err)
		}
		if got.Name != "Dark Mode v2" {
			t.Errorf("name not updated: %q", got.Name)
		}
		if got.Key != "dark-mode" {
			t.Errorf("key changed unexpectedly: %q", got.Key)
		}
		if got.Type != domain.FlagTypeBool {
			t.Errorf("type changed unexpectedly: %q", got.Type)
		}
		if got.DefaultVariantKey != "true" {
			t.Errorf("default_variant_key: got %q", got.DefaultVariantKey)
		}
	})

	t.Run("GetByKey absent returns ErrNotFound", func(t *testing.T) {
		_, err := flagRepo.GetByKey(ctx, projID, "ghost-flag")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("ListByProject returns flags for correct project only", func(t *testing.T) {
		// Insert a flag under proj2
		if err := flagRepo.Create(ctx, &domain.Flag{
			ID:                "flag-bbbb-0001",
			ProjectID:         projID2,
			Key:               "beta-feature",
			Name:              "Beta",
			Type:              domain.FlagTypeBool,
			Variants:          boolVariants,
			DefaultVariantKey: "false",
			CreatedAt:         time.Now().UTC(),
		}); err != nil {
			t.Fatalf("Create for proj2: %v", err)
		}

		list1, err := flagRepo.ListByProject(ctx, projID)
		if err != nil {
			t.Fatalf("ListByProject proj1: %v", err)
		}
		for _, f := range list1 {
			if f.ProjectID != projID {
				t.Errorf("got flag from wrong project: %s", f.ProjectID)
			}
		}

		list2, err := flagRepo.ListByProject(ctx, projID2)
		if err != nil {
			t.Fatalf("ListByProject proj2: %v", err)
		}
		if len(list2) == 0 {
			t.Error("expected at least one flag for proj2")
		}
	})

	t.Run("ListByProject empty returns empty slice never nil", func(t *testing.T) {
		list, err := flagRepo.ListByProject(ctx, "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("ListByProject: %v", err)
		}
		if list == nil {
			t.Error("expected empty slice, got nil")
		}
	})

	t.Run("Delete then GetByKey returns ErrNotFound", func(t *testing.T) {
		f := &domain.Flag{
			ID:                "flag-aaaa-del1",
			ProjectID:         projID,
			Key:               "to-delete",
			Name:              "To Delete",
			Type:              domain.FlagTypeBool,
			Variants:          boolVariants,
			DefaultVariantKey: "false",
			CreatedAt:         time.Now().UTC(),
		}
		if err := flagRepo.Create(ctx, f); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := flagRepo.Delete(ctx, f.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		_, err := flagRepo.GetByKey(ctx, projID, "to-delete")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound after delete, got %v", err)
		}
	})

	t.Run("Delete absent ID returns ErrNotFound", func(t *testing.T) {
		err := flagRepo.Delete(ctx, "00000000-0000-0000-0000-000000000000")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}
