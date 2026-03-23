package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

// apiKeyService is the use-case interface required by APIKeyHandler.
type apiKeyService interface {
	Create(ctx context.Context, projectID, environmentID, name string, tier domain.ToolCapabilityTier) (*app.APIKeyCreateResult, error)
	List(ctx context.Context, projectID, environmentID string) ([]app.APIKeyView, error)
	Revoke(ctx context.Context, id string) error
}

// APIKeyHandler handles HTTP requests for API key management.
type APIKeyHandler struct {
	svc      apiKeyService
	projects projectResolver
	envs     environmentResolver
}

// NewAPIKeyHandler constructs an APIKeyHandler.
func NewAPIKeyHandler(svc apiKeyService, projects projectResolver, envs environmentResolver) *APIKeyHandler {
	return &APIKeyHandler{svc: svc, projects: projects, envs: envs}
}

// RegisterRoutes registers API key management routes on mux behind the provided auth middleware.
func (h *APIKeyHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("POST /api/v1/projects/{slug}/environments/{env_slug}/api-keys",
		auth(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/projects/{slug}/environments/{env_slug}/api-keys",
		auth(http.HandlerFunc(h.list)))
	mux.Handle("DELETE /api/v1/projects/{slug}/environments/{env_slug}/api-keys/{key_id}",
		auth(http.HandlerFunc(h.revoke)))
}

type apiKeyResponse struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	DisplayPrefix  string     `json:"display_prefix"`
	CapabilityTier string     `json:"capability_tier"`
	CreatedAt      time.Time  `json:"created_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}

type apiKeyCreateResponse struct {
	apiKeyResponse
	Key string `json:"key"`
}

func (h *APIKeyHandler) create(w http.ResponseWriter, r *http.Request) {
	proj, env, ok := resolveProjectAndEnv(r.Context(), w, h.projects, h.envs, r.PathValue("slug"), r.PathValue("env_slug"))
	if !ok {
		return
	}

	var body struct {
		Name           string `json:"name"`
		CapabilityTier string `json:"capability_tier"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}
	if body.Name == "" {
		WriteError(w, newBadRequest("name is required"))
		return
	}

	tier := domain.TierRead
	if body.CapabilityTier != "" {
		t := domain.ToolCapabilityTier(body.CapabilityTier)
		if !t.Valid() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_capability_tier","message":"capability_tier must be one of: read, write, destructive"}`))
			return
		}
		tier = t
	}

	result, err := h.svc.Create(r.Context(), proj.ID, env.ID, body.Name, tier)
	if err != nil {
		WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, apiKeyCreateResponse{
		apiKeyResponse: apiKeyResponse{
			ID:             result.ID,
			Name:           result.Name,
			DisplayPrefix:  result.DisplayPrefix,
			CapabilityTier: result.CapabilityTier,
			CreatedAt:      result.CreatedAt.UTC(),
		},
		Key: result.Plaintext,
	})
}

func (h *APIKeyHandler) list(w http.ResponseWriter, r *http.Request) {
	proj, env, ok := resolveProjectAndEnv(r.Context(), w, h.projects, h.envs, r.PathValue("slug"), r.PathValue("env_slug"))
	if !ok {
		return
	}

	views, err := h.svc.List(r.Context(), proj.ID, env.ID)
	if err != nil {
		WriteError(w, err)
		return
	}

	items := make([]apiKeyResponse, len(views))
	for i, v := range views {
		items[i] = apiKeyResponse{
			ID:             v.ID,
			Name:           v.Name,
			DisplayPrefix:  v.DisplayPrefix,
			CapabilityTier: v.CapabilityTier,
			CreatedAt:      v.CreatedAt.UTC(),
			RevokedAt:      v.RevokedAt,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"api_keys": items})
}

func (h *APIKeyHandler) revoke(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Revoke(r.Context(), r.PathValue("key_id")); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
