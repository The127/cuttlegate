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
)

// EvalResult is the outcome of evaluating a flag for a given context.
type EvalResult struct {
	VariantKey      string
	Reason          EvalReason
	MatchedRuleID   string // empty when Reason != ReasonRuleMatch
	MatchedRuleName string // empty when Reason != ReasonRuleMatch
}

// Evaluate returns the variant and reason for a flag given a user context.
//
// state may be nil, which is treated as disabled (flag never toggled in this environment).
// rules are sorted by Priority ascending before evaluation; slice order does not matter.
//
// segmentSlugs is the set of segment slugs the calling user belongs to, pre-loaded by the
// service layer — the evaluator does no IO. nil is treated as an empty set: in_segment
// conditions always miss, not_in_segment conditions always hit (user is not in any segment).
func Evaluate(flag *Flag, state *FlagEnvironmentState, rules []*Rule, ctx EvalContext, segmentSlugs Set) EvalResult {
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
		if matchesAll(rule.Conditions, ctx, segmentSlugs) {
			variantKey := rule.VariantKey
			if len(rule.Rollout) > 0 {
				if ctx.UserID == "" {
					continue // no user key → can't bucket, skip rule
				}
				variantKey = resolveRollout(rule.Rollout, Bucket(flag.Key, ctx.UserID))
			}
			return EvalResult{
				VariantKey:      variantKey,
				Reason:          ReasonRuleMatch,
				MatchedRuleID:   rule.ID,
				MatchedRuleName: rule.Name,
			}
		}
	}

	return EvalResult{VariantKey: flag.DefaultVariantKey, Reason: ReasonDefault}
}

// resolveRollout picks a variant key from a weighted rollout based on a bucket value [0, 99].
// Weights must sum to 100 (enforced by Rule.Validate).
func resolveRollout(entries []RolloutEntry, bucket int) string {
	cumulative := 0
	for _, e := range entries {
		cumulative += e.Weight
		if bucket < cumulative {
			return e.VariantKey
		}
	}
	// Fallback: should not happen if weights sum to 100, but return last entry to be safe.
	return entries[len(entries)-1].VariantKey
}

// matchesAll returns true when every condition in the slice matches ctx.
func matchesAll(conditions []Condition, ctx EvalContext, segmentSlugs Set) bool {
	for _, c := range conditions {
		if !matchesCondition(c, ctx, segmentSlugs) {
			return false
		}
	}
	return true
}

// matchesCondition evaluates a single condition against ctx.
// A missing attribute never matches for attribute-based operators.
// For segment operators, segmentSlugs (nil treated as empty) determines membership.
func matchesCondition(c Condition, ctx EvalContext, segmentSlugs Set) bool {
	switch c.Operator {
	case OperatorInSegment:
		if len(c.Values) == 0 {
			return false
		}
		return segmentSlugs.Contains(c.Values[0])
	case OperatorNotInSegment:
		if len(c.Values) == 0 {
			return false
		}
		return !segmentSlugs.Contains(c.Values[0])
	}

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
