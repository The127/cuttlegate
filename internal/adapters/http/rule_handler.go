package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
)

// ruleService is the use-case interface required by RuleHandler.
type ruleService interface {
	Create(ctx context.Context, flagID, environmentID string, priority int, conditions []domain.Condition, variantKey, name string) (*domain.Rule, error)
	List(ctx context.Context, flagID, environmentID string) ([]*domain.Rule, error)
	Update(ctx context.Context, rule *domain.Rule) (*domain.Rule, error)
	Delete(ctx context.Context, id string) error
}

// flagResolver resolves a flag key within a project to a domain.Flag.
type flagResolver interface {
	GetByKey(ctx context.Context, projectID, key string) (*domain.Flag, error)
}

// RuleHandler handles HTTP requests for targeting rule CRUD.
type RuleHandler struct {
	svc      ruleService
	projects projectResolver
	flags    flagResolver
	envs     environmentResolver
}

// NewRuleHandler constructs a RuleHandler.
func NewRuleHandler(svc ruleService, projects projectResolver, flags flagResolver, envs environmentResolver) *RuleHandler {
	return &RuleHandler{svc: svc, projects: projects, flags: flags, envs: envs}
}

// RegisterRoutes registers all rule routes on mux behind the provided auth middleware.
func (h *RuleHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("POST /api/v1/projects/{slug}/flags/{key}/environments/{env_slug}/rules", auth(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/projects/{slug}/flags/{key}/environments/{env_slug}/rules", auth(http.HandlerFunc(h.list)))
	mux.Handle("PATCH /api/v1/projects/{slug}/flags/{key}/environments/{env_slug}/rules/{ruleID}", auth(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/projects/{slug}/flags/{key}/environments/{env_slug}/rules/{ruleID}", auth(http.HandlerFunc(h.delete)))
}

// conditionJSON is the JSON representation of a domain.Condition.
type conditionJSON struct {
	Attribute string   `json:"attribute"`
	Operator  string   `json:"operator"`
	Values    []string `json:"values"`
}

// ruleResponse is the JSON representation of a domain.Rule.
type ruleResponse struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Priority   int             `json:"priority"`
	Conditions []conditionJSON `json:"conditions"`
	VariantKey string          `json:"variantKey"`
	Enabled    bool            `json:"enabled"`
	CreatedAt  time.Time       `json:"createdAt"`
}

func toRuleResponse(r *domain.Rule) ruleResponse {
	conditions := make([]conditionJSON, len(r.Conditions))
	for i, c := range r.Conditions {
		conditions[i] = conditionJSON{
			Attribute: c.Attribute,
			Operator:  string(c.Operator),
			Values:    c.Values,
		}
	}
	return ruleResponse{
		ID:         r.ID,
		Name:       r.Name,
		Priority:   r.Priority,
		Conditions: conditions,
		VariantKey: r.VariantKey,
		Enabled:    r.Enabled,
		CreatedAt:  r.CreatedAt.UTC(),
	}
}

func conditionsFromJSON(raw []conditionJSON) []domain.Condition {
	conditions := make([]domain.Condition, len(raw))
	for i, c := range raw {
		conditions[i] = domain.Condition{
			Attribute: c.Attribute,
			Operator:  domain.Operator(c.Operator),
			Values:    c.Values,
		}
	}
	return conditions
}

// resolveContext resolves project, flag, and environment from path values.
// Returns false and writes an error response if any lookup fails.
func (h *RuleHandler) resolveContext(ctx context.Context, w http.ResponseWriter, projSlug, flagKey, envSlug string) (*domain.Project, *domain.Flag, *domain.Environment, bool) {
	proj, env, ok := resolveProjectAndEnv(ctx, w, h.projects, h.envs, projSlug, envSlug)
	if !ok {
		return nil, nil, nil, false
	}
	flag, err := h.flags.GetByKey(ctx, proj.ID, flagKey)
	if err != nil {
		WriteError(w, err)
		return nil, nil, nil, false
	}
	return proj, flag, env, true
}

func (h *RuleHandler) create(w http.ResponseWriter, r *http.Request) {
	_, flag, env, ok := h.resolveContext(r.Context(), w, r.PathValue("slug"), r.PathValue("key"), r.PathValue("env_slug"))
	if !ok {
		return
	}

	var body struct {
		Conditions []conditionJSON `json:"conditions"`
		VariantKey string          `json:"variantKey"`
		Priority   int             `json:"priority"`
		Name       string          `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}

	rule, err := h.svc.Create(r.Context(), flag.ID, env.ID, body.Priority, conditionsFromJSON(body.Conditions), body.VariantKey, body.Name)
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toRuleResponse(rule))
}

func (h *RuleHandler) list(w http.ResponseWriter, r *http.Request) {
	_, flag, env, ok := h.resolveContext(r.Context(), w, r.PathValue("slug"), r.PathValue("key"), r.PathValue("env_slug"))
	if !ok {
		return
	}
	rules, err := h.svc.List(r.Context(), flag.ID, env.ID)
	if err != nil {
		WriteError(w, err)
		return
	}
	items := make([]ruleResponse, 0, len(rules))
	for _, rule := range rules {
		items = append(items, toRuleResponse(rule))
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": items})
}

func (h *RuleHandler) update(w http.ResponseWriter, r *http.Request) {
	_, flag, env, ok := h.resolveContext(r.Context(), w, r.PathValue("slug"), r.PathValue("key"), r.PathValue("env_slug"))
	if !ok {
		return
	}

	var body struct {
		Conditions []conditionJSON `json:"conditions"`
		VariantKey string          `json:"variantKey"`
		Priority   int             `json:"priority"`
		Enabled    bool            `json:"enabled"`
		Name       string          `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, newBadRequest("invalid request body"))
		return
	}

	rule := &domain.Rule{
		ID:            r.PathValue("ruleID"),
		FlagID:        flag.ID,
		EnvironmentID: env.ID,
		Name:          body.Name,
		Conditions:    conditionsFromJSON(body.Conditions),
		VariantKey:    body.VariantKey,
		Priority:      body.Priority,
		Enabled:       body.Enabled,
	}

	updated, err := h.svc.Update(r.Context(), rule)
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toRuleResponse(updated))
}

func (h *RuleHandler) delete(w http.ResponseWriter, r *http.Request) {
	_, _, _, ok := h.resolveContext(r.Context(), w, r.PathValue("slug"), r.PathValue("key"), r.PathValue("env_slug"))
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), r.PathValue("ruleID")); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
