package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/The127/cuttlegate/internal/app"
	"github.com/The127/cuttlegate/internal/domain"
)

// fakeProjectRepository is an in-memory implementation of ports.ProjectRepository.
type fakeProjectRepository struct {
	bySlug map[string]*domain.Project
	byID   map[string]*domain.Project
}

func newFakeProjectRepository() *fakeProjectRepository {
	return &fakeProjectRepository{
		bySlug: make(map[string]*domain.Project),
		byID:   make(map[string]*domain.Project),
	}
}

func (f *fakeProjectRepository) Create(_ context.Context, p domain.Project) error {
	if _, exists := f.bySlug[p.Slug]; exists {
		return domain.ErrConflict
	}
	cp := p
	f.bySlug[p.Slug] = &cp
	f.byID[p.ID] = &cp
	return nil
}

func (f *fakeProjectRepository) GetBySlug(_ context.Context, slug string) (*domain.Project, error) {
	p, ok := f.bySlug[slug]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (f *fakeProjectRepository) List(_ context.Context) ([]*domain.Project, error) {
	result := make([]*domain.Project, 0, len(f.bySlug))
	for _, p := range f.bySlug {
		cp := *p
		result = append(result, &cp)
	}
	return result, nil
}

func (f *fakeProjectRepository) UpdateName(_ context.Context, id, name string) error {
	p, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	p.Name = name
	return nil
}

func (f *fakeProjectRepository) Delete(_ context.Context, id string) error {
	p, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	delete(f.bySlug, p.Slug)
	delete(f.byID, id)
	return nil
}

func TestProjectService_Create_Succeeds(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	p, err := svc.Create(authCtx("editor-1", domain.RoleEditor), "Acme Corp", "acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID == "" {
		t.Error("expected a non-empty ID")
	}
	if p.Slug != "acme" {
		t.Errorf("slug: got %q, want %q", p.Slug, "acme")
	}
	if p.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestProjectService_Create_AddsCreatorAsAdminMember(t *testing.T) {
	memberRepo := newFakeProjectMemberRepository()
	svc := app.NewProjectService(newFakeProjectRepository(), memberRepo)
	ctx := authCtx("creator-1", domain.RoleEditor)
	p, err := svc.Create(ctx, "Acme Corp", "acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	members, err := memberRepo.ListMembers(ctx, p.ID)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
	if members[0].UserID != "creator-1" {
		t.Errorf("member user ID: got %q, want %q", members[0].UserID, "creator-1")
	}
	if members[0].Role != domain.RoleAdmin {
		t.Errorf("member role: got %q, want %q", members[0].Role, domain.RoleAdmin)
	}
}

func TestProjectService_Create_DuplicateSlug_ReturnsErrConflict(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	ctx := authCtx("editor-1", domain.RoleEditor)
	if _, err := svc.Create(ctx, "Acme Corp", "acme"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := svc.Create(ctx, "Acme Duplicate", "acme")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestProjectService_GetBySlug_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	_, err := svc.GetBySlug(context.Background(), "ghost")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestProjectService_List_EmptyReturnsEmptySlice(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	list, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(list) != 0 {
		t.Errorf("expected 0 projects, got %d", len(list))
	}
}

func TestProjectService_Delete_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	err := svc.Delete(authCtx("admin-1", domain.RoleAdmin), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestProjectService_UpdateName_Succeeds(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	ctx := authCtx("admin-1", domain.RoleAdmin)
	p, err := svc.Create(ctx, "Acme", "acme")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	updated, err := svc.UpdateName(ctx, "acme", "Acme Corp")
	if err != nil {
		t.Fatalf("UpdateName: %v", err)
	}
	if updated.Name != "Acme Corp" {
		t.Errorf("name: got %q, want %q", updated.Name, "Acme Corp")
	}
	if updated.ID != p.ID {
		t.Errorf("ID changed unexpectedly")
	}
	if updated.Slug != "acme" {
		t.Errorf("slug changed unexpectedly: got %q", updated.Slug)
	}
}

func TestProjectService_UpdateName_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	_, err := svc.UpdateName(authCtx("admin-1", domain.RoleAdmin), "ghost", "Whatever")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestProjectService_DeleteBySlug_Succeeds(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	editorCtx := authCtx("editor-1", domain.RoleEditor)
	adminCtx := authCtx("admin-1", domain.RoleAdmin)
	if _, err := svc.Create(editorCtx, "Acme", "acme"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.DeleteBySlug(adminCtx, "acme"); err != nil {
		t.Fatalf("DeleteBySlug: %v", err)
	}
	_, err := svc.GetBySlug(context.Background(), "acme")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestProjectService_DeleteBySlug_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	err := svc.DeleteBySlug(authCtx("admin-1", domain.RoleAdmin), "ghost")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── RBAC ──────────────────────────────────────────────────────────────────────

func TestProjectService_Create_ViewerForbidden(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	_, err := svc.Create(authCtx("viewer-1", domain.RoleViewer), "Acme", "acme")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestProjectService_UpdateName_ViewerForbidden(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	_, err := svc.UpdateName(authCtx("viewer-1", domain.RoleViewer), "acme", "New Name")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestProjectService_DeleteBySlug_ViewerForbidden(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	err := svc.DeleteBySlug(authCtx("viewer-1", domain.RoleViewer), "acme")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestProjectService_Create_NoAuthContextForbidden(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	_, err := svc.Create(context.Background(), "Acme", "acme")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

// @auth-bypass — editor is blocked on rename
func TestProjectService_UpdateName_EditorForbidden(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	_, err := svc.UpdateName(authCtx("editor-1", domain.RoleEditor), "acme", "New Name")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

// @auth-bypass — editor is blocked on delete
func TestProjectService_DeleteBySlug_EditorForbidden(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	err := svc.DeleteBySlug(authCtx("editor-1", domain.RoleEditor), "acme")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

// @happy — admin can rename
func TestProjectService_UpdateName_AdminSucceeds(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	editorCtx := authCtx("editor-1", domain.RoleEditor)
	adminCtx := authCtx("admin-1", domain.RoleAdmin)
	if _, err := svc.Create(editorCtx, "Acme", "acme"); err != nil {
		t.Fatalf("create: %v", err)
	}
	updated, err := svc.UpdateName(adminCtx, "acme", "Acme Corp")
	if err != nil {
		t.Fatalf("UpdateName: %v", err)
	}
	if updated.Name != "Acme Corp" {
		t.Errorf("name: got %q, want %q", updated.Name, "Acme Corp")
	}
}

// @happy — admin can delete
func TestProjectService_DeleteBySlug_AdminSucceeds(t *testing.T) {
	svc := app.NewProjectService(newFakeProjectRepository(), newFakeProjectMemberRepository())
	editorCtx := authCtx("editor-1", domain.RoleEditor)
	adminCtx := authCtx("admin-1", domain.RoleAdmin)
	if _, err := svc.Create(editorCtx, "Acme", "acme"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.DeleteBySlug(adminCtx, "acme"); err != nil {
		t.Fatalf("DeleteBySlug: %v", err)
	}
	_, err := svc.GetBySlug(context.Background(), "acme")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
