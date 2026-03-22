package cuttlegate

// EvalContext is the user context sent with every evaluation request.
//
// Attributes values must be JSON-serialisable. Non-serialisable values
// (e.g. channels, functions) will cause the evaluation request to fail.
type EvalContext struct {
	UserID     string         `json:"user_id"`
	Attributes map[string]any `json:"attributes"`
}

// EvalResult is the result of evaluating a single flag.
type EvalResult struct {
	Key         string `json:"key"`
	Enabled     bool   `json:"enabled"`
	Value       string `json:"value"` // empty string for bool flags
	Reason      string `json:"reason"`
	EvaluatedAt string `json:"evaluated_at"`
}

// FlagResult is the result of evaluating a single flag by key.
type FlagResult struct {
	Enabled bool
	Value   string // empty string for bool flags
	Reason  string
}
