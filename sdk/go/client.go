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
	// Evaluate evaluates all flags for the given context and returns results for
	// every flag in the project/environment. ctx is used for cancellation and
	// deadline propagation.
	Evaluate(ctx context.Context, evalCtx EvalContext) ([]EvalResult, error)

	// EvaluateFlag evaluates a single flag by key. It calls Evaluate internally
	// and filters the result. Returns FlagResult with Reason "not_found" if the
	// key is absent — never returns an error for a missing key.
	EvaluateFlag(ctx context.Context, key string, evalCtx EvalContext) (FlagResult, error)
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

	// HTTPClient is an optional custom HTTP client. If nil, a default client
	// with the configured Timeout is used. If provided, the caller owns the
	// client's timeout — the Timeout field is ignored.
	HTTPClient *http.Client

	// Timeout is the request timeout applied when HTTPClient is nil.
	// Defaults to 10s.
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

	return &client{
		baseURL:      cfg.BaseURL,
		serviceToken: cfg.ServiceToken,
		project:      cfg.Project,
		environment:  cfg.Environment,
		httpClient:   httpClient,
	}, nil
}

// client is the unexported concrete implementation of Client.
type client struct {
	baseURL      string
	serviceToken string
	project      string
	environment  string
	httpClient   *http.Client
}

type bulkEvalRequest struct {
	Context EvalContext `json:"context"`
}

type bulkEvalResponseFlag struct {
	Key     string  `json:"key"`
	Enabled bool    `json:"enabled"`
	Value   *string `json:"value"`
	Reason  string  `json:"reason"`
	Type    string  `json:"type"`
}

type bulkEvalResponse struct {
	Flags       []bulkEvalResponseFlag `json:"flags"`
	EvaluatedAt string                 `json:"evaluated_at"`
}

func (c *client) Evaluate(ctx context.Context, evalCtx EvalContext) ([]EvalResult, error) {
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

	results := make([]EvalResult, 0, len(parsed.Flags))
	for _, f := range parsed.Flags {
		value := ""
		if f.Value != nil {
			value = *f.Value
		}
		results = append(results, EvalResult{
			Key:         f.Key,
			Enabled:     f.Enabled,
			Value:       value,
			Reason:      f.Reason,
			EvaluatedAt: parsed.EvaluatedAt,
		})
	}
	return results, nil
}

func (c *client) EvaluateFlag(ctx context.Context, key string, evalCtx EvalContext) (FlagResult, error) {
	results, err := c.Evaluate(ctx, evalCtx)
	if err != nil {
		return FlagResult{}, err
	}
	for _, r := range results {
		if r.Key == key {
			return FlagResult{Enabled: r.Enabled, Value: r.Value, Reason: r.Reason}, nil
		}
	}
	return FlagResult{Enabled: false, Value: "", Reason: "not_found"}, nil
}
