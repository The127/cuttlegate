package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpadapter "github.com/karo/cuttlegate/internal/adapters/http"
	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// fakeSegmentService is a test double for the segmentService interface.
type fakeSegmentService struct {
	seg *domain.Segment // returned by Create and GetBySlug when err is nil
	err error           // returned by all methods when set
}

func newFakeSegmentService() *fakeSegmentService {
	return &fakeSegmentService{
		seg: &domain.Segment{
			ID:        "seg-1",
			Slug:      "beta-users",
			Name:      "Beta Users",
			ProjectID: "proj-acme",
			CreatedAt: time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC),
		},
	}
}

func (f *fakeSegmentService) Create(_ context.Context, _, _, _ string) (*domain.Segment, error) {
	return f.seg, f.err
}

func (f *fakeSegmentService) GetBySlug(_ context.Context, _, _ string) (*domain.Segment, error) {
	return f.seg, f.err
}

func (f *fakeSegmentService) List(_ context.Context, _ string) ([]*domain.Segment, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []*domain.Segment{f.seg}, nil
}

func (f *fakeSegmentService) ListWithCount(_ context.Context, _ string) ([]*ports.SegmentWithCount, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []*ports.SegmentWithCount{{Segment: f.seg, MemberCount: 3}}, nil
}

func (f *fakeSegmentService) UpdateName(_ context.Context, _, _ string) error {
	return f.err
}

func (f *fakeSegmentService) Delete(_ context.Context, _, _ string) error {
	return f.err
}

func (f *fakeSegmentService) SetMembers(_ context.Context, _ string, _ []string) error {
	return f.err
}

func (f *fakeSegmentService) ListMembers(_ context.Context, _ string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []string{}, nil
}

func newSegmentMux(svc *fakeSegmentService, auth func(http.Handler) http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	httpadapter.NewSegmentHandler(svc, newFakeResolver("acme")).RegisterRoutes(mux, auth)
	return mux
}

const segBase = "/api/v1/projects/acme/segments"

// ── Auth: all routes require authentication ───────────────────────────────────

func TestSegmentHandler_Unauthenticated_Returns401(t *testing.T) {
	routes := []struct {
		method, path, body string
	}{
		{http.MethodPost, segBase, `{"slug":"beta","name":"Beta"}`},
		{http.MethodGet, segBase, ""},
		{http.MethodGet, segBase + "/beta-users", ""},
		{http.MethodPatch, segBase + "/beta-users", `{"name":"Beta v2"}`},
		{http.MethodDelete, segBase + "/beta-users", ""},
		{http.MethodPut, segBase + "/beta-users/members", `{"members":["u1"]}`},
		{http.MethodGet, segBase + "/beta-users/members", ""},
	}
	mux := newSegmentMux(newFakeSegmentService(), requireAuth401)
	for _, tc := range routes {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: got %d, want 401", tc.method, tc.path, rec.Code)
		}
	}
}

// ── RBAC: Editor-gated writes ─────────────────────────────────────────────────

func TestSegmentHandler_Create_Forbidden_Returns403(t *testing.T) {
	svc := newFakeSegmentService()
	svc.err = domain.ErrForbidden
	mux := newSegmentMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodPost, segBase,
		strings.NewReader(`{"slug":"beta","name":"Beta"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

func TestSegmentHandler_Update_Forbidden_Returns403(t *testing.T) {
	svc := newFakeSegmentService()
	svc.err = domain.ErrForbidden
	mux := newSegmentMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodPatch, segBase+"/beta-users",
		strings.NewReader(`{"name":"Beta v2"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

func TestSegmentHandler_Delete_Forbidden_Returns403(t *testing.T) {
	svc := newFakeSegmentService()
	svc.err = domain.ErrForbidden
	mux := newSegmentMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodDelete, segBase+"/beta-users", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

func TestSegmentHandler_SetMembers_Forbidden_Returns403(t *testing.T) {
	svc := newFakeSegmentService()
	svc.err = domain.ErrForbidden
	mux := newSegmentMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodPut, segBase+"/beta-users/members",
		strings.NewReader(`{"members":["u1"]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

// ── RBAC: Viewer-gated reads ──────────────────────────────────────────────────

func TestSegmentHandler_List_Forbidden_Returns403(t *testing.T) {
	svc := newFakeSegmentService()
	svc.err = domain.ErrForbidden
	mux := newSegmentMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodGet, segBase, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

func TestSegmentHandler_Get_Forbidden_Returns403(t *testing.T) {
	svc := newFakeSegmentService()
	svc.err = domain.ErrForbidden
	mux := newSegmentMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodGet, segBase+"/beta-users", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

func TestSegmentHandler_ListMembers_Forbidden_Returns403(t *testing.T) {
	svc := newFakeSegmentService()
	svc.err = domain.ErrForbidden
	mux := newSegmentMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodGet, segBase+"/beta-users/members", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

// @happy — list response includes memberCount field
func TestSegmentHandler_List_IncludesMemberCount(t *testing.T) {
	mux := newSegmentMux(newFakeSegmentService(), noopAuth)

	req := httptest.NewRequest(http.MethodGet, segBase, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}

	var body struct {
		Segments []struct {
			ID          string `json:"id"`
			Slug        string `json:"slug"`
			MemberCount int    `json:"memberCount"`
		} `json:"segments"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(body.Segments))
	}
	seg := body.Segments[0]
	if seg.Slug != "beta-users" {
		t.Errorf("slug: got %q, want %q", seg.Slug, "beta-users")
	}
	// fakeSegmentService.ListWithCount returns MemberCount = 3
	if seg.MemberCount != 3 {
		t.Errorf("memberCount: got %d, want 3", seg.MemberCount)
	}
}
