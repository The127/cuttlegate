package domain

import "time"

// ProjectMember represents a user's membership and role within a project.
type ProjectMember struct {
	ProjectID string
	UserID    string
	Role      Role
	CreatedAt time.Time
}
