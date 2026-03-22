package domain

import "time"

// FlagEvaluationStats holds aggregated evaluation statistics for a single
// flag in a specific environment.
type FlagEvaluationStats struct {
	FlagID          string
	EnvironmentID   string
	FlagKey         string
	EvaluationCount int64
	LastEvaluatedAt time.Time
}
