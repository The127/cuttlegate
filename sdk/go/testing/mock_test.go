package cuttlegatetesting_test

import (
	"context"
	"testing"

	cuttlegate "github.com/The127/cuttlegate/sdk/go"
	cuttlegatetesting "github.com/The127/cuttlegate/sdk/go/testing"
)

var ctx = context.Background()
var evalCtx = cuttlegate.EvalContext{UserID: "u1", Attributes: map[string]any{}}

func TestMockClient_Enable(t *testing.T) {
	// @happy
	mock := cuttlegatetesting.NewMockClient()
	mock.Enable("dark-mode")

	result, err := mock.EvaluateFlag(ctx, "dark-mode", evalCtx)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Enabled {
		t.Error("expected Enabled=true after Enable")
	}
	if result.Reason != "mock" {
		t.Errorf("expected Reason=mock, got %q", result.Reason)
	}
}

func TestMockClient_Disable(t *testing.T) {
	// @happy
	mock := cuttlegatetesting.NewMockClient()
	mock.Enable("dark-mode")
	mock.Disable("dark-mode")

	result, err := mock.EvaluateFlag(ctx, "dark-mode", evalCtx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Enabled {
		t.Error("expected Enabled=false after Disable")
	}
	if result.Reason != "mock" {
		t.Errorf("expected Reason=mock, got %q", result.Reason)
	}
}

func TestMockClient_SetVariant(t *testing.T) {
	// @happy
	mock := cuttlegatetesting.NewMockClient()
	mock.SetVariant("banner-text", "holiday")

	result, err := mock.EvaluateFlag(ctx, "banner-text", evalCtx)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Enabled {
		t.Error("expected Enabled=true after SetVariant")
	}
	if result.Value != "holiday" {
		t.Errorf("expected Value=holiday, got %q", result.Value)
	}
}

func TestMockClient_UnknownFlag_ReturnsDefault(t *testing.T) {
	// @edge
	mock := cuttlegatetesting.NewMockClient()

	result, err := mock.EvaluateFlag(ctx, "unknown", evalCtx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Enabled {
		t.Error("expected Enabled=false for unknown flag")
	}
	if result.Reason != "mock_default" {
		t.Errorf("expected Reason=mock_default, got %q", result.Reason)
	}
}

func TestMockClient_AssertEvaluated_Pass(t *testing.T) {
	// @happy
	mock := cuttlegatetesting.NewMockClient()
	mock.Enable("dark-mode")
	mock.EvaluateFlag(ctx, "dark-mode", evalCtx)

	if err := mock.AssertEvaluated("dark-mode"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMockClient_AssertEvaluated_Fail(t *testing.T) {
	// @error-path
	mock := cuttlegatetesting.NewMockClient()

	if err := mock.AssertEvaluated("dark-mode"); err == nil {
		t.Error("expected error when flag was not evaluated")
	}
}

func TestMockClient_AssertNotEvaluated_Pass(t *testing.T) {
	// @happy
	mock := cuttlegatetesting.NewMockClient()

	if err := mock.AssertNotEvaluated("dark-mode"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMockClient_AssertNotEvaluated_Fail(t *testing.T) {
	// @error-path
	mock := cuttlegatetesting.NewMockClient()
	mock.EvaluateFlag(ctx, "dark-mode", evalCtx)

	if err := mock.AssertNotEvaluated("dark-mode"); err == nil {
		t.Error("expected error when flag was evaluated")
	}
}

func TestMockClient_Reset(t *testing.T) {
	// @happy
	mock := cuttlegatetesting.NewMockClient()
	mock.Enable("dark-mode")
	mock.EvaluateFlag(ctx, "dark-mode", evalCtx)
	mock.Reset()

	if err := mock.AssertEvaluated("dark-mode"); err == nil {
		t.Error("expected AssertEvaluated to fail after Reset")
	}
	result, _ := mock.EvaluateFlag(ctx, "dark-mode", evalCtx)
	if result.Enabled {
		t.Error("expected flag disabled after Reset")
	}
}

func TestMockClient_EvaluateAll_ReturnsBulk(t *testing.T) {
	// @happy
	mock := cuttlegatetesting.NewMockClient()
	mock.Enable("dark-mode")
	mock.SetVariant("banner-text", "holiday")

	results, err := mock.EvaluateAll(ctx, evalCtx)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestMockClient_Evaluate_ReturnsMockDefault(t *testing.T) {
	// @edge — unknown flags return mock_default, not an error
	mock := cuttlegatetesting.NewMockClient()

	result, err := mock.Evaluate(ctx, "unknown-flag", evalCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Enabled {
		t.Error("expected Enabled=false for unknown flag")
	}
	if result.Reason != "mock_default" {
		t.Errorf("expected Reason=mock_default, got %q", result.Reason)
	}
}

func TestMockClient_Bool_ReturnsTrue(t *testing.T) {
	// @happy
	mock := cuttlegatetesting.NewMockClient()
	mock.Enable("dark-mode") // Enable sets Variant="true"

	val, err := mock.Bool(ctx, "dark-mode", evalCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !val {
		t.Error("expected true for enabled bool flag")
	}
}

func TestMockClient_String_ReturnsValue(t *testing.T) {
	// @happy
	mock := cuttlegatetesting.NewMockClient()
	mock.SetVariant("banner-text", "holiday")

	val, err := mock.String(ctx, "banner-text", evalCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "holiday" {
		t.Errorf("expected holiday, got %q", val)
	}
}

func TestMockClient_ImplementsClientInterface(t *testing.T) {
	// @happy — compile-time check
	var _ cuttlegate.Client = cuttlegatetesting.NewMockClient()
}
