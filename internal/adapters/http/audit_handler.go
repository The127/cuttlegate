package httpadapter

import (
	"context"
	"encoding/base64"
	"net/http"
	"strconv"
	"time"

	"github.com/The127/cuttlegate/internal/domain"
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

type auditEntryResponse struct {
	ID              string    `json:"id"`
	OccurredAt      time.Time `json:"occurred_at"`
	ActorID         string    `json:"actor_id"`
	ActorEmail      string    `json:"actor_email"`
	Action          string    `json:"action"`
	EntityType      string    `json:"entity_type,omitempty"`
	EntityID        string    `json:"entity_id,omitempty"`
	FlagKey         string    `json:"flag_key,omitempty"`
	EnvironmentSlug string    `json:"environment_slug,omitempty"`
	Source          string    `json:"source,omitempty"`
	BeforeState     string    `json:"before_state,omitempty"`
	AfterState      string    `json:"after_state,omitempty"`
	ProjectSlug     string    `json:"project_slug"`
}

type auditListResponse struct {
	Entries    []auditEntryResponse `json:"entries"`
	NextCursor *string              `json:"next_cursor"`
}

func toAuditEntryResponse(e *domain.AuditEvent, projectSlug string) auditEntryResponse {
	return auditEntryResponse{
		ID:              e.ID,
		OccurredAt:      e.OccurredAt.UTC(),
		ActorID:         e.ActorID,
		ActorEmail:      e.ActorEmail,
		Action:          e.Action,
		EntityType:      e.EntityType,
		EntityID:        e.EntityID,
		FlagKey:         e.EntityKey,
		EnvironmentSlug: e.EnvironmentSlug,
		Source:          e.Source,
		BeforeState:     e.BeforeState,
		AfterState:      e.AfterState,
		ProjectSlug:     projectSlug,
	}
}

// encodeCursor encodes a timestamp as an opaque base64 cursor string.
func encodeCursor(t time.Time) string {
	return base64.RawURLEncoding.EncodeToString([]byte(t.UTC().Format(time.RFC3339Nano)))
}

// decodeCursor decodes an opaque base64 cursor string back to a time.Time.
func decodeCursor(s string) (time.Time, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, string(b))
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

	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		t, err := decodeCursor(cursor)
		if err != nil {
			WriteError(w, newBadRequest("invalid cursor"))
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

	// Normalise the filter so we know the effective limit for cursor logic.
	effective := domain.NormalizeAuditFilter(filter)

	entries := make([]auditEntryResponse, 0, len(events))
	for _, e := range events {
		entries = append(entries, toAuditEntryResponse(e, proj.Slug))
	}

	var nextCursor *string
	if len(events) == effective.Limit {
		c := encodeCursor(events[len(events)-1].OccurredAt)
		nextCursor = &c
	}

	writeJSON(w, http.StatusOK, auditListResponse{
		Entries:    entries,
		NextCursor: nextCursor,
	})
}
