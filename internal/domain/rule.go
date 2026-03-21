package domain

import "time"

// Operator is a comparison operator used in a targeting rule condition.
type Operator string

const (
	OperatorEq         Operator = "eq"
	OperatorNeq        Operator = "neq"
	OperatorContains   Operator = "contains"
	OperatorStartsWith Operator = "starts_with"
	OperatorEndsWith   Operator = "ends_with"
	OperatorIn         Operator = "in"
	OperatorNotIn      Operator = "not_in"
)

// scalarOperators require exactly one value.
var scalarOperators = map[Operator]bool{
	OperatorEq:         true,
	OperatorNeq:        true,
	OperatorContains:   true,
	OperatorStartsWith: true,
	OperatorEndsWith:   true,
}

// listOperators require one or more values.
var listOperators = map[Operator]bool{
	OperatorIn:    true,
	OperatorNotIn: true,
}

// Condition is a single attribute test within a targeting rule.
// All conditions in a rule are ANDed — all must match for the rule to fire.
type Condition struct {
	Attribute string // user context attribute key, e.g. "plan", "country"
	Operator  Operator
	Values    []string // single-element for scalar operators; one or more for in/not_in
}

// Validate returns an error if the condition is not well-formed.
func (c Condition) Validate() error {
	if c.Attribute == "" {
		return &ValidationError{Field: "attribute", Message: "must not be empty"}
	}
	if scalarOperators[c.Operator] {
		if len(c.Values) != 1 {
			return &ValidationError{Field: "values", Message: "operator " + string(c.Operator) + " requires exactly one value"}
		}
		return nil
	}
	if listOperators[c.Operator] {
		if len(c.Values) == 0 {
			return &ValidationError{Field: "values", Message: "operator " + string(c.Operator) + " requires at least one value"}
		}
		return nil
	}
	return &ValidationError{Field: "operator", Message: "unsupported operator: " + string(c.Operator)}
}

// Rule is a targeting rule attached to a flag in a specific environment.
// Rules are evaluated in ascending Priority order; the first matching rule wins.
type Rule struct {
	ID            string
	FlagID        string
	EnvironmentID string
	Priority      int // lower = evaluated first; 0 is valid
	Conditions    []Condition
	VariantKey    string // variant to return when all conditions match
	Enabled       bool
	CreatedAt     time.Time
}

// Validate returns an error if the rule is not well-formed.
func (r Rule) Validate() error {
	if len(r.Conditions) == 0 {
		return &ValidationError{Field: "conditions", Message: "rule must have at least one condition"}
	}
	if r.VariantKey == "" {
		return &ValidationError{Field: "variantKey", Message: "must not be empty"}
	}
	for _, c := range r.Conditions {
		if err := c.Validate(); err != nil {
			return err
		}
	}
	return nil
}
