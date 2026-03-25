package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/karo/cuttlegate/internal/domain"
)

// flagVariantService is the use-case interface required by FlagVariantHandler.
type flagVariantService interface {
	AddVariant(ctx context.Context, projectID, flagKey string, v domain.Variant) (*domain.Flag, error)
	RenameVariant(ctx context.Context, projectID, flagKey, variantKey, newName string) (*domain.Flag, error)
	DeleteVariant(ctx context.Context, projectID, flagKey, variantKey string, force bool) (*domain.Flag, error)
}

// FlagVariantHandler handles HTTP requests for variant management.
type FlagVariantHandler struct {
	svc      flagVariantService
	projects projectResolver
}

// NewFlagVariantHandler constructs a FlagVariantHandler.
func NewFlagVariantHandler(svc flagVariantService, projects projectResolver) *FlagVariantHandler {
	return &FlagVariantHandler{svc: svc, projects: projects}
}

// RegisterRoutes registers all variant routes on mux behind the provided auth middleware.
func (h *FlagVariantHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("POST /api/v1/projects/{slug}/flags/{key}/variants", auth(http.HandlerFunc(h.add)))
	mux.Handle("PATCH /api/v1/projects/{slug}/flags/{key}/variants/{variant_key}", auth(http.HandlerFunc(h.rename)))
	mux.Handle("DELETE /api/v1/projects/{slug}/flags/{key}/variants/{variant_key}", auth(http.HandlerFunc(h.delete)))
}

func (h *FlagVariantHandler) add(w http.ResponseWriter, r *http.Request) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		WriteError(w, err)
		return
	}

	var body struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}
	if body.Key == "" {
		WriteError(w, newBadRequest("key is required"))
		return
	}

	f, err := h.svc.AddVariant(r.Context(), proj.ID, r.PathValue("key"), domain.Variant{Key: body.Key, Name: body.Name})
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toFlagResponse(f))
}

func (h *FlagVariantHandler) rename(w http.ResponseWriter, r *http.Request) {
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
	if body.Name == nil {
		WriteError(w, newBadRequest("name is required"))
		return
	}

	f, err := h.svc.RenameVariant(r.Context(), proj.ID, r.PathValue("key"), r.PathValue("variant_key"), *body.Name)
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toFlagResponse(f))
}

func (h *FlagVariantHandler) delete(w http.ResponseWriter, r *http.Request) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		WriteError(w, err)
		return
	}

	force := r.URL.Query().Get("force") == "true"
	f, err := h.svc.DeleteVariant(r.Context(), proj.ID, r.PathValue("key"), r.PathValue("variant_key"), force)
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toFlagResponse(f))
}
