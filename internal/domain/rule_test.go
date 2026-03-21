package domain

import (
	"testing"
)

func TestCondition_Validate(t *testing.T) {
	tests := []struct {
		name    string
		c       Condition
		wantErr string
	}{
		{
			name:    "empty attribute",
			c:       Condition{Attribute: "", Operator: OperatorEq, Values: []string{"x"}},
			wantErr: "attribute: must not be empty",
		},
		{
			name:    "unsupported operator",
			c:       Condition{Attribute: "plan", Operator: "unknown", Values: []string{"x"}},
			wantErr: "operator: unsupported operator: unknown",
		},
		// scalar operators
		{
			name:    "eq with zero values",
			c:       Condition{Attribute: "plan", Operator: OperatorEq, Values: []string{}},
			wantErr: "values: operator eq requires exactly one value",
		},
		{
			name:    "eq with two values",
			c:       Condition{Attribute: "plan", Operator: OperatorEq, Values: []string{"a", "b"}},
			wantErr: "values: operator eq requires exactly one value",
		},
		{
			name: "eq with one value — valid",
			c:    Condition{Attribute: "plan", Operator: OperatorEq, Values: []string{"pro"}},
		},
		{
			name: "neq with one value — valid",
			c:    Condition{Attribute: "plan", Operator: OperatorNeq, Values: []string{"free"}},
		},
		{
			name: "contains with one value — valid",
			c:    Condition{Attribute: "email", Operator: OperatorContains, Values: []string{"@example"}},
		},
		{
			name: "starts_with with one value — valid",
			c:    Condition{Attribute: "email", Operator: OperatorStartsWith, Values: []string{"admin"}},
		},
		{
			name: "ends_with with one value — valid",
			c:    Condition{Attribute: "email", Operator: OperatorEndsWith, Values: []string{".com"}},
		},
		// list operators
		{
			name:    "in with zero values",
			c:       Condition{Attribute: "country", Operator: OperatorIn, Values: []string{}},
			wantErr: "values: operator in requires at least one value",
		},
		{
			name: "in with one value — valid",
			c:    Condition{Attribute: "country", Operator: OperatorIn, Values: []string{"DE"}},
		},
		{
			name: "in with multiple values — valid",
			c:    Condition{Attribute: "country", Operator: OperatorIn, Values: []string{"DE", "AT", "CH"}},
		},
		{
			name:    "not_in with zero values",
			c:       Condition{Attribute: "country", Operator: OperatorNotIn, Values: []string{}},
			wantErr: "values: operator not_in requires at least one value",
		},
		{
			name: "not_in with multiple values — valid",
			c:    Condition{Attribute: "country", Operator: OperatorNotIn, Values: []string{"US", "CA"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.c.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestRule_Validate(t *testing.T) {
	validCondition := Condition{Attribute: "plan", Operator: OperatorEq, Values: []string{"pro"}}

	tests := []struct {
		name    string
		r       Rule
		wantErr string
	}{
		{
			name: "valid rule",
			r:    Rule{Conditions: []Condition{validCondition}, VariantKey: "variant-a"},
		},
		{
			name: "priority zero is valid",
			r:    Rule{Priority: 0, Conditions: []Condition{validCondition}, VariantKey: "variant-a"},
		},
		{
			name:    "no conditions",
			r:       Rule{Conditions: []Condition{}, VariantKey: "variant-a"},
			wantErr: "conditions: rule must have at least one condition",
		},
		{
			name:    "empty variant key",
			r:       Rule{Conditions: []Condition{validCondition}, VariantKey: ""},
			wantErr: "variantKey: must not be empty",
		},
		{
			name: "invalid condition propagates error",
			r: Rule{
				Conditions: []Condition{{Attribute: "", Operator: OperatorEq, Values: []string{"x"}}},
				VariantKey: "variant-a",
			},
			wantErr: "attribute: must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.r.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}
