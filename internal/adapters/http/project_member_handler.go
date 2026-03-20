package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
)

// projectMemberService is the use-case interface required by ProjectMemberHandler.
type projectMemberService interface {
	ListMembers(ctx context.Context, projectSlug string) ([]*domain.ProjectMember, error)
	AddMember(ctx context.Context, projectSlug, userID string, role domain.Role) (*domain.ProjectMember, error)
	UpdateRole(ctx context.Context, projectSlug, userID string, role domain.Role) error
	RemoveMember(ctx context.Context, projectSlug, userID string) error
}

// ProjectMemberHandler handles HTTP requests for project membership management.
type ProjectMemberHandler struct {
	svc projectMemberService
}

// NewProjectMemberHandler constructs a ProjectMemberHandler.
func NewProjectMemberHandler(svc projectMemberService) *ProjectMemberHandler {
	return &ProjectMemberHandler{svc: svc}
}

// RegisterRoutes registers all membership routes on mux behind the provided auth middleware.
func (h *ProjectMemberHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/projects/{slug}/members", auth(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/projects/{slug}/members", auth(http.HandlerFunc(h.add)))
	mux.Handle("PATCH /api/v1/projects/{slug}/members/{user_id}", auth(http.HandlerFunc(h.updateRole)))
	mux.Handle("DELETE /api/v1/projects/{slug}/members/{user_id}", auth(http.HandlerFunc(h.remove)))
}

// memberResponse is the JSON representation of a project member.
type memberResponse struct {
	ProjectID string    `json:"project_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func toMemberResponse(m *domain.ProjectMember) memberResponse {
	return memberResponse{
		ProjectID: m.ProjectID,
		UserID:    m.UserID,
		Role:      string(m.Role),
		CreatedAt: m.CreatedAt.UTC(),
	}
}

func (h *ProjectMemberHandler) list(w http.ResponseWriter, r *http.Request) {
	members, err := h.svc.ListMembers(r.Context(), r.PathValue("slug"))
	if err != nil {
		WriteError(w, err)
		return
	}
	items := make([]memberResponse, 0, len(members))
	for _, m := range members {
		items = append(items, toMemberResponse(m))
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": items})
}

func (h *ProjectMemberHandler) add(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}
	if body.UserID == "" || body.Role == "" {
		WriteError(w, newBadRequest("user_id and role are required"))
		return
	}
	role := domain.Role(body.Role)
	if !role.Valid() {
		WriteError(w, newBadRequest("invalid role"))
		return
	}
	m, err := h.svc.AddMember(r.Context(), r.PathValue("slug"), body.UserID, role)
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toMemberResponse(m))
}

func (h *ProjectMemberHandler) updateRole(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}
	if body.Role == "" {
		WriteError(w, newBadRequest("role is required"))
		return
	}
	role := domain.Role(body.Role)
	if !role.Valid() {
		WriteError(w, newBadRequest("invalid role"))
		return
	}
	if err := h.svc.UpdateRole(r.Context(), r.PathValue("slug"), r.PathValue("user_id"), role); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProjectMemberHandler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveMember(r.Context(), r.PathValue("slug"), r.PathValue("user_id")); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
