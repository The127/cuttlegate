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
