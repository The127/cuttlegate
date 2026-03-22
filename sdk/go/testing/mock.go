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

	// Enable marks a flag as enabled with no variant value.
	Enable(key string)

	// Disable removes a flag, causing it to return enabled=false with reason "mock_default".
	Disable(key string)

	// SetVariant enables a flag and sets its variant value.
	SetVariant(key, value string)

	// AssertEvaluated panics (via testing.T-style error) if the flag was not evaluated.
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

func (m *mockClient) Evaluate(ctx context.Context, _ cuttlegate.EvalContext) ([]cuttlegate.EvalResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	results := make([]cuttlegate.EvalResult, 0, len(m.flags))
	for key, cfg := range m.flags {
		m.evaluated[key] = struct{}{}
		results = append(results, cuttlegate.EvalResult{
			Key:         key,
			Enabled:     cfg.enabled,
			Value:       cfg.value,
			Reason:      "mock",
			EvaluatedAt: mockEvaluatedAt,
		})
	}
	return results, nil
}

func (m *mockClient) EvaluateFlag(ctx context.Context, key string, _ cuttlegate.EvalContext) (cuttlegate.FlagResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.evaluated[key] = struct{}{}
	cfg, ok := m.flags[key]
	if !ok {
		return cuttlegate.FlagResult{Enabled: false, Value: "", Reason: "mock_default"}, nil
	}
	return cuttlegate.FlagResult{Enabled: cfg.enabled, Value: cfg.value, Reason: "mock"}, nil
}

func (m *mockClient) Enable(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flags[key] = flagConfig{enabled: true, value: ""}
}

func (m *mockClient) Disable(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.flags, key)
}

func (m *mockClient) SetVariant(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flags[key] = flagConfig{enabled: true, value: value}
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
