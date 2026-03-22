package domain

import "time"

// EvaluationEvent records a single flag evaluation. Instances are
// append-only — they are never modified after creation.
type EvaluationEvent struct {
	ID              string
	FlagKey         string
	ProjectID       string
	EnvironmentID   string
	InputContext    string    // JSON-encoded EvalContext.Attributes (serialised by app layer)
	UserID          string    // EvalContext.UserID
	MatchedRuleID   string    // empty if no rule matched
	MatchedRuleName string    // empty if no rule matched
	VariantKey      string
	Reason          EvalReason
	OccurredAt      time.Time
}
