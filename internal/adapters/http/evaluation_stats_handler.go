package httpadapter

import (
	"context"
	"net/http"
	"time"

	"github.com/karo/cuttlegate/internal/app"
)

// evaluationStatsService is the use-case interface required by EvaluationStatsHandler.
type evaluationStatsService interface {
	GetStats(ctx context.Context, projectID, environmentID, flagKey string) (*app.FlagStatsView, error)
}

// EvaluationStatsHandler handles HTTP requests for flag evaluation statistics.
type EvaluationStatsHandler struct {
	svc      evaluationStatsService
	projects projectResolver
	envs     environmentResolver
}

// NewEvaluationStatsHandler constructs an EvaluationStatsHandler.
func NewEvaluationStatsHandler(svc evaluationStatsService, projects projectResolver, envs environmentResolver) *EvaluationStatsHandler {
	return &EvaluationStatsHandler{svc: svc, projects: projects, envs: envs}
}

// RegisterRoutes registers the evaluation stats routes on mux behind the provided auth middleware.
func (h *EvaluationStatsHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/projects/{slug}/environments/{env_slug}/flags/{key}/stats",
		auth(http.HandlerFunc(h.getStats)))
}

type flagStatsResponse struct {
	LastEvaluatedAt *string `json:"last_evaluated_at"`
	EvaluationCount int64   `json:"evaluation_count"`
}

func (h *EvaluationStatsHandler) getStats(w http.ResponseWriter, r *http.Request) {
	proj, env, ok := resolveProjectAndEnv(r.Context(), w, h.projects, h.envs, r.PathValue("slug"), r.PathValue("env_slug"))
	if !ok {
		return
	}

	view, err := h.svc.GetStats(r.Context(), proj.ID, env.ID, r.PathValue("key"))
	if err != nil {
		WriteError(w, err)
		return
	}

	resp := flagStatsResponse{
		EvaluationCount: view.EvaluationCount,
	}
	if view.LastEvaluatedAt != nil {
		s := view.LastEvaluatedAt.Format(time.RFC3339)
		resp.LastEvaluatedAt = &s
	}

	writeJSON(w, http.StatusOK, resp)
}
