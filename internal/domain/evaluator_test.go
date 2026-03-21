package domain

import (
	"testing"
	"time"
)

func testFlag() *Flag {
	return &Flag{
		ID:                "flag-1",
		ProjectID:         "proj-1",
		Key:               "my-flag",
		Type:              FlagTypeString,
		DefaultVariantKey: "default",
		Variants:          []Variant{{Key: "default"}, {Key: "variant-a"}, {Key: "variant-b"}},
		CreatedAt:         time.Now(),
	}
}

func enabledState() *FlagEnvironmentState {
	return &FlagEnvironmentState{FlagID: "flag-1", EnvironmentID: "env-1", Enabled: true}
}

func disabledState() *FlagEnvironmentState {
	return &FlagEnvironmentState{FlagID: "flag-1", EnvironmentID: "env-1", Enabled: false}
}

func rule(priority int, variantKey string, conditions ...Condition) *Rule {
	return &Rule{
		ID:            "rule-" + variantKey,
		FlagID:        "flag-1",
		EnvironmentID: "env-1",
		Priority:      priority,
		Conditions:    conditions,
		VariantKey:    variantKey,
		Enabled:       true,
	}
}

func cond(attr string, op Operator, values ...string) Condition {
	return Condition{Attribute: attr, Operator: op, Values: values}
}

func TestEvaluate_DisabledFlag(t *testing.T) {
	flag := testFlag()

	t.Run("state disabled", func(t *testing.T) {
		result := Evaluate(flag, disabledState(), nil, EvalContext{}, nil)
		if result.Reason != ReasonDisabled || result.VariantKey != flag.DefaultVariantKey {
			t.Fatalf("got %+v", result)
		}
	})

	t.Run("state nil", func(t *testing.T) {
		result := Evaluate(flag, nil, nil, EvalContext{}, nil)
		if result.Reason != ReasonDisabled || result.VariantKey != flag.DefaultVariantKey {
			t.Fatalf("got %+v", result)
		}
	})
}

func TestEvaluate_NoRules(t *testing.T) {
	result := Evaluate(testFlag(), enabledState(), nil, EvalContext{}, nil)
	if result.Reason != ReasonDefault || result.VariantKey != "default" {
		t.Fatalf("got %+v", result)
	}
}

func TestEvaluate_RuleMatch(t *testing.T) {
	flag := testFlag()
	r := rule(0, "variant-a", cond("plan", OperatorEq, "pro"))
	ctx := EvalContext{Attributes: map[string]string{"plan": "pro"}}

	result := Evaluate(flag, enabledState(), []*Rule{r}, ctx, nil)
	if result.Reason != ReasonRuleMatch || result.VariantKey != "variant-a" {
		t.Fatalf("got %+v", result)
	}
}

func TestEvaluate_FirstMatchWins(t *testing.T) {
	flag := testFlag()
	r1 := rule(0, "variant-a", cond("plan", OperatorEq, "free")) // does not match
	r2 := rule(1, "variant-b", cond("plan", OperatorEq, "pro"))  // matches
	ctx := EvalContext{Attributes: map[string]string{"plan": "pro"}}

	result := Evaluate(flag, enabledState(), []*Rule{r1, r2}, ctx, nil)
	if result.VariantKey != "variant-b" || result.Reason != ReasonRuleMatch {
		t.Fatalf("got %+v", result)
	}
}

func TestEvaluate_PriorityOrdering(t *testing.T) {
	flag := testFlag()
	// priority 1 matches, priority 10 also matches — priority 1 must win regardless of slice order
	r10 := rule(10, "variant-b", cond("plan", OperatorEq, "pro"))
	r1 := rule(1, "variant-a", cond("plan", OperatorEq, "pro"))
	ctx := EvalContext{Attributes: map[string]string{"plan": "pro"}}

	result := Evaluate(flag, enabledState(), []*Rule{r10, r1}, ctx, nil) // r10 first in slice
	if result.VariantKey != "variant-a" {
		t.Fatalf("expected variant-a (priority 1), got %+v", result)
	}
}

func TestEvaluate_DisabledRuleSkipped(t *testing.T) {
	flag := testFlag()
	r := rule(0, "variant-a", cond("plan", OperatorEq, "pro"))
	r.Enabled = false
	ctx := EvalContext{Attributes: map[string]string{"plan": "pro"}}

	result := Evaluate(flag, enabledState(), []*Rule{r}, ctx, nil)
	if result.Reason != ReasonDefault {
		t.Fatalf("disabled rule should be skipped, got %+v", result)
	}
}

func TestEvaluate_MissingAttributeNoMatch(t *testing.T) {
	flag := testFlag()
	r := rule(0, "variant-a", cond("plan", OperatorIn, "pro", "enterprise"))
	ctx := EvalContext{Attributes: map[string]string{}} // "plan" absent

	result := Evaluate(flag, enabledState(), []*Rule{r}, ctx, nil)
	if result.Reason != ReasonDefault {
		t.Fatalf("missing attribute should not match, got %+v", result)
	}
}

func TestEvaluate_Operators(t *testing.T) {
	flag := testFlag()
	state := enabledState()

	tests := []struct {
		name      string
		condition Condition
		attrs     map[string]string
		wantMatch bool
	}{
		{"eq match", cond("plan", OperatorEq, "pro"), map[string]string{"plan": "pro"}, true},
		{"eq no match", cond("plan", OperatorEq, "pro"), map[string]string{"plan": "free"}, false},
		{"neq match", cond("plan", OperatorNeq, "free"), map[string]string{"plan": "pro"}, true},
		{"neq no match", cond("plan", OperatorNeq, "pro"), map[string]string{"plan": "pro"}, false},
		{"contains match", cond("email", OperatorContains, "@acme"), map[string]string{"email": "user@acme.com"}, true},
		{"contains no match", cond("email", OperatorContains, "@acme"), map[string]string{"email": "user@other.com"}, false},
		{"starts_with match", cond("email", OperatorStartsWith, "admin"), map[string]string{"email": "admin@acme.com"}, true},
		{"starts_with no match", cond("email", OperatorStartsWith, "admin"), map[string]string{"email": "user@acme.com"}, false},
		{"ends_with match", cond("email", OperatorEndsWith, ".de"), map[string]string{"email": "user@acme.de"}, true},
		{"ends_with no match", cond("email", OperatorEndsWith, ".de"), map[string]string{"email": "user@acme.com"}, false},
		{"in match", cond("country", OperatorIn, "DE", "AT", "CH"), map[string]string{"country": "AT"}, true},
		{"in no match", cond("country", OperatorIn, "DE", "AT", "CH"), map[string]string{"country": "US"}, false},
		{"not_in match", cond("country", OperatorNotIn, "US", "CA"), map[string]string{"country": "DE"}, true},
		{"not_in no match", cond("country", OperatorNotIn, "US", "CA"), map[string]string{"country": "US"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := rule(0, "variant-a", tt.condition)
			ctx := EvalContext{Attributes: tt.attrs}
			result := Evaluate(flag, state, []*Rule{r}, ctx, nil)
			matched := result.Reason == ReasonRuleMatch
			if matched != tt.wantMatch {
				t.Fatalf("wantMatch=%v got reason=%s", tt.wantMatch, result.Reason)
			}
		})
	}
}

func TestEvaluate_SegmentOperators(t *testing.T) {
	flag := testFlag()
	state := enabledState()
	ctx := EvalContext{UserID: "user-1", Attributes: map[string]string{}}

	inBeta := map[string]struct{}{"beta": {}}

	tests := []struct {
		name         string
		condition    Condition
		segmentSlugs map[string]struct{}
		wantMatch    bool
	}{
		{"in_segment: user is member", cond("", OperatorInSegment, "beta"), inBeta, true},
		{"in_segment: user not member", cond("", OperatorInSegment, "beta"), nil, false},
		{"in_segment: wrong segment", cond("", OperatorInSegment, "alpha"), inBeta, false},
		{"not_in_segment: user not member", cond("", OperatorNotInSegment, "beta"), nil, true},
		{"not_in_segment: user is member", cond("", OperatorNotInSegment, "beta"), inBeta, false},
		{"not_in_segment: wrong segment (not member)", cond("", OperatorNotInSegment, "alpha"), inBeta, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := rule(0, "variant-a", tt.condition)
			result := Evaluate(flag, state, []*Rule{r}, ctx, tt.segmentSlugs)
			matched := result.Reason == ReasonRuleMatch
			if matched != tt.wantMatch {
				t.Fatalf("wantMatch=%v got reason=%s", tt.wantMatch, result.Reason)
			}
		})
	}
}
