package cuttlegate

import "time"

// EvalContext is the user context sent with every evaluation request.
//
// Attributes values must be JSON-serialisable. Non-serialisable values
// (e.g. channels, functions) will cause the evaluation request to fail.
type EvalContext struct {
	UserID     string         `json:"user_id"`
	Attributes map[string]any `json:"attributes"`
}

// EvalResult is the result of evaluating a single flag.
//
// Variant is the variant key; for bool flags it is "true" or "false".
// Value is the string representation of the variant value; for bool flags
// it is empty — use Variant or Bool() instead.
// Reason describes why this result was produced: "targeting_rule", "default",
// "disabled", or "percentage_rollout".
type EvalResult struct {
	Key         string `json:"key"`
	Enabled     bool   `json:"enabled"`
	Value       string `json:"value"`   // string value; empty for bool flags
	Variant     string `json:"variant"` // variant key; "true"/"false" for bool flags
	Reason      string `json:"reason"`
	EvaluatedAt string `json:"evaluated_at"`
}

// FlagResult is the result of evaluating a single flag by key via EvaluateFlag.
// Prefer Evaluate, Bool, or String for new code — they return structured errors
// rather than encoding not-found as a Reason string.
type FlagResult struct {
	Enabled bool
	Value   string // string value; empty for bool flags
	Variant string // variant key; "true"/"false" for bool flags
	Reason  string
}

// FlagUpdate is a real-time flag state change received from the SSE stream.
// It is delivered on the updates channel returned by Client.Subscribe.
type FlagUpdate struct {
	Key       string    // flag key that changed
	Enabled   bool      // new enabled state
	UpdatedAt time.Time // when the change occurred (UTC)
}
