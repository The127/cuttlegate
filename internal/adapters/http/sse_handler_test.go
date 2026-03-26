package httpadapter_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpadapter "github.com/The127/cuttlegate/internal/adapters/http"
	"github.com/The127/cuttlegate/internal/domain"
)

// flagTestEvent is a DomainEvent that also satisfies the flagChangeEvent
// interface expected by the SSE handler for filtering and payload extraction.
type flagTestEvent struct {
	occurredAt  time.Time
	projectSlug string
	envSlug     string
	flagKey     string
	enabled     bool
}

func (e flagTestEvent) EventType() string     { return "flag.state_changed" }
func (e flagTestEvent) OccurredAt() time.Time { return e.occurredAt }
func (e flagTestEvent) ProjectSlug() string   { return e.projectSlug }
func (e flagTestEvent) EnvironmentSlug() string {
	return e.envSlug
}
func (e flagTestEvent) FlagKey() string { return e.flagKey }
func (e flagTestEvent) Enabled() bool   { return e.enabled }

func newSSEMux(broker *httpadapter.Broker, auth func(http.Handler) http.Handler) *http.ServeMux {
	proj := &fakeProjResolver{projects: map[string]*domain.Project{
		"alpha": {ID: "proj-alpha", Name: "Alpha", Slug: "alpha"},
	}}
	envs := &fakeEnvResolver{envs: map[string]*domain.Environment{
		"proj-alpha/staging":    {ID: "env-staging", ProjectID: "proj-alpha", Slug: "staging", Name: "Staging"},
		"proj-alpha/production": {ID: "env-prod", ProjectID: "proj-alpha", Slug: "production", Name: "Production"},
	}}
	mux := http.NewServeMux()
	httpadapter.NewSSEHandler(broker, proj, envs).RegisterRoutes(mux, auth)
	return mux
}

func noAuth(h http.Handler) http.Handler { return h }

func TestSSEHandler_StreamsEventToClient(t *testing.T) {
	broker := httpadapter.NewBroker(8)
	defer broker.Shutdown()

	mux := newSSEMux(broker, noAuth)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/v1/projects/alpha/environments/staging/flags/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET stream: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/event-stream")
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", cc, "no-cache")
	}

	ts := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	evt := flagTestEvent{
		occurredAt:  ts,
		projectSlug: "alpha",
		envSlug:     "staging",
		flagKey:     "dark-mode",
		enabled:     true,
	}
	if err := broker.Publish(context.Background(), evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		var got map[string]any
		if err := json.Unmarshal([]byte(payload), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got["type"] != "flag.state_changed" {
			t.Errorf("type = %v, want flag.state_changed", got["type"])
		}
		if got["project"] != "alpha" {
			t.Errorf("project = %v, want alpha", got["project"])
		}
		if got["environment"] != "staging" {
			t.Errorf("environment = %v, want staging", got["environment"])
		}
		if got["flag_key"] != "dark-mode" {
			t.Errorf("flag_key = %v, want dark-mode", got["flag_key"])
		}
		if got["enabled"] != true {
			t.Errorf("enabled = %v, want true", got["enabled"])
		}
		if got["occurred_at"] != "2026-03-21T12:00:00Z" {
			t.Errorf("occurred_at = %v, want 2026-03-21T12:00:00Z", got["occurred_at"])
		}
		return // success
	}
	t.Fatal("no SSE data line received")
}

func TestSSEHandler_FiltersEventsByProjectAndEnvironment(t *testing.T) {
	broker := httpadapter.NewBroker(8)
	defer broker.Shutdown()

	mux := newSSEMux(broker, noAuth)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Connect to staging.
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/v1/projects/alpha/environments/staging/flags/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET stream: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	// Publish an event for production — should NOT reach the staging client.
	prodEvt := flagTestEvent{
		occurredAt:  time.Now(),
		projectSlug: "alpha",
		envSlug:     "production",
		flagKey:     "beta-flag",
		enabled:     false,
	}
	if err := broker.Publish(context.Background(), prodEvt); err != nil {
		t.Fatalf("Publish prod: %v", err)
	}

	// Publish an event for staging — should reach the client.
	stagingEvt := flagTestEvent{
		occurredAt:  time.Now(),
		projectSlug: "alpha",
		envSlug:     "staging",
		flagKey:     "dark-mode",
		enabled:     true,
	}
	if err := broker.Publish(context.Background(), stagingEvt); err != nil {
		t.Fatalf("Publish staging: %v", err)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		var got map[string]any
		if err := json.Unmarshal([]byte(payload), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		// The first data line must be the staging event, not the production event.
		if got["environment"] != "staging" {
			t.Errorf("received event for environment %v, want staging", got["environment"])
		}
		if got["flag_key"] != "dark-mode" {
			t.Errorf("flag_key = %v, want dark-mode", got["flag_key"])
		}
		return
	}
	t.Fatal("no SSE data line received")
}

func TestSSEHandler_InvalidProjectReturns404(t *testing.T) {
	broker := httpadapter.NewBroker(8)
	defer broker.Shutdown()

	mux := newSSEMux(broker, noAuth)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/projects/nonexistent/environments/staging/flags/stream")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestSSEHandler_InvalidEnvironmentReturns404(t *testing.T) {
	broker := httpadapter.NewBroker(8)
	defer broker.Shutdown()

	mux := newSSEMux(broker, noAuth)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/projects/alpha/environments/nonexistent/flags/stream")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestSSEHandler_LastEventIDAccepted(t *testing.T) {
	broker := httpadapter.NewBroker(8)
	defer broker.Shutdown()

	mux := newSSEMux(broker, noAuth)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/v1/projects/alpha/environments/staging/flags/stream", nil)
	req.Header.Set("Last-Event-ID", "abc123")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

// nonFlusherWriter is a ResponseWriter that deliberately does not implement http.Flusher.
type nonFlusherWriter struct {
	code   int
	header http.Header
}

func (w *nonFlusherWriter) Header() http.Header       { return w.header }
func (w *nonFlusherWriter) Write([]byte) (int, error) { return 0, nil }
func (w *nonFlusherWriter) WriteHeader(code int)      { w.code = code }

func TestSSEHandler_FlusherNotAvailableReturns500(t *testing.T) {
	broker := httpadapter.NewBroker(8)
	defer broker.Shutdown()

	proj := &fakeProjResolver{projects: map[string]*domain.Project{
		"alpha": {ID: "proj-alpha", Name: "Alpha", Slug: "alpha"},
	}}
	envs := &fakeEnvResolver{envs: map[string]*domain.Environment{
		"proj-alpha/staging": {ID: "env-staging", ProjectID: "proj-alpha", Slug: "staging", Name: "Staging"},
	}}
	handler := httpadapter.NewSSEHandler(broker, proj, envs)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, noAuth)

	w := &nonFlusherWriter{header: make(http.Header)}
	r := httptest.NewRequest("GET", "/api/v1/projects/alpha/environments/staging/flags/stream", nil)
	mux.ServeHTTP(w, r)

	if w.code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.code)
	}
}
