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

func (f *fakeEnvironmentRepository) UpdateName(_ context.Context, id, name string) error {
	e, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	e.Name = name
	return nil
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

func newEnvironmentService() (*app.EnvironmentService, *fakeProjectRepository) {
	projRepo := newFakeProjectRepository()
	envRepo := newFakeEnvironmentRepository()
	return app.NewEnvironmentService(envRepo, projRepo), projRepo
}

func TestEnvironmentService_Create_Succeeds(t *testing.T) {
	svc, projRepo := newEnvironmentService()
	ctx := authCtx("editor-1", domain.RoleEditor)

	proj := domain.Project{ID: "proj-1", Name: "Proj One", Slug: "proj-one"}
	projRepo.bySlug["proj-one"] = &proj
	projRepo.byID["proj-1"] = &proj

	e, err := svc.Create(ctx, "proj-one", "Staging", "staging")
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

func TestEnvironmentService_Create_NonExistentProject_ReturnsErrNotFound(t *testing.T) {
	svc, _ := newEnvironmentService()
	_, err := svc.Create(authCtx("editor-1", domain.RoleEditor), "ghost-proj", "Staging", "staging")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestEnvironmentService_Create_DuplicateSlug_ReturnsErrConflict(t *testing.T) {
	svc, projRepo := newEnvironmentService()
	ctx := authCtx("editor-1", domain.RoleEditor)

	proj := domain.Project{ID: "proj-1", Name: "Proj One", Slug: "proj-one"}
	projRepo.bySlug["proj-one"] = &proj
	projRepo.byID["proj-1"] = &proj

	if _, err := svc.Create(ctx, "proj-one", "Staging", "staging"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := svc.Create(ctx, "proj-one", "Staging Dup", "staging")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestEnvironmentService_Create_SameSlugDifferentProjects_Succeeds(t *testing.T) {
	svc, projRepo := newEnvironmentService()
	ctx := authCtx("editor-1", domain.RoleEditor)

	p1 := domain.Project{ID: "proj-1", Name: "Proj One", Slug: "proj-one"}
	p2 := domain.Project{ID: "proj-2", Name: "Proj Two", Slug: "proj-two"}
	projRepo.bySlug["proj-one"] = &p1
	projRepo.byID["proj-1"] = &p1
	projRepo.bySlug["proj-two"] = &p2
	projRepo.byID["proj-2"] = &p2

	if _, err := svc.Create(ctx, "proj-one", "Staging", "staging"); err != nil {
		t.Fatalf("create under proj-one: %v", err)
	}
	if _, err := svc.Create(ctx, "proj-two", "Staging", "staging"); err != nil {
		t.Errorf("expected success for same slug under different project, got %v", err)
	}
}

func TestEnvironmentService_GetBySlug_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc, _ := newEnvironmentService()
	_, err := svc.GetBySlug(context.Background(), "proj-1", "ghost")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestEnvironmentService_ListByProject_EmptyReturnsEmptySlice(t *testing.T) {
	svc, _ := newEnvironmentService()
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
	svc, _ := newEnvironmentService()
	err := svc.Delete(authCtx("editor-1", domain.RoleEditor), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestEnvironmentService_DeleteBySlug_Succeeds(t *testing.T) {
	svc, projRepo := newEnvironmentService()
	ctx := authCtx("editor-1", domain.RoleEditor)

	proj := domain.Project{ID: "proj-1", Name: "Proj One", Slug: "proj-one"}
	projRepo.bySlug["proj-one"] = &proj
	projRepo.byID["proj-1"] = &proj

	if _, err := svc.Create(ctx, "proj-one", "Staging", "staging"); err != nil {
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
	svc, _ := newEnvironmentService()
	err := svc.DeleteBySlug(authCtx("editor-1", domain.RoleEditor), "proj-1", "ghost")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── RBAC ──────────────────────────────────────────────────────────────────────

func TestEnvironmentService_Create_ViewerForbidden(t *testing.T) {
	svc, projRepo := newEnvironmentService()
	proj := domain.Project{ID: "proj-1", Name: "Proj One", Slug: "proj-one"}
	projRepo.bySlug["proj-one"] = &proj
	projRepo.byID["proj-1"] = &proj

	_, err := svc.Create(authCtx("viewer-1", domain.RoleViewer), "proj-one", "Staging", "staging")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestEnvironmentService_DeleteBySlug_ViewerForbidden(t *testing.T) {
	svc, _ := newEnvironmentService()
	err := svc.DeleteBySlug(authCtx("viewer-1", domain.RoleViewer), "proj-1", "staging")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

// ── UpdateName ────────────────────────────────────────────────────────────────

func TestEnvironmentService_UpdateName_Succeeds(t *testing.T) {
	svc, projRepo := newEnvironmentService()
	ctx := authCtx("admin-1", domain.RoleAdmin)

	proj := domain.Project{ID: "proj-1", Name: "Proj One", Slug: "proj-one"}
	projRepo.bySlug["proj-one"] = &proj
	projRepo.byID["proj-1"] = &proj

	if _, err := svc.Create(authCtx("editor-1", domain.RoleEditor), "proj-one", "Staging", "staging"); err != nil {
		t.Fatalf("create: %v", err)
	}

	e, err := svc.UpdateName(ctx, "proj-1", "staging", "Pre-Production")
	if err != nil {
		t.Fatalf("UpdateName: %v", err)
	}
	if e.Name != "Pre-Production" {
		t.Errorf("Name: got %q, want Pre-Production", e.Name)
	}
	if e.Slug != "staging" {
		t.Errorf("Slug should remain staging, got %q", e.Slug)
	}
}

func TestEnvironmentService_UpdateName_EmptyName_ReturnsValidationError(t *testing.T) {
	svc, _ := newEnvironmentService()
	ctx := authCtx("admin-1", domain.RoleAdmin)

	_, err := svc.UpdateName(ctx, "proj-1", "staging", "")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestEnvironmentService_UpdateName_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc, _ := newEnvironmentService()
	ctx := authCtx("admin-1", domain.RoleAdmin)

	_, err := svc.UpdateName(ctx, "proj-1", "ghost", "NewName")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestEnvironmentService_UpdateName_ViewerForbidden(t *testing.T) {
	svc, _ := newEnvironmentService()
	_, err := svc.UpdateName(authCtx("viewer-1", domain.RoleViewer), "proj-1", "staging", "X")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestEnvironmentService_UpdateName_EditorForbidden(t *testing.T) {
	svc, _ := newEnvironmentService()
	_, err := svc.UpdateName(authCtx("editor-1", domain.RoleEditor), "proj-1", "staging", "X")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestEnvironmentService_Create_NoAuthContextForbidden(t *testing.T) {
	svc, projRepo := newEnvironmentService()
	proj := domain.Project{ID: "proj-1", Name: "Proj One", Slug: "proj-one"}
	projRepo.bySlug["proj-one"] = &proj
	projRepo.byID["proj-1"] = &proj

	_, err := svc.Create(context.Background(), "proj-one", "Staging", "staging")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}
