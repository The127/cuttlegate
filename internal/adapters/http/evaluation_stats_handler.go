package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/karo/cuttlegate/internal/app"
)

// evaluationStatsService is the use-case interface required by EvaluationStatsHandler.
type evaluationStatsService interface {
	GetStats(ctx context.Context, projectID, environmentID, flagKey string) (*app.FlagStatsView, error)
	GetBuckets(ctx context.Context, projectID, environmentID, envSlug, flagKey, window, bucketSize string) (*app.EvaluationBucketsView, error)
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
// Both routes use OIDC-only auth — the caller must pass RequireBearer (not RequireBearerOrAPIKey).
func (h *EvaluationStatsHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/projects/{slug}/environments/{env_slug}/flags/{key}/stats",
		auth(http.HandlerFunc(h.getStats)))
	mux.Handle("GET /api/v1/projects/{slug}/environments/{env_slug}/flags/{key}/stats/buckets",
		auth(http.HandlerFunc(h.getBuckets)))
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

// bucketResponse is the JSON shape for a single time bucket.
type bucketResponse struct {
	TS       string           `json:"ts"`
	Total    int64            `json:"total"`
	Variants map[string]int64 `json:"variants"`
}

// bucketsResponse is the JSON shape for the GET .../stats/buckets response.
type bucketsResponse struct {
	FlagKey     string           `json:"flag_key"`
	Environment string           `json:"environment"`
	Window      string           `json:"window"`
	BucketSize  string           `json:"bucket_size"`
	Buckets     []bucketResponse `json:"buckets"`
}

func (h *EvaluationStatsHandler) getBuckets(w http.ResponseWriter, r *http.Request) {
	window := r.URL.Query().Get("window")
	bucketSize := r.URL.Query().Get("bucket")
	if window == "" || bucketSize == "" {
		writeInvalidParameter(w)
		return
	}

	proj, env, ok := resolveProjectAndEnv(r.Context(), w, h.projects, h.envs, r.PathValue("slug"), r.PathValue("env_slug"))
	if !ok {
		return
	}

	view, err := h.svc.GetBuckets(r.Context(), proj.ID, env.ID, env.Slug, r.PathValue("key"), window, bucketSize)
	if err != nil {
		if errors.Is(err, app.ErrInvalidParameter) {
			writeInvalidParameter(w)
			return
		}
		WriteError(w, err)
		return
	}

	buckets := make([]bucketResponse, len(view.Buckets))
	for i, b := range view.Buckets {
		variants := b.Variants
		if variants == nil {
			variants = map[string]int64{}
		}
		buckets[i] = bucketResponse{
			TS:       b.Timestamp.UTC().Format(time.RFC3339),
			Total:    b.Total,
			Variants: variants,
		}
	}

	writeJSON(w, http.StatusOK, bucketsResponse{
		FlagKey:     view.FlagKey,
		Environment: view.Environment,
		Window:      view.Window,
		BucketSize:  view.BucketSize,
		Buckets:     buckets,
	})
}

// writeInvalidParameter writes a JSON 400 response for invalid query parameters.
func writeInvalidParameter(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(`{"error":"invalid_parameter"}` + "\n")) //nolint:errcheck
}
