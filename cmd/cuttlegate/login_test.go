package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeviceAuthRequestForm(t *testing.T) {
	form := DeviceAuthRequest("cuttlegate")

	if got := form.Get("client_id"); got != "cuttlegate" {
		t.Errorf("client_id = %q, want %q", got, "cuttlegate")
	}
	if got := form.Get("scope"); got != "openid profile email" {
		t.Errorf("scope = %q, want %q", got, "openid profile email")
	}
}

func TestDiscoverEndpoints(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"device_authorization_endpoint": "https://example.com/device/authorize",
			"token_endpoint":                "https://example.com/token",
		})
	}))
	defer ts.Close()

	disco, err := discover(ts.URL)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if disco.DeviceAuthorizationEndpoint != "https://example.com/device/authorize" {
		t.Errorf("device_authorization_endpoint = %q", disco.DeviceAuthorizationEndpoint)
	}
	if disco.TokenEndpoint != "https://example.com/token" {
		t.Errorf("token_endpoint = %q", disco.TokenEndpoint)
	}
}

func TestPollTokenPending(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "authorization_pending",
		})
	}))
	defer ts.Close()

	tok, retry, err := pollToken(ts.URL, "cuttlegate", "device123")
	if err != nil {
		t.Fatalf("pollToken: %v", err)
	}
	if tok != nil {
		t.Errorf("expected nil token, got %+v", tok)
	}
	if retry != "authorization_pending" {
		t.Errorf("retry = %q, want %q", retry, "authorization_pending")
	}
}

func TestPollTokenSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "at-123",
			"refresh_token": "rt-456",
			"id_token":      "id-789",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer ts.Close()

	tok, retry, err := pollToken(ts.URL, "cuttlegate", "device123")
	if err != nil {
		t.Fatalf("pollToken: %v", err)
	}
	if retry != "" {
		t.Errorf("retry = %q, want empty", retry)
	}
	if tok.AccessToken != "at-123" {
		t.Errorf("access_token = %q", tok.AccessToken)
	}
	if tok.RefreshToken != "rt-456" {
		t.Errorf("refresh_token = %q", tok.RefreshToken)
	}
}
