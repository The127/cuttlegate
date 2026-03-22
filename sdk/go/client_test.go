package cuttlegate_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	cuttlegate "github.com/karo/cuttlegate/sdk/go"
)

// bulkResponse is a valid evaluate API response for use in tests.
var bulkResponse = map[string]any{
	"flags": []map[string]any{
		{"key": "dark-mode", "enabled": true, "value": nil, "value_key": "true", "reason": "rule_match", "type": "bool"},
		{"key": "banner-text", "enabled": true, "value": "holiday", "value_key": "holiday", "reason": "default", "type": "string"},
	},
	"evaluated_at": "2026-03-21T10:00:00Z",
}

func serveBulk(t *testing.T, body any, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Errorf("test server encode: %v", err)
		}
	}))
}

func validConfig(baseURL string) cuttlegate.Config {
	return cuttlegate.Config{
		BaseURL:      baseURL,
		ServiceToken: "svc_test",
		Project:      "my-project",
		Environment:  "production",
	}
}

// --- @happy: NewClient validation ---

func TestNewClient_ValidConfig(t *testing.T) {
	// @happy
	srv := serveBulk(t, bulkResponse, 200)
	defer srv.Close()

	c, err := cuttlegate.NewClient(validConfig(srv.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_MissingBaseURL(t *testing.T) {
	// @error-path
	_, err := cuttlegate.NewClient(cuttlegate.Config{
		ServiceToken: "tok",
		Project:      "proj",
		Environment:  "prod",
	})
	if err == nil {
		t.Fatal("expected error for missing BaseURL")
	}
}

func TestNewClient_MissingServiceToken(t *testing.T) {
	// @error-path
	_, err := cuttlegate.NewClient(cuttlegate.Config{
		BaseURL:     "http://localhost",
		Project:     "proj",
		Environment: "prod",
	})
	if err == nil {
		t.Fatal("expected error for missing ServiceToken")
	}
}

func TestNewClient_MissingProject(t *testing.T) {
	// @error-path
	_, err := cuttlegate.NewClient(cuttlegate.Config{
		BaseURL:      "http://localhost",
		ServiceToken: "tok",
		Environment:  "prod",
	})
	if err == nil {
		t.Fatal("expected error for missing Project")
	}
}

func TestNewClient_MissingEnvironment(t *testing.T) {
	// @error-path
	_, err := cuttlegate.NewClient(cuttlegate.Config{
		BaseURL:      "http://localhost",
		ServiceToken: "tok",
		Project:      "proj",
	})
	if err == nil {
		t.Fatal("expected error for missing Environment")
	}
}

func TestNewClient_NoNetworkCallsAtInit(t *testing.T) {
	// @happy — validation must be local only
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	defer srv.Close()

	_, err := cuttlegate.NewClient(validConfig(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("NewClient must not make network calls")
	}
}

// --- @happy: Evaluate ---

func TestEvaluate_ReturnsBulkResults(t *testing.T) {
	// @happy
	srv := serveBulk(t, bulkResponse, 200)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	results, err := c.Evaluate(context.Background(), cuttlegate.EvalContext{UserID: "u1", Attributes: map[string]any{"plan": "pro"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Key != "dark-mode" || !results[0].Enabled || results[0].Reason != "rule_match" {
		t.Errorf("unexpected result[0]: %+v", results[0])
	}
	// @happy: value_key is "true" for bool flag, Value is "" (nil in wire format)
	if results[0].ValueKey != "true" {
		t.Errorf("expected ValueKey=true for bool flag, got %q", results[0].ValueKey)
	}
	if results[1].Key != "banner-text" || results[1].Value != "holiday" {
		t.Errorf("unexpected result[1]: %+v", results[1])
	}
	// @happy: value_key equals value for string flag
	if results[1].ValueKey != "holiday" {
		t.Errorf("expected ValueKey=holiday for string flag, got %q", results[1].ValueKey)
	}
}

func TestEvaluate_SetsAuthorizationHeader(t *testing.T) {
	// @happy
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bulkResponse)
	}))
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	c.Evaluate(context.Background(), cuttlegate.EvalContext{UserID: "u1"})

	if gotAuth != "Bearer svc_test" {
		t.Errorf("expected Bearer svc_test, got %q", gotAuth)
	}
}

// --- @auth-bypass: 401/403 ---

func TestEvaluate_Returns_AuthError_On401(t *testing.T) {
	// @auth-bypass
	srv := serveBulk(t, map[string]string{"error": "unauthorized"}, 401)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	_, err := c.Evaluate(context.Background(), cuttlegate.EvalContext{UserID: "u1"})

	var authErr *cuttlegate.AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *AuthError, got %T: %v", err, err)
	}
	if authErr.StatusCode != 401 {
		t.Errorf("expected StatusCode 401, got %d", authErr.StatusCode)
	}
}

func TestEvaluate_Returns_AuthError_On403(t *testing.T) {
	// @auth-bypass
	srv := serveBulk(t, map[string]string{"error": "forbidden"}, 403)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	_, err := c.Evaluate(context.Background(), cuttlegate.EvalContext{UserID: "u1"})

	var authErr *cuttlegate.AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *AuthError, got %T: %v", err, err)
	}
	if authErr.StatusCode != 403 {
		t.Errorf("expected StatusCode 403, got %d", authErr.StatusCode)
	}
}

// --- @error-path: ServerError ---

func TestEvaluate_Returns_ServerError_On500(t *testing.T) {
	// @error-path
	srv := serveBulk(t, map[string]string{"error": "internal"}, 500)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	_, err := c.Evaluate(context.Background(), cuttlegate.EvalContext{UserID: "u1"})

	var srvErr *cuttlegate.ServerError
	if !errors.As(err, &srvErr) {
		t.Fatalf("expected *ServerError, got %T: %v", err, err)
	}
	if srvErr.StatusCode != 500 {
		t.Errorf("expected StatusCode 500, got %d", srvErr.StatusCode)
	}
}

// --- @error-path: NotFoundError ---

func TestEvaluate_Returns_NotFoundError_On404(t *testing.T) {
	// @error-path
	srv := serveBulk(t, map[string]string{"error": "not_found"}, 404)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	_, err := c.Evaluate(context.Background(), cuttlegate.EvalContext{UserID: "u1"})

	var nfErr *cuttlegate.NotFoundError
	if !errors.As(err, &nfErr) {
		t.Fatalf("expected *NotFoundError, got %T: %v", err, err)
	}
	if nfErr.Resource != "project" {
		t.Errorf("expected Resource=project, got %q", nfErr.Resource)
	}
}

// --- @happy: EvaluateFlag ---

func TestEvaluateFlag_ReturnsSingleFlag(t *testing.T) {
	// @happy
	srv := serveBulk(t, bulkResponse, 200)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	result, err := c.EvaluateFlag(context.Background(), "dark-mode", cuttlegate.EvalContext{UserID: "u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Enabled || result.Reason != "rule_match" {
		t.Errorf("unexpected result: %+v", result)
	}
	// @happy: value_key is present on FlagResult
	if result.ValueKey != "true" {
		t.Errorf("expected ValueKey=true for bool flag, got %q", result.ValueKey)
	}
}

func TestEvaluateFlag_ReturnsNotFoundReasonForMissingKey(t *testing.T) {
	// @edge
	srv := serveBulk(t, bulkResponse, 200)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	result, err := c.EvaluateFlag(context.Background(), "nonexistent-flag", cuttlegate.EvalContext{UserID: "u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Enabled {
		t.Error("expected Enabled=false for missing key")
	}
	if result.Reason != "not_found" {
		t.Errorf("expected Reason=not_found, got %q", result.Reason)
	}
}

// --- @edge: context cancellation ---

func TestEvaluate_RespectsContextCancellation(t *testing.T) {
	// @edge
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the client cancels.
		<-r.Context().Done()
		w.WriteHeader(200)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	_, err := c.Evaluate(ctx, cuttlegate.EvalContext{UserID: "u1"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// --- @happy: custom HTTPClient ---

func TestNewClient_AcceptsCustomHTTPClient(t *testing.T) {
	// @happy
	srv := serveBulk(t, bulkResponse, 200)
	defer srv.Close()

	cfg := validConfig(srv.URL)
	cfg.HTTPClient = &http.Client{}
	c, err := cuttlegate.NewClient(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := c.Evaluate(context.Background(), cuttlegate.EvalContext{UserID: "u1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
