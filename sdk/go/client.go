package cuttlegate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultTimeout = 10 * time.Second

// Client is the interface implemented by both the real client and test mocks.
// Consumers should type variables as Client, not as the concrete type returned
// by NewClient.
type Client interface {
	// EvaluateAll evaluates all flags for the given context and returns a map
	// from flag key to EvalResult. Uses the bulk evaluation endpoint — one HTTP
	// request regardless of how many flags exist. ctx is used for cancellation
	// and deadline propagation.
	EvaluateAll(ctx context.Context, evalCtx EvalContext) (map[string]EvalResult, error)

	// Evaluate evaluates a single flag by key. Returns NotFoundError if the key
	// does not exist in the project — never silently defaults.
	Evaluate(ctx context.Context, key string, evalCtx EvalContext) (EvalResult, error)

	// Bool evaluates a flag and returns its boolean value. Returns false and
	// NotFoundError if the flag key does not exist. Never silently defaults.
	// Each call makes one HTTP round trip — prefer EvaluateAll when evaluating multiple flags.
	Bool(ctx context.Context, key string, evalCtx EvalContext) (bool, error)

	// String evaluates a flag and returns its string value. Returns "" and
	// NotFoundError if the flag key does not exist. Never silently defaults.
	// Each call makes one HTTP round trip — prefer EvaluateAll when evaluating multiple flags.
	String(ctx context.Context, key string, evalCtx EvalContext) (string, error)

	// Deprecated: EvaluateFlag evaluates a single flag by key. Returns FlagResult with
	// Reason "not_found" if the key is absent — does not return an error for a
	// missing key. Use Evaluate, Bool, or String for new code.
	EvaluateFlag(ctx context.Context, key string, evalCtx EvalContext) (FlagResult, error)

	// Subscribe opens a real-time stream of flag state changes for the given key.
	// Returns an update channel, an error channel, and an error if configuration
	// is invalid. Both channels are closed when ctx is cancelled.
	//
	// The update channel delivers FlagUpdate values when the flag state changes.
	// The error channel delivers non-fatal errors (reconnect attempts, parse errors)
	// and a terminal AuthError before closing on 401/403.
	//
	// Multiple independent Subscribe calls on the same key return independent
	// streams — cancelling one does not affect the others.
	Subscribe(ctx context.Context, key string) (<-chan FlagUpdate, <-chan error, error)
}

// Config holds the configuration for a Cuttlegate client.
type Config struct {
	// BaseURL is the base URL of the Cuttlegate server. Required.
	// Example: "https://flags.example.com"
	BaseURL string

	// ServiceToken is the service account token for authentication. Required.
	ServiceToken string

	// Project is the project slug. Required.
	Project string

	// Environment is the environment slug to evaluate against. Required.
	// Example: "production"
	Environment string

	// HTTPClient is an optional custom HTTP client for evaluation requests.
	// If nil, a default client with the configured Timeout is used. If provided,
	// the caller owns the client's timeout — the Timeout field is ignored.
	HTTPClient *http.Client

	// StreamHTTPClient is an optional custom HTTP client for SSE streaming
	// connections (Subscribe). It must not have a short timeout — SSE
	// connections are long-lived. If nil, a client with no timeout is used.
	StreamHTTPClient *http.Client

	// Timeout is the request timeout applied to evaluation requests when HTTPClient is nil.
	// Defaults to 10s. Does not apply to SSE streaming connections.
	Timeout time.Duration
}

// NewClient constructs an authenticated Cuttlegate client.
// Returns an error if any required field is missing. No network calls are made.
func NewClient(cfg Config) (Client, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("cuttlegate: BaseURL is required")
	}
	if cfg.ServiceToken == "" {
		return nil, fmt.Errorf("cuttlegate: ServiceToken is required")
	}
	if cfg.Project == "" {
		return nil, fmt.Errorf("cuttlegate: Project is required")
	}
	if cfg.Environment == "" {
		return nil, fmt.Errorf("cuttlegate: Environment is required")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}

	streamHTTPClient := cfg.StreamHTTPClient
	if streamHTTPClient == nil {
		streamHTTPClient = &http.Client{} // no timeout — SSE connections are long-lived
	}

	return &client{
		baseURL:          cfg.BaseURL,
		serviceToken:     cfg.ServiceToken,
		project:          cfg.Project,
		environment:      cfg.Environment,
		httpClient:       httpClient,
		streamHTTPClient: streamHTTPClient,
	}, nil
}

// client is the unexported concrete implementation of Client.
type client struct {
	baseURL          string
	serviceToken     string
	project          string
	environment      string
	httpClient       *http.Client
	streamHTTPClient *http.Client
}

type bulkEvalRequest struct {
	Context EvalContext `json:"context"`
}

type bulkEvalResponseFlag struct {
	Key      string  `json:"key"`
	Enabled  bool    `json:"enabled"`
	Value    *string `json:"value"`
	ValueKey string  `json:"value_key"`
	Reason   string  `json:"reason"`
	Type     string  `json:"type"`
}

type bulkEvalResponse struct {
	Flags       []bulkEvalResponseFlag `json:"flags"`
	EvaluatedAt string                 `json:"evaluated_at"`
}

// doBulkEval makes the HTTP request to the bulk evaluation endpoint and returns
// parsed results keyed by flag key.
func (c *client) doBulkEval(ctx context.Context, evalCtx EvalContext) (map[string]EvalResult, error) {
	url := fmt.Sprintf(
		"%s/api/v1/projects/%s/environments/%s/evaluate",
		c.baseURL, c.project, c.environment,
	)

	body, err := json.Marshal(bulkEvalRequest{Context: evalCtx})
	if err != nil {
		return nil, fmt.Errorf("cuttlegate: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cuttlegate: failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.serviceToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cuttlegate: request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, &AuthError{StatusCode: resp.StatusCode, Message: http.StatusText(resp.StatusCode)}
	case http.StatusNotFound:
		return nil, &NotFoundError{Resource: "project", Key: c.project}
	}
	if resp.StatusCode >= 500 {
		return nil, &ServerError{StatusCode: resp.StatusCode, Message: http.StatusText(resp.StatusCode)}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cuttlegate: unexpected status %d", resp.StatusCode)
	}

	var parsed bulkEvalResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("cuttlegate: failed to decode response: %w", err)
	}

	results := make(map[string]EvalResult, len(parsed.Flags))
	for _, f := range parsed.Flags {
		// value_key is always present for servers that support it; fall back to
		// dereferencing Value for older servers that do not yet send value_key.
		variant := f.ValueKey
		value := ""
		if f.Value != nil {
			value = *f.Value
		}
		if variant == "" {
			variant = value
		}
		results[f.Key] = EvalResult{
			Key:         f.Key,
			Enabled:     f.Enabled,
			Value:       value,
			Variant:     variant,
			Reason:      f.Reason,
			EvaluatedAt: parsed.EvaluatedAt,
		}
	}
	return results, nil
}

func (c *client) EvaluateAll(ctx context.Context, evalCtx EvalContext) (map[string]EvalResult, error) {
	return c.doBulkEval(ctx, evalCtx)
}

func (c *client) Evaluate(ctx context.Context, key string, evalCtx EvalContext) (EvalResult, error) {
	results, err := c.doBulkEval(ctx, evalCtx)
	if err != nil {
		return EvalResult{}, err
	}
	r, ok := results[key]
	if !ok {
		return EvalResult{}, &NotFoundError{Resource: "flag", Key: key}
	}
	return r, nil
}

func (c *client) Bool(ctx context.Context, key string, evalCtx EvalContext) (bool, error) {
	result, err := c.Evaluate(ctx, key, evalCtx)
	if err != nil {
		return false, err
	}
	return result.Variant == "true", nil
}

func (c *client) String(ctx context.Context, key string, evalCtx EvalContext) (string, error) {
	result, err := c.Evaluate(ctx, key, evalCtx)
	if err != nil {
		return "", err
	}
	return result.Value, nil
}

func (c *client) EvaluateFlag(ctx context.Context, key string, evalCtx EvalContext) (FlagResult, error) {
	results, err := c.doBulkEval(ctx, evalCtx)
	if err != nil {
		return FlagResult{}, err
	}
	r, ok := results[key]
	if !ok {
		return FlagResult{Enabled: false, Value: "", Variant: "", Reason: "not_found"}, nil
	}
	return FlagResult{Enabled: r.Enabled, Value: r.Value, Variant: r.Variant, Reason: r.Reason}, nil
}
