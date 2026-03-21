package httpadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/karo/cuttlegate/internal/domain/ports"
)

// flagChangeEvent is a narrow interface for domain events that carry
// flag-state-change data. The SSE handler type-asserts incoming
// ports.DomainEvent values to this interface; events that do not satisfy
// it are silently dropped. This decouples the handler from any concrete
// event struct (which lives in the domain layer and may be introduced
// by a different issue).
type flagChangeEvent interface {
	ProjectSlug() string
	EnvironmentSlug() string
	FlagKey() string
	Enabled() bool
}

// sseEventPayload is the JSON wire format for a single SSE data line.
type sseEventPayload struct {
	Type        string `json:"type"`
	Project     string `json:"project"`
	Environment string `json:"environment"`
	FlagKey     string `json:"flag_key"`
	Enabled     bool   `json:"enabled"`
	OccurredAt  string `json:"occurred_at"`
}

// SSEHandler streams flag state change events to connected clients over
// Server-Sent Events. It subscribes to a Broker, filters events by
// project/environment, and writes SSE-formatted lines.
type SSEHandler struct {
	broker   *Broker
	projects projectResolver
	envs     environmentResolver
}

// NewSSEHandler constructs an SSEHandler.
func NewSSEHandler(broker *Broker, projects projectResolver, envs environmentResolver) *SSEHandler {
	return &SSEHandler{broker: broker, projects: projects, envs: envs}
}

// RegisterRoutes registers the SSE streaming route on mux behind the provided auth middleware.
func (h *SSEHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/projects/{slug}/environments/{env_slug}/flags/stream", auth(http.HandlerFunc(h.stream)))
}

func (h *SSEHandler) stream(w http.ResponseWriter, r *http.Request) {
	projSlug := r.PathValue("slug")
	envSlug := r.PathValue("env_slug")

	_, _, ok := resolveProjectAndEnv(r.Context(), w, h.projects, h.envs, projSlug, envSlug)
	if !ok {
		return
	}

	// Assert http.Flusher support.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Subscribe to broker.
	ch, unsub := h.broker.Subscribe()
	if ch == nil {
		http.Error(w, "broker unavailable", http.StatusInternalServerError)
		return
	}
	defer unsub()

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Accept Last-Event-ID without error — no replay.
	_ = r.Header.Get("Last-Event-ID")

	h.streamLoop(r.Context(), w, flusher, ch, projSlug, envSlug)
}

func (h *SSEHandler) streamLoop(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, ch <-chan ports.DomainEvent, projSlug, envSlug string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-ch:
			if !ok {
				// Broker shut down — channel closed.
				return
			}
			h.handleEvent(w, flusher, event, projSlug, envSlug)

		case <-ticker.C:
			fmt.Fprintf(w, ": keep-alive\n\n") //nolint:errcheck
			flusher.Flush()
		}
	}
}

func (h *SSEHandler) handleEvent(w http.ResponseWriter, flusher http.Flusher, event ports.DomainEvent, projSlug, envSlug string) {
	fce, ok := event.(flagChangeEvent)
	if !ok {
		return
	}

	// Filter: only forward events matching this client's project/environment.
	if fce.ProjectSlug() != projSlug || fce.EnvironmentSlug() != envSlug {
		return
	}

	payload := sseEventPayload{
		Type:        event.EventType(),
		Project:     fce.ProjectSlug(),
		Environment: fce.EnvironmentSlug(),
		FlagKey:     fce.FlagKey(),
		Enabled:     fce.Enabled(),
		OccurredAt:  event.OccurredAt().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("sse: marshal error: %v", err)
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", data) //nolint:errcheck
	flusher.Flush()
}
