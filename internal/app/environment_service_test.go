package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

// fakeEnvironmentRepository is an in-memory implementation of ports.EnvironmentRepository.
type fakeEnvironmentRepository struct {
	byKey map[string]*domain.Environment // key: projectID+"/"+slug
	byID  map[string]*domain.Environment
}

func newFakeEnvironmentRepository() *fakeEnvironmentRepository {
	return &fakeEnvironmentRepository{
		byKey: make(map[string]*domain.Environment),
		byID:  make(map[string]*domain.Environment),
	}
}

func envKey(projectID, slug string) string { return projectID + "/" + slug }

func (f *fakeEnvironmentRepository) Create(_ context.Context, e domain.Environment) error {
	k := envKey(e.ProjectID, e.Slug)
	if _, exists := f.byKey[k]; exists {
		return domain.ErrConflict
	}
	cp := e
	f.byKey[k] = &cp
	f.byID[e.ID] = &cp
	return nil
}

func (f *fakeEnvironmentRepository) GetBySlug(_ context.Context, projectID, slug string) (*domain.Environment, error) {
	e, ok := f.byKey[envKey(projectID, slug)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *e
	return &cp, nil
}

func (f *fakeEnvironmentRepository) ListByProject(_ context.Context, projectID string) ([]*domain.Environment, error) {
	result := make([]*domain.Environment, 0)
	for _, e := range f.byKey {
		if e.ProjectID == projectID {
			cp := *e
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (f *fakeEnvironmentRepository) Delete(_ context.Context, id string) error {
	e, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	delete(f.byKey, envKey(e.ProjectID, e.Slug))
	delete(f.byID, id)
	return nil
}

func newEnvironmentService() *app.EnvironmentService {
	return app.NewEnvironmentService(newFakeEnvironmentRepository())
}

func TestEnvironmentService_Create_Succeeds(t *testing.T) {
	svc := newEnvironmentService()
	ctx := authCtx("editor-1", domain.RoleEditor)

	e, err := svc.Create(ctx, "proj-1", "Staging", "staging")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.ID == "" {
		t.Error("expected non-empty ID")
	}
	if e.ProjectID != "proj-1" {
		t.Errorf("ProjectID: got %q, want %q", e.ProjectID, "proj-1")
	}
	if e.Slug != "staging" {
		t.Errorf("Slug: got %q, want %q", e.Slug, "staging")
	}
	if e.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestEnvironmentService_Create_DuplicateSlug_ReturnsErrConflict(t *testing.T) {
	svc := newEnvironmentService()
	ctx := authCtx("editor-1", domain.RoleEditor)

	if _, err := svc.Create(ctx, "proj-1", "Staging", "staging"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := svc.Create(ctx, "proj-1", "Staging Dup", "staging")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestEnvironmentService_Create_SameSlugDifferentProjects_Succeeds(t *testing.T) {
	svc := newEnvironmentService()
	ctx := authCtx("editor-1", domain.RoleEditor)

	if _, err := svc.Create(ctx, "proj-1", "Staging", "staging"); err != nil {
		t.Fatalf("create under proj-1: %v", err)
	}
	if _, err := svc.Create(ctx, "proj-2", "Staging", "staging"); err != nil {
		t.Errorf("expected success for same slug under different project, got %v", err)
	}
}

func TestEnvironmentService_GetBySlug_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := newEnvironmentService()
	_, err := svc.GetBySlug(context.Background(), "proj-1", "ghost")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestEnvironmentService_ListByProject_EmptyReturnsEmptySlice(t *testing.T) {
	svc := newEnvironmentService()
	list, err := svc.ListByProject(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(list) != 0 {
		t.Errorf("expected 0 environments, got %d", len(list))
	}
}

func TestEnvironmentService_Delete_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := newEnvironmentService()
	err := svc.Delete(authCtx("editor-1", domain.RoleEditor), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestEnvironmentService_DeleteBySlug_Succeeds(t *testing.T) {
	svc := newEnvironmentService()
	ctx := authCtx("editor-1", domain.RoleEditor)

	if _, err := svc.Create(ctx, "proj-1", "Staging", "staging"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.DeleteBySlug(ctx, "proj-1", "staging"); err != nil {
		t.Fatalf("DeleteBySlug: %v", err)
	}
	_, err := svc.GetBySlug(ctx, "proj-1", "staging")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestEnvironmentService_DeleteBySlug_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := newEnvironmentService()
	err := svc.DeleteBySlug(authCtx("editor-1", domain.RoleEditor), "proj-1", "ghost")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── RBAC ──────────────────────────────────────────────────────────────────────

func TestEnvironmentService_Create_ViewerForbidden(t *testing.T) {
	svc := newEnvironmentService()
	_, err := svc.Create(authCtx("viewer-1", domain.RoleViewer), "proj-1", "Staging", "staging")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestEnvironmentService_DeleteBySlug_ViewerForbidden(t *testing.T) {
	svc := newEnvironmentService()
	err := svc.DeleteBySlug(authCtx("viewer-1", domain.RoleViewer), "proj-1", "staging")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestEnvironmentService_Create_NoAuthContextForbidden(t *testing.T) {
	svc := newEnvironmentService()
	_, err := svc.Create(context.Background(), "proj-1", "Staging", "staging")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}
