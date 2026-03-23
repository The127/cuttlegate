package domain

import "time"

// EvaluationBucket holds time-bucketed evaluation counts for a single time
// slot. Variants maps each variant key to its evaluation count within the
// bucket. An empty bucket (no evaluations in the slot) has Total == 0 and
// Variants == nil or empty map.
type EvaluationBucket struct {
	Timestamp time.Time
	Total     int64
	Variants  map[string]int64
}
