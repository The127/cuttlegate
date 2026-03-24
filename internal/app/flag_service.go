package app

import (
	"context"
	"log"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// FlagEnvironmentView combines a flag with its enabled state for a specific environment.
type FlagEnvironmentView struct {
	Flag    *domain.Flag
	Enabled bool
}

// FlagService orchestrates flag use cases.
type FlagService struct {
	repo      ports.FlagRepository
	envRepo   ports.EnvironmentRepository
	stateRepo ports.FlagEnvironmentStateRepository
	publisher ports.EventPublisher
	auditRepo ports.AuditRepository
}

// NewFlagService constructs a FlagService.
func NewFlagService(repo ports.FlagRepository, envRepo ports.EnvironmentRepository, stateRepo ports.FlagEnvironmentStateRepository, publisher ports.EventPublisher, auditRepo ports.AuditRepository) *FlagService {
	return &FlagService{repo: repo, envRepo: envRepo, stateRepo: stateRepo, publisher: publisher, auditRepo: auditRepo}
}

type auditEntry struct {
	Action          string
	EntityID        string
	EntityKey       string
	ProjectID       string
	EnvironmentSlug string
	Before          string
	After           string
	Source          string
}

// recordAudit persists an audit event on a best-effort basis.
// Failures are logged but never block the calling mutation.
func (s *FlagService) recordAudit(ctx context.Context, entry auditEntry) {
	if err := s.recordAuditErr(ctx, entry); err != nil {
		log.Printf("audit: failed to record %s for %s/%s: %v", entry.Action, entry.ProjectID, entry.EntityKey, err)
	}
}

// recordAuditErr persists an audit event and returns any error.
// Callers that need fail-hard audit semantics (e.g. MCP write tools) use this directly.
func (s *FlagService) recordAuditErr(ctx context.Context, entry auditEntry) error {
	ac, _ := domain.AuthContextFrom(ctx)
	id, err := newUUID()
	if err != nil {
		return err
	}
	event := &domain.AuditEvent{
		ID:              id,
		ProjectID:       entry.ProjectID,
		ActorID:         ac.UserID,
		Action:          entry.Action,
		EntityType:      "flag",
		EntityID:        entry.EntityID,
		EntityKey:       entry.EntityKey,
		EnvironmentSlug: entry.EnvironmentSlug,
		BeforeState:     entry.Before,
		AfterState:      entry.After,
		Source:          entry.Source,
		OccurredAt:      time.Now().UTC(),
	}
	return s.auditRepo.Record(ctx, event)
}

// Create validates the flag, assigns a UUID and creation timestamp, persists it, then
// auto-creates a disabled FlagEnvironmentState row for every existing environment in the project.
// If state row creation fails the flag insert is compensated (deleted).
// Requires at least editor role.
func (s *FlagService) Create(ctx context.Context, flag *domain.Flag) error {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return err
	}
	if err := flag.Validate(); err != nil {
		return err
	}
	id, err := newUUID()
	if err != nil {
		return err
	}
	flag.ID = id
	flag.CreatedAt = time.Now().UTC()
	if err := s.repo.Create(ctx, flag); err != nil {
		return err
	}
	envs, err := s.envRepo.ListByProject(ctx, flag.ProjectID)
	if err != nil {
		_ = s.repo.Delete(ctx, flag.ID)
		return err
	}
	if len(envs) == 0 {
		s.recordAudit(ctx, auditEntry{Action: "flag.created", EntityID: flag.ID, EntityKey: flag.Key, ProjectID: flag.ProjectID, After: flag.Key})
		return nil
	}
	states := make([]*domain.FlagEnvironmentState, len(envs))
	for i, e := range envs {
		states[i] = &domain.FlagEnvironmentState{
			FlagID:        flag.ID,
			EnvironmentID: e.ID,
			Enabled:       false,
		}
	}
	if err := s.stateRepo.CreateBatch(ctx, states); err != nil {
		_ = s.repo.Delete(ctx, flag.ID)
		return err
	}
	s.recordAudit(ctx, auditEntry{Action: "flag.created", EntityID: flag.ID, EntityKey: flag.Key, ProjectID: flag.ProjectID, After: flag.Key})
	return nil
}

// GetByKey retrieves a flag by project ID and key.
func (s *FlagService) GetByKey(ctx context.Context, projectID, key string) (*domain.Flag, error) {
	return s.repo.GetByKey(ctx, projectID, key)
}

// ListByProject returns all flags for a project.
func (s *FlagService) ListByProject(ctx context.Context, projectID string) ([]*domain.Flag, error) {
	return s.repo.ListByProject(ctx, projectID)
}

// ListByProjectPaginated returns a paginated, filtered, sorted list of flags for a project.
func (s *FlagService) ListByProjectPaginated(ctx context.Context, projectID string, filter domain.FlagListFilter) ([]*domain.Flag, int, error) {
	return s.repo.ListByProjectPaginated(ctx, projectID, filter)
}

// ListByEnvironment returns all flags that have a state row for the given environment,
// each combined with its enabled state. Flags created before the environment existed
// have no state row and are absent from the result.
func (s *FlagService) ListByEnvironment(ctx context.Context, projectID, environmentID string) ([]*FlagEnvironmentView, error) {
	states, err := s.stateRepo.ListByEnvironment(ctx, environmentID)
	if err != nil {
		return nil, err
	}
	if len(states) == 0 {
		return []*FlagEnvironmentView{}, nil
	}
	enabledByFlagID := make(map[string]bool, len(states))
	for _, st := range states {
		enabledByFlagID[st.FlagID] = st.Enabled
	}
	flags, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]*FlagEnvironmentView, 0, len(states))
	for _, f := range flags {
		if enabled, ok := enabledByFlagID[f.ID]; ok {
			cp := *f
			result = append(result, &FlagEnvironmentView{Flag: &cp, Enabled: enabled})
		}
	}
	return result, nil
}

// AddVariant appends a new variant to an existing flag.
// Returns ErrImmutableVariants for bool flags, ErrConflict for duplicate variant keys.
// Requires at least editor role.
func (s *FlagService) AddVariant(ctx context.Context, projectID, flagKey string, v domain.Variant) (*domain.Flag, error) {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return nil, err
	}
	f, err := s.repo.GetByKey(ctx, projectID, flagKey)
	if err != nil {
		return nil, err
	}
	if f.Type == domain.FlagTypeBool {
		return nil, domain.ErrImmutableVariants
	}
	for _, existing := range f.Variants {
		if existing.Key == v.Key {
			return nil, domain.ErrConflict
		}
	}
	f.Variants = append(f.Variants, v)
	if err := f.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, f); err != nil {
		return nil, err
	}
	s.recordAudit(ctx, auditEntry{Action: "flag.variant_added", EntityID: f.ID, EntityKey: f.Key, ProjectID: f.ProjectID, After: v.Key})
	return f, nil
}

// RenameVariant updates the name of an existing variant. The variant key is immutable.
// Returns ErrNotFound if the variant key does not exist.
// Requires at least editor role.
func (s *FlagService) RenameVariant(ctx context.Context, projectID, flagKey, variantKey, newName string) (*domain.Flag, error) {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return nil, err
	}
	f, err := s.repo.GetByKey(ctx, projectID, flagKey)
	if err != nil {
		return nil, err
	}
	found := false
	for i, v := range f.Variants {
		if v.Key == variantKey {
			f.Variants[i].Name = newName
			found = true
			break
		}
	}
	if !found {
		return nil, domain.ErrNotFound
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, f); err != nil {
		return nil, err
	}
	s.recordAudit(ctx, auditEntry{Action: "flag.variant_renamed", EntityID: f.ID, EntityKey: f.Key, ProjectID: f.ProjectID, Before: variantKey, After: newName})
	return f, nil
}

// DeleteVariant removes a variant from a flag.
// Returns ErrImmutableVariants for bool flags, ErrDefaultVariant if the variant is the default,
// ErrLastVariant if it is the only remaining variant, ErrNotFound if the key does not exist.
// Requires at least editor role.
func (s *FlagService) DeleteVariant(ctx context.Context, projectID, flagKey, variantKey string) (*domain.Flag, error) {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return nil, err
	}
	f, err := s.repo.GetByKey(ctx, projectID, flagKey)
	if err != nil {
		return nil, err
	}
	if f.Type == domain.FlagTypeBool {
		return nil, domain.ErrImmutableVariants
	}
	if variantKey == f.DefaultVariantKey {
		return nil, domain.ErrDefaultVariant
	}
	if len(f.Variants) == 1 {
		return nil, domain.ErrLastVariant
	}
	idx := -1
	for i, v := range f.Variants {
		if v.Key == variantKey {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, domain.ErrNotFound
	}
	f.Variants = append(f.Variants[:idx], f.Variants[idx+1:]...)
	if err := f.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, f); err != nil {
		return nil, err
	}
	s.recordAudit(ctx, auditEntry{Action: "flag.variant_deleted", EntityID: f.ID, EntityKey: f.Key, ProjectID: f.ProjectID, Before: variantKey})
	return f, nil
}

// Update validates and persists updated flag fields. Only Name, Variants, and DefaultVariantKey are mutable.
// Requires at least editor role.
func (s *FlagService) Update(ctx context.Context, flag *domain.Flag) error {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return err
	}
	if err := flag.Validate(); err != nil {
		return err
	}
	if err := s.repo.Update(ctx, flag); err != nil {
		return err
	}
	s.recordAudit(ctx, auditEntry{Action: "flag.updated", EntityID: flag.ID, EntityKey: flag.Key, ProjectID: flag.ProjectID})
	return nil
}

// GetByKeyAndEnvironment returns a flag combined with its enabled state for a specific environment.
// If no state row exists the flag is returned with Enabled: false.
func (s *FlagService) GetByKeyAndEnvironment(ctx context.Context, projectID, environmentID, flagKey string) (*FlagEnvironmentView, error) {
	f, err := s.repo.GetByKey(ctx, projectID, flagKey)
	if err != nil {
		return nil, err
	}
	state, err := s.stateRepo.GetByFlagAndEnvironment(ctx, f.ID, environmentID)
	if err != nil {
		return nil, err
	}
	enabled := false
	if state != nil {
		enabled = state.Enabled
	}
	return &FlagEnvironmentView{Flag: f, Enabled: enabled}, nil
}

// SetEnabledParams holds the parameters for SetEnabled.
type SetEnabledParams struct {
	ProjectID     string
	EnvironmentID string
	FlagKey       string
	Enabled       bool
	ProjectSlug   string
	EnvSlug       string
	// Source identifies the origin of the mutation. Empty for HTTP-originated calls (best-effort audit).
	// Set to "mcp" for MCP-originated calls — audit failure is then treated as a hard error.
	Source string
}

// SetEnabled enables or disables a flag in a specific environment.
// Returns ErrNotFound if no state row exists (flag created before this environment).
// On success, publishes a FlagStateChangedEvent. Publish failure is logged, not returned.
// Requires at least editor role.
func (s *FlagService) SetEnabled(ctx context.Context, params SetEnabledParams) error {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return err
	}
	f, err := s.repo.GetByKey(ctx, params.ProjectID, params.FlagKey)
	if err != nil {
		return err
	}
	if err := s.stateRepo.SetEnabled(ctx, f.ID, params.EnvironmentID, params.Enabled); err != nil {
		return err
	}
	event := domain.NewFlagStateChangedEvent(params.ProjectSlug, params.EnvSlug, params.FlagKey, params.Enabled)
	if err := s.publisher.Publish(ctx, event); err != nil {
		log.Printf("failed to publish FlagStateChangedEvent for %s/%s/%s: %v", params.ProjectSlug, params.EnvSlug, params.FlagKey, err)
	}
	before := "disabled"
	after := "enabled"
	if !params.Enabled {
		before = "enabled"
		after = "disabled"
	}
	entry := auditEntry{Action: "flag.state_changed", EntityID: f.ID, EntityKey: params.FlagKey, ProjectID: params.ProjectID, EnvironmentSlug: params.EnvSlug, Before: before, After: after, Source: params.Source}
	if params.Source != "" {
		// MCP path: audit failure is a hard error — do not silently succeed the mutation.
		return s.recordAuditErr(ctx, entry)
	}
	s.recordAudit(ctx, entry)
	return nil
}

// Delete removes a flag by ID.
// Requires at least editor role.
func (s *FlagService) Delete(ctx context.Context, id string) error {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.recordAudit(ctx, auditEntry{Action: "flag.deleted", EntityID: id})
	return nil
}

// DeleteByKey removes a flag identified by project ID and key.
// Requires at least editor role.
func (s *FlagService) DeleteByKey(ctx context.Context, projectID, key string) error {
	if _, err := requireRole(ctx, domain.RoleEditor); err != nil {
		return err
	}
	f, err := s.repo.GetByKey(ctx, projectID, key)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, f.ID); err != nil {
		return err
	}
	s.recordAudit(ctx, auditEntry{Action: "flag.deleted", EntityID: f.ID, EntityKey: f.Key, ProjectID: f.ProjectID, Before: f.Key})
	return nil
}
