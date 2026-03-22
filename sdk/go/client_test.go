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

// --- @happy: EvaluateAll ---

func TestEvaluateAll_ReturnsBulkResults(t *testing.T) {
	// @happy
	srv := serveBulk(t, bulkResponse, 200)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	results, err := c.EvaluateAll(context.Background(), cuttlegate.EvalContext{UserID: "u1", Attributes: map[string]any{"plan": "pro"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	dm, ok := results["dark-mode"]
	if !ok || !dm.Enabled || dm.Reason != "rule_match" {
		t.Errorf("unexpected dark-mode result: %+v", dm)
	}
	// @happy: Variant is "true" for bool flag
	if dm.Variant != "true" {
		t.Errorf("expected Variant=true for bool flag, got %q", dm.Variant)
	}
	bt, ok := results["banner-text"]
	if !ok || bt.Value != "holiday" {
		t.Errorf("unexpected banner-text result: %+v", bt)
	}
	// @happy: Variant equals value for string flag
	if bt.Variant != "holiday" {
		t.Errorf("expected Variant=holiday for string flag, got %q", bt.Variant)
	}
}

func TestEvaluateAll_UsesBulkEndpoint(t *testing.T) {
	// @happy — EvaluateAll must make exactly one HTTP request
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bulkResponse)
	}))
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	_, err := c.EvaluateAll(context.Background(), cuttlegate.EvalContext{UserID: "u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requestCount != 1 {
		t.Errorf("expected exactly 1 HTTP request, got %d", requestCount)
	}
}

func TestEvaluateAll_SetsAuthorizationHeader(t *testing.T) {
	// @happy
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bulkResponse)
	}))
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	c.EvaluateAll(context.Background(), cuttlegate.EvalContext{UserID: "u1"})

	if gotAuth != "Bearer svc_test" {
		t.Errorf("expected Bearer svc_test, got %q", gotAuth)
	}
}

// --- @happy: Evaluate (single flag) ---

func TestEvaluate_ReturnsSingleFlag(t *testing.T) {
	// @happy
	srv := serveBulk(t, bulkResponse, 200)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	result, err := c.Evaluate(context.Background(), "dark-mode", cuttlegate.EvalContext{UserID: "u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Enabled || result.Reason != "rule_match" {
		t.Errorf("unexpected result: %+v", result)
	}
	if result.Variant != "true" {
		t.Errorf("expected Variant=true, got %q", result.Variant)
	}
}

func TestEvaluate_FlagNotFound_ReturnsNotFoundError(t *testing.T) {
	// @error-path: missing flag key must return NotFoundError, never silent default
	srv := serveBulk(t, bulkResponse, 200)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	_, err := c.Evaluate(context.Background(), "ghost-flag", cuttlegate.EvalContext{UserID: "u1"})
	if err == nil {
		t.Fatal("expected NotFoundError for missing flag key, got nil")
	}
	var nfErr *cuttlegate.NotFoundError
	if !errors.As(err, &nfErr) {
		t.Fatalf("expected *NotFoundError, got %T: %v", err, err)
	}
	if nfErr.Resource != "flag" {
		t.Errorf("expected Resource=flag, got %q", nfErr.Resource)
	}
	if nfErr.Key != "ghost-flag" {
		t.Errorf("expected Key=ghost-flag, got %q", nfErr.Key)
	}
}

func TestEvaluate_ProjectNotFound_ReturnsNotFoundError(t *testing.T) {
	// @error-path: 404 from server means project not found (not flag)
	srv := serveBulk(t, map[string]string{"error": "not_found"}, 404)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	_, err := c.Evaluate(context.Background(), "any-flag", cuttlegate.EvalContext{UserID: "u1"})

	var nfErr *cuttlegate.NotFoundError
	if !errors.As(err, &nfErr) {
		t.Fatalf("expected *NotFoundError, got %T: %v", err, err)
	}
	if nfErr.Resource != "project" {
		t.Errorf("expected Resource=project, got %q", nfErr.Resource)
	}
}

// --- @happy: Bool ---

func TestBool_ReturnsTrue(t *testing.T) {
	// @happy: dark-mode has value_key="true"
	srv := serveBulk(t, bulkResponse, 200)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	val, err := c.Bool(context.Background(), "dark-mode", cuttlegate.EvalContext{UserID: "u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !val {
		t.Error("expected true for dark-mode flag")
	}
}

func TestBool_FlagNotFound_ReturnsNotFoundError(t *testing.T) {
	// @error-path: Bool must return false + NotFoundError, never silent false
	srv := serveBulk(t, bulkResponse, 200)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	val, err := c.Bool(context.Background(), "ghost-flag", cuttlegate.EvalContext{UserID: "u1"})
	if err == nil {
		t.Fatal("expected NotFoundError, got nil")
	}
	var nfErr *cuttlegate.NotFoundError
	if !errors.As(err, &nfErr) {
		t.Fatalf("expected *NotFoundError, got %T: %v", err, err)
	}
	if nfErr.Resource != "flag" {
		t.Errorf("expected Resource=flag, got %q", nfErr.Resource)
	}
	if val {
		t.Error("expected false zero value on error")
	}
}

// --- @happy: String ---

func TestString_ReturnsValue(t *testing.T) {
	// @happy: banner-text has value="holiday"
	srv := serveBulk(t, bulkResponse, 200)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	val, err := c.String(context.Background(), "banner-text", cuttlegate.EvalContext{UserID: "u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "holiday" {
		t.Errorf("expected holiday, got %q", val)
	}
}

func TestString_FlagNotFound_ReturnsNotFoundError(t *testing.T) {
	// @error-path: String must return "" + NotFoundError, never silent empty string
	srv := serveBulk(t, bulkResponse, 200)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	val, err := c.String(context.Background(), "ghost-flag", cuttlegate.EvalContext{UserID: "u1"})
	if err == nil {
		t.Fatal("expected NotFoundError, got nil")
	}
	var nfErr *cuttlegate.NotFoundError
	if !errors.As(err, &nfErr) {
		t.Fatalf("expected *NotFoundError, got %T: %v", err, err)
	}
	if nfErr.Resource != "flag" {
		t.Errorf("expected Resource=flag, got %q", nfErr.Resource)
	}
	if val != "" {
		t.Error("expected empty string zero value on error")
	}
}

// --- @auth-bypass: 401/403 ---

func TestEvaluateAll_Returns_AuthError_On401(t *testing.T) {
	// @auth-bypass
	srv := serveBulk(t, map[string]string{"error": "unauthorized"}, 401)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	_, err := c.EvaluateAll(context.Background(), cuttlegate.EvalContext{UserID: "u1"})

	var authErr *cuttlegate.AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *AuthError, got %T: %v", err, err)
	}
	if authErr.StatusCode != 401 {
		t.Errorf("expected StatusCode 401, got %d", authErr.StatusCode)
	}
}

func TestEvaluateAll_Returns_AuthError_On403(t *testing.T) {
	// @auth-bypass
	srv := serveBulk(t, map[string]string{"error": "forbidden"}, 403)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	_, err := c.EvaluateAll(context.Background(), cuttlegate.EvalContext{UserID: "u1"})

	var authErr *cuttlegate.AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *AuthError, got %T: %v", err, err)
	}
	if authErr.StatusCode != 403 {
		t.Errorf("expected StatusCode 403, got %d", authErr.StatusCode)
	}
}

// --- @error-path: ServerError ---

func TestEvaluateAll_Returns_ServerError_On500(t *testing.T) {
	// @error-path
	srv := serveBulk(t, map[string]string{"error": "internal"}, 500)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	_, err := c.EvaluateAll(context.Background(), cuttlegate.EvalContext{UserID: "u1"})

	var srvErr *cuttlegate.ServerError
	if !errors.As(err, &srvErr) {
		t.Fatalf("expected *ServerError, got %T: %v", err, err)
	}
	if srvErr.StatusCode != 500 {
		t.Errorf("expected StatusCode 500, got %d", srvErr.StatusCode)
	}
}

// --- @error-path: NotFoundError distinguishes flag vs project ---

func TestEvaluateAll_Returns_NotFoundError_On404(t *testing.T) {
	// @error-path
	srv := serveBulk(t, map[string]string{"error": "not_found"}, 404)
	defer srv.Close()

	c, _ := cuttlegate.NewClient(validConfig(srv.URL))
	_, err := c.EvaluateAll(context.Background(), cuttlegate.EvalContext{UserID: "u1"})

	var nfErr *cuttlegate.NotFoundError
	if !errors.As(err, &nfErr) {
		t.Fatalf("expected *NotFoundError, got %T: %v", err, err)
	}
	if nfErr.Resource != "project" {
		t.Errorf("expected Resource=project, got %q", nfErr.Resource)
	}
}

// --- @happy: EvaluateFlag (legacy) ---

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
	// @happy: Variant is present on FlagResult
	if result.Variant != "true" {
		t.Errorf("expected Variant=true for bool flag, got %q", result.Variant)
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

func TestEvaluateAll_RespectsContextCancellation(t *testing.T) {
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
	_, err := c.EvaluateAll(ctx, cuttlegate.EvalContext{UserID: "u1"})
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
	if _, err := c.EvaluateAll(context.Background(), cuttlegate.EvalContext{UserID: "u1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
