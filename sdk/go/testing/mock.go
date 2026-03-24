package cuttlegatetesting

import (
	"context"
	"fmt"
	"sync"

	cuttlegate "github.com/karo/cuttlegate/sdk/go"
)

const mockEvaluatedAt = "1970-01-01T00:00:00Z"

// MockClient extends cuttlegate.Client with helpers for controlling flag state
// and asserting evaluation calls in tests.
type MockClient interface {
	cuttlegate.Client

	// Enable marks a flag as enabled with Variant="true".
	Enable(key string)

	// Disable removes a flag, causing it to return enabled=false with reason "mock_default".
	Disable(key string)

	// SetVariant enables a flag and sets its variant value.
	SetVariant(key, value string)

	// AssertEvaluated returns an error if the flag was not evaluated.
	// Intended for use with testing.TB — call as mock.AssertEvaluated("my-flag") in a test.
	AssertEvaluated(key string) error

	// AssertNotEvaluated returns an error if the flag was evaluated.
	AssertNotEvaluated(key string) error

	// Reset clears all flag state and recorded evaluations.
	Reset()
}

type flagConfig struct {
	enabled bool
	value   string
	variant string
}

type mockClient struct {
	mu        sync.Mutex
	flags     map[string]flagConfig
	evaluated map[string]struct{}
}

// NewMockClient returns a MockClient with no flags set.
// Use Enable, Disable, or SetVariant before calling code under test.
//
// Example:
//
//	mock := cuttlegatetesting.NewMockClient()
//	mock.Enable("dark-mode")
//
//	result, err := myService.GetFeatures(ctx, mock)
//	if err != nil { t.Fatal(err) }
//	if err := mock.AssertEvaluated("dark-mode"); err != nil { t.Error(err) }
func NewMockClient() MockClient {
	return &mockClient{
		flags:     make(map[string]flagConfig),
		evaluated: make(map[string]struct{}),
	}
}

func (m *mockClient) EvaluateAll(ctx context.Context, _ cuttlegate.EvalContext) (map[string]cuttlegate.EvalResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	results := make(map[string]cuttlegate.EvalResult, len(m.flags))
	for key, cfg := range m.flags {
		m.evaluated[key] = struct{}{}
		results[key] = cuttlegate.EvalResult{
			Key:         key,
			Enabled:     cfg.enabled,
			Value:       cfg.value,
			Variant:     cfg.variant,
			Reason:      "mock",
			EvaluatedAt: mockEvaluatedAt,
		}
	}
	return results, nil
}

func (m *mockClient) Evaluate(ctx context.Context, key string, _ cuttlegate.EvalContext) (cuttlegate.EvalResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.evaluated[key] = struct{}{}
	cfg, ok := m.flags[key]
	if !ok {
		return cuttlegate.EvalResult{
			Key:         key,
			Enabled:     false,
			Value:       "",
			Variant:     "",
			Reason:      "mock_default",
			EvaluatedAt: mockEvaluatedAt,
		}, nil
	}
	return cuttlegate.EvalResult{
		Key:         key,
		Enabled:     cfg.enabled,
		Value:       cfg.value,
		Variant:     cfg.variant,
		Reason:      "mock",
		EvaluatedAt: mockEvaluatedAt,
	}, nil
}

func (m *mockClient) Bool(ctx context.Context, key string, evalCtx cuttlegate.EvalContext) (bool, error) {
	result, err := m.Evaluate(ctx, key, evalCtx)
	if err != nil {
		return false, err
	}
	return result.Variant == "true", nil
}

func (m *mockClient) String(ctx context.Context, key string, evalCtx cuttlegate.EvalContext) (string, error) {
	result, err := m.Evaluate(ctx, key, evalCtx)
	if err != nil {
		return "", err
	}
	return result.Variant, nil
}

func (m *mockClient) EvaluateFlag(ctx context.Context, key string, _ cuttlegate.EvalContext) (cuttlegate.FlagResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.evaluated[key] = struct{}{}
	cfg, ok := m.flags[key]
	if !ok {
		return cuttlegate.FlagResult{Enabled: false, Value: "", Variant: "", Reason: "mock_default"}, nil
	}
	return cuttlegate.FlagResult{Enabled: cfg.enabled, Value: cfg.value, Variant: cfg.variant, Reason: "mock"}, nil
}

func (m *mockClient) Enable(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flags[key] = flagConfig{enabled: true, value: "", variant: "true"}
}

func (m *mockClient) Disable(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flags[key] = flagConfig{enabled: false, value: "", variant: "false"}
}

func (m *mockClient) SetVariant(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flags[key] = flagConfig{enabled: true, value: value, variant: value}
}

func (m *mockClient) AssertEvaluated(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.evaluated[key]; !ok {
		return fmt.Errorf("cuttlegatetesting: expected flag %q to have been evaluated, but it was not", key)
	}
	return nil
}

func (m *mockClient) AssertNotEvaluated(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.evaluated[key]; ok {
		return fmt.Errorf("cuttlegatetesting: expected flag %q NOT to have been evaluated, but it was", key)
	}
	return nil
}

func (m *mockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flags = make(map[string]flagConfig)
	m.evaluated = make(map[string]struct{})
}

// Subscribe returns an immediately-closed update channel and error channel.
// The mock does not simulate real-time streaming — it satisfies the interface
// for code that type-checks against cuttlegate.Client. Use a real httptest.Server
// if your test needs to exercise the Subscribe code path.
func (m *mockClient) Subscribe(ctx context.Context, key string) (<-chan cuttlegate.FlagUpdate, <-chan error, error) {
	updates := make(chan cuttlegate.FlagUpdate)
	errs := make(chan error)
	close(updates)
	close(errs)
	return updates, errs, nil
}
