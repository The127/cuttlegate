package openfeature_test

import (
	"context"
	"testing"

	of "github.com/open-feature/go-sdk/openfeature"

	cuttlegateof "github.com/karo/cuttlegate/sdk/go/openfeature"
	cuttlegatetesting "github.com/karo/cuttlegate/sdk/go/testing"
)

func TestProvider_ImplementsFeatureProvider(t *testing.T) {
	mock := cuttlegatetesting.NewMockClient()
	var _ of.FeatureProvider = cuttlegateof.NewProvider(mock)
}

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

	result := provider.BooleanEvaluation(context.Background(), "dark-mode", false,
		of.FlattenedContext{"targetingKey": "u1"})

	if result.Value != true {
		t.Errorf("expected true, got %v", result.Value)
	}
	if result.Variant != "true" {
		t.Errorf("expected variant=true, got %q", result.Variant)
	}
	if result.Reason != of.TargetingMatchReason {
		t.Errorf("expected TARGETING_MATCH, got %q", result.Reason)
	}
}

func TestProvider_BooleanEvaluation_Disabled(t *testing.T) {
	mock := cuttlegatetesting.NewMockClient()
	mock.Disable("dark-mode")
	provider := cuttlegateof.NewProvider(mock)

	result := provider.BooleanEvaluation(context.Background(), "dark-mode", true,
		of.FlattenedContext{"targetingKey": "u1"})

	if result.Value != false {
		t.Errorf("expected false, got %v", result.Value)
	}
}

func TestProvider_StringEvaluation(t *testing.T) {
	mock := cuttlegatetesting.NewMockClient()
	mock.SetVariant("color", "blue")
	provider := cuttlegateof.NewProvider(mock)

	result := provider.StringEvaluation(context.Background(), "color", "red",
		of.FlattenedContext{"targetingKey": "u1"})

	if result.Value != "blue" {
		t.Errorf("expected blue, got %v", result.Value)
	}
	if result.Reason != of.TargetingMatchReason {
		t.Errorf("expected TARGETING_MATCH, got %q", result.Reason)
	}
}

// Conformance: register provider with the OpenFeature API and resolve through it.
func TestProvider_Conformance_Boolean(t *testing.T) {
	mock := cuttlegatetesting.NewMockClient()
	mock.Enable("feature-x")
	provider := cuttlegateof.NewProvider(mock)

	if err := of.SetNamedProviderAndWait("conformance-test", provider); err != nil {
		t.Fatalf("SetProviderAndWait: %v", err)
	}
	defer of.Shutdown()

	client := of.NewClient("conformance-test")
	val, err := client.BooleanValue(context.Background(), "feature-x", false, of.NewEvaluationContext("u1", nil))
	if err != nil {
		t.Fatalf("BooleanValue: %v", err)
	}
	if val != true {
		t.Errorf("expected true, got %v", val)
	}
}

func TestProvider_Conformance_String(t *testing.T) {
	mock := cuttlegatetesting.NewMockClient()
	mock.SetVariant("theme", "dark")
	provider := cuttlegateof.NewProvider(mock)

	if err := of.SetNamedProviderAndWait("conformance-str", provider); err != nil {
		t.Fatalf("SetProviderAndWait: %v", err)
	}

	client := of.NewClient("conformance-str")
	val, err := client.StringValue(context.Background(), "theme", "light", of.NewEvaluationContext("u1", nil))
	if err != nil {
		t.Fatalf("StringValue: %v", err)
	}
	if val != "dark" {
		t.Errorf("expected dark, got %v", val)
	}
}

func TestProvider_ContextMapping(t *testing.T) {
	mock := cuttlegatetesting.NewMockClient()
	mock.Enable("flag")
	provider := cuttlegateof.NewProvider(mock)

	result := provider.BooleanEvaluation(context.Background(), "flag", false,
		of.FlattenedContext{"targetingKey": "user-42", "plan": "pro"})

	if result.Value != true {
		t.Errorf("expected true, got %v", result.Value)
	}
	if err := mock.AssertEvaluated("flag"); err != nil {
		t.Errorf("flag should have been evaluated: %v", err)
	}
}
