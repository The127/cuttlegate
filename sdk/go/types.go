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
// Variant is the primary field for the variant key. For bool flags it is
// "true" or "false"; for all other flag types it is the variant key string.
// Reason describes why this result was produced: "targeting_rule", "default",
// "disabled", or "percentage_rollout".
type EvalResult struct {
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`

	// Deprecated: Value is empty for bool flags. Use Variant for the raw
	// variant key, Bool() for boolean evaluation, or String() for string flags.
	Value string `json:"value"`

	// Variant is the variant key. For bool flags: "true" or "false".
	Variant     string `json:"variant"`
	Reason      string `json:"reason"`
	EvaluatedAt string `json:"evaluated_at"`
}

// Deprecated: FlagResult is the result of evaluating a single flag by key via EvaluateFlag.
// Use Evaluate, Bool, or String for new code — they return structured errors
// rather than encoding not-found as a Reason string.
type FlagResult struct {
	Enabled bool

	// Deprecated: Value is empty for bool flags. Use Variant for the raw
	// variant key, or switch to client.Bool() / client.String() / client.Evaluate()
	// in place of EvaluateFlag.
	Value string

	// Variant is the variant key. For bool flags: "true" or "false".
	Variant string
	Reason  string
}

// FlagDefault specifies fallback state for a flag when the server is unreachable.
type FlagDefault struct {
	Enabled bool
	Variant string
}

// FlagUpdate is a real-time flag state change received from the SSE stream.
// It is delivered on the updates channel returned by Client.Subscribe.
type FlagUpdate struct {
	Key       string    // flag key that changed
	Enabled   bool      // new enabled state
	UpdatedAt time.Time // when the change occurred (UTC)
}
