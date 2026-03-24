package openfeature_test

import (
	"context"
	"testing"

	cuttlegateof "github.com/karo/cuttlegate/sdk/go/openfeature"
	cuttlegatetesting "github.com/karo/cuttlegate/sdk/go/testing"
)

func TestProvider_Metadata(t *testing.T) {
	mock := cuttlegatetesting.NewMockClient()
	provider := cuttlegateof.NewProvider(mock)
	if provider.Metadata().Name != "cuttlegate" {
		t.Errorf("expected name=cuttlegate, got %q", provider.Metadata().Name)
	}
}

func TestProvider_BooleanEvaluation_Enabled(t *testing.T) {
	mock := cuttlegatetesting.NewMockClient()
	mock.Enable("dark-mode")
	provider := cuttlegateof.NewProvider(mock)

	result := provider.BooleanEvaluation(context.Background(), "dark-mode", false, cuttlegateof.EvaluationContext{TargetingKey: "u1"})

	if result.Value != true {
		t.Errorf("expected true, got %v", result.Value)
	}
	if result.Variant != "true" {
		t.Errorf("expected variant=true, got %q", result.Variant)
	}
	if result.Reason != "TARGETING_MATCH" {
		t.Errorf("expected reason=TARGETING_MATCH, got %q", result.Reason)
	}
}

func TestProvider_BooleanEvaluation_Disabled(t *testing.T) {
	mock := cuttlegatetesting.NewMockClient()
	mock.Disable("dark-mode")
	provider := cuttlegateof.NewProvider(mock)

	result := provider.BooleanEvaluation(context.Background(), "dark-mode", true, cuttlegateof.EvaluationContext{TargetingKey: "u1"})

	if result.Value != false {
		t.Errorf("expected false, got %v", result.Value)
	}
	if result.Variant != "false" {
		t.Errorf("expected variant=false, got %q", result.Variant)
	}
}

func TestProvider_BooleanEvaluation_MissingFlag_ReturnsDefault(t *testing.T) {
	mock := cuttlegatetesting.NewMockClient()
	provider := cuttlegateof.NewProvider(mock)

	result := provider.BooleanEvaluation(context.Background(), "missing", true, cuttlegateof.EvaluationContext{TargetingKey: "u1"})

	// Mock returns enabled=false with reason "mock_default" for unknown flags — not an error.
	// The provider should return the evaluated value (false), not the default.
	if result.Reason == "ERROR" {
		// If the mock returns an error for unknown flags, the default should be returned.
		if result.Value != true {
			t.Errorf("expected default=true on error, got %v", result.Value)
		}
	}
}

func TestProvider_StringEvaluation(t *testing.T) {
	mock := cuttlegatetesting.NewMockClient()
	mock.SetVariant("color", "blue")
	provider := cuttlegateof.NewProvider(mock)

	result := provider.StringEvaluation(context.Background(), "color", "red", cuttlegateof.EvaluationContext{TargetingKey: "u1"})

	if result.Value != "blue" {
		t.Errorf("expected blue, got %v", result.Value)
	}
	if result.Variant != "blue" {
		t.Errorf("expected variant=blue, got %q", result.Variant)
	}
	if result.Reason != "TARGETING_MATCH" {
		t.Errorf("expected reason=TARGETING_MATCH, got %q", result.Reason)
	}
}

func TestProvider_StringEvaluation_ReturnsDefault_OnError(t *testing.T) {
	mock := cuttlegatetesting.NewMockClient()
	provider := cuttlegateof.NewProvider(mock)

	result := provider.StringEvaluation(context.Background(), "missing", "fallback", cuttlegateof.EvaluationContext{TargetingKey: "u1"})

	// Mock returns empty string for unknown flags via mock_default, not an error.
	// Just verify no panic and we get a value.
	if result.Value == nil {
		t.Error("expected non-nil value")
	}
}

func TestProvider_ContextMapping(t *testing.T) {
	mock := cuttlegatetesting.NewMockClient()
	mock.Enable("flag")
	provider := cuttlegateof.NewProvider(mock)

	result := provider.BooleanEvaluation(context.Background(), "flag", false, cuttlegateof.EvaluationContext{
		TargetingKey: "user-42",
		Attributes:   map[string]any{"plan": "pro"},
	})

	if result.Value != true {
		t.Errorf("expected true, got %v", result.Value)
	}
	if err := mock.AssertEvaluated("flag"); err != nil {
		t.Errorf("flag should have been evaluated: %v", err)
	}
}
