package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

// evaluationService is the use-case interface required by EvaluationHandler.
type evaluationService interface {
	Evaluate(ctx context.Context, projectID, environmentID, flagKey string, evalCtx domain.EvalContext) (*app.EvalView, error)
}

// EvaluationHandler handles flag evaluation HTTP requests.
type EvaluationHandler struct {
	svc      evaluationService
	projects projectResolver
	envs     environmentResolver
}

// NewEvaluationHandler constructs an EvaluationHandler.
func NewEvaluationHandler(svc evaluationService, projects projectResolver, envs environmentResolver) *EvaluationHandler {
	return &EvaluationHandler{svc: svc, projects: projects, envs: envs}
}

// RegisterRoutes registers the evaluation route on mux behind the provided auth middleware.
func (h *EvaluationHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("POST /api/v1/projects/{slug}/environments/{env_slug}/flags/{key}/evaluate",
		auth(http.HandlerFunc(h.evaluate)))
}

type evaluateRequest struct {
	Context *evalContextJSON `json:"context"`
}

type evalContextJSON struct {
	UserID     string            `json:"user_id"`
	Attributes map[string]string `json:"attributes"`
}

type evaluateResponse struct {
	Key     string  `json:"key"`
	Enabled bool    `json:"enabled"`
	Value   *string `json:"value"`
	Reason  string  `json:"reason"`
	Type    string  `json:"type"`
}

func (h *EvaluationHandler) evaluate(w http.ResponseWriter, r *http.Request) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		WriteError(w, err)
		return
	}
	env, err := h.envs.GetBySlug(r.Context(), proj.ID, r.PathValue("env_slug"))
	if err != nil {
		WriteError(w, err)
		return
	}

	var body evaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}
	if body.Context == nil {
		WriteError(w, newBadRequest("context is required"))
		return
	}

	attrs := body.Context.Attributes
	if attrs == nil {
		attrs = map[string]string{}
	}
	evalCtx := domain.EvalContext{
		UserID:     body.Context.UserID,
		Attributes: attrs,
	}

	view, err := h.svc.Evaluate(r.Context(), proj.ID, env.ID, r.PathValue("key"), evalCtx)
	if err != nil {
		WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, evaluateResponse{
		Key:     view.Key,
		Enabled: view.Enabled,
		Value:   view.Value,
		Reason:  string(view.Reason),
		Type:    string(view.Type),
	})
}
