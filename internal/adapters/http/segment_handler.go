package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
)

// segmentService is the use-case interface required by SegmentHandler.
type segmentService interface {
	Create(ctx context.Context, projectID, slug, name string) (*domain.Segment, error)
	GetBySlug(ctx context.Context, projectID, slug string) (*domain.Segment, error)
	List(ctx context.Context, projectID string) ([]*domain.Segment, error)
	UpdateName(ctx context.Context, id, name string) error
	Delete(ctx context.Context, projectID, slug string) error
	SetMembers(ctx context.Context, segmentID string, userKeys []string) error
	ListMembers(ctx context.Context, segmentID string) ([]string, error)
}

// SegmentHandler handles HTTP requests for user segment CRUD.
type SegmentHandler struct {
	svc      segmentService
	projects projectResolver
}

// NewSegmentHandler constructs a SegmentHandler.
func NewSegmentHandler(svc segmentService, projects projectResolver) *SegmentHandler {
	return &SegmentHandler{svc: svc, projects: projects}
}

// RegisterRoutes registers all segment routes on mux behind the provided auth middleware.
func (h *SegmentHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("POST /api/v1/projects/{slug}/segments", auth(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/projects/{slug}/segments", auth(http.HandlerFunc(h.list)))
	mux.Handle("GET /api/v1/projects/{slug}/segments/{segmentSlug}", auth(http.HandlerFunc(h.get)))
	mux.Handle("PATCH /api/v1/projects/{slug}/segments/{segmentSlug}", auth(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/projects/{slug}/segments/{segmentSlug}", auth(http.HandlerFunc(h.delete)))
	mux.Handle("PUT /api/v1/projects/{slug}/segments/{segmentSlug}/members", auth(http.HandlerFunc(h.setMembers)))
	mux.Handle("GET /api/v1/projects/{slug}/segments/{segmentSlug}/members", auth(http.HandlerFunc(h.listMembers)))
}

// segmentResponse is the JSON representation of a domain.Segment.
type segmentResponse struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	ProjectID string    `json:"projectId"`
	CreatedAt time.Time `json:"createdAt"`
}

func toSegmentResponse(s *domain.Segment) segmentResponse {
	return segmentResponse{
		ID:        s.ID,
		Slug:      s.Slug,
		Name:      s.Name,
		ProjectID: s.ProjectID,
		CreatedAt: s.CreatedAt.UTC(),
	}
}

func (h *SegmentHandler) create(w http.ResponseWriter, r *http.Request) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		WriteError(w, err)
		return
	}

	var body struct {
		Slug string `json:"slug"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}

	seg, err := h.svc.Create(r.Context(), proj.ID, body.Slug, body.Name)
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toSegmentResponse(seg))
}

func (h *SegmentHandler) list(w http.ResponseWriter, r *http.Request) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		WriteError(w, err)
		return
	}

	segments, err := h.svc.List(r.Context(), proj.ID)
	if err != nil {
		WriteError(w, err)
		return
	}

	items := make([]segmentResponse, 0, len(segments))
	for _, s := range segments {
		items = append(items, toSegmentResponse(s))
	}
	writeJSON(w, http.StatusOK, map[string]any{"segments": items})
}

func (h *SegmentHandler) get(w http.ResponseWriter, r *http.Request) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		WriteError(w, err)
		return
	}

	seg, err := h.svc.GetBySlug(r.Context(), proj.ID, r.PathValue("segmentSlug"))
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toSegmentResponse(seg))
}

func (h *SegmentHandler) update(w http.ResponseWriter, r *http.Request) {
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

	seg, err := h.svc.GetBySlug(r.Context(), proj.ID, r.PathValue("segmentSlug"))
	if err != nil {
		WriteError(w, err)
		return
	}

	if body.Name == nil {
		writeJSON(w, http.StatusOK, toSegmentResponse(seg))
		return
	}

	if err := h.svc.UpdateName(r.Context(), seg.ID, *body.Name); err != nil {
		WriteError(w, err)
		return
	}
	seg.Name = *body.Name
	writeJSON(w, http.StatusOK, toSegmentResponse(seg))
}

func (h *SegmentHandler) delete(w http.ResponseWriter, r *http.Request) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		WriteError(w, err)
		return
	}

	if err := h.svc.Delete(r.Context(), proj.ID, r.PathValue("segmentSlug")); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SegmentHandler) setMembers(w http.ResponseWriter, r *http.Request) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		WriteError(w, err)
		return
	}

	seg, err := h.svc.GetBySlug(r.Context(), proj.ID, r.PathValue("segmentSlug"))
	if err != nil {
		WriteError(w, err)
		return
	}

	var body struct {
		Members []string `json:"members"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}
	if body.Members == nil {
		body.Members = []string{}
	}

	if err := h.svc.SetMembers(r.Context(), seg.ID, body.Members); err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": body.Members})
}

func (h *SegmentHandler) listMembers(w http.ResponseWriter, r *http.Request) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		WriteError(w, err)
		return
	}

	seg, err := h.svc.GetBySlug(r.Context(), proj.ID, r.PathValue("segmentSlug"))
	if err != nil {
		WriteError(w, err)
		return
	}

	members, err := h.svc.ListMembers(r.Context(), seg.ID)
	if err != nil {
		WriteError(w, err)
		return
	}
	if members == nil {
		members = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": members})
}
