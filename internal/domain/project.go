package domain

import "time"

// Project represents a logical grouping of feature flags and environments.
type Project struct {
	ID        string
	Name      string
	Slug      string
	CreatedAt time.Time
}

// Validate returns a ValidationError if the project's invariants are violated.
func (p Project) Validate() error {
	if p.Name == "" {
		return &ValidationError{Field: "name", Message: "must not be empty"}
	}
	if !slugRe.MatchString(p.Slug) {
		return &ValidationError{Field: "slug", Message: "must match ^[a-z0-9][a-z0-9-]*$"}
	}
	return nil
}
