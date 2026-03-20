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
)

// fakeProjectMemberService is a test double for the projectMemberService interface.
type fakeProjectMemberService struct {
	members map[string][]*domain.ProjectMember // keyed by projectSlug
	err     error                              // if set, all calls return this error
}

func newFakeProjectMemberService() *fakeProjectMemberService {
	return &fakeProjectMemberService{members: make(map[string][]*domain.ProjectMember)}
}

func (f *fakeProjectMemberService) ListMembers(_ context.Context, slug string) ([]*domain.ProjectMember, error) {
	if f.err != nil {
		return nil, f.err
	}
	result := make([]*domain.ProjectMember, 0, len(f.members[slug]))
	for _, m := range f.members[slug] {
		cp := *m
		result = append(result, &cp)
	}
	return result, nil
}

func (f *fakeProjectMemberService) AddMember(_ context.Context, slug, userID string, role domain.Role) (*domain.ProjectMember, error) {
	if f.err != nil {
		return nil, f.err
	}
	m := &domain.ProjectMember{
		ProjectID: "proj-" + slug,
		UserID:    userID,
		Role:      role,
		CreatedAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
	}
	f.members[slug] = append(f.members[slug], m)
	return m, nil
}

func (f *fakeProjectMemberService) UpdateRole(_ context.Context, slug, userID string, role domain.Role) error {
	if f.err != nil {
		return f.err
	}
	for _, m := range f.members[slug] {
		if m.UserID == userID {
			m.Role = role
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeProjectMemberService) RemoveMember(_ context.Context, slug, userID string) error {
	if f.err != nil {
		return f.err
	}
	list := f.members[slug]
	for i, m := range list {
		if m.UserID == userID {
			f.members[slug] = append(list[:i], list[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func newMemberMux(svc *fakeProjectMemberService, auth func(http.Handler) http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	httpadapter.NewProjectMemberHandler(svc).RegisterRoutes(mux, auth)
	return mux
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestProjectMemberHandler_List_ReturnsWrappedArray(t *testing.T) {
	svc := newFakeProjectMemberService()
	svc.members["acme"] = []*domain.ProjectMember{
		{ProjectID: "proj-acme", UserID: "user-1", Role: domain.RoleAdmin, CreatedAt: time.Now()},
	}
	mux := newMemberMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/acme/members", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	arr, ok := body["members"].([]any)
	if !ok {
		t.Fatal("response missing 'members' array")
	}
	if len(arr) != 1 {
		t.Errorf("expected 1 member, got %d", len(arr))
	}
}

// ── Add ───────────────────────────────────────────────────────────────────────

func TestProjectMemberHandler_Add_Succeeds(t *testing.T) {
	mux := newMemberMux(newFakeProjectMemberService(), noopAuth)
	body := `{"user_id":"user-1","role":"viewer"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201", rec.Code)
	}
	var m map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if m["user_id"] != "user-1" {
		t.Errorf("user_id: got %v", m["user_id"])
	}
	if m["role"] != "viewer" {
		t.Errorf("role: got %v", m["role"])
	}
}

func TestProjectMemberHandler_Add_MissingFields_Returns400(t *testing.T) {
	mux := newMemberMux(newFakeProjectMemberService(), noopAuth)
	body := `{"user_id":"user-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/members", strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestProjectMemberHandler_Add_InvalidRole_Returns400(t *testing.T) {
	mux := newMemberMux(newFakeProjectMemberService(), noopAuth)
	body := `{"user_id":"user-1","role":"superuser"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/members", strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestProjectMemberHandler_Add_ForbiddenReturns403(t *testing.T) {
	svc := newFakeProjectMemberService()
	svc.err = domain.ErrForbidden
	mux := newMemberMux(svc, noopAuth)

	body := `{"user_id":"user-1","role":"viewer"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/acme/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

// ── Update role ───────────────────────────────────────────────────────────────

func TestProjectMemberHandler_UpdateRole_Succeeds(t *testing.T) {
	svc := newFakeProjectMemberService()
	svc.members["acme"] = []*domain.ProjectMember{
		{ProjectID: "proj-acme", UserID: "user-1", Role: domain.RoleViewer},
	}
	mux := newMemberMux(svc, noopAuth)

	body := `{"role":"editor"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/members/user-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", rec.Code)
	}
}

func TestProjectMemberHandler_UpdateRole_ForbiddenReturns403(t *testing.T) {
	svc := newFakeProjectMemberService()
	svc.err = domain.ErrForbidden
	mux := newMemberMux(svc, noopAuth)

	body := `{"role":"editor"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/members/user-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
}

// ── Remove ────────────────────────────────────────────────────────────────────

func TestProjectMemberHandler_Remove_Succeeds(t *testing.T) {
	svc := newFakeProjectMemberService()
	svc.members["acme"] = []*domain.ProjectMember{
		{ProjectID: "proj-acme", UserID: "user-1", Role: domain.RoleViewer},
	}
	mux := newMemberMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/acme/members/user-1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", rec.Code)
	}
}

// ── Scenario 12: ErrLastAdmin → 409 ──────────────────────────────────────────

func TestProjectMemberHandler_Remove_LastAdminReturns409(t *testing.T) {
	svc := newFakeProjectMemberService()
	svc.err = domain.ErrLastAdmin
	mux := newMemberMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/acme/members/admin-1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status: got %d, want 409", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "last_admin" {
		t.Errorf("error code: got %v, want last_admin", body["error"])
	}
}

// ── Scenario 11: all routes require auth ─────────────────────────────────────

func TestProjectMemberHandler_Unauthenticated_Returns401(t *testing.T) {
	routes := []struct {
		method, path, body string
	}{
		{http.MethodGet, "/api/v1/projects/acme/members", ""},
		{http.MethodPost, "/api/v1/projects/acme/members", `{"user_id":"u","role":"viewer"}`},
		{http.MethodPatch, "/api/v1/projects/acme/members/user-1", `{"role":"editor"}`},
		{http.MethodDelete, "/api/v1/projects/acme/members/user-1", ""},
	}
	mux := newMemberMux(newFakeProjectMemberService(), requireAuth401)

	for _, tc := range routes {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: got %d, want 401", tc.method, tc.path, rec.Code)
		}
	}
}
