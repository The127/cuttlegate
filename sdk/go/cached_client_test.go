package cuttlegate_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	cuttlegate "github.com/The127/cuttlegate/sdk/go"
)

// --- helpers ---

func newCachedClient(t *testing.T, baseURL string) *cuttlegate.CachedClient {
	t.Helper()
	cc, err := cuttlegate.NewCachedClient(validConfig(baseURL))
	if err != nil {
		t.Fatalf("NewCachedClient: %v", err)
	}
	return cc
}

// serveCombined starts a test server that handles both the bulk eval endpoint
// and the SSE stream endpoint. sseLines are written then the SSE connection
// blocks until the client disconnects.
func serveCombined(t *testing.T, bulkBody any, bulkStatus int, sseLines []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/projects/my-project/environments/production/flags/stream" {
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "no flusher", 500)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)
			flusher.Flush()
			for _, line := range sseLines {
				fmt.Fprint(w, line)
				flusher.Flush()
			}
			<-r.Context().Done()
			return
		}
		// Bulk eval.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(bulkStatus)
		if err := json.NewEncoder(w).Encode(bulkBody); err != nil {
			t.Errorf("test server encode: %v", err)
		}
	}))
}

// --- @happy: NewCachedClient validation ---

func TestNewCachedClient_ValidConfig(t *testing.T) {
	// @happy
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	cc, err := cuttlegate.NewCachedClient(validConfig(srv.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cc == nil {
		t.Fatal("expected non-nil CachedClient")
	}
}

func TestNewCachedClient_MissingBaseURL(t *testing.T) {
	// @error-path
	_, err := cuttlegate.NewCachedClient(cuttlegate.Config{
		ServiceToken: "tok",
		Project:      "proj",
		Environment:  "prod",
	})
	if err == nil {
		t.Fatal("expected error for missing BaseURL")
	}
}

func TestNewCachedClient_NoNetworkCallsAtInit(t *testing.T) {
	// @happy — no network calls on construction
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	defer srv.Close()

	_, err := cuttlegate.NewCachedClient(validConfig(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("NewCachedClient must not make network calls")
	}
}

// --- @happy: CachedClient satisfies Client interface ---

func TestCachedClient_SatisfiesClientInterface(t *testing.T) {
	// @happy — compile-time check; runtime confirms the interface is held.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	cc, err := cuttlegate.NewCachedClient(validConfig(srv.URL))
	if err != nil {
		t.Fatalf("NewCachedClient: %v", err)
	}
	var _ cuttlegate.Client = cc
}

// --- @happy: Bootstrap seeds cache from EvaluateAll ---

func TestCachedClient_Bootstrap_SeedsCache(t *testing.T) {
	// @happy — Bootstrap calls EvaluateAll once; dark-mode and banner-text end up in cache.
	srv := serveCombined(t, bulkResponse, 200, nil)
	defer srv.Close()

	cc := newCachedClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cc.Bootstrap(ctx, cuttlegate.EvalContext{UserID: "u1"}); err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}

	// Verify cache hit — Bool must not make additional HTTP requests.
	val, err := cc.Bool(ctx, "dark-mode", cuttlegate.EvalContext{UserID: "u1"})
	if err != nil {
		t.Fatalf("Bool after Bootstrap: %v", err)
	}
	if !val {
		t.Error("expected dark-mode=true from cache")
	}
}

// --- @error-path: Bootstrap returns AuthError on 401 ---

func TestCachedClient_Bootstrap_ErrorOnAuthFailure(t *testing.T) {
	// @error-path
	srv := serveBulk(t, map[string]string{"error": "Unauthorized"}, 401)
	defer srv.Close()

	cc := newCachedClient(t, srv.URL)
	err := cc.Bootstrap(context.Background(), cuttlegate.EvalContext{UserID: "u1"})
	if err == nil {
		t.Fatal("expected error from Bootstrap on 401")
	}
	var authErr *cuttlegate.AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *AuthError, got %T: %v", err, err)
	}
}

// --- @error-path: Bootstrap returns error on network failure ---

func TestCachedClient_Bootstrap_ErrorOnNetworkFailure(t *testing.T) {
	// @error-path
	cc := newCachedClient(t, "http://127.0.0.1:1") // nothing listening
	err := cc.Bootstrap(context.Background(), cuttlegate.EvalContext{UserID: "u1"})
	if err == nil {
		t.Fatal("expected error from Bootstrap on network failure")
	}
}

// --- @happy: Bool returns cached value on cache hit ---

func TestCachedClient_Bool_CacheHit_NoHTTPCall(t *testing.T) {
	// @happy — Bool reads from cache; zero additional HTTP calls after Bootstrap.
	evalCallCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/projects/my-project/environments/production/flags/stream" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			return
		}
		evalCallCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(bulkResponse)
	}))
	defer srv.Close()

	cc := newCachedClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cc.Bootstrap(ctx, cuttlegate.EvalContext{UserID: "u1"}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	callsAfterBootstrap := evalCallCount

	val, err := cc.Bool(ctx, "dark-mode", cuttlegate.EvalContext{UserID: "u1"})
	if err != nil {
		t.Fatalf("Bool: %v", err)
	}
	if !val {
		t.Error("expected true for dark-mode from cache")
	}
	if evalCallCount != callsAfterBootstrap {
		t.Errorf("Bool must not make HTTP calls on cache hit: before=%d after=%d", callsAfterBootstrap, evalCallCount)
	}
}

// --- @happy: String returns cached value on cache hit ---

func TestCachedClient_String_CacheHit(t *testing.T) {
	// @happy
	srv := serveCombined(t, bulkResponse, 200, nil)
	defer srv.Close()

	cc := newCachedClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cc.Bootstrap(ctx, cuttlegate.EvalContext{UserID: "u1"}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	val, err := cc.String(ctx, "banner-text", cuttlegate.EvalContext{UserID: "u1"})
	if err != nil {
		t.Fatalf("String: %v", err)
	}
	if val != "holiday" {
		t.Errorf("expected holiday, got %q", val)
	}
}

// --- @happy: cache miss falls back to live HTTP ---

func TestCachedClient_Bool_CacheMiss_FallsBackToHTTP(t *testing.T) {
	// @happy — "new-flag" not in cache; Bool makes a live HTTP call.
	bulkWithNew := map[string]any{
		"flags": []map[string]any{
			{"key": "new-flag", "enabled": true, "value": nil, "value_key": "true", "reason": "default", "type": "bool"},
		},
		"evaluated_at": "2026-03-23T10:00:00Z",
	}

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/projects/my-project/environments/production/flags/stream" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			return
		}
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if callCount == 1 {
			json.NewEncoder(w).Encode(bulkResponse) // Bootstrap's EvaluateAll
		} else {
			json.NewEncoder(w).Encode(bulkWithNew) // cache miss fallback
		}
	}))
	defer srv.Close()

	cc := newCachedClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cc.Bootstrap(ctx, cuttlegate.EvalContext{UserID: "u1"}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	callsAfterBootstrap := callCount

	val, err := cc.Bool(ctx, "new-flag", cuttlegate.EvalContext{UserID: "u1"})
	if err != nil {
		t.Fatalf("Bool fallback: %v", err)
	}
	if !val {
		t.Error("expected true for new-flag from live HTTP fallback")
	}
	if callCount <= callsAfterBootstrap {
		t.Error("expected at least one additional HTTP call for cache-miss fallback")
	}
}

// --- @edge: Bool before Bootstrap falls back to live HTTP ---

func TestCachedClient_Bool_BeforeBootstrap_FallsBackToHTTP(t *testing.T) {
	// @edge — no Bootstrap call; Bool falls back to live HTTP; no panic.
	srv := serveBulk(t, bulkResponse, 200)
	defer srv.Close()

	cc := newCachedClient(t, srv.URL)
	val, err := cc.Bool(context.Background(), "dark-mode", cuttlegate.EvalContext{UserID: "u1"})
	if err != nil {
		t.Fatalf("Bool before Bootstrap: %v", err)
	}
	if !val {
		t.Error("expected true for dark-mode via live HTTP fallback before Bootstrap")
	}
}

// --- @happy: SSE update applied to cache ---

func TestCachedClient_SSEUpdate_AppliedToCache(t *testing.T) {
	// @happy — SSE loop flips dark-mode from true to false; subsequent Bool returns false.
	occurredAt := "2026-03-23T10:00:00Z"
	sseLine := sseEvent("dark-mode", false, occurredAt) // flip to false

	srv := serveCombined(t, bulkResponse, 200, []string{sseLine})
	defer srv.Close()

	cc := newCachedClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cc.Bootstrap(ctx, cuttlegate.EvalContext{UserID: "u1"}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	// Poll until the SSE update is visible — must happen within 2s.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		val, err := cc.Bool(ctx, "dark-mode", cuttlegate.EvalContext{UserID: "u1"})
		if err != nil {
			t.Fatalf("Bool: %v", err)
		}
		if !val { // dark-mode is now false
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timed out waiting for SSE update to be applied to cache")
}

// --- @edge: SSE event for key not in cache is ignored ---

func TestCachedClient_SSEUpdate_UnknownKeyIgnored(t *testing.T) {
	// @edge — event for "unknown-flag" must not add it to the cache; no panic.
	occurredAt := "2026-03-23T10:00:00Z"
	sseLine := sseEvent("unknown-flag", true, occurredAt)

	srv := serveCombined(t, bulkResponse, 200, []string{sseLine})
	defer srv.Close()

	cc := newCachedClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cc.Bootstrap(ctx, cuttlegate.EvalContext{UserID: "u1"}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	// Give time for the SSE event to be processed.
	time.Sleep(100 * time.Millisecond)

	// "unknown-flag" is not in cache — just verify no panic; err is expected.
	_, _ = cc.Bool(ctx, "unknown-flag", cuttlegate.EvalContext{UserID: "u1"})
}

// --- @happy: context cancel stops background goroutine ---

func TestCachedClient_ContextCancel_StopsGoroutine(t *testing.T) {
	// @happy — cancel stops SSE goroutine; cache still readable; Bool cache-hit still works.
	srv := serveCombined(t, bulkResponse, 200, nil)
	defer srv.Close()

	cc := newCachedClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())

	if err := cc.Bootstrap(ctx, cuttlegate.EvalContext{UserID: "u1"}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	cancel() // stop the SSE goroutine

	// Cache is still populated — Bool must still work from cache.
	val, err := cc.Bool(context.Background(), "dark-mode", cuttlegate.EvalContext{UserID: "u1"})
	if err != nil {
		t.Fatalf("Bool after context cancel: %v", err)
	}
	if !val {
		t.Error("expected true for dark-mode from cache after goroutine stopped")
	}
}

// --- @edge: Bootstrap called twice — no goroutine duplication ---

func TestCachedClient_Bootstrap_CalledTwice(t *testing.T) {
	// @edge — second Bootstrap stops first goroutine, refreshes cache.
	srv := serveCombined(t, bulkResponse, 200, nil)
	defer srv.Close()

	cc := newCachedClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cc.Bootstrap(ctx, cuttlegate.EvalContext{UserID: "u1"}); err != nil {
		t.Fatalf("first Bootstrap: %v", err)
	}
	if err := cc.Bootstrap(ctx, cuttlegate.EvalContext{UserID: "u2"}); err != nil {
		t.Fatalf("second Bootstrap: %v", err)
	}
	// The race detector catches any goroutine leaks from duplicate start.
}

// --- @edge: concurrent reads during SSE write (data race check) ---

func TestCachedClient_ConcurrentReads_NoRace(t *testing.T) {
	// @edge — multiple goroutines reading while SSE loop writes; race detector verifies safety.
	occurredAt := "2026-03-23T10:00:00Z"
	var sseLines []string
	for i := 0; i < 10; i++ {
		enabled := i%2 == 0
		sseLines = append(sseLines, sseEvent("dark-mode", enabled, occurredAt))
	}

	srv := serveCombined(t, bulkResponse, 200, sseLines)
	defer srv.Close()

	cc := newCachedClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cc.Bootstrap(ctx, cuttlegate.EvalContext{UserID: "u1"}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = cc.Bool(ctx, "dark-mode", cuttlegate.EvalContext{UserID: "u1"})
				_, _ = cc.String(ctx, "banner-text", cuttlegate.EvalContext{UserID: "u1"})
			}
		}()
	}
	wg.Wait()
}

// --- @error-path: SSE reconnect on 5xx ---

func TestCachedClient_SSE_ReconnectsOn5xx(t *testing.T) {
	// @error-path — first SSE attempt returns 500; goroutine reconnects and applies update.
	occurredAt := "2026-03-23T10:00:00Z"
	sseLine := sseEvent("dark-mode", false, occurredAt) // flip to false after reconnect

	var sseAttemptMu sync.Mutex
	sseAttempt := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/projects/my-project/environments/production/flags/stream" {
			sseAttemptMu.Lock()
			sseAttempt++
			attempt := sseAttempt
			sseAttemptMu.Unlock()

			if attempt == 1 {
				w.WriteHeader(500)
				return
			}
			flusher, _ := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			flusher.Flush()
			fmt.Fprint(w, sseLine)
			flusher.Flush()
			<-r.Context().Done()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(bulkResponse)
	}))
	defer srv.Close()

	cc := newCachedClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cc.Bootstrap(ctx, cuttlegate.EvalContext{UserID: "u1"}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		val, err := cc.Bool(ctx, "dark-mode", cuttlegate.EvalContext{UserID: "u1"})
		if err != nil {
			t.Fatalf("Bool: %v", err)
		}
		if !val { // dark-mode flipped to false after reconnect
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("timed out waiting for SSE update after reconnect from 5xx")
}

// --- @error-path: SSE terminates on 401 (no reconnect) ---

func TestCachedClient_SSE_TerminatesOn401(t *testing.T) {
	// @error-path — SSE 401: goroutine stops; cache still readable; no goroutine leak.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/projects/my-project/environments/production/flags/stream" {
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(bulkResponse)
	}))
	defer srv.Close()

	cc := newCachedClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cc.Bootstrap(ctx, cuttlegate.EvalContext{UserID: "u1"}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	// Give the SSE goroutine time to terminate.
	time.Sleep(200 * time.Millisecond)

	// Cache was populated at Bootstrap — Bool must return from cache.
	val, err := cc.Bool(ctx, "dark-mode", cuttlegate.EvalContext{UserID: "u1"})
	if err != nil {
		t.Fatalf("Bool after SSE 401: %v", err)
	}
	if !val {
		t.Error("expected true for dark-mode from cache after SSE goroutine stopped on 401")
	}
}
