package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

// promotionService is the use-case interface required by PromotionHandler.
type promotionService interface {
	PromoteFlagState(ctx context.Context, projectID, sourceEnvID, targetEnvID, flagKey string) (*app.FlagPromotionDiff, error)
	PromoteAllFlags(ctx context.Context, projectID, sourceEnvID, targetEnvID string) ([]*app.FlagPromotionDiff, error)
}

// PromotionHandler handles HTTP requests for flag state promotion.
type PromotionHandler struct {
	svc      promotionService
	projects projectResolver
	envs     environmentResolver
}

// NewPromotionHandler constructs a PromotionHandler.
func NewPromotionHandler(svc promotionService, projects projectResolver, envs environmentResolver) *PromotionHandler {
	return &PromotionHandler{svc: svc, projects: projects, envs: envs}
}

// RegisterRoutes registers promotion endpoints on mux behind the provided auth middleware.
func (h *PromotionHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("POST /api/v1/projects/{slug}/environments/{env_slug}/flags/{key}/promote", auth(http.HandlerFunc(h.promoteFlag)))
	mux.Handle("POST /api/v1/projects/{slug}/environments/{env_slug}/promote", auth(http.HandlerFunc(h.promoteAllFlags)))
}

// flagPromotionDiffJSON is the JSON representation of a FlagPromotionDiff.
type flagPromotionDiffJSON struct {
	FlagKey       string `json:"flag_key"`
	EnabledBefore bool   `json:"enabled_before"`
	EnabledAfter  bool   `json:"enabled_after"`
	RulesAdded    int    `json:"rules_added"`
	RulesRemoved  int    `json:"rules_removed"`
}

func toPromotionDiffJSON(d *app.FlagPromotionDiff) flagPromotionDiffJSON {
	return flagPromotionDiffJSON{
		FlagKey:       d.FlagKey,
		EnabledBefore: d.EnabledBefore,
		EnabledAfter:  d.EnabledAfter,
		RulesAdded:    d.RulesAdded,
		RulesRemoved:  d.RulesRemoved,
	}
}

// resolveTargetEnv decodes the request body, validates target_env_slug, and resolves it
// to an Environment. Returns false and writes an error response if anything fails.
func (h *PromotionHandler) resolveTargetEnv(w http.ResponseWriter, r *http.Request, projectID string) (*domain.Environment, bool) {
	var body struct {
		TargetEnvSlug string `json:"target_env_slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return nil, false
	}
	if body.TargetEnvSlug == "" {
		WriteError(w, newBadRequest("target_env_slug is required"))
		return nil, false
	}
	env, err := h.envs.GetBySlug(r.Context(), projectID, body.TargetEnvSlug)
	if err != nil {
		WriteError(w, err)
		return nil, false
	}
	return env, true
}

func (h *PromotionHandler) promoteFlag(w http.ResponseWriter, r *http.Request) {
	proj, srcEnv, ok := resolveProjectAndEnv(r.Context(), w, h.projects, h.envs, r.PathValue("slug"), r.PathValue("env_slug"))
	if !ok {
		return
	}
	targetEnv, ok := h.resolveTargetEnv(w, r, proj.ID)
	if !ok {
		return
	}
	diff, err := h.svc.PromoteFlagState(r.Context(), proj.ID, srcEnv.ID, targetEnv.ID, r.PathValue("key"))
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toPromotionDiffJSON(diff))
}

func (h *PromotionHandler) promoteAllFlags(w http.ResponseWriter, r *http.Request) {
	proj, srcEnv, ok := resolveProjectAndEnv(r.Context(), w, h.projects, h.envs, r.PathValue("slug"), r.PathValue("env_slug"))
	if !ok {
		return
	}
	targetEnv, ok := h.resolveTargetEnv(w, r, proj.ID)
	if !ok {
		return
	}
	diffs, err := h.svc.PromoteAllFlags(r.Context(), proj.ID, srcEnv.ID, targetEnv.ID)
	if err != nil {
		WriteError(w, err)
		return
	}
	items := make([]flagPromotionDiffJSON, len(diffs))
	for i, d := range diffs {
		items[i] = toPromotionDiffJSON(d)
	}
	writeJSON(w, http.StatusOK, map[string]any{"flags": items})
}
