package domain

import "time"

// Environment represents a deployment target (e.g. production, staging) within a project.
type Environment struct {
	ID        string
	ProjectID string
	Name      string
	Slug      string
	CreatedAt time.Time
}

// Validate returns a ValidationError if the environment's invariants are violated.
func (e Environment) Validate() error {
	if e.Name == "" {
		return &ValidationError{Field: "name", Message: "must not be empty"}
	}
	if !slugRe.MatchString(e.Slug) {
		return &ValidationError{Field: "slug", Message: "must match ^[a-z0-9][a-z0-9-]*$"}
	}
	return nil
}
