package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
)

// projectService is the use-case interface required by ProjectHandler.
type projectService interface {
	Create(ctx context.Context, name, slug string) (*domain.Project, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Project, error)
	List(ctx context.Context) ([]*domain.Project, error)
	UpdateName(ctx context.Context, slug, name string) (*domain.Project, error)
	DeleteBySlug(ctx context.Context, slug string) error
}

// ProjectHandler handles HTTP requests for project CRUD.
type ProjectHandler struct {
	svc projectService
}

// NewProjectHandler constructs a ProjectHandler.
func NewProjectHandler(svc projectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

// RegisterRoutes registers all project routes on mux behind the provided auth middleware.
func (h *ProjectHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/projects", auth(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/projects", auth(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/projects/{slug}", auth(http.HandlerFunc(h.get)))
	mux.Handle("PATCH /api/v1/projects/{slug}", auth(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/projects/{slug}", auth(http.HandlerFunc(h.delete)))
}

// projectResponse is the JSON representation of a project.
type projectResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

func toProjectResponse(p *domain.Project) projectResponse {
	return projectResponse{
		ID:        p.ID,
		Name:      p.Name,
		Slug:      p.Slug,
		CreatedAt: p.CreatedAt.UTC(),
	}
}

func (h *ProjectHandler) list(w http.ResponseWriter, r *http.Request) {
	projects, err := h.svc.List(r.Context())
	if err != nil {
		WriteError(w, err)
		return
	}
	items := make([]projectResponse, 0, len(projects))
	for _, p := range projects {
		items = append(items, toProjectResponse(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": items})
}

func (h *ProjectHandler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}
	if body.Name == "" || body.Slug == "" {
		WriteError(w, newBadRequest("name and slug are required"))
		return
	}
	p, err := h.svc.Create(r.Context(), body.Name, body.Slug)
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProjectResponse(p))
}

func (h *ProjectHandler) get(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	p, err := h.svc.GetBySlug(r.Context(), slug)
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProjectResponse(p))
}

func (h *ProjectHandler) update(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	var body struct {
		Name *string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}

	// No name field in body — no-op: return current state.
	if body.Name == nil {
		p, err := h.svc.GetBySlug(r.Context(), slug)
		if err != nil {
			WriteError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toProjectResponse(p))
		return
	}

	if *body.Name == "" {
		WriteError(w, newBadRequest("name must not be empty"))
		return
	}

	p, err := h.svc.UpdateName(r.Context(), slug, *body.Name)
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProjectResponse(p))
}

func (h *ProjectHandler) delete(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if err := h.svc.DeleteBySlug(r.Context(), slug); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
