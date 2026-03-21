package app_test

import (
	"context"
	"errors"
	"testing"

	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

func newSegmentSvc() *app.SegmentService {
	return app.NewSegmentService(dbadapter.NewFakeSegmentRepository())
}

// ── Create scenarios ──────────────────────────────────────────────────────────

func TestSegmentService_Create_Succeeds(t *testing.T) {
	svc := newSegmentSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	seg, err := svc.Create(ctx, "proj-1", "beta-users", "Beta Users")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if seg.ID == "" {
		t.Error("expected non-empty ID")
	}
	if seg.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if seg.Slug != "beta-users" || seg.Name != "Beta Users" {
		t.Errorf("unexpected segment: %+v", seg)
	}
}

func TestSegmentService_Create_InvalidSlug_ReturnsError(t *testing.T) {
	svc := newSegmentSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	_, err := svc.Create(ctx, "proj-1", "INVALID SLUG", "Beta")
	if err == nil {
		t.Error("expected validation error for invalid slug, got nil")
	}
}

func TestSegmentService_Create_EmptyName_ReturnsError(t *testing.T) {
	svc := newSegmentSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	_, err := svc.Create(ctx, "proj-1", "beta", "")
	if err == nil {
		t.Error("expected validation error for empty name, got nil")
	}
}

func TestSegmentService_Create_ViewerReturnsForbidden(t *testing.T) {
	svc := newSegmentSvc()
	_, err := svc.Create(authCtx("viewer-1", domain.RoleViewer), "proj-1", "beta", "Beta")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

// ── List / GetBySlug scenarios ────────────────────────────────────────────────

func TestSegmentService_List_Empty_DoesNotError(t *testing.T) {
	svc := newSegmentSvc()
	_, err := svc.List(authCtx("viewer-1", domain.RoleViewer), "proj-1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestSegmentService_Create_DuplicateSlug_ReturnsConflict(t *testing.T) {
	svc := newSegmentSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	_, err := svc.Create(ctx, "proj-1", "beta", "Beta Users")
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err = svc.Create(ctx, "proj-1", "beta", "Beta Users Again")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict for duplicate slug, got %v", err)
	}
}

func TestSegmentService_GetBySlug_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := newSegmentSvc()
	_, err := svc.GetBySlug(authCtx("viewer-1", domain.RoleViewer), "proj-1", "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSegmentService_GetBySlug_ReturnsSegment(t *testing.T) {
	svc := newSegmentSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	created, err := svc.Create(ctx, "proj-1", "beta", "Beta Users")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	found, err := svc.GetBySlug(ctx, "proj-1", "beta")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, found.ID)
	}
}

// ── UpdateName scenarios ──────────────────────────────────────────────────────

func TestSegmentService_UpdateName_Succeeds(t *testing.T) {
	svc := newSegmentSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	seg, _ := svc.Create(ctx, "proj-1", "beta", "Beta")
	if err := svc.UpdateName(ctx, seg.ID, "Beta Users"); err != nil {
		t.Fatalf("UpdateName: %v", err)
	}
}

func TestSegmentService_UpdateName_EmptyName_ReturnsError(t *testing.T) {
	svc := newSegmentSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	if err := svc.UpdateName(ctx, "any-id", ""); err == nil {
		t.Error("expected error for empty name, got nil")
	}
}

// ── Delete scenarios ──────────────────────────────────────────────────────────

func TestSegmentService_Delete_Succeeds(t *testing.T) {
	svc := newSegmentSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	_, err := svc.Create(ctx, "proj-1", "beta", "Beta")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Delete(ctx, "proj-1", "beta"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = svc.GetBySlug(ctx, "proj-1", "beta")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestSegmentService_Delete_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := newSegmentSvc()
	err := svc.Delete(authCtx("editor-1", domain.RoleEditor), "proj-1", "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── Members scenarios ─────────────────────────────────────────────────────────

func TestSegmentService_SetAndListMembers(t *testing.T) {
	svc := newSegmentSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	seg, _ := svc.Create(ctx, "proj-1", "beta", "Beta")

	if err := svc.SetMembers(ctx, seg.ID, []string{"alice", "bob"}); err != nil {
		t.Fatalf("SetMembers: %v", err)
	}

	members, err := svc.ListMembers(ctx, seg.ID)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}
}

func TestSegmentService_SetMembers_ClearsExisting(t *testing.T) {
	svc := newSegmentSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	seg, _ := svc.Create(ctx, "proj-1", "beta", "Beta")

	_ = svc.SetMembers(ctx, seg.ID, []string{"alice", "bob"})
	_ = svc.SetMembers(ctx, seg.ID, []string{"carol"})

	members, _ := svc.ListMembers(ctx, seg.ID)
	if len(members) != 1 || members[0] != "carol" {
		t.Errorf("expected [carol], got %v", members)
	}
}

func TestSegmentService_SetMembers_ViewerReturnsForbidden(t *testing.T) {
	svc := newSegmentSvc()
	err := svc.SetMembers(authCtx("viewer-1", domain.RoleViewer), "seg-1", []string{"alice"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestSegmentService_List_ViewerNoAuth_ReturnsForbidden(t *testing.T) {
	svc := newSegmentSvc()
	_, err := svc.List(context.Background(), "proj-1")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden for missing auth, got %v", err)
	}
}
