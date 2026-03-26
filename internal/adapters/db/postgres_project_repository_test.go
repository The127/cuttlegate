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

func TestPostgresProjectRepository_CreateGetListDelete(t *testing.T) {
	db := newTestDB(t)
	repo := dbadapter.NewPostgresProjectRepository(db)
	ctx := context.Background()

	p := domain.Project{
		ID:        "11111111-1111-4111-8111-111111111111",
		Name:      "Test Acme",
		Slug:      "test-acme",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}

	// Create
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// GetBySlug
	got, err := repo.GetBySlug(ctx, "test-acme")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if got.ID != p.ID || got.Name != p.Name || got.Slug != p.Slug {
		t.Errorf("project mismatch: got %+v, want %+v", got, p)
	}

	// List
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list == nil {
		t.Error("List returned nil, expected a slice")
	}
	if len(list) == 0 {
		t.Error("expected at least one project in list")
	}

	// Duplicate slug → ErrConflict
	err = repo.Create(ctx, domain.Project{
		ID:        "22222222-2222-4222-8222-222222222222",
		Name:      "Duplicate",
		Slug:      "test-acme",
		CreatedAt: time.Now().UTC(),
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}

	// Delete
	if err := repo.Delete(ctx, p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// GetBySlug after delete → ErrNotFound
	_, err = repo.GetBySlug(ctx, "test-acme")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}

	// Delete non-existent → ErrNotFound
	err = repo.Delete(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound for missing ID, got %v", err)
	}
}

func TestPostgresProjectRepository_List_NeverNil(t *testing.T) {
	db := newTestDB(t)
	repo := dbadapter.NewPostgresProjectRepository(db)

	list, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list == nil {
		t.Error("List returned nil, expected a slice")
	}
}

func TestPostgresProjectRepository_GetBySlug_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := dbadapter.NewPostgresProjectRepository(db)

	_, err := repo.GetBySlug(context.Background(), "definitely-does-not-exist-xyz")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
