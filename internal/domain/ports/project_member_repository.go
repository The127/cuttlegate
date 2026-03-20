package ports

import (
	"context"

	"github.com/karo/cuttlegate/internal/domain"
)

// ProjectMemberRepository is the port for persisting and retrieving project membership entities.
type ProjectMemberRepository interface {
	// AddMember adds a new member to a project. Returns ErrConflict if the user is already a member.
	AddMember(ctx context.Context, member *domain.ProjectMember) error
	// ListMembers returns all members of a project. Returns an empty (non-nil) slice if there are none.
	ListMembers(ctx context.Context, projectID string) ([]*domain.ProjectMember, error)
	// UpdateRole changes the role of an existing project member. Returns ErrNotFound if not a member.
	UpdateRole(ctx context.Context, projectID, userID string, role domain.Role) error
	// RemoveMember removes a member from a project. Returns ErrNotFound if not a member.
	RemoveMember(ctx context.Context, projectID, userID string) error
}
