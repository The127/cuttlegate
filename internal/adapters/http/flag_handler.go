package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
)

// flagService is the use-case interface required by FlagHandler.
type flagService interface {
	Create(ctx context.Context, flag *domain.Flag) error
	GetByKey(ctx context.Context, projectID, key string) (*domain.Flag, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.Flag, error)
	Update(ctx context.Context, flag *domain.Flag) error
	DeleteByKey(ctx context.Context, projectID, key string) error
}

// projectResolver resolves a project slug to a domain.Project.
type projectResolver interface {
	GetBySlug(ctx context.Context, slug string) (*domain.Project, error)
}

// FlagHandler handles HTTP requests for flag CRUD.
type FlagHandler struct {
	svc      flagService
	projects projectResolver
}

// NewFlagHandler constructs a FlagHandler.
func NewFlagHandler(svc flagService, projects projectResolver) *FlagHandler {
	return &FlagHandler{svc: svc, projects: projects}
}

// RegisterRoutes registers all flag routes on mux behind the provided auth middleware.
func (h *FlagHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/projects/{slug}/flags", auth(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/projects/{slug}/flags", auth(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/projects/{slug}/flags/{key}", auth(http.HandlerFunc(h.get)))
	mux.Handle("PATCH /api/v1/projects/{slug}/flags/{key}", auth(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/projects/{slug}/flags/{key}", auth(http.HandlerFunc(h.delete)))
}

// variantJSON is the JSON representation of a domain.Variant.
type variantJSON struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// flagResponse is the JSON representation of a flag.
type flagResponse struct {
	ID                string        `json:"id"`
	ProjectID         string        `json:"project_id"`
	Key               string        `json:"key"`
	Name              string        `json:"name"`
	Type              string        `json:"type"`
	Variants          []variantJSON `json:"variants"`
	DefaultVariantKey string        `json:"default_variant_key"`
	CreatedAt         time.Time     `json:"created_at"`
}

func toFlagResponse(f *domain.Flag) flagResponse {
	variants := make([]variantJSON, len(f.Variants))
	for i, v := range f.Variants {
		variants[i] = variantJSON{Key: v.Key, Name: v.Name}
	}
	return flagResponse{
		ID:                f.ID,
		ProjectID:         f.ProjectID,
		Key:               f.Key,
		Name:              f.Name,
		Type:              string(f.Type),
		Variants:          variants,
		DefaultVariantKey: f.DefaultVariantKey,
		CreatedAt:         f.CreatedAt.UTC(),
	}
}

func (h *FlagHandler) resolveProject(ctx context.Context, w http.ResponseWriter, slug string) (*domain.Project, bool) {
	proj, err := h.projects.GetBySlug(ctx, slug)
	if err != nil {
		WriteError(w, err)
		return nil, false
	}
	return proj, true
}

func (h *FlagHandler) list(w http.ResponseWriter, r *http.Request) {
	proj, ok := h.resolveProject(r.Context(), w, r.PathValue("slug"))
	if !ok {
		return
	}
	flags, err := h.svc.ListByProject(r.Context(), proj.ID)
	if err != nil {
		WriteError(w, err)
		return
	}
	items := make([]flagResponse, 0, len(flags))
	for _, f := range flags {
		items = append(items, toFlagResponse(f))
	}
	writeJSON(w, http.StatusOK, map[string]any{"flags": items})
}

func (h *FlagHandler) create(w http.ResponseWriter, r *http.Request) {
	proj, ok := h.resolveProject(r.Context(), w, r.PathValue("slug"))
	if !ok {
		return
	}

	var body struct {
		Key               string        `json:"key"`
		Name              string        `json:"name"`
		Type              string        `json:"type"`
		Variants          []variantJSON `json:"variants"`
		DefaultVariantKey string        `json:"default_variant_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}

	variants := make([]domain.Variant, len(body.Variants))
	for i, v := range body.Variants {
		variants[i] = domain.Variant{Key: v.Key, Name: v.Name}
	}

	flag := &domain.Flag{
		ProjectID:         proj.ID,
		Key:               body.Key,
		Name:              body.Name,
		Type:              domain.FlagType(body.Type),
		Variants:          variants,
		DefaultVariantKey: body.DefaultVariantKey,
	}

	if err := h.svc.Create(r.Context(), flag); err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toFlagResponse(flag))
}

func (h *FlagHandler) get(w http.ResponseWriter, r *http.Request) {
	proj, ok := h.resolveProject(r.Context(), w, r.PathValue("slug"))
	if !ok {
		return
	}
	flag, err := h.svc.GetByKey(r.Context(), proj.ID, r.PathValue("key"))
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toFlagResponse(flag))
}

func (h *FlagHandler) update(w http.ResponseWriter, r *http.Request) {
	proj, ok := h.resolveProject(r.Context(), w, r.PathValue("slug"))
	if !ok {
		return
	}

	flag, err := h.svc.GetByKey(r.Context(), proj.ID, r.PathValue("key"))
	if err != nil {
		WriteError(w, err)
		return
	}

	var body struct {
		Name              *string       `json:"name"`
		Variants          []variantJSON `json:"variants"`
		DefaultVariantKey *string       `json:"default_variant_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}

	if body.Name != nil {
		flag.Name = *body.Name
	}
	if body.Variants != nil {
		variants := make([]domain.Variant, len(body.Variants))
		for i, v := range body.Variants {
			variants[i] = domain.Variant{Key: v.Key, Name: v.Name}
		}
		flag.Variants = variants
	}
	if body.DefaultVariantKey != nil {
		flag.DefaultVariantKey = *body.DefaultVariantKey
	}

	if err := h.svc.Update(r.Context(), flag); err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toFlagResponse(flag))
}

func (h *FlagHandler) delete(w http.ResponseWriter, r *http.Request) {
	proj, ok := h.resolveProject(r.Context(), w, r.PathValue("slug"))
	if !ok {
		return
	}
	if err := h.svc.DeleteByKey(r.Context(), proj.ID, r.PathValue("key")); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
