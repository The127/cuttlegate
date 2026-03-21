package domain

import "time"

// Segment is a named group of users used for targeted feature flag evaluation.
// Membership is loaded separately via SegmentRepository to avoid loading all
// members on every segment read.
type Segment struct {
	ID        string
	Slug      string
	Name      string
	ProjectID string
	CreatedAt time.Time
}

// Validate returns an error if the segment is not well-formed.
func (s *Segment) Validate() error {
	if s.Name == "" {
		return &ValidationError{Field: "name", Message: "must not be empty"}
	}
	if !keyRe.MatchString(s.Slug) {
		return &ValidationError{Field: "slug", Message: `must match ^[a-z0-9][a-z0-9-]*$`}
	}
	return nil
}
