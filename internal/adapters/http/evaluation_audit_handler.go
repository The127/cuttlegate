package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/The127/cuttlegate/internal/app"
	"github.com/The127/cuttlegate/internal/domain"
	"github.com/The127/cuttlegate/internal/domain/ports"
)

// evaluationAuditService is the use-case interface required by EvaluationAuditHandler.
type evaluationAuditService interface {
	ListEvaluations(ctx context.Context, projectID, environmentID, flagKey string, filter ports.EvaluationFilter) ([]*app.EvaluationEventView, error)
}

// EvaluationAuditHandler handles evaluation audit trail HTTP requests.
type EvaluationAuditHandler struct {
	svc      evaluationAuditService
	projects projectResolver
	envs     environmentResolver
}

// NewEvaluationAuditHandler constructs an EvaluationAuditHandler.
func NewEvaluationAuditHandler(svc evaluationAuditService, projects projectResolver, envs environmentResolver) *EvaluationAuditHandler {
	return &EvaluationAuditHandler{svc: svc, projects: projects, envs: envs}
}

// RegisterRoutes registers the evaluation audit routes on mux.
func (h *EvaluationAuditHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/projects/{slug}/environments/{env_slug}/flags/{key}/evaluations",
		auth(http.HandlerFunc(h.list)))
}

type evalAuditItem struct {
	ID           string      `json:"id"`
	OccurredAt   string      `json:"occurred_at"`
	FlagKey      string      `json:"flag_key"`
	UserID       string      `json:"user_id"`
	InputContext interface{} `json:"input_context"`
	MatchedRule  interface{} `json:"matched_rule"` // null or {"id":"...","name":"..."}
	VariantKey   string      `json:"variant_key"`
	Reason       string      `json:"reason"`
}

type matchedRuleJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type evalAuditListResponse struct {
	Items      []evalAuditItem `json:"items"`
	NextCursor *string         `json:"next_cursor"` // null when no more pages
}

func (h *EvaluationAuditHandler) list(w http.ResponseWriter, r *http.Request) {
	proj, err := h.projects.GetBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		if isNotFound(err) {
			WriteError(w, domain.ErrForbidden)
			return
		}
		WriteError(w, err)
		return
	}
	env, err := h.envs.GetBySlug(r.Context(), proj.ID, r.PathValue("env_slug"))
	if err != nil {
		if isNotFound(err) {
			WriteError(w, domain.ErrForbidden)
			return
		}
		WriteError(w, err)
		return
	}

	filter := ports.EvaluationFilter{}

	if before := r.URL.Query().Get("before"); before != "" {
		t, err := time.Parse(time.RFC3339, before)
		if err != nil {
			WriteError(w, newBadRequest("invalid_cursor"))
			return
		}
		filter.Before = t.UTC()
	}

	views, err := h.svc.ListEvaluations(r.Context(), proj.ID, env.ID, r.PathValue("key"), filter)
	if err != nil {
		WriteError(w, err)
		return
	}

	items := make([]evalAuditItem, len(views))
	for i, v := range views {
		item := evalAuditItem{
			ID:         v.ID,
			OccurredAt: v.OccurredAt.UTC().Format(time.RFC3339),
			FlagKey:    v.FlagKey,
			UserID:     v.UserID,
			Reason:     v.Reason,
			VariantKey: v.VariantKey,
		}

		// InputContext: re-parse the JSON string so it serialises as an object, not a string.
		item.InputContext = jsonRawOrEmpty(v.InputContext)

		// MatchedRule: null when no rule matched.
		if v.MatchedRuleID != "" {
			item.MatchedRule = matchedRuleJSON{ID: v.MatchedRuleID, Name: v.MatchedRuleName}
		}

		items[i] = item
	}

	// Compute next cursor: timestamp of the oldest item in this page.
	var nextCursor *string
	if len(views) == ports.DefaultEvaluationLimit {
		oldest := views[len(views)-1].OccurredAt.UTC().Format(time.RFC3339)
		nextCursor = &oldest
	}

	writeJSON(w, http.StatusOK, evalAuditListResponse{
		Items:      items,
		NextCursor: nextCursor,
	})
}

// isNotFound returns true when err is ErrNotFound.
func isNotFound(err error) bool {
	return errors.Is(err, domain.ErrNotFound)
}

// jsonRawOrEmpty parses s as JSON and returns json.RawMessage so the value
// is embedded as a JSON object rather than a quoted string. Falls back to {}
// on parse failure.
func jsonRawOrEmpty(s string) json.RawMessage {
	if json.Valid([]byte(s)) {
		return json.RawMessage(s)
	}
	return json.RawMessage("{}")
}
