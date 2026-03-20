package domain

import "time"

// Project represents a logical grouping of feature flags and environments.
type Project struct {
	ID        string
	Name      string
	Slug      string
	CreatedAt time.Time
}
