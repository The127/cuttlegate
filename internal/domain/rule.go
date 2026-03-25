package domain

import "time"

// Operator is a comparison operator used in a targeting rule condition.
type Operator string

const (
	OperatorEq           Operator = "eq"
	OperatorNeq          Operator = "neq"
	OperatorContains     Operator = "contains"
	OperatorStartsWith   Operator = "starts_with"
	OperatorEndsWith     Operator = "ends_with"
	OperatorIn           Operator = "in"
	OperatorNotIn        Operator = "not_in"
	OperatorInSegment    Operator = "in_segment"
	OperatorNotInSegment Operator = "not_in_segment"
)

// scalarOperators require exactly one value.
var scalarOperators = map[Operator]bool{
	OperatorEq:           true,
	OperatorNeq:          true,
	OperatorContains:     true,
	OperatorStartsWith:   true,
	OperatorEndsWith:     true,
	OperatorInSegment:    true, // Values[0] is the segment slug
	OperatorNotInSegment: true, // Values[0] is the segment slug
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
	// Segment operators reference a slug in Values[0]; Attribute is unused.
	isSegmentOp := c.Operator == OperatorInSegment || c.Operator == OperatorNotInSegment
	if !isSegmentOp && c.Attribute == "" {
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

// RolloutEntry assigns a weight to a variant for percentage-based rollouts.
// All weights in a rule's Rollout slice must sum to 100.
type RolloutEntry struct {
	VariantKey string
	Weight     int // 1–100; all entries must sum to 100
}

// Rule is a targeting rule attached to a flag in a specific environment.
// Rules are evaluated in ascending Priority order; the first matching rule wins.
type Rule struct {
	ID            string
	FlagID        string
	EnvironmentID string
	Name          string // human-readable label; may be empty; max 255 characters
	Priority      int    // lower = evaluated first; 0 is valid
	Conditions    []Condition
	VariantKey    string         // variant to return when all conditions match (used when Rollout is nil)
	Rollout       []RolloutEntry // optional weighted variant distribution; when set, replaces VariantKey
	Enabled       bool
	CreatedAt     time.Time
}

// Validate returns an error if the rule is not well-formed.
func (r Rule) Validate() error {
	if len(r.Name) > 255 {
		return &ValidationError{Field: "name", Message: "must not exceed 255 characters"}
	}
	if len(r.Conditions) == 0 {
		return &ValidationError{Field: "conditions", Message: "rule must have at least one condition"}
	}
	if len(r.Rollout) > 0 {
		total := 0
		for _, entry := range r.Rollout {
			if entry.VariantKey == "" {
				return &ValidationError{Field: "rollout", Message: "each rollout entry must have a non-empty variant key"}
			}
			if entry.Weight < 1 {
				return &ValidationError{Field: "rollout", Message: "each rollout entry weight must be at least 1"}
			}
			total += entry.Weight
		}
		if total != 100 {
			return &ValidationError{Field: "rollout", Message: "rollout weights must sum to 100"}
		}
	} else if r.VariantKey == "" {
		return &ValidationError{Field: "variantKey", Message: "must not be empty"}
	}
	for _, c := range r.Conditions {
		if err := c.Validate(); err != nil {
			return err
		}
	}
	return nil
}
