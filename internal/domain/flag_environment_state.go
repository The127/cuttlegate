package domain

// FlagEnvironmentState records whether a flag is enabled in a specific environment.
type FlagEnvironmentState struct {
	FlagID        string
	EnvironmentID string
	Enabled       bool
}
