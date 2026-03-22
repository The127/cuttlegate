package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

// evaluationService is the use-case interface required by EvaluationHandler.
type evaluationService interface {
	Evaluate(ctx context.Context, projectID, environmentID, flagKey string, evalCtx domain.EvalContext) (*app.EvalView, error)
	EvaluateAll(ctx context.Context, projectID, environmentID string, evalCtx domain.EvalContext) ([]app.EvalView, time.Time, error)
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

// RegisterRoutes registers the evaluation routes on mux behind the provided auth middleware.
func (h *EvaluationHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("POST /api/v1/projects/{slug}/environments/{env_slug}/flags/{key}/evaluate",
		auth(http.HandlerFunc(h.evaluate)))
	mux.Handle("POST /api/v1/projects/{slug}/environments/{env_slug}/evaluate",
		auth(http.HandlerFunc(h.evaluateAll)))
}

type evaluateRequest struct {
	Context *evalContextJSON `json:"context"`
}

type evalContextJSON struct {
	UserID     string            `json:"user_id"`
	Attributes map[string]string `json:"attributes"`
}

type evaluateResponse struct {
	Key      string  `json:"key"`
	Enabled  bool    `json:"enabled"`
	Value    *string `json:"value"`
	ValueKey string  `json:"value_key"`
	Reason   string  `json:"reason"`
	Type     string  `json:"type"`
}

func (h *EvaluationHandler) resolveEvalRequest(w http.ResponseWriter, r *http.Request) (*domain.Project, *domain.Environment, domain.EvalContext, bool) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			WriteError(w, domain.ErrForbidden)
			return nil, nil, domain.EvalContext{}, false
		}
		WriteError(w, err)
		return nil, nil, domain.EvalContext{}, false
	}
	env, err := h.envs.GetBySlug(r.Context(), proj.ID, r.PathValue("env_slug"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			WriteError(w, domain.ErrForbidden)
			return nil, nil, domain.EvalContext{}, false
		}
		WriteError(w, err)
		return nil, nil, domain.EvalContext{}, false
	}

	if !apiKeyScopeAllows(r.Context(), proj.ID, env.ID) {
		WriteError(w, domain.ErrForbidden)
		return nil, nil, domain.EvalContext{}, false
	}

	var body evaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return nil, nil, domain.EvalContext{}, false
	}
	if body.Context == nil {
		WriteError(w, newBadRequest("context is required"))
		return nil, nil, domain.EvalContext{}, false
	}

	attrs := body.Context.Attributes
	if attrs == nil {
		attrs = map[string]string{}
	}
	evalCtx := domain.EvalContext{
		UserID:     body.Context.UserID,
		Attributes: attrs,
	}

	return proj, env, evalCtx, true
}

func (h *EvaluationHandler) evaluate(w http.ResponseWriter, r *http.Request) {
	proj, env, evalCtx, ok := h.resolveEvalRequest(w, r)
	if !ok {
		return
	}

	view, err := h.svc.Evaluate(r.Context(), proj.ID, env.ID, r.PathValue("key"), evalCtx)
	if err != nil {
		WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, evaluateResponse{
		Key:      view.Key,
		Enabled:  view.Enabled,
		Value:    view.Value,
		ValueKey: view.ValueKey,
		Reason:   string(view.Reason),
		Type:     string(view.Type),
	})
}

type evaluateAllResponse struct {
	Flags       []evaluateResponse `json:"flags"`
	EvaluatedAt string             `json:"evaluated_at"`
}

func (h *EvaluationHandler) evaluateAll(w http.ResponseWriter, r *http.Request) {
	proj, env, evalCtx, ok := h.resolveEvalRequest(w, r)
	if !ok {
		return
	}

	views, evaluatedAt, err := h.svc.EvaluateAll(r.Context(), proj.ID, env.ID, evalCtx)
	if err != nil {
		WriteError(w, err)
		return
	}

	flags := make([]evaluateResponse, len(views))
	for i, v := range views {
		flags[i] = evaluateResponse{
			Key:      v.Key,
			Enabled:  v.Enabled,
			Value:    v.Value,
			ValueKey: v.ValueKey,
			Reason:   string(v.Reason),
			Type:     string(v.Type),
		}
	}

	writeJSON(w, http.StatusOK, evaluateAllResponse{
		Flags:       flags,
		EvaluatedAt: evaluatedAt.Format(time.RFC3339),
	})
}
