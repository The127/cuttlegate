package app

import (
	"context"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// ProjectMemberView is a project member record enriched with cached user profile data.
type ProjectMemberView struct {
	ProjectID string
	UserID    string
	Role      domain.Role
	CreatedAt time.Time
	Name      string
	Email     string
}

// ProjectMemberService orchestrates project membership use cases.
type ProjectMemberService struct {
	repo     ports.ProjectMemberRepository
	project  ports.ProjectRepository
	userRepo ports.UserRepository
}

// NewProjectMemberService constructs a ProjectMemberService.
func NewProjectMemberService(repo ports.ProjectMemberRepository, project ports.ProjectRepository, userRepo ports.UserRepository) *ProjectMemberService {
	return &ProjectMemberService{repo: repo, project: project, userRepo: userRepo}
}

// ListMembers returns all members of a project enriched with cached user profiles.
// Requires at least viewer role.
func (s *ProjectMemberService) ListMembers(ctx context.Context, projectSlug string) ([]ProjectMemberView, error) {
	if _, err := requireRole(ctx, domain.RoleViewer); err != nil {
		return nil, err
	}
	proj, err := s.project.GetBySlug(ctx, projectSlug)
	if err != nil {
		return nil, err
	}
	members, err := s.repo.ListMembers(ctx, proj.ID)
	if err != nil {
		return nil, err
	}
	views := make([]ProjectMemberView, 0, len(members))
	for _, m := range members {
		v := ProjectMemberView{
			ProjectID: m.ProjectID,
			UserID:    m.UserID,
			Role:      m.Role,
			CreatedAt: m.CreatedAt,
		}
		if u, err := s.userRepo.GetByID(ctx, m.UserID); err == nil && u != nil {
			v.Name = u.Name
			v.Email = u.Email
		}
		views = append(views, v)
	}
	return views, nil
}

// AddMember adds a new member to a project. Requires admin role.
// The caller cannot assign a role higher than their own.
func (s *ProjectMemberService) AddMember(ctx context.Context, projectSlug, userID string, role domain.Role) (ProjectMemberView, error) {
	ac, err := requireRole(ctx, domain.RoleAdmin)
	if err != nil {
		return ProjectMemberView{}, err
	}
	if roleLevel[role] > roleLevel[ac.Role] {
		return ProjectMemberView{}, domain.ErrForbidden
	}
	proj, err := s.project.GetBySlug(ctx, projectSlug)
	if err != nil {
		return ProjectMemberView{}, err
	}
	m := &domain.ProjectMember{
		ProjectID: proj.ID,
		UserID:    userID,
		Role:      role,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.AddMember(ctx, m); err != nil {
		return ProjectMemberView{}, err
	}
	v := ProjectMemberView{
		ProjectID: m.ProjectID,
		UserID:    m.UserID,
		Role:      m.Role,
		CreatedAt: m.CreatedAt,
	}
	if u, err := s.userRepo.GetByID(ctx, userID); err == nil && u != nil {
		v.Name = u.Name
		v.Email = u.Email
	}
	return v, nil
}

// UpdateRole changes the role of an existing project member. Requires admin role.
// The caller cannot assign a role higher than their own.
func (s *ProjectMemberService) UpdateRole(ctx context.Context, projectSlug, userID string, role domain.Role) error {
	ac, err := requireRole(ctx, domain.RoleAdmin)
	if err != nil {
		return err
	}
	if roleLevel[role] > roleLevel[ac.Role] {
		return domain.ErrForbidden
	}
	proj, err := s.project.GetBySlug(ctx, projectSlug)
	if err != nil {
		return err
	}
	return s.repo.UpdateRole(ctx, proj.ID, userID, role)
}

// RemoveMember removes a member from a project. Requires admin role.
// Returns ErrLastAdmin if removing the member would leave the project with no admins.
func (s *ProjectMemberService) RemoveMember(ctx context.Context, projectSlug, userID string) error {
	if _, err := requireRole(ctx, domain.RoleAdmin); err != nil {
		return err
	}
	proj, err := s.project.GetBySlug(ctx, projectSlug)
	if err != nil {
		return err
	}
	members, err := s.repo.ListMembers(ctx, proj.ID)
	if err != nil {
		return err
	}
	var found bool
	var targetIsAdmin bool
	adminCount := 0
	for _, m := range members {
		if m.Role == domain.RoleAdmin {
			adminCount++
		}
		if m.UserID == userID {
			found = true
			targetIsAdmin = m.Role == domain.RoleAdmin
		}
	}
	if !found {
		return domain.ErrNotFound
	}
	if targetIsAdmin && adminCount == 1 {
		return domain.ErrLastAdmin
	}
	return s.repo.RemoveMember(ctx, proj.ID, userID)
}
