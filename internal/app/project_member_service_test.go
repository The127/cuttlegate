package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

// fakeProjectMemberRepository is an in-memory implementation of ports.ProjectMemberRepository.
type fakeProjectMemberRepository struct {
	members map[string][]*domain.ProjectMember // keyed by projectID
}

func newFakeProjectMemberRepository() *fakeProjectMemberRepository {
	return &fakeProjectMemberRepository{
		members: make(map[string][]*domain.ProjectMember),
	}
}

func (f *fakeProjectMemberRepository) AddMember(_ context.Context, m *domain.ProjectMember) error {
	for _, existing := range f.members[m.ProjectID] {
		if existing.UserID == m.UserID {
			return domain.ErrConflict
		}
	}
	cp := *m
	f.members[m.ProjectID] = append(f.members[m.ProjectID], &cp)
	return nil
}

func (f *fakeProjectMemberRepository) ListMembers(_ context.Context, projectID string) ([]*domain.ProjectMember, error) {
	result := make([]*domain.ProjectMember, 0, len(f.members[projectID]))
	for _, m := range f.members[projectID] {
		cp := *m
		result = append(result, &cp)
	}
	return result, nil
}

func (f *fakeProjectMemberRepository) UpdateRole(_ context.Context, projectID, userID string, role domain.Role) error {
	for _, m := range f.members[projectID] {
		if m.UserID == userID {
			m.Role = role
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeProjectMemberRepository) RemoveMember(_ context.Context, projectID, userID string) error {
	list := f.members[projectID]
	for i, m := range list {
		if m.UserID == userID {
			f.members[projectID] = append(list[:i], list[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

// newMemberSvc is a convenience constructor for tests.
func newMemberSvc() (*app.ProjectMemberService, *fakeProjectRepository, *fakeProjectMemberRepository) {
	projRepo := newFakeProjectRepository()
	memberRepo := newFakeProjectMemberRepository()
	svc := app.NewProjectMemberService(memberRepo, projRepo)
	return svc, projRepo, memberRepo
}

// seedProject seeds a project and returns its ID.
func seedProject(projRepo *fakeProjectRepository, slug string) string {
	p := domain.Project{ID: "proj-" + slug, Name: slug, Slug: slug}
	projRepo.bySlug[slug] = &p
	projRepo.byID[p.ID] = &p
	return p.ID
}

// authCtx returns a context with the given role for the given user.
func authCtx(userID string, role domain.Role) context.Context {
	return domain.NewAuthContext(context.Background(), domain.AuthContext{UserID: userID, Role: role})
}

// ── Scenario 1: viewer can list ───────────────────────────────────────────────

func TestProjectMemberService_ListMembers_ViewerCanList(t *testing.T) {
	svc, projRepo, _ := newMemberSvc()
	seedProject(projRepo, "acme")

	members, err := svc.ListMembers(authCtx("viewer-1", domain.RoleViewer), "acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if members == nil {
		t.Error("expected non-nil slice")
	}
}

// ── Scenario 2: editor cannot add ────────────────────────────────────────────

func TestProjectMemberService_AddMember_EditorForbidden(t *testing.T) {
	svc, projRepo, _ := newMemberSvc()
	seedProject(projRepo, "acme")

	_, err := svc.AddMember(authCtx("editor-1", domain.RoleEditor), "acme", "new-user", domain.RoleViewer)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

// ── Scenario 3: admin can add ─────────────────────────────────────────────────

func TestProjectMemberService_AddMember_AdminSucceeds(t *testing.T) {
	svc, projRepo, _ := newMemberSvc()
	seedProject(projRepo, "acme")

	m, err := svc.AddMember(authCtx("admin-1", domain.RoleAdmin), "acme", "new-user", domain.RoleViewer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.UserID != "new-user" {
		t.Errorf("UserID: got %q, want %q", m.UserID, "new-user")
	}
	if m.Role != domain.RoleViewer {
		t.Errorf("Role: got %q, want %q", m.Role, domain.RoleViewer)
	}
}

// ── Scenario 4: duplicate member returns ErrConflict ─────────────────────────

func TestProjectMemberService_AddMember_DuplicateReturnsErrConflict(t *testing.T) {
	svc, projRepo, _ := newMemberSvc()
	seedProject(projRepo, "acme")
	ctx := authCtx("admin-1", domain.RoleAdmin)

	if _, err := svc.AddMember(ctx, "acme", "user-123", domain.RoleViewer); err != nil {
		t.Fatalf("first add: %v", err)
	}
	_, err := svc.AddMember(ctx, "acme", "user-123", domain.RoleViewer)
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

// ── Scenario 5: admin can change role ────────────────────────────────────────

func TestProjectMemberService_UpdateRole_AdminSucceeds(t *testing.T) {
	svc, projRepo, memberRepo := newMemberSvc()
	projID := seedProject(projRepo, "acme")
	memberRepo.members[projID] = []*domain.ProjectMember{
		{ProjectID: projID, UserID: "user-123", Role: domain.RoleViewer},
	}

	err := svc.UpdateRole(authCtx("admin-1", domain.RoleAdmin), "acme", "user-123", domain.RoleEditor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if memberRepo.members[projID][0].Role != domain.RoleEditor {
		t.Errorf("role not updated: got %q", memberRepo.members[projID][0].Role)
	}
}

// ── Scenario 6: editor cannot change roles ────────────────────────────────────

func TestProjectMemberService_UpdateRole_EditorForbidden(t *testing.T) {
	svc, projRepo, _ := newMemberSvc()
	seedProject(projRepo, "acme")

	err := svc.UpdateRole(authCtx("editor-1", domain.RoleEditor), "acme", "user-123", domain.RoleViewer)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

// ── Scenario 7: removing last admin returns ErrLastAdmin ─────────────────────

func TestProjectMemberService_RemoveMember_LastAdminReturnsErrLastAdmin(t *testing.T) {
	svc, projRepo, memberRepo := newMemberSvc()
	projID := seedProject(projRepo, "acme")
	memberRepo.members[projID] = []*domain.ProjectMember{
		{ProjectID: projID, UserID: "admin-1", Role: domain.RoleAdmin},
	}

	err := svc.RemoveMember(authCtx("admin-1", domain.RoleAdmin), "acme", "admin-1")
	if !errors.Is(err, domain.ErrLastAdmin) {
		t.Errorf("expected ErrLastAdmin, got %v", err)
	}
	// member must not have been removed
	if len(memberRepo.members[projID]) != 1 {
		t.Error("member was removed despite ErrLastAdmin")
	}
}

// ── Scenario 8: removing non-admin member succeeds ───────────────────────────

func TestProjectMemberService_RemoveMember_NonAdminSucceeds(t *testing.T) {
	svc, projRepo, memberRepo := newMemberSvc()
	projID := seedProject(projRepo, "acme")
	memberRepo.members[projID] = []*domain.ProjectMember{
		{ProjectID: projID, UserID: "admin-1", Role: domain.RoleAdmin},
		{ProjectID: projID, UserID: "user-456", Role: domain.RoleViewer},
	}

	err := svc.RemoveMember(authCtx("admin-1", domain.RoleAdmin), "acme", "user-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(memberRepo.members[projID]) != 1 {
		t.Errorf("expected 1 member remaining, got %d", len(memberRepo.members[projID]))
	}
}

// ── Scenario 9: removing non-member returns ErrNotFound ──────────────────────

func TestProjectMemberService_RemoveMember_NotMemberReturnsErrNotFound(t *testing.T) {
	svc, projRepo, memberRepo := newMemberSvc()
	projID := seedProject(projRepo, "acme")
	// seed one admin so the project isn't empty
	memberRepo.members[projID] = []*domain.ProjectMember{
		{ProjectID: projID, UserID: "admin-1", Role: domain.RoleAdmin},
	}

	err := svc.RemoveMember(authCtx("admin-1", domain.RoleAdmin), "acme", "ghost")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── Scenario 10: any OIDC sub value can be added ─────────────────────────────

func TestProjectMemberService_AddMember_ArbitraryOIDCSubSucceeds(t *testing.T) {
	svc, projRepo, _ := newMemberSvc()
	seedProject(projRepo, "acme")

	_, err := svc.AddMember(authCtx("admin-1", domain.RoleAdmin), "acme", "arbitrary-oidc-sub-value", domain.RoleViewer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
