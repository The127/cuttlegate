//go:build integration

package dbadapter_test

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
	"github.com/karo/cuttlegate/internal/domain"
)

func TestPostgresSegmentRepository(t *testing.T) {
	db := newTestDB(t)
	projRepo := dbadapter.NewPostgresProjectRepository(db)
	segRepo := dbadapter.NewPostgresSegmentRepository(db)
	ctx := context.Background()

	const projID = "eeeeeeee-1111-4111-8111-111111111111"

	if err := projRepo.Create(ctx, domain.Project{
		ID:        projID,
		Name:      "seg-test-proj",
		Slug:      "seg-test-proj",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	newSeg := func(id, slug string) *domain.Segment {
		return &domain.Segment{
			ID:        id,
			Slug:      slug,
			Name:      slug + "-name",
			ProjectID: projID,
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
	}

	t.Run("Create and GetBySlug roundtrip", func(t *testing.T) {
		s := newSeg("00000042-0000-4000-8000-000000000001", "beta-users")
		if err := segRepo.Create(ctx, s); err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := segRepo.GetBySlug(ctx, projID, "beta-users")
		if err != nil {
			t.Fatalf("GetBySlug: %v", err)
		}
		if got.ID != s.ID || got.Slug != s.Slug || got.Name != s.Name || got.ProjectID != s.ProjectID {
			t.Errorf("segment mismatch: got %+v", got)
		}
	})

	t.Run("Duplicate (project_id, slug) returns ErrConflict", func(t *testing.T) {
		s := newSeg("00000042-0000-4000-8000-000000000002", "beta-users")
		err := segRepo.Create(ctx, s)
		if !errors.Is(err, domain.ErrConflict) {
			t.Errorf("expected ErrConflict, got %v", err)
		}
	})

	t.Run("GetBySlug absent returns ErrNotFound", func(t *testing.T) {
		_, err := segRepo.GetBySlug(ctx, projID, "ghost-segment")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("List returns segments for correct project only", func(t *testing.T) {
		// Create a second segment in the same project and one in a different project.
		if err := segRepo.Create(ctx, newSeg("00000042-0000-4000-8000-000000000003", "early-access")); err != nil {
			t.Fatalf("Create second segment: %v", err)
		}

		const otherProjID = "eeeeeeee-2222-4222-8222-222222222222"
		if err := projRepo.Create(ctx, domain.Project{
			ID:        otherProjID,
			Name:      "other-proj",
			Slug:      "other-proj",
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}); err != nil {
			t.Fatalf("seed other project: %v", err)
		}
		other := &domain.Segment{
			ID:        "00000042-0000-4000-8000-000000000004",
			Slug:      "other-segment",
			Name:      "Other",
			ProjectID: otherProjID,
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
		if err := segRepo.Create(ctx, other); err != nil {
			t.Fatalf("Create other-project segment: %v", err)
		}

		list, err := segRepo.List(ctx, projID)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(list) < 2 {
			t.Errorf("expected at least 2 segments for proj, got %d", len(list))
		}
		for _, s := range list {
			if s.ProjectID != projID {
				t.Errorf("got segment from wrong project: %s", s.ProjectID)
			}
		}
	})

	t.Run("List empty project returns empty slice never nil", func(t *testing.T) {
		list, err := segRepo.List(ctx, "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if list == nil {
			t.Error("expected empty slice, got nil")
		}
	})

	t.Run("UpdateName changes name, absent ID returns ErrNotFound", func(t *testing.T) {
		s := newSeg("00000042-0000-4000-8000-000000000005", "canary")
		if err := segRepo.Create(ctx, s); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := segRepo.UpdateName(ctx, s.ID, "canary-updated"); err != nil {
			t.Fatalf("UpdateName: %v", err)
		}
		got, err := segRepo.GetBySlug(ctx, projID, "canary")
		if err != nil {
			t.Fatalf("GetBySlug after update: %v", err)
		}
		if got.Name != "canary-updated" {
			t.Errorf("name not updated: %q", got.Name)
		}

		err = segRepo.UpdateName(ctx, "00000000-0000-0000-0000-000000000000", "nope")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound for absent ID, got %v", err)
		}
	})

	t.Run("Delete removes segment, absent ID returns ErrNotFound", func(t *testing.T) {
		s := newSeg("00000042-0000-4000-8000-000000000006", "to-delete")
		if err := segRepo.Create(ctx, s); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := segRepo.Delete(ctx, s.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		_, err := segRepo.GetBySlug(ctx, projID, "to-delete")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound after delete, got %v", err)
		}

		err = segRepo.Delete(ctx, "00000000-0000-0000-0000-000000000000")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound for absent ID, got %v", err)
		}
	})

	t.Run("SetMembers and ListMembers roundtrip", func(t *testing.T) {
		s := newSeg("00000042-0000-4000-8000-000000000007", "members-seg")
		if err := segRepo.Create(ctx, s); err != nil {
			t.Fatalf("Create: %v", err)
		}

		// Initial set.
		if err := segRepo.SetMembers(ctx, s.ID, []string{"alice", "bob", "carol"}); err != nil {
			t.Fatalf("SetMembers: %v", err)
		}
		members, err := segRepo.ListMembers(ctx, s.ID)
		if err != nil {
			t.Fatalf("ListMembers: %v", err)
		}
		sort.Strings(members)
		if len(members) != 3 || members[0] != "alice" || members[1] != "bob" || members[2] != "carol" {
			t.Errorf("unexpected members: %v", members)
		}

		// Replace — removes carol, adds dave.
		if err := segRepo.SetMembers(ctx, s.ID, []string{"alice", "bob", "dave"}); err != nil {
			t.Fatalf("SetMembers replace: %v", err)
		}
		members, err = segRepo.ListMembers(ctx, s.ID)
		if err != nil {
			t.Fatalf("ListMembers after replace: %v", err)
		}
		sort.Strings(members)
		if len(members) != 3 || members[2] != "dave" {
			t.Errorf("unexpected members after replace: %v", members)
		}

		// Clear all members with empty slice.
		if err := segRepo.SetMembers(ctx, s.ID, []string{}); err != nil {
			t.Fatalf("SetMembers clear: %v", err)
		}
		members, err = segRepo.ListMembers(ctx, s.ID)
		if err != nil {
			t.Fatalf("ListMembers after clear: %v", err)
		}
		if len(members) != 0 {
			t.Errorf("expected empty members after clear, got %v", members)
		}
	})

	t.Run("ListMembers empty segment returns empty slice never nil", func(t *testing.T) {
		s := newSeg("00000042-0000-4000-8000-000000000008", "empty-members")
		if err := segRepo.Create(ctx, s); err != nil {
			t.Fatalf("Create: %v", err)
		}
		members, err := segRepo.ListMembers(ctx, s.ID)
		if err != nil {
			t.Fatalf("ListMembers: %v", err)
		}
		if members == nil {
			t.Error("expected empty slice, got nil")
		}
	})

	t.Run("IsMember returns correct membership status", func(t *testing.T) {
		s := newSeg("00000042-0000-4000-8000-000000000009", "ismember-seg")
		if err := segRepo.Create(ctx, s); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := segRepo.SetMembers(ctx, s.ID, []string{"alice"}); err != nil {
			t.Fatalf("SetMembers: %v", err)
		}

		ok, err := segRepo.IsMember(ctx, s.ID, "alice")
		if err != nil {
			t.Fatalf("IsMember alice: %v", err)
		}
		if !ok {
			t.Error("expected alice to be a member")
		}

		ok, err = segRepo.IsMember(ctx, s.ID, "bob")
		if err != nil {
			t.Fatalf("IsMember bob: %v", err)
		}
		if ok {
			t.Error("expected bob not to be a member")
		}
	})

	t.Run("ListWithCount returns correct member counts", func(t *testing.T) {
		const countProjID = "eeeeeeee-3333-4333-8333-333333333333"
		if err := projRepo.Create(ctx, domain.Project{
			ID:        countProjID,
			Name:      "count-proj",
			Slug:      "count-proj",
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}); err != nil {
			t.Fatalf("seed count project: %v", err)
		}

		// Segment with 2 members.
		sA := newSeg("00000042-0000-4000-8000-000000000010", "count-a")
		sA.ProjectID = countProjID
		if err := segRepo.Create(ctx, sA); err != nil {
			t.Fatalf("Create sA: %v", err)
		}
		if err := segRepo.SetMembers(ctx, sA.ID, []string{"u1", "u2"}); err != nil {
			t.Fatalf("SetMembers sA: %v", err)
		}

		// Segment with 0 members.
		sB := newSeg("00000042-0000-4000-8000-000000000011", "count-b")
		sB.ProjectID = countProjID
		if err := segRepo.Create(ctx, sB); err != nil {
			t.Fatalf("Create sB: %v", err)
		}

		items, err := segRepo.ListWithCount(ctx, countProjID)
		if err != nil {
			t.Fatalf("ListWithCount: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(items))
		}

		bySlug := make(map[string]int, len(items))
		for _, item := range items {
			bySlug[item.Segment.Slug] = item.MemberCount
		}
		if bySlug["count-a"] != 2 {
			t.Errorf("count-a: expected 2 members, got %d", bySlug["count-a"])
		}
		if bySlug["count-b"] != 0 {
			t.Errorf("count-b: expected 0 members, got %d", bySlug["count-b"])
		}
	})
}
