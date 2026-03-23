package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHealthHandler_Degraded_NilConn covers the @edge scenario: server started without
// DATABASE_URL (conn == nil). The handler must return 503 with the degraded body.
func TestHealthHandler_Degraded_NilConn(t *testing.T) {
	handler := healthHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want %d", res.StatusCode, http.StatusServiceUnavailable)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", ct, "application/json")
	}
	body := rec.Body.String()
	want := `{"status":"degraded","reason":"database"}`
	if body != want {
		t.Errorf("body: got %q, want %q", body, want)
	}
}

// TestHealthHandler_NoAuth covers the @happy scenario: /health must be accessible without
// an Authorization header. The handler itself has no auth wrapper — this test confirms the
// function does not gate on auth context.
func TestHealthHandler_NoAuth(t *testing.T) {
	handler := healthHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	// Deliberately no Authorization header.
	rec := httptest.NewRecorder()
	handler(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		t.Errorf("expected no auth gate, got %d", res.StatusCode)
	}
}
