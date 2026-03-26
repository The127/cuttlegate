package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpadapter "github.com/The127/cuttlegate/internal/adapters/http"
	"github.com/The127/cuttlegate/internal/app"
	"github.com/The127/cuttlegate/internal/domain"
)

// fakeProjectMemberService is a test double for the projectMemberService interface.
type fakeProjectMemberService struct {
	members map[string][]app.ProjectMemberView // keyed by projectSlug
	err     error                              // if set, all calls return this error
}

func newFakeProjectMemberService() *fakeProjectMemberService {
	return &fakeProjectMemberService{members: make(map[string][]app.ProjectMemberView)}
}

func (f *fakeProjectMemberService) ListMembers(_ context.Context, slug string) ([]app.ProjectMemberView, error) {
	if f.err != nil {
		return nil, f.err
	}
	result := make([]app.ProjectMemberView, len(f.members[slug]))
	copy(result, f.members[slug])
	return result, nil
}

func (f *fakeProjectMemberService) AddMember(_ context.Context, slug, userID string, role domain.Role) (app.ProjectMemberView, error) {
	if f.err != nil {
		return app.ProjectMemberView{}, f.err
	}
	v := app.ProjectMemberView{
		ProjectID: "proj-" + slug,
		UserID:    userID,
		Role:      role,
		CreatedAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
		Name:      "",
		Email:     "",
	}
	f.members[slug] = append(f.members[slug], v)
	return v, nil
}

func (f *fakeProjectMemberService) UpdateRole(_ context.Context, slug, userID string, role domain.Role) error {
	if f.err != nil {
		return f.err
	}
	for i, m := range f.members[slug] {
		if m.UserID == userID {
			f.members[slug][i].Role = role
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
	svc.members["acme"] = []app.ProjectMemberView{
		{ProjectID: "proj-acme", UserID: "user-1", Role: domain.RoleAdmin, CreatedAt: time.Now(), Name: "Alice", Email: "alice@example.com"},
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
	m := arr[0].(map[string]any)
	if m["name"] != "Alice" {
		t.Errorf("name: got %v, want Alice", m["name"])
	}
	if m["email"] != "alice@example.com" {
		t.Errorf("email: got %v, want alice@example.com", m["email"])
	}
}

func TestProjectMemberHandler_List_UnknownProfile_ReturnsEmptyStrings(t *testing.T) {
	svc := newFakeProjectMemberService()
	svc.members["acme"] = []app.ProjectMemberView{
		{ProjectID: "proj-acme", UserID: "user-1", Role: domain.RoleViewer, CreatedAt: time.Now()},
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
	arr := body["members"].([]any)
	m := arr[0].(map[string]any)
	// name and email must be present as empty strings, not absent or null
	if _, ok := m["name"]; !ok {
		t.Error("name field absent from response")
	}
	if _, ok := m["email"]; !ok {
		t.Error("email field absent from response")
	}
	if m["name"] != "" {
		t.Errorf("name: got %v, want empty string", m["name"])
	}
	if m["email"] != "" {
		t.Errorf("email: got %v, want empty string", m["email"])
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
	// name and email must always be present
	if _, ok := m["name"]; !ok {
		t.Error("name field absent from add response")
	}
	if _, ok := m["email"]; !ok {
		t.Error("email field absent from add response")
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
	svc.members["acme"] = []app.ProjectMemberView{
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

func TestProjectMemberHandler_UpdateRole_LastAdminReturns409(t *testing.T) {
	svc := newFakeProjectMemberService()
	svc.err = domain.ErrLastAdmin
	mux := newMemberMux(svc, noopAuth)

	body := `{"role":"viewer"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/acme/members/admin-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status: got %d, want 409", rec.Code)
	}
	var body2 map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body2["error"] != "last_admin" {
		t.Errorf("error code: got %v, want last_admin", body2["error"])
	}
}

// ── Remove ────────────────────────────────────────────────────────────────────

func TestProjectMemberHandler_Remove_Succeeds(t *testing.T) {
	svc := newFakeProjectMemberService()
	svc.members["acme"] = []app.ProjectMemberView{
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

// ── RBAC ──────────────────────────────────────────────────────────────────────

func TestProjectMemberHandler_Remove_Forbidden_Returns403(t *testing.T) {
	svc := newFakeProjectMemberService()
	svc.err = domain.ErrForbidden
	mux := newMemberMux(svc, noopAuth)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/acme/members/user-1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", rec.Code)
	}
	var b map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&b); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if b["error"] != "forbidden" {
		t.Errorf("error code: got %v, want forbidden", b["error"])
	}
}
