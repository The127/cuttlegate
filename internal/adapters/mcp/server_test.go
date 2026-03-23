package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karo/cuttlegate/internal/adapters/mcp"
	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

// ── Test doubles ──────────────────────────────────────────────────────────────

type fakeAPIKeyAuth struct {
	key *domain.APIKey
	err error
}

func (f *fakeAPIKeyAuth) AuthenticateMCP(_ context.Context, _ string) (*domain.APIKey, error) {
	return f.key, f.err
}

type fakeAPIKeyRepo struct {
	key *domain.APIKey
	err error
}

func (f *fakeAPIKeyRepo) Create(_ context.Context, _ *domain.APIKey) error { return nil }
func (f *fakeAPIKeyRepo) GetByID(_ context.Context, _ string) (*domain.APIKey, error) {
	return f.key, f.err
}
func (f *fakeAPIKeyRepo) GetByHash(_ context.Context, _ [32]byte) (*domain.APIKey, error) {
	return f.key, f.err
}
func (f *fakeAPIKeyRepo) ListByEnvironment(_ context.Context, _, _ string) ([]*domain.APIKey, error) {
	return nil, nil
}
func (f *fakeAPIKeyRepo) Revoke(_ context.Context, _ string) error { return nil }
func (f *fakeAPIKeyRepo) UpdateCapabilityTier(_ context.Context, _ string, _ domain.ToolCapabilityTier) error {
	return nil
}

type fakeFlagSvc struct {
	views      []*app.FlagEnvironmentView
	listErr    error
	setErr     error
	setCalled  bool
	setEnabled bool
}

func (f *fakeFlagSvc) ListByEnvironment(_ context.Context, _, _ string) ([]*app.FlagEnvironmentView, error) {
	return f.views, f.listErr
}
func (f *fakeFlagSvc) SetEnabled(_ context.Context, params app.SetEnabledParams) error {
	f.setCalled = true
	f.setEnabled = params.Enabled
	return f.setErr
}

type fakeEvalSvc struct {
	view *app.EvalView
	err  error
}

func (f *fakeEvalSvc) Evaluate(_ context.Context, _, _, _ string, _ domain.EvalContext) (*app.EvalView, error) {
	return f.view, f.err
}

type fakeProjectSvc struct {
	proj *domain.Project
	err  error
}

func (f *fakeProjectSvc) GetBySlug(_ context.Context, _ string) (*domain.Project, error) {
	return f.proj, f.err
}

type fakeEnvSvc struct {
	env *domain.Environment
	err error
}

func (f *fakeEnvSvc) GetBySlug(_ context.Context, _, _ string) (*domain.Environment, error) {
	return f.env, f.err
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func makeWriteKey(id, projectID, environmentID string) *domain.APIKey {
	key := &domain.APIKey{
		ID:             id,
		ProjectID:      projectID,
		EnvironmentID:  environmentID,
		CapabilityTier: domain.TierWrite,
	}
	return key
}

func makeReadKey(id, projectID, environmentID string) *domain.APIKey {
	return &domain.APIKey{
		ID:             id,
		ProjectID:      projectID,
		EnvironmentID:  environmentID,
		CapabilityTier: domain.TierRead,
	}
}

func buildServer(authKey *domain.APIKey, authErr error, repoKey *domain.APIKey, repoErr error,
	flagSvc *fakeFlagSvc, evalSvc *fakeEvalSvc,
	proj *domain.Project, projErr error, env *domain.Environment, envErr error) *mcp.Server {
	return mcp.NewServer(
		&fakeAPIKeyAuth{key: authKey, err: authErr},
		&fakeAPIKeyRepo{key: repoKey, err: repoErr},
		flagSvc,
		evalSvc,
		&fakeProjectSvc{proj: proj, err: projErr},
		&fakeEnvSvc{env: env, err: envErr},
	)
}

func doPost(t *testing.T, srv *mcp.Server, bearer, method string, params any) *httptest.ResponseRecorder {
	t.Helper()
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		reqBody["params"] = params
	}
	b, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/message?session_id=test123", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	mux.ServeHTTP(w, req)
	return w
}

// ── Authentication tests ───────────────────────────────────────────────────────

// @auth-bypass — no Authorization header on POST /message
func TestHandleMessage_NoBearer_ReturnsUnauthenticated(t *testing.T) {
	flagSvc := &fakeFlagSvc{}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, nil, nil, domain.ErrNotFound, flagSvc, evalSvc, nil, nil, nil, nil)

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	req := httptest.NewRequest("POST", "/message?session_id=test", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errVal, _ := resp["error"].(string)
	if errVal != "unauthenticated" {
		t.Errorf("expected unauthenticated, got %q (full: %v)", errVal, resp)
	}
}

// @auth-bypass — invalid API key on POST /message
func TestHandleMessage_InvalidKey_ReturnsUnauthenticated(t *testing.T) {
	flagSvc := &fakeFlagSvc{}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, nil, nil, domain.ErrNotFound, flagSvc, evalSvc, nil, nil, nil, nil)

	w := doPost(t, srv, "cg_invalid", "tools/list", nil)

	// Live key check fails — result contains unauthenticated
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck

	// The response is a JSON-RPC result with unauthenticated error body
	result, _ := resp["result"].(map[string]any)
	if result == nil {
		t.Fatalf("expected result key in response, got: %v", resp)
	}
	errVal, _ := result["error"].(string)
	if errVal != "unauthenticated" {
		t.Errorf("expected unauthenticated, got %q", errVal)
	}
}

// @happy — tool list filtered by tier at connection time (read-tier)
func TestHandleToolsList_ReadTier_OnlyReadTools(t *testing.T) {
	key := makeReadKey("key-id", "proj-id", "env-id")
	flagSvc := &fakeFlagSvc{}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, nil, key, nil, flagSvc, evalSvc, nil, nil, nil, nil)

	w := doPost(t, srv, "cg_validkey", "tools/list", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	result, _ := resp["result"].(map[string]any)
	tools, _ := result["tools"].([]any)

	names := make(map[string]bool)
	for _, tool := range tools {
		m, _ := tool.(map[string]any)
		names[m["name"].(string)] = true
	}

	if !names["list_flags"] || !names["evaluate_flag"] {
		t.Errorf("read tier should see list_flags and evaluate_flag; got %v", names)
	}
	if names["enable_flag"] || names["disable_flag"] {
		t.Errorf("read tier must NOT see enable_flag or disable_flag; got %v", names)
	}
}

// @happy — tool list for write-tier shows all tools
func TestHandleToolsList_WriteTier_AllTools(t *testing.T) {
	key := makeWriteKey("key-id", "proj-id", "env-id")
	flagSvc := &fakeFlagSvc{}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, nil, key, nil, flagSvc, evalSvc, nil, nil, nil, nil)

	w := doPost(t, srv, "cg_validkey", "tools/list", nil)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	result, _ := resp["result"].(map[string]any)
	tools, _ := result["tools"].([]any)

	names := make(map[string]bool)
	for _, tool := range tools {
		m, _ := tool.(map[string]any)
		names[m["name"].(string)] = true
	}

	for _, expected := range []string{"list_flags", "evaluate_flag", "enable_flag", "disable_flag"} {
		if !names[expected] {
			t.Errorf("write tier should see %s; got %v", expected, names)
		}
	}
}

// ── list_flags tests ───────────────────────────────────────────────────────────

// @edge — environment with no flags returns empty array
func TestListFlags_EmptyEnvironment_ReturnsEmptyArray(t *testing.T) {
	key := makeReadKey("key-id", "proj-id", "env-id")
	proj := &domain.Project{ID: "proj-id", Slug: "acme"}
	env := &domain.Environment{ID: "env-id", Slug: "staging"}
	flagSvc := &fakeFlagSvc{views: []*app.FlagEnvironmentView{}}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, nil, key, nil, flagSvc, evalSvc, proj, nil, env, nil)

	w := doPost(t, srv, "cg_valid", "tools/call", map[string]any{
		"name":      "list_flags",
		"arguments": map[string]any{"project_slug": "acme", "environment_slug": "staging"},
	})

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	result, _ := resp["result"].(map[string]any)
	content, _ := result["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("expected content")
	}
	textItem, _ := content[0].(map[string]any)
	text, _ := textItem["text"].(string)

	var flags []any
	if err := json.Unmarshal([]byte(text), &flags); err != nil {
		t.Fatalf("parse result: %v, text was: %s", err, text)
	}
	if len(flags) != 0 {
		t.Errorf("expected empty array, got %v", flags)
	}
}

// @error-path — unknown project slug
func TestListFlags_UnknownProject_ReturnsNotFound(t *testing.T) {
	key := makeReadKey("key-id", "proj-id", "env-id")
	flagSvc := &fakeFlagSvc{}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, nil, key, nil, flagSvc, evalSvc, nil, domain.ErrNotFound, nil, nil)

	w := doPost(t, srv, "cg_valid", "tools/call", map[string]any{
		"name":      "list_flags",
		"arguments": map[string]any{"project_slug": "does-not-exist", "environment_slug": "production"},
	})

	assertToolError(t, w, "not_found")
}

// @error-path — API key scoped to different project
func TestListFlags_WrongProjectScope_ReturnsForbidden(t *testing.T) {
	key := makeReadKey("key-id", "other-proj-id", "env-id")
	proj := &domain.Project{ID: "different-proj-id", Slug: "other-project"}
	env := &domain.Environment{ID: "env-id", Slug: "production"}
	flagSvc := &fakeFlagSvc{}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, nil, key, nil, flagSvc, evalSvc, proj, nil, env, nil)

	w := doPost(t, srv, "cg_valid", "tools/call", map[string]any{
		"name":      "list_flags",
		"arguments": map[string]any{"project_slug": "other-project", "environment_slug": "production"},
	})

	assertToolError(t, w, "forbidden")
}

// @happy — list_flags with read-tier key
func TestListFlags_HappyPath_ReturnsFlags(t *testing.T) {
	key := makeReadKey("key-id", "proj-id", "env-id")
	proj := &domain.Project{ID: "proj-id", Slug: "acme"}
	env := &domain.Environment{ID: "env-id", Slug: "production"}
	flagSvc := &fakeFlagSvc{
		views: []*app.FlagEnvironmentView{
			{Flag: &domain.Flag{Key: "dark-mode", Type: domain.FlagTypeBool, DefaultVariantKey: "false"}, Enabled: true},
		},
	}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, nil, key, nil, flagSvc, evalSvc, proj, nil, env, nil)

	w := doPost(t, srv, "cg_valid", "tools/call", map[string]any{
		"name":      "list_flags",
		"arguments": map[string]any{"project_slug": "acme", "environment_slug": "production"},
	})

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	result, _ := resp["result"].(map[string]any)
	content, _ := result["content"].([]any)
	textItem, _ := content[0].(map[string]any)
	text, _ := textItem["text"].(string)

	var flags []map[string]any
	if err := json.Unmarshal([]byte(text), &flags); err != nil {
		t.Fatalf("parse flags: %v, text: %s", err, text)
	}
	if len(flags) != 1 {
		t.Fatalf("expected 1 flag, got %d", len(flags))
	}
	if flags[0]["key"] != "dark-mode" {
		t.Errorf("expected key=dark-mode, got %v", flags[0]["key"])
	}
	if flags[0]["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", flags[0]["enabled"])
	}
}

// ── evaluate_flag tests ────────────────────────────────────────────────────────

// @happy — evaluate disabled flag
func TestEvaluateFlag_DisabledFlag_ReturnsDisabledReason(t *testing.T) {
	key := makeReadKey("key-id", "proj-id", "env-id")
	proj := &domain.Project{ID: "proj-id", Slug: "acme"}
	env := &domain.Environment{ID: "env-id", Slug: "production"}
	evalSvc := &fakeEvalSvc{
		view: &app.EvalView{Key: "dark-mode", Enabled: false, ValueKey: "false", Reason: domain.ReasonDisabled},
	}
	flagSvc := &fakeFlagSvc{}
	srv := buildServer(nil, nil, key, nil, flagSvc, evalSvc, proj, nil, env, nil)

	w := doPost(t, srv, "cg_valid", "tools/call", map[string]any{
		"name": "evaluate_flag",
		"arguments": map[string]any{
			"project_slug":     "acme",
			"environment_slug": "production",
			"key":              "dark-mode",
			"eval_context":     map[string]any{"user_id": "u1", "attributes": map[string]any{}},
		},
	})

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	result, _ := resp["result"].(map[string]any)
	content, _ := result["content"].([]any)
	textItem, _ := content[0].(map[string]any)
	text, _ := textItem["text"].(string)

	var evalResult map[string]any
	json.Unmarshal([]byte(text), &evalResult) //nolint:errcheck
	if evalResult["reason"] != "disabled" {
		t.Errorf("expected reason=disabled, got %v", evalResult["reason"])
	}
	if evalResult["enabled"] != false {
		t.Errorf("expected enabled=false, got %v", evalResult["enabled"])
	}
}

// @edge — unknown flag key returns not_found
func TestEvaluateFlag_UnknownKey_ReturnsNotFound(t *testing.T) {
	key := makeReadKey("key-id", "proj-id", "env-id")
	proj := &domain.Project{ID: "proj-id", Slug: "acme"}
	env := &domain.Environment{ID: "env-id", Slug: "production"}
	evalSvc := &fakeEvalSvc{err: domain.ErrNotFound}
	flagSvc := &fakeFlagSvc{}
	srv := buildServer(nil, nil, key, nil, flagSvc, evalSvc, proj, nil, env, nil)

	w := doPost(t, srv, "cg_valid", "tools/call", map[string]any{
		"name": "evaluate_flag",
		"arguments": map[string]any{
			"project_slug":     "acme",
			"environment_slug": "production",
			"key":              "nonexistent-flag",
			"eval_context":     map[string]any{},
		},
	})

	assertToolError(t, w, "not_found")
}

// @error-path — context.attributes with potentially injected string value succeeds
func TestEvaluateFlag_InjectedAttribute_TreatedAsOpaqueString(t *testing.T) {
	key := makeReadKey("key-id", "proj-id", "env-id")
	proj := &domain.Project{ID: "proj-id", Slug: "acme"}
	env := &domain.Environment{ID: "env-id", Slug: "production"}
	evalSvc := &fakeEvalSvc{
		view: &app.EvalView{Key: "my-flag", Enabled: false, ValueKey: "false", Reason: domain.ReasonDisabled},
	}
	flagSvc := &fakeFlagSvc{}
	srv := buildServer(nil, nil, key, nil, flagSvc, evalSvc, proj, nil, env, nil)

	w := doPost(t, srv, "cg_valid", "tools/call", map[string]any{
		"name": "evaluate_flag",
		"arguments": map[string]any{
			"project_slug":     "acme",
			"environment_slug": "production",
			"key":              "my-flag",
			"eval_context":     map[string]any{"attributes": map[string]any{"plan": "<injected>"}},
		},
	})

	// Should succeed (200) — not reject the call
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	result, _ := resp["result"].(map[string]any)
	if result == nil {
		t.Fatal("expected result")
	}
	// Should not be an error result
	isErr, _ := result["isError"].(bool)
	if isErr {
		t.Error("injected attribute should not cause an error")
	}
}

// ── enable_flag / disable_flag tests ──────────────────────────────────────────

// @auth-bypass — write tool called with read-tier key
func TestEnableFlag_ReadTierKey_ReturnsInsufficientCapability(t *testing.T) {
	key := makeReadKey("key-id", "proj-id", "env-id")
	flagSvc := &fakeFlagSvc{}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, nil, key, nil, flagSvc, evalSvc, nil, nil, nil, nil)

	w := doPost(t, srv, "cg_valid", "tools/call", map[string]any{
		"name":      "enable_flag",
		"arguments": map[string]any{"project_slug": "acme", "environment_slug": "production", "key": "checkout-v2"},
	})

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	result, _ := resp["result"].(map[string]any)
	content, _ := result["content"].([]any)
	textItem, _ := content[0].(map[string]any)
	text, _ := textItem["text"].(string)

	var errBody map[string]any
	json.Unmarshal([]byte(text), &errBody) //nolint:errcheck
	if errBody["error"] != "insufficient_capability" {
		t.Errorf("expected insufficient_capability, got %v", errBody["error"])
	}
	if errBody["required"] != "write" {
		t.Errorf("expected required=write, got %v", errBody["required"])
	}
	if errBody["provided"] != "read" {
		t.Errorf("expected provided=read, got %v", errBody["provided"])
	}
}

// @auth-bypass — write tool with revoked key
func TestEnableFlag_RevokedKey_ReturnsUnauthenticated(t *testing.T) {
	flagSvc := &fakeFlagSvc{}
	evalSvc := &fakeEvalSvc{}
	// Repo returns not-found (simulates revoked/missing)
	srv := buildServer(nil, nil, nil, domain.ErrNotFound, flagSvc, evalSvc, nil, nil, nil, nil)

	w := doPost(t, srv, "cg_revoked", "tools/call", map[string]any{
		"name":      "enable_flag",
		"arguments": map[string]any{"project_slug": "acme", "environment_slug": "production", "key": "f1"},
	})

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	result, _ := resp["result"].(map[string]any)
	if result == nil {
		t.Fatal("expected result")
	}
	errVal, _ := result["error"].(string)
	if errVal != "unauthenticated" {
		t.Errorf("expected unauthenticated, got %q", errVal)
	}
}

// @auth-bypass — write tool after tier downgrade mid-connection
func TestEnableFlag_TierDowngradedLiveCheck_ReturnsInsufficientCapability(t *testing.T) {
	// Connection-time key was write; live DB returns read-tier (downgraded)
	liveKey := makeReadKey("key-id", "proj-id", "env-id")
	flagSvc := &fakeFlagSvc{}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, nil, liveKey, nil, flagSvc, evalSvc, nil, nil, nil, nil)

	w := doPost(t, srv, "cg_valid", "tools/call", map[string]any{
		"name":      "enable_flag",
		"arguments": map[string]any{"project_slug": "acme", "environment_slug": "production", "key": "f1"},
	})

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	result, _ := resp["result"].(map[string]any)
	content, _ := result["content"].([]any)
	if len(content) > 0 {
		textItem, _ := content[0].(map[string]any)
		text, _ := textItem["text"].(string)
		var errBody map[string]any
		json.Unmarshal([]byte(text), &errBody) //nolint:errcheck
		if errBody["error"] != "insufficient_capability" {
			t.Errorf("expected insufficient_capability, got %v", errBody)
		}
	}
}

// @happy — enable flag with write-tier key
func TestEnableFlag_WriteTierKey_EnablesFlag(t *testing.T) {
	key := makeWriteKey("key-id", "proj-id", "env-id")
	proj := &domain.Project{ID: "proj-id", Slug: "acme"}
	env := &domain.Environment{ID: "env-id", Slug: "production"}
	flagSvc := &fakeFlagSvc{}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, nil, key, nil, flagSvc, evalSvc, proj, nil, env, nil)

	w := doPost(t, srv, "cg_valid", "tools/call", map[string]any{
		"name":      "enable_flag",
		"arguments": map[string]any{"project_slug": "acme", "environment_slug": "production", "key": "checkout-v2"},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !flagSvc.setCalled {
		t.Error("SetEnabled was not called")
	}
	if !flagSvc.setEnabled {
		t.Error("SetEnabled was called with enabled=false, expected true")
	}
}

// @happy — disable flag with write-tier key
func TestDisableFlag_WriteTierKey_DisablesFlag(t *testing.T) {
	key := makeWriteKey("key-id", "proj-id", "env-id")
	proj := &domain.Project{ID: "proj-id", Slug: "acme"}
	env := &domain.Environment{ID: "env-id", Slug: "production"}
	flagSvc := &fakeFlagSvc{}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, nil, key, nil, flagSvc, evalSvc, proj, nil, env, nil)

	doPost(t, srv, "cg_valid", "tools/call", map[string]any{
		"name":      "disable_flag",
		"arguments": map[string]any{"project_slug": "acme", "environment_slug": "production", "key": "checkout-v2"},
	})

	if !flagSvc.setCalled {
		t.Error("SetEnabled was not called")
	}
	if flagSvc.setEnabled {
		t.Error("SetEnabled was called with enabled=true, expected false")
	}
}

// @error-path — enable_flag for unknown flag key
func TestEnableFlag_UnknownFlagKey_ReturnsNotFound(t *testing.T) {
	key := makeWriteKey("key-id", "proj-id", "env-id")
	proj := &domain.Project{ID: "proj-id", Slug: "acme"}
	env := &domain.Environment{ID: "env-id", Slug: "production"}
	flagSvc := &fakeFlagSvc{setErr: domain.ErrNotFound}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, nil, key, nil, flagSvc, evalSvc, proj, nil, env, nil)

	w := doPost(t, srv, "cg_valid", "tools/call", map[string]any{
		"name":      "enable_flag",
		"arguments": map[string]any{"project_slug": "acme", "environment_slug": "production", "key": "nonexistent"},
	})

	assertToolError(t, w, "not_found")
}

// @error-path — audit service unavailable on write call returns internal_error
func TestEnableFlag_AuditFailure_ReturnsInternalError(t *testing.T) {
	key := makeWriteKey("key-id", "proj-id", "env-id")
	proj := &domain.Project{ID: "proj-id", Slug: "acme"}
	env := &domain.Environment{ID: "env-id", Slug: "production"}
	flagSvc := &fakeFlagSvc{setErr: errors.New("audit unavailable")}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, nil, key, nil, flagSvc, evalSvc, proj, nil, env, nil)

	w := doPost(t, srv, "cg_valid", "tools/call", map[string]any{
		"name":      "enable_flag",
		"arguments": map[string]any{"project_slug": "acme", "environment_slug": "production", "key": "my-flag"},
	})

	assertToolError(t, w, "internal_error")
}

// ── initialize tests ───────────────────────────────────────────────────────────

func TestHandleInitialize_ReturnsProtocolVersion(t *testing.T) {
	key := makeReadKey("key-id", "proj-id", "env-id")
	flagSvc := &fakeFlagSvc{}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, nil, key, nil, flagSvc, evalSvc, nil, nil, nil, nil)

	w := doPost(t, srv, "cg_valid", "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
	})

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	result, _ := resp["result"].(map[string]any)
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("expected protocolVersion=2024-11-05, got %v", result["protocolVersion"])
	}
}

// ── SSE endpoint tests ────────────────────────────────────────────────────────

// @auth-bypass — no Authorization header on SSE endpoint
func TestHandleSSE_NoBearer_ReturnsUnauthorized(t *testing.T) {
	flagSvc := &fakeFlagSvc{}
	evalSvc := &fakeEvalSvc{}
	srv := buildServer(nil, errors.New("no key"), nil, domain.ErrNotFound, flagSvc, evalSvc, nil, nil, nil, nil)

	req := httptest.NewRequest("GET", "/sse", nil)
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	var errBody map[string]any
	json.NewDecoder(w.Body).Decode(&errBody) //nolint:errcheck
	if errBody["error"] != "unauthenticated" {
		t.Errorf("expected unauthenticated, got %v", errBody)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func assertToolError(t *testing.T, w *httptest.ResponseRecorder, expectedError string) {
	t.Helper()
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	result, _ := resp["result"].(map[string]any)
	if result == nil {
		t.Fatalf("expected result, got: %v", resp)
	}
	content, _ := result["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("expected content in result: %v", result)
	}
	textItem, _ := content[0].(map[string]any)
	text, _ := textItem["text"].(string)
	var errBody map[string]any
	json.Unmarshal([]byte(text), &errBody) //nolint:errcheck
	if errBody["error"] != expectedError {
		t.Errorf("expected error=%q, got %v (full body: %s)", expectedError, errBody["error"], text)
	}
}
