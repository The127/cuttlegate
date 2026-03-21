package httpadapter

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
)

// auditService is the use-case interface required by AuditHandler.
type auditService interface {
	ListByProject(ctx context.Context, projectID string, filter domain.AuditFilter) ([]*domain.AuditEvent, error)
}

// AuditHandler handles HTTP requests for the audit log.
type AuditHandler struct {
	svc     auditService
	projSvc projectResolver
}

// NewAuditHandler constructs an AuditHandler.
func NewAuditHandler(svc auditService, projSvc projectResolver) *AuditHandler {
	return &AuditHandler{svc: svc, projSvc: projSvc}
}

// RegisterRoutes registers audit routes on mux behind the provided auth middleware.
func (h *AuditHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/projects/{slug}/audit", auth(http.HandlerFunc(h.list)))
}

type auditEventResponse struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ActorID     string    `json:"actor_id"`
	Action      string    `json:"action"`
	EntityType  string    `json:"entity_type"`
	EntityID    string    `json:"entity_id"`
	EntityKey   string    `json:"entity_key,omitempty"`
	BeforeState string    `json:"before_state,omitempty"`
	AfterState  string    `json:"after_state,omitempty"`
	OccurredAt  time.Time `json:"occurred_at"`
}

func toAuditEventResponse(e *domain.AuditEvent) auditEventResponse {
	return auditEventResponse{
		ID:          e.ID,
		ProjectID:   e.ProjectID,
		ActorID:     e.ActorID,
		Action:      e.Action,
		EntityType:  e.EntityType,
		EntityID:    e.EntityID,
		EntityKey:   e.EntityKey,
		BeforeState: e.BeforeState,
		AfterState:  e.AfterState,
		OccurredAt:  e.OccurredAt.UTC(),
	}
}

func (h *AuditHandler) list(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	proj, err := h.projSvc.GetBySlug(r.Context(), slug)
	if err != nil {
		WriteError(w, err)
		return
	}

	filter := domain.AuditFilter{
		FlagKey: r.URL.Query().Get("flag_key"),
	}
	if before := r.URL.Query().Get("before"); before != "" {
		t, err := time.Parse(time.RFC3339, before)
		if err != nil {
			WriteError(w, newBadRequest("invalid before parameter: must be RFC3339"))
			return
		}
		filter.Before = t
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			WriteError(w, newBadRequest("invalid limit parameter: must be a positive integer"))
			return
		}
		filter.Limit = limit
	}

	events, err := h.svc.ListByProject(r.Context(), proj.ID, filter)
	if err != nil {
		WriteError(w, err)
		return
	}

	items := make([]auditEventResponse, 0, len(events))
	for _, e := range events {
		items = append(items, toAuditEventResponse(e))
	}
	writeJSON(w, http.StatusOK, map[string]any{"audit_events": items})
}
