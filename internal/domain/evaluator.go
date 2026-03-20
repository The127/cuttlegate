package domain

import (
	"sort"
	"strings"
)

// EvalContext holds the user attributes used to evaluate targeting rules.
type EvalContext struct {
	UserID     string
	Attributes map[string]string
}

// EvalReason describes why a particular variant was returned.
type EvalReason string

const (
	ReasonDisabled  EvalReason = "disabled"
	ReasonDefault   EvalReason = "default"
	ReasonRuleMatch EvalReason = "rule_match"
	ReasonRollout   EvalReason = "rollout" // reserved — not used until #43/#44
)

// EvalResult is the outcome of evaluating a flag for a given context.
type EvalResult struct {
	VariantKey string
	Reason     EvalReason
}

// Evaluate returns the variant and reason for a flag given a user context.
//
// state may be nil, which is treated as disabled (flag never toggled in this environment).
// rules are sorted by Priority ascending before evaluation; slice order does not matter.
func Evaluate(flag *Flag, state *FlagEnvironmentState, rules []*Rule, ctx EvalContext) EvalResult {
	if state == nil || !state.Enabled {
		return EvalResult{VariantKey: flag.DefaultVariantKey, Reason: ReasonDisabled}
	}

	sorted := make([]*Rule, len(rules))
	copy(sorted, rules)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	for _, rule := range sorted {
		if !rule.Enabled {
			continue
		}
		if matchesAll(rule.Conditions, ctx) {
			return EvalResult{VariantKey: rule.VariantKey, Reason: ReasonRuleMatch}
		}
	}

	return EvalResult{VariantKey: flag.DefaultVariantKey, Reason: ReasonDefault}
}

// matchesAll returns true when every condition in the slice matches ctx.
func matchesAll(conditions []Condition, ctx EvalContext) bool {
	for _, c := range conditions {
		if !matchesCondition(c, ctx) {
			return false
		}
	}
	return true
}

// matchesCondition evaluates a single condition against ctx.
// A missing attribute never matches.
func matchesCondition(c Condition, ctx EvalContext) bool {
	val, ok := ctx.Attributes[c.Attribute]
	if !ok {
		return false
	}

	switch c.Operator {
	case OperatorEq:
		return val == c.Values[0]
	case OperatorNeq:
		return val != c.Values[0]
	case OperatorContains:
		return strings.Contains(val, c.Values[0])
	case OperatorStartsWith:
		return strings.HasPrefix(val, c.Values[0])
	case OperatorEndsWith:
		return strings.HasSuffix(val, c.Values[0])
	case OperatorIn:
		for _, v := range c.Values {
			if val == v {
				return true
			}
		}
		return false
	case OperatorNotIn:
		for _, v := range c.Values {
			if val == v {
				return false
			}
		}
		return true
	default:
		return false
	}
}
