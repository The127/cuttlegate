package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

// flagEnvService is the use-case interface required by FlagEnvironmentHandler.
type flagEnvService interface {
	ListByEnvironment(ctx context.Context, projectID, environmentID string) ([]*app.FlagEnvironmentView, error)
	SetEnabled(ctx context.Context, projectID, environmentID, flagKey string, enabled bool) error
}

// environmentResolver resolves an environment slug within a project to a domain.Environment.
type environmentResolver interface {
	GetBySlug(ctx context.Context, projectID, slug string) (*domain.Environment, error)
}

// FlagEnvironmentHandler handles HTTP requests for per-environment flag state.
type FlagEnvironmentHandler struct {
	svc      flagEnvService
	projects projectResolver
	envs     environmentResolver
}

// NewFlagEnvironmentHandler constructs a FlagEnvironmentHandler.
func NewFlagEnvironmentHandler(svc flagEnvService, projects projectResolver, envs environmentResolver) *FlagEnvironmentHandler {
	return &FlagEnvironmentHandler{svc: svc, projects: projects, envs: envs}
}

// RegisterRoutes registers environment-scoped flag routes on mux behind the provided auth middleware.
func (h *FlagEnvironmentHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/projects/{slug}/environments/{env_slug}/flags", auth(http.HandlerFunc(h.list)))
	mux.Handle("PATCH /api/v1/projects/{slug}/environments/{env_slug}/flags/{key}", auth(http.HandlerFunc(h.setEnabled)))
}

// flagEnvResponse is the JSON representation of a flag with its per-environment enabled state.
type flagEnvResponse struct {
	ID                string        `json:"id"`
	ProjectID         string        `json:"project_id"`
	Key               string        `json:"key"`
	Name              string        `json:"name"`
	Type              string        `json:"type"`
	Variants          []variantJSON `json:"variants"`
	DefaultVariantKey string        `json:"default_variant_key"`
	Enabled           bool          `json:"enabled"`
}

func toFlagEnvResponse(v *app.FlagEnvironmentView) flagEnvResponse {
	variants := make([]variantJSON, len(v.Flag.Variants))
	for i, vr := range v.Flag.Variants {
		variants[i] = variantJSON{Key: vr.Key, Name: vr.Name}
	}
	return flagEnvResponse{
		ID:                v.Flag.ID,
		ProjectID:         v.Flag.ProjectID,
		Key:               v.Flag.Key,
		Name:              v.Flag.Name,
		Type:              string(v.Flag.Type),
		Variants:          variants,
		DefaultVariantKey: v.Flag.DefaultVariantKey,
		Enabled:           v.Enabled,
	}
}

func (h *FlagEnvironmentHandler) resolveProjectAndEnv(ctx context.Context, w http.ResponseWriter, projSlug, envSlug string) (*domain.Project, *domain.Environment, bool) {
	proj, err := h.projects.GetBySlug(ctx, projSlug)
	if err != nil {
		WriteError(w, err)
		return nil, nil, false
	}
	env, err := h.envs.GetBySlug(ctx, proj.ID, envSlug)
	if err != nil {
		WriteError(w, err)
		return nil, nil, false
	}
	return proj, env, true
}

func (h *FlagEnvironmentHandler) list(w http.ResponseWriter, r *http.Request) {
	proj, env, ok := h.resolveProjectAndEnv(r.Context(), w, r.PathValue("slug"), r.PathValue("env_slug"))
	if !ok {
		return
	}
	views, err := h.svc.ListByEnvironment(r.Context(), proj.ID, env.ID)
	if err != nil {
		WriteError(w, err)
		return
	}
	items := make([]flagEnvResponse, 0, len(views))
	for _, v := range views {
		items = append(items, toFlagEnvResponse(v))
	}
	writeJSON(w, http.StatusOK, map[string]any{"flags": items})
}

func (h *FlagEnvironmentHandler) setEnabled(w http.ResponseWriter, r *http.Request) {
	proj, env, ok := h.resolveProjectAndEnv(r.Context(), w, r.PathValue("slug"), r.PathValue("env_slug"))
	if !ok {
		return
	}

	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}
	if body.Enabled == nil {
		WriteError(w, newBadRequest("enabled is required"))
		return
	}

	if err := h.svc.SetEnabled(r.Context(), proj.ID, env.ID, r.PathValue("key"), *body.Enabled); err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": *body.Enabled})
}
