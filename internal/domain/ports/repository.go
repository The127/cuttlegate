package ports

import (
	"context"

	"github.com/karo/cuttlegate/internal/domain"
)

// FlagRepository is the port for persisting and retrieving feature flag entities.
type FlagRepository interface {
	Create(ctx context.Context, flag *domain.Flag) error
	GetByKey(ctx context.Context, projectID, key string) (*domain.Flag, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.Flag, error)
	// ListByProjectPaginated returns a page of flags matching the filter criteria,
	// along with the total count of matching flags (for pagination metadata).
	ListByProjectPaginated(ctx context.Context, projectID string, filter domain.FlagListFilter) ([]*domain.Flag, int, error)
	Update(ctx context.Context, flag *domain.Flag) error
	Delete(ctx context.Context, id string) error
}

// ProjectRepository is the port for persisting and retrieving project entities.
type ProjectRepository interface {
	Create(ctx context.Context, project domain.Project) error
	GetBySlug(ctx context.Context, slug string) (*domain.Project, error)
	List(ctx context.Context) ([]*domain.Project, error)
	UpdateName(ctx context.Context, id, name string) error
	Delete(ctx context.Context, id string) error
}

// EnvironmentRepository is the port for persisting and retrieving environment entities.
type EnvironmentRepository interface {
	Create(ctx context.Context, env domain.Environment) error
	GetBySlug(ctx context.Context, projectID, slug string) (*domain.Environment, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.Environment, error)
	UpdateName(ctx context.Context, id, name string) error
	Delete(ctx context.Context, id string) error
}

// RuleRepository is the port for persisting and retrieving targeting rules.
type RuleRepository interface {
	// ListByFlag returns all rules for a flag across all environments.
	ListByFlag(ctx context.Context, flagID string) ([]*domain.Rule, error)
	ListByFlagEnvironment(ctx context.Context, flagID, environmentID string) ([]*domain.Rule, error)
	// ListByEnvironment returns all rules for an environment ordered by flag_id then priority ascending.
	// Used by EvaluateAll to batch-load rules in one query instead of one per flag.
	ListByEnvironment(ctx context.Context, environmentID string) ([]*domain.Rule, error)
	Upsert(ctx context.Context, rule *domain.Rule) error
	Delete(ctx context.Context, id string) error
	// DeleteByFlagEnvironment removes all rules for a given flag+environment pair in one operation.
	DeleteByFlagEnvironment(ctx context.Context, flagID, environmentID string) error
}

// SegmentWithCount pairs a segment with its precomputed member count.
// Used by ListWithCount to avoid a second round-trip per segment.
type SegmentWithCount struct {
	Segment     *domain.Segment
	MemberCount int
}

// SegmentRepository is the port for persisting and retrieving user segments.
type SegmentRepository interface {
	Create(ctx context.Context, segment *domain.Segment) error
	GetBySlug(ctx context.Context, projectID, slug string) (*domain.Segment, error)
	List(ctx context.Context, projectID string) ([]*domain.Segment, error)
	// ListWithCount returns segments for a project with their member counts,
	// computed in a single query via LEFT JOIN + COUNT.
	ListWithCount(ctx context.Context, projectID string) ([]*SegmentWithCount, error)
	UpdateName(ctx context.Context, id, name string) error
	Delete(ctx context.Context, id string) error
	// SetMembers bulk-replaces all members of a segment. An empty slice clears all members.
	SetMembers(ctx context.Context, segmentID string, userKeys []string) error
	ListMembers(ctx context.Context, segmentID string) ([]string, error)
	IsMember(ctx context.Context, segmentID string, userKey string) (bool, error)
}
