package mcp

import (
	"context"
	"encoding/json"

	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

// listFlagsArgs are the input arguments for the list_flags tool.
type listFlagsArgs struct {
	ProjectSlug     string `json:"project_slug"`
	EnvironmentSlug string `json:"environment_slug"`
}

// listFlagResult is one entry in the list_flags response.
type listFlagResult struct {
	Key      string `json:"key"`
	Enabled  bool   `json:"enabled"`
	ValueKey string `json:"value_key"`
	Type     string `json:"type"`
}

func (s *Server) callListFlags(ctx context.Context, id json.RawMessage, rawArgs json.RawMessage, key *domain.APIKey) jsonRPCResponse {
	var args listFlagsArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: toolErrorResult(mcpError{Error: "internal_error"})}
	}

	projectID, environmentID, _, aerr, ok := s.resolveProjectAndEnv(ctx, args.ProjectSlug, args.EnvironmentSlug, key)
	if !ok {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: toolErrorResult(aerr)}
	}

	views, err := s.flagSvc.ListByEnvironment(ctx, projectID, environmentID)
	if err != nil {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: toolErrorResult(mcpError{Error: "internal_error"})}
	}

	results := make([]listFlagResult, 0, len(views))
	for _, v := range views {
		valueKey := v.Flag.DefaultVariantKey
		results = append(results, listFlagResult{
			Key:      v.Flag.Key,
			Enabled:  v.Enabled,
			ValueKey: valueKey,
			Type:     string(v.Flag.Type),
		})
	}

	return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: toolSuccessResult(results)}
}

// evalFlagArgs are the input arguments for the evaluate_flag tool.
type evalFlagArgs struct {
	ProjectSlug     string      `json:"project_slug"`
	EnvironmentSlug string      `json:"environment_slug"`
	Key             string      `json:"key"`
	EvalContext     evalContext `json:"eval_context"`
}

// evalContext mirrors the MCP tool parameter shape.
type evalContext struct {
	UserID     string            `json:"user_id"`
	Attributes map[string]string `json:"attributes"`
}

// evalFlagResult is the evaluate_flag response shape.
type evalFlagResult struct {
	Key      string `json:"key"`
	Enabled  bool   `json:"enabled"`
	ValueKey string `json:"value_key"`
	Reason   string `json:"reason"`
}

func (s *Server) callEvaluateFlag(ctx context.Context, id json.RawMessage, rawArgs json.RawMessage, key *domain.APIKey) jsonRPCResponse {
	var args evalFlagArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: toolErrorResult(mcpError{Error: "internal_error"})}
	}

	// Rate limiting keyed on the API key UUID.
	ac, _ := domain.AuthContextFrom(ctx)
	if !s.rateLimiter.allow(ac.UserID) {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: toolErrorResult(mcpError{Error: "internal_error"})}
	}

	projectID, environmentID, _, aerr, ok := s.resolveProjectAndEnv(ctx, args.ProjectSlug, args.EnvironmentSlug, key)
	if !ok {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: toolErrorResult(aerr)}
	}

	evalCtx := domain.EvalContext{
		UserID:     args.EvalContext.UserID,
		Attributes: args.EvalContext.Attributes,
	}
	if evalCtx.Attributes == nil {
		evalCtx.Attributes = map[string]string{}
	}

	view, err := s.evalSvc.Evaluate(ctx, projectID, environmentID, args.Key, evalCtx)
	if err != nil {
		if isDomainNotFound(err) {
			return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: toolErrorResult(mcpError{Error: "not_found"})}
		}
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: toolErrorResult(mcpError{Error: "internal_error"})}
	}

	return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: toolSuccessResult(evalFlagResult{
		Key:      view.Key,
		Enabled:  view.Enabled,
		ValueKey: view.ValueKey,
		Reason:   string(view.Reason),
	})}
}

// setEnabledArgs are the input arguments for enable_flag and disable_flag.
type setEnabledArgs struct {
	ProjectSlug     string `json:"project_slug"`
	EnvironmentSlug string `json:"environment_slug"`
	Key             string `json:"key"`
}

func (s *Server) callSetEnabled(ctx context.Context, id json.RawMessage, rawArgs json.RawMessage, key *domain.APIKey, enabled bool) jsonRPCResponse {
	var args setEnabledArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: toolErrorResult(mcpError{Error: "internal_error"})}
	}

	projectID, environmentID, envSlug, aerr, ok := s.resolveProjectAndEnv(ctx, args.ProjectSlug, args.EnvironmentSlug, key)
	if !ok {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: toolErrorResult(aerr)}
	}

	err := s.flagSvc.SetEnabled(ctx, app.SetEnabledParams{
		ProjectID:     projectID,
		EnvironmentID: environmentID,
		FlagKey:       args.Key,
		Enabled:       enabled,
		ProjectSlug:   args.ProjectSlug,
		EnvSlug:       envSlug,
		Source:        "mcp",
	})
	if err != nil {
		if isDomainNotFound(err) {
			return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: toolErrorResult(mcpError{Error: "not_found"})}
		}
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: toolErrorResult(mcpError{Error: "internal_error"})}
	}

	state := "enabled"
	if !enabled {
		state = "disabled"
	}
	return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: toolSuccessResult(map[string]any{
		"key":     args.Key,
		"enabled": enabled,
		"state":   state,
	})}
}

// isDomainNotFound reports whether err is domain.ErrNotFound.
func isDomainNotFound(err error) bool {
	return err != nil && err.Error() == domain.ErrNotFound.Error()
}
