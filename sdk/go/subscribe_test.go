package cuttlegate_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"context"

	cuttlegate "github.com/karo/cuttlegate/sdk/go"
)

// sseEvent formats a single SSE data line.
func sseEvent(flagKey string, enabled bool, occurredAt string) string {
	return fmt.Sprintf(
		`data: {"type":"flag.state_changed","project":"my-project","environment":"production","flag_key":%q,"enabled":%v,"occurred_at":%q}`+"\n\n",
		flagKey, enabled, occurredAt,
	)
}

// serveSSE starts a test server that writes the provided SSE lines then blocks
// until the client disconnects (or ctx is cancelled).
func serveSSE(t *testing.T, lines []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flusher", 500)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		for _, line := range lines {
			fmt.Fprint(w, line)
			flusher.Flush()
		}
		// Block until the client disconnects.
		<-r.Context().Done()
	}))
}

// serveSSEStatus starts a test server that always responds with the given HTTP status.
func serveSSEStatus(t *testing.T, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
}

func newSubscribeClient(t *testing.T, baseURL string) cuttlegate.Client {
	t.Helper()
	c, err := cuttlegate.NewClient(cuttlegate.Config{
		BaseURL:      baseURL,
		ServiceToken: "svc_test",
		Project:      "my-project",
		Environment:  "production",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// drainAfterCancel reads both channels to exhaustion after cancelling ctx.
// Returns the collected updates and errors. Uses a short deadline to prevent
// test hangs — channels must close promptly after cancel.
func drainAfterCancel(t *testing.T, cancel context.CancelFunc, updates <-chan cuttlegate.FlagUpdate, errs <-chan error) ([]cuttlegate.FlagUpdate, []error) {
	t.Helper()
	cancel()

	deadline := time.After(2 * time.Second)
	var gotUpdates []cuttlegate.FlagUpdate
	var gotErrs []error

	updatesDone := false
	errsDone := false
	for !updatesDone || !errsDone {
		select {
		case u, ok := <-updates:
			if !ok {
				updatesDone = true
				updates = nil
				continue
			}
			gotUpdates = append(gotUpdates, u)
		case e, ok := <-errs:
			if !ok {
				errsDone = true
				errs = nil
				continue
			}
			gotErrs = append(gotErrs, e)
		case <-deadline:
			t.Error("timed out waiting for channels to close after context cancel")
			return gotUpdates, gotErrs
		}
	}
	return gotUpdates, gotErrs
}

// --- @happy: receives flag update ---

func TestSubscribe_ReceivesFlagUpdate(t *testing.T) {
	// @happy: Subscribe receives a flag.state_changed event and delivers it.
	occurredAt := "2026-03-22T10:00:00Z"
	srv := serveSSE(t, []string{sseEvent("dark-mode", true, occurredAt)})
	defer srv.Close()

	c := newSubscribeClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updates, errs, err := c.Subscribe(ctx, "dark-mode")
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	select {
	case u := <-updates:
		if u.Key != "dark-mode" {
			t.Errorf("expected Key=dark-mode, got %q", u.Key)
		}
		if !u.Enabled {
			t.Errorf("expected Enabled=true")
		}
		expectedAt, _ := time.Parse(time.RFC3339, occurredAt)
		if !u.UpdatedAt.Equal(expectedAt) {
			t.Errorf("expected UpdatedAt=%v, got %v", expectedAt, u.UpdatedAt)
		}
	case e := <-errs:
		t.Fatalf("unexpected error: %v", e)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for flag update")
	}
}

// --- @edge: context cancel closes both channels ---

func TestSubscribe_ContextCancel_ClosesBothChannels(t *testing.T) {
	// @edge: cancelling ctx must close both channels cleanly.
	srv := serveSSE(t, nil) // no events — just blocks
	defer srv.Close()

	c := newSubscribeClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())

	updates, errs, err := c.Subscribe(ctx, "dark-mode")
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	_, _ = drainAfterCancel(t, cancel, updates, errs)
	// drainAfterCancel verifies both channels closed; if it didn't time out, we pass.
}

// --- @edge: key filtering ---

func TestSubscribe_FiltersOtherKeys(t *testing.T) {
	// @edge: events for other flag keys must be dropped.
	occurredAt := "2026-03-22T10:00:00Z"
	srv := serveSSE(t, []string{
		sseEvent("other-flag", true, occurredAt),   // should be filtered
		sseEvent("dark-mode", false, occurredAt),   // should be delivered
	})
	defer srv.Close()

	c := newSubscribeClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updates, errs, err := c.Subscribe(ctx, "dark-mode")
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	select {
	case u := <-updates:
		if u.Key != "dark-mode" {
			t.Errorf("expected filtered update for dark-mode, got key=%q", u.Key)
		}
	case e := <-errs:
		t.Fatalf("unexpected error: %v", e)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for filtered update")
	}
}

// --- @auth-bypass: 401 closes channels with AuthError ---

func TestSubscribe_401_ClosesWithAuthError(t *testing.T) {
	// @auth-bypass: 401 must deliver AuthError and close both channels.
	srv := serveSSEStatus(t, 401)
	defer srv.Close()

	c := newSubscribeClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updates, errs, err := c.Subscribe(ctx, "dark-mode")
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	// Collect errors until both channels close.
	var gotErrs []error
	deadline := time.After(2 * time.Second)
	updatesDone, errsDone := false, false
	for !updatesDone || !errsDone {
		select {
		case _, ok := <-updates:
			if !ok {
				updatesDone = true
				updates = nil
			}
		case e, ok := <-errs:
			if !ok {
				errsDone = true
				errs = nil
				continue
			}
			gotErrs = append(gotErrs, e)
		case <-deadline:
			t.Fatal("timed out waiting for channels to close on 401")
		}
	}

	var authErr *cuttlegate.AuthError
	found := false
	for _, e := range gotErrs {
		if errors.As(e, &authErr) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected *AuthError in error channel, got: %v", gotErrs)
	}
	if authErr != nil && authErr.StatusCode != 401 {
		t.Errorf("expected StatusCode 401, got %d", authErr.StatusCode)
	}
}

// --- @auth-bypass: 403 closes channels with AuthError ---

func TestSubscribe_403_ClosesWithAuthError(t *testing.T) {
	// @auth-bypass: 403 must deliver AuthError and close both channels.
	srv := serveSSEStatus(t, 403)
	defer srv.Close()

	c := newSubscribeClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updates, errs, err := c.Subscribe(ctx, "dark-mode")
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	var gotErrs []error
	deadline := time.After(2 * time.Second)
	updatesDone, errsDone := false, false
	for !updatesDone || !errsDone {
		select {
		case _, ok := <-updates:
			if !ok {
				updatesDone = true
				updates = nil
			}
		case e, ok := <-errs:
			if !ok {
				errsDone = true
				errs = nil
				continue
			}
			gotErrs = append(gotErrs, e)
		case <-deadline:
			t.Fatal("timed out waiting for channels to close on 403")
		}
	}

	var authErr *cuttlegate.AuthError
	found := false
	for _, e := range gotErrs {
		if errors.As(e, &authErr) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected *AuthError in error channel, got: %v", gotErrs)
	}
	if authErr != nil && authErr.StatusCode != 403 {
		t.Errorf("expected StatusCode 403, got %d", authErr.StatusCode)
	}
}

// --- @error-path: 5xx triggers reconnect ---

func TestSubscribe_5xx_Reconnects(t *testing.T) {
	// @error-path: 5xx must trigger reconnect; SDK must eventually deliver
	// the update after server recovers.
	occurredAt := "2026-03-22T10:00:00Z"
	attempt := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt == 1 {
			w.WriteHeader(500)
			return
		}
		// Second attempt: serve the SSE event.
		flusher, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()
		fmt.Fprint(w, sseEvent("dark-mode", true, occurredAt))
		flusher.Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := newSubscribeClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updates, errs, err := c.Subscribe(ctx, "dark-mode")
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	select {
	case u := <-updates:
		if u.Key != "dark-mode" {
			t.Errorf("expected dark-mode, got %q", u.Key)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for update after reconnect")
	// drain errors silently — we expect a ServerError on attempt 1
	case e := <-errs:
		var srvErr *cuttlegate.ServerError
		if !errors.As(e, &srvErr) {
			t.Fatalf("unexpected non-ServerError: %v", e)
		}
		// ServerError expected on first attempt — wait for the update on reconnect.
		select {
		case u := <-updates:
			if u.Key != "dark-mode" {
				t.Errorf("expected dark-mode, got %q", u.Key)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for update after reconnect")
		}
	}
}

// --- @error-path: invalid JSON in SSE data ---

func TestSubscribe_InvalidJSON_NonFatalError(t *testing.T) {
	// @error-path: malformed JSON in data line must send a non-fatal error
	// and not close the stream.
	occurredAt := "2026-03-22T10:00:00Z"
	srv := serveSSE(t, []string{
		"data: not-valid-json\n\n",
		sseEvent("dark-mode", true, occurredAt),
	})
	defer srv.Close()

	c := newSubscribeClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updates, errs, err := c.Subscribe(ctx, "dark-mode")
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	// We should get a non-fatal error followed by the valid update.
	gotError := false
	deadline := time.After(2 * time.Second)
	for {
		select {
		case u, ok := <-updates:
			if !ok {
				t.Fatal("updates channel closed unexpectedly")
			}
			if u.Key == "dark-mode" {
				// Valid update received — test passes.
				return
			}
		case _, ok := <-errs:
			if !ok {
				if !gotError {
					t.Fatal("error channel closed without delivering a non-fatal error")
				}
				return
			}
			gotError = true
		case <-deadline:
			t.Fatal("timed out waiting for non-fatal error and subsequent update")
		}
	}
}

// --- @happy: multiple independent Subscribe calls ---

func TestSubscribe_MultipleIndependentStreams(t *testing.T) {
	// @happy: two Subscribe calls on the same key return independent streams.
	occurredAt := "2026-03-22T10:00:00Z"
	srv := serveSSE(t, []string{
		sseEvent("dark-mode", true, occurredAt),
	})
	defer srv.Close()

	c := newSubscribeClient(t, srv.URL)
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()

	updates1, _, err := c.Subscribe(ctx1, "dark-mode")
	if err != nil {
		t.Fatalf("Subscribe 1 returned error: %v", err)
	}
	updates2, _, err := c.Subscribe(ctx2, "dark-mode")
	if err != nil {
		t.Fatalf("Subscribe 2 returned error: %v", err)
	}

	// Cancel stream 1 — stream 2 must still work.
	cancel1()

	// Drain stream 1 channels.
	deadline := time.After(2 * time.Second)
	select {
	case <-updates1:
	case <-deadline:
		// It's okay if updates1 delivers nothing before closing.
	}

	// Stream 2 should still receive updates.
	select {
	case u := <-updates2:
		if u.Key != "dark-mode" {
			t.Errorf("stream 2: expected dark-mode, got %q", u.Key)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stream 2: timed out waiting for update after stream 1 cancelled")
	}
}

// --- @happy: sets Authorization header ---

func TestSubscribe_SetsAuthorizationHeader(t *testing.T) {
	// @happy: SSE request must include Authorization: Bearer <token>
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := newSubscribeClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())

	updates, errs, err := c.Subscribe(ctx, "dark-mode")
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	// Wait briefly for the request to arrive.
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Drain to prevent goroutine leak.
	for range updates {
	}
	for range errs {
	}

	if gotAuth != "Bearer svc_test" {
		t.Errorf("expected Authorization: Bearer svc_test, got %q", gotAuth)
	}
}

// --- @error-path: empty key ---

func TestSubscribe_EmptyKey_ReturnsError(t *testing.T) {
	// @error-path: empty key must return an immediate error.
	c := newSubscribeClient(t, "http://localhost:9")
	_, _, err := c.Subscribe(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty key, got nil")
	}
}
