package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/The127/cuttlegate/internal/domain"
)

// environmentService is the use-case interface required by EnvironmentHandler.
type environmentService interface {
	Create(ctx context.Context, projectSlug, name, envSlug string) (*domain.Environment, error)
	GetBySlug(ctx context.Context, projectID, slug string) (*domain.Environment, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.Environment, error)
	UpdateName(ctx context.Context, projectID, slug, name string) (*domain.Environment, error)
	DeleteBySlug(ctx context.Context, projectID, slug string) error
}

// EnvironmentHandler handles HTTP requests for environment CRUD.
type EnvironmentHandler struct {
	svc      environmentService
	projects projectResolver
}

// NewEnvironmentHandler constructs an EnvironmentHandler.
func NewEnvironmentHandler(svc environmentService, projects projectResolver) *EnvironmentHandler {
	return &EnvironmentHandler{svc: svc, projects: projects}
}

// RegisterRoutes registers all environment routes on mux behind the provided auth middleware.
func (h *EnvironmentHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/projects/{slug}/environments", auth(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/projects/{slug}/environments", auth(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/projects/{slug}/environments/{env_slug}", auth(http.HandlerFunc(h.get)))
	mux.Handle("PATCH /api/v1/projects/{slug}/environments/{env_slug}", auth(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/projects/{slug}/environments/{env_slug}", auth(http.HandlerFunc(h.delete)))
}

// environmentResponse is the JSON representation of an environment.
type environmentResponse struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

func toEnvironmentResponse(e *domain.Environment) environmentResponse {
	return environmentResponse{
		ID:        e.ID,
		ProjectID: e.ProjectID,
		Name:      e.Name,
		Slug:      e.Slug,
		CreatedAt: e.CreatedAt.UTC(),
	}
}

func (h *EnvironmentHandler) list(w http.ResponseWriter, r *http.Request) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		WriteError(w, err)
		return
	}
	envs, err := h.svc.ListByProject(r.Context(), proj.ID)
	if err != nil {
		WriteError(w, err)
		return
	}
	items := make([]environmentResponse, 0, len(envs))
	for _, e := range envs {
		items = append(items, toEnvironmentResponse(e))
	}
	writeJSON(w, http.StatusOK, map[string]any{"environments": items})
}

func (h *EnvironmentHandler) create(w http.ResponseWriter, r *http.Request) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		WriteError(w, err)
		return
	}
	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}
	e, err := h.svc.Create(r.Context(), proj.ID, body.Name, body.Slug)
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toEnvironmentResponse(e))
}

func (h *EnvironmentHandler) get(w http.ResponseWriter, r *http.Request) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		WriteError(w, err)
		return
	}
	e, err := h.svc.GetBySlug(r.Context(), proj.ID, r.PathValue("env_slug"))
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toEnvironmentResponse(e))
}

func (h *EnvironmentHandler) update(w http.ResponseWriter, r *http.Request) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		WriteError(w, err)
		return
	}

	var body struct {
		Name *string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}

	// No name field in body — no-op: return current state.
	if body.Name == nil {
		e, err := h.svc.GetBySlug(r.Context(), proj.ID, r.PathValue("env_slug"))
		if err != nil {
			WriteError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toEnvironmentResponse(e))
		return
	}

	if *body.Name == "" {
		WriteError(w, newBadRequest("name must not be empty"))
		return
	}

	e, err := h.svc.UpdateName(r.Context(), proj.ID, r.PathValue("env_slug"), *body.Name)
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toEnvironmentResponse(e))
}

func (h *EnvironmentHandler) delete(w http.ResponseWriter, r *http.Request) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		WriteError(w, err)
		return
	}
	if err := h.svc.DeleteBySlug(r.Context(), proj.ID, r.PathValue("env_slug")); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
