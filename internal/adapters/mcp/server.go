package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/The127/cuttlegate/internal/domain"
	"github.com/The127/cuttlegate/internal/domain/ports"
)

// Server is the MCP HTTP+SSE server (2024-11-05 protocol).
// It binds to a dedicated address and exposes Cuttlegate flag operations as MCP tools.
type Server struct {
	apiKeySvc   apiKeyAuthenticator
	keyRepo     ports.APIKeyRepository
	flagSvc     flagService
	evalSvc     evaluationService
	projectSvc  projectResolver
	envSvc      environmentResolver
	rateLimiter *fixedWindowRateLimiter

	mu       sync.Mutex
	sessions map[string]*session // session ID → session
}

// NewServer constructs an MCP Server.
func NewServer(
	apiKeySvc apiKeyAuthenticator,
	keyRepo ports.APIKeyRepository,
	flagSvc flagService,
	evalSvc evaluationService,
	projectSvc projectResolver,
	envSvc environmentResolver,
) *Server {
	return &Server{
		apiKeySvc:   apiKeySvc,
		keyRepo:     keyRepo,
		flagSvc:     flagSvc,
		evalSvc:     evalSvc,
		projectSvc:  projectSvc,
		envSvc:      envSvc,
		rateLimiter: newFixedWindowRateLimiter(600, 60),
		sessions:    make(map[string]*session),
	}
}

// RegisterRoutes registers the MCP HTTP endpoints on mux.
// GET  /sse      — SSE stream establishment + tool list
// POST /message  — JSON-RPC 2.0 message handling
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /sse", s.handleSSE)
	mux.HandleFunc("POST /message", s.handleMessage)
}

// jsonRPCRequest is a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// jsonRPCResponse is a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// jsonRPCError is a JSON-RPC 2.0 error object.
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// mcpError is the MCP-specific error body embedded in tool results.
type mcpError struct {
	Error    string `json:"error"`
	Required string `json:"required,omitempty"`
	Provided string `json:"provided,omitempty"`
}

func rpcError(id json.RawMessage, code int, msg string) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jsonRPCError{Code: code, Message: msg},
	}
}

func rpcResult(id json.RawMessage, result any) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

// extractBearer extracts the plaintext API key from "Authorization: Bearer <key>".
// Returns empty string if absent or malformed.
func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}

// authenticateRequest authenticates the request using the Bearer token.
// Returns the live API key on success; writes an error response and returns nil on failure.
func (s *Server) authenticateRequest(w http.ResponseWriter, r *http.Request) *domain.APIKey {
	plaintext := extractBearer(r)
	if plaintext == "" {
		writeToolError(w, http.StatusUnauthorized, mcpError{Error: "unauthenticated"})
		return nil
	}
	key, err := s.apiKeySvc.AuthenticateMCP(r.Context(), plaintext)
	if err != nil {
		writeToolError(w, http.StatusUnauthorized, mcpError{Error: "unauthenticated"})
		return nil
	}
	return key
}

// liveKeyCheck performs a per-call live DB lookup to verify the key is still valid
// and returns its current capability tier. This is the security gate per ADR 0028.
func (s *Server) liveKeyCheck(ctx context.Context, plaintext string) (*domain.APIKey, error) {
	hash := domain.HashAPIKey(plaintext)
	key, err := s.keyRepo.GetByHash(ctx, hash)
	if err != nil {
		return nil, errors.New("unauthenticated")
	}
	if key.Revoked() {
		return nil, errors.New("unauthenticated")
	}
	return key, nil
}

// handleSSE establishes the MCP SSE stream and performs connection-time authentication.
// It sends the initial endpoint event and keeps the connection open.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	key := s.authenticateRequest(w, r)
	if key == nil {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	sessID := newSessionID()
	sess := &session{
		id:    sessID,
		keyID: key.ID,
		tier:  key.CapabilityTier,
	}

	s.mu.Lock()
	s.sessions[sessID] = sess
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.sessions, sessID)
		s.mu.Unlock()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Send endpoint event with the message URL including the session ID.
	_, _ = fmt.Fprintf(w, "event: endpoint\ndata: /message?session_id=%s\n\n", sessID)
	flusher.Flush()

	// Block until client disconnects.
	<-r.Context().Done()
}

// handleMessage processes a JSON-RPC 2.0 MCP message from the client.
func (s *Server) handleMessage(w http.ResponseWriter, r *http.Request) {
	// Auth header required on POST as well (per-call auth check).
	plaintext := extractBearer(r)
	if plaintext == "" {
		writeToolError(w, http.StatusUnauthorized, mcpError{Error: "unauthenticated"})
		return
	}

	sessID := r.URL.Query().Get("session_id")
	if sessID == "" {
		writeJSON(w, http.StatusBadRequest, rpcError(nil, -32600, "missing session_id"))
		return
	}

	s.mu.Lock()
	sess := s.sessions[sessID] // may be nil if client sends before SSE stream is established
	s.mu.Unlock()

	var req jsonRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusOK, rpcError(nil, -32700, "parse error"))
		return
	}

	// Per-call live key lookup — this is the security gate.
	liveKey, err := s.liveKeyCheck(r.Context(), plaintext)
	if err != nil {
		writeJSON(w, http.StatusOK, rpcResult(req.ID, mcpError{Error: "unauthenticated"}))
		return
	}

	// Update session tier from live lookup (handles downgrades).
	if sess != nil {
		sess.tier = liveKey.CapabilityTier
	}

	ctx := domain.NewAuthContext(r.Context(), domain.AuthContext{
		UserID: liveKey.ID,
		Role:   roleForTier(liveKey.CapabilityTier),
	})

	var resp jsonRPCResponse
	switch req.Method {
	case "initialize":
		resp = s.handleInitialize(ctx, req)
	case "tools/list":
		resp = s.handleToolsList(ctx, req, liveKey.CapabilityTier)
	case "tools/call":
		resp = s.handleToolsCall(ctx, req, liveKey, plaintext)
	default:
		resp = rpcError(req.ID, -32601, "method not found")
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleInitialize responds to the MCP initialize handshake.
func (s *Server) handleInitialize(_ context.Context, req jsonRPCRequest) jsonRPCResponse {
	return rpcResult(req.ID, map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "cuttlegate",
			"version": "1.0.0",
		},
	})
}

// handleToolsList returns the tools visible to this credential's capability tier.
func (s *Server) handleToolsList(_ context.Context, req jsonRPCRequest, tier domain.ToolCapabilityTier) jsonRPCResponse {
	tools := buildToolList(tier)
	return rpcResult(req.ID, map[string]any{"tools": tools})
}

// toolCallParams is the JSON-RPC params for tools/call.
type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// handleToolsCall dispatches a tool call to the appropriate handler.
func (s *Server) handleToolsCall(ctx context.Context, req jsonRPCRequest, key *domain.APIKey, plaintext string) jsonRPCResponse {
	var p toolCallParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return rpcError(req.ID, -32602, "invalid params")
	}

	// Determine required tier for this tool.
	requiredTier, known := toolTier(p.Name)
	if !known {
		return rpcResult(req.ID, toolErrorResult(mcpError{Error: "not_found"}))
	}

	// Per-call capability enforcement.
	if !key.CapabilityTier.Permits(requiredTier) {
		return rpcResult(req.ID, toolErrorResult(mcpError{
			Error:    "insufficient_capability",
			Required: string(requiredTier),
			Provided: string(key.CapabilityTier),
		}))
	}

	switch p.Name {
	case "list_flags":
		return s.callListFlags(ctx, req.ID, p.Arguments, key)
	case "evaluate_flag":
		return s.callEvaluateFlag(ctx, req.ID, p.Arguments, key)
	case "enable_flag":
		return s.callSetEnabled(ctx, req.ID, p.Arguments, key, true)
	case "disable_flag":
		return s.callSetEnabled(ctx, req.ID, p.Arguments, key, false)
	default:
		return rpcResult(req.ID, toolErrorResult(mcpError{Error: "not_found"}))
	}
}

// toolErrorResult wraps an mcpError as an MCP tool result (not a JSON-RPC error).
func toolErrorResult(e mcpError) map[string]any {
	b, _ := json.Marshal(e)
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": string(b)},
		},
		"isError": true,
	}
}

// toolSuccessResult wraps a value as an MCP tool result.
func toolSuccessResult(v any) map[string]any {
	b, _ := json.Marshal(v)
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": string(b)},
		},
	}
}

// roleForTier maps a ToolCapabilityTier to the appropriate RBAC Role for app service calls.
func roleForTier(tier domain.ToolCapabilityTier) domain.Role {
	switch tier {
	case domain.TierRead:
		return domain.RoleViewer
	default:
		return domain.RoleEditor
	}
}

// writeToolError writes an MCP error body as JSON.
func writeToolError(w http.ResponseWriter, status int, e mcpError) {
	b, _ := json.Marshal(e)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

// writeJSON writes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("mcp: failed to marshal response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

// resolveProjectAndEnv resolves project slug and environment slug to their IDs,
// enforcing that they match the key's scoped project and environment.
func (s *Server) resolveProjectAndEnv(ctx context.Context, projectSlug, environmentSlug string, key *domain.APIKey) (projectID, environmentID, envSlug string, aerr mcpError, ok bool) {
	proj, err := s.projectSvc.GetBySlug(ctx, projectSlug)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", "", "", mcpError{Error: "not_found"}, false
		}
		return "", "", "", mcpError{Error: "internal_error"}, false
	}
	if proj.ID != key.ProjectID {
		return "", "", "", mcpError{Error: "forbidden"}, false
	}

	env, err := s.envSvc.GetBySlug(ctx, proj.ID, environmentSlug)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", "", "", mcpError{Error: "not_found"}, false
		}
		return "", "", "", mcpError{Error: "internal_error"}, false
	}
	if env.ID != key.EnvironmentID {
		return "", "", "", mcpError{Error: "forbidden"}, false
	}

	return proj.ID, env.ID, env.Slug, mcpError{}, true
}
