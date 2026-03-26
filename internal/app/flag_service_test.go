package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/The127/cuttlegate/internal/app"
	"github.com/The127/cuttlegate/internal/domain"
	"github.com/The127/cuttlegate/internal/domain/ports"
)

// noOpPublisher is an EventPublisher that does nothing. Used by existing tests
// that don't care about event publishing.
type noOpPublisher struct{}

func (noOpPublisher) Publish(_ context.Context, _ ports.DomainEvent) error { return nil }

// noOpAuditRepository is an AuditRepository that does nothing. Used by tests
// that don't care about audit recording.
type noOpAuditRepository struct{}

func (noOpAuditRepository) Record(_ context.Context, _ *domain.AuditEvent) error { return nil }
func (noOpAuditRepository) ListByProject(_ context.Context, _ string, _ domain.AuditFilter) ([]*domain.AuditEvent, error) {
	return nil, nil
}

// spyPublisher records all published events for assertion.
type spyPublisher struct {
	events []ports.DomainEvent
	err    error // if set, Publish returns this error
}

func (s *spyPublisher) Publish(_ context.Context, event ports.DomainEvent) error {
	if s.err != nil {
		return s.err
	}
	s.events = append(s.events, event)
	return nil
}

// fakeFlagEnvironmentStateRepository is an in-memory implementation of ports.FlagEnvironmentStateRepository.
type fakeFlagEnvironmentStateRepository struct {
	states   map[string]*domain.FlagEnvironmentState // key: flagID+"/"+envID
	batchErr error                                   // if set, CreateBatch returns this error
}

func newFakeFlagEnvironmentStateRepository() *fakeFlagEnvironmentStateRepository {
	return &fakeFlagEnvironmentStateRepository{states: make(map[string]*domain.FlagEnvironmentState)}
}

func stateKey(flagID, envID string) string { return flagID + "/" + envID }

func (f *fakeFlagEnvironmentStateRepository) CreateBatch(_ context.Context, states []*domain.FlagEnvironmentState) error {
	if f.batchErr != nil {
		return f.batchErr
	}
	for _, s := range states {
		cp := *s
		f.states[stateKey(s.FlagID, s.EnvironmentID)] = &cp
	}
	return nil
}

func (f *fakeFlagEnvironmentStateRepository) ListByEnvironment(_ context.Context, environmentID string) ([]*domain.FlagEnvironmentState, error) {
	result := make([]*domain.FlagEnvironmentState, 0)
	for _, s := range f.states {
		if s.EnvironmentID == environmentID {
			cp := *s
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (f *fakeFlagEnvironmentStateRepository) GetByFlagAndEnvironment(_ context.Context, flagID, environmentID string) (*domain.FlagEnvironmentState, error) {
	s, ok := f.states[stateKey(flagID, environmentID)]
	if !ok {
		return nil, nil
	}
	cp := *s
	return &cp, nil
}

func (f *fakeFlagEnvironmentStateRepository) SetEnabled(_ context.Context, flagID, environmentID string, enabled bool) error {
	k := stateKey(flagID, environmentID)
	f.states[k] = &domain.FlagEnvironmentState{FlagID: flagID, EnvironmentID: environmentID, Enabled: enabled}
	return nil
}

func (f *fakeFlagEnvironmentStateRepository) Upsert(_ context.Context, state *domain.FlagEnvironmentState) error {
	k := stateKey(state.FlagID, state.EnvironmentID)
	cp := *state
	f.states[k] = &cp
	return nil
}

// newFlagSvc constructs a FlagService with in-memory fakes.
func newFlagSvc() *app.FlagService {
	return app.NewFlagService(newFakeFlagRepository(), newFakeEnvironmentRepository(), newFakeFlagEnvironmentStateRepository(), newFakeRuleRepository(), noOpPublisher{}, noOpAuditRepository{})
}

var testVariants = []domain.Variant{
	{Key: "true", Name: "On"},
	{Key: "false", Name: "Off"},
}

// fakeFlagRepository is an in-memory implementation of ports.FlagRepository.
type fakeFlagRepository struct {
	byKey map[string]*domain.Flag // key: projectID+"/"+key
	byID  map[string]*domain.Flag
}

func newFakeFlagRepository() *fakeFlagRepository {
	return &fakeFlagRepository{
		byKey: make(map[string]*domain.Flag),
		byID:  make(map[string]*domain.Flag),
	}
}

func flagKey(projectID, key string) string { return projectID + "/" + key }

func (f *fakeFlagRepository) Create(_ context.Context, flag *domain.Flag) error {
	k := flagKey(flag.ProjectID, flag.Key)
	if _, exists := f.byKey[k]; exists {
		return domain.ErrConflict
	}
	cp := *flag
	f.byKey[k] = &cp
	f.byID[flag.ID] = &cp
	return nil
}

func (f *fakeFlagRepository) GetByKey(_ context.Context, projectID, key string) (*domain.Flag, error) {
	flag, ok := f.byKey[flagKey(projectID, key)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *flag
	return &cp, nil
}

func (f *fakeFlagRepository) ListByProject(_ context.Context, projectID string) ([]*domain.Flag, error) {
	result := make([]*domain.Flag, 0)
	for _, flag := range f.byKey {
		if flag.ProjectID == projectID {
			cp := *flag
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (f *fakeFlagRepository) ListByProjectPaginated(ctx context.Context, projectID string, filter domain.FlagListFilter) ([]*domain.Flag, int, error) {
	filter.Normalize()
	all, _ := f.ListByProject(ctx, projectID)
	return all, len(all), nil
}

func (f *fakeFlagRepository) Update(_ context.Context, flag *domain.Flag) error {
	existing, ok := f.byID[flag.ID]
	if !ok {
		return domain.ErrNotFound
	}
	existing.Name = flag.Name
	existing.Variants = flag.Variants
	existing.DefaultVariantKey = flag.DefaultVariantKey
	return nil
}

func (f *fakeFlagRepository) Delete(_ context.Context, id string) error {
	flag, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	delete(f.byKey, flagKey(flag.ProjectID, flag.Key))
	delete(f.byID, id)
	return nil
}

func validBoolFlag(projectID string) *domain.Flag {
	return &domain.Flag{
		ProjectID:         projectID,
		Key:               "dark-mode",
		Name:              "Dark Mode",
		Type:              domain.FlagTypeBool,
		Variants:          testVariants,
		DefaultVariantKey: "false",
	}
}

func TestFlagService_Create_Succeeds(t *testing.T) {
	svc := newFlagSvc()
	f := validBoolFlag("proj-1")
	if err := svc.Create(authCtx("admin-1", domain.RoleAdmin), f); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if f.ID == "" {
		t.Error("expected non-empty ID after Create")
	}
	if f.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestFlagService_Create_InvalidFlag_ReturnsError(t *testing.T) {
	svc := newFlagSvc()
	f := &domain.Flag{
		ProjectID: "proj-1",
		Key:       "bad key!",
		Type:      domain.FlagTypeBool,
		Variants:  testVariants,
	}
	if err := svc.Create(authCtx("admin-1", domain.RoleAdmin), f); err == nil {
		t.Error("expected validation error, got nil")
	}
}

func TestFlagService_Create_DuplicateKey_ReturnsErrConflict(t *testing.T) {
	svc := newFlagSvc()
	ctx := authCtx("admin-1", domain.RoleAdmin)
	if err := svc.Create(ctx, validBoolFlag("proj-1")); err != nil {
		t.Fatalf("first create: %v", err)
	}
	err := svc.Create(ctx, validBoolFlag("proj-1"))
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestFlagService_GetByKey_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := newFlagSvc()
	_, err := svc.GetByKey(context.Background(), "proj-1", "ghost")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFlagService_ListByProject_EmptyReturnsEmptySlice(t *testing.T) {
	svc := newFlagSvc()
	list, err := svc.ListByProject(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if list == nil {
		t.Error("expected empty slice, got nil")
	}
}

func TestFlagService_Update_Succeeds(t *testing.T) {
	svc := newFlagSvc()
	ctx := authCtx("admin-1", domain.RoleAdmin)
	f := validBoolFlag("proj-1")
	if err := svc.Create(ctx, f); err != nil {
		t.Fatalf("Create: %v", err)
	}
	f.Name = "Dark Mode Beta"
	if err := svc.Update(ctx, f); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err := svc.GetByKey(ctx, "proj-1", "dark-mode")
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if got.Name != "Dark Mode Beta" {
		t.Errorf("name: got %q", got.Name)
	}
}

func TestFlagService_DeleteByKey_Succeeds(t *testing.T) {
	svc := newFlagSvc()
	ctx := authCtx("admin-1", domain.RoleAdmin)
	if err := svc.Create(ctx, validBoolFlag("proj-1")); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.DeleteByKey(ctx, "proj-1", "dark-mode"); err != nil {
		t.Fatalf("DeleteByKey: %v", err)
	}
	_, err := svc.GetByKey(ctx, "proj-1", "dark-mode")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestFlagService_DeleteByKey_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := newFlagSvc()
	err := svc.DeleteByKey(authCtx("admin-1", domain.RoleAdmin), "proj-1", "ghost")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── RBAC scenarios ────────────────────────────────────────────────────────────

// Scenario 1 & 4: viewer calling a write use-case returns ErrForbidden (no HTTP)
func TestFlagService_Create_ViewerReturnsForbidden(t *testing.T) {
	svc := newFlagSvc()
	err := svc.Create(authCtx("viewer-1", domain.RoleViewer), validBoolFlag("proj-1"))
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

// Scenario 2: editor can create/update/delete flags
func TestFlagService_Create_EditorSucceeds(t *testing.T) {
	svc := newFlagSvc()
	if err := svc.Create(authCtx("editor-1", domain.RoleEditor), validBoolFlag("proj-1")); err != nil {
		t.Errorf("expected no error for editor, got %v", err)
	}
}

func TestFlagService_Update_EditorSucceeds(t *testing.T) {
	svc := newFlagSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	f := validBoolFlag("proj-1")
	if err := svc.Create(ctx, f); err != nil {
		t.Fatalf("Create: %v", err)
	}
	f.Name = "Updated"
	if err := svc.Update(ctx, f); err != nil {
		t.Errorf("expected no error for editor, got %v", err)
	}
}

func TestFlagService_DeleteByKey_EditorSucceeds(t *testing.T) {
	svc := newFlagSvc()
	ctx := authCtx("editor-1", domain.RoleEditor)
	if err := svc.Create(ctx, validBoolFlag("proj-1")); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.DeleteByKey(ctx, "proj-1", "dark-mode"); err != nil {
		t.Errorf("expected no error for editor, got %v", err)
	}
}

// Scenario 5: admin can perform any write operation
func TestFlagService_Create_AdminSucceeds(t *testing.T) {
	svc := newFlagSvc()
	if err := svc.Create(authCtx("admin-1", domain.RoleAdmin), validBoolFlag("proj-1")); err != nil {
		t.Errorf("expected no error for admin, got %v", err)
	}
}

// Scenario 6: missing AuthContext returns ErrForbidden
func TestFlagService_Create_NoAuthContextReturnsForbidden(t *testing.T) {
	svc := newFlagSvc()
	err := svc.Create(context.Background(), validBoolFlag("proj-1"))
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden for missing auth, got %v", err)
	}
}

// Scenario 8: wrapped ErrForbidden still satisfies errors.Is
func TestFlagService_Create_ViewerWrappedForbidden(t *testing.T) {
	svc := newFlagSvc()
	err := svc.Create(authCtx("viewer-1", domain.RoleViewer), validBoolFlag("proj-1"))
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("errors.Is(err, ErrForbidden) must be true, got %v", err)
	}
}

func TestFlagService_Create_NoEnvironments_Succeeds(t *testing.T) {
	svc := newFlagSvc() // no environments seeded
	f := validBoolFlag("proj-1")
	if err := svc.Create(authCtx("admin-1", domain.RoleAdmin), f); err != nil {
		t.Fatalf("expected flag creation without environments to succeed, got %v", err)
	}
	if f.ID == "" {
		t.Error("expected non-empty ID after Create")
	}
}

// ── FlagEnvironmentState scenarios ───────────────────────────────────────────

// Scenario 1: creating a flag auto-creates a disabled state row for every existing environment.
func TestFlagService_Create_AutoCreatesStateRowsForAllEnvironments(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	envRepo := newFakeEnvironmentRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	svc := app.NewFlagService(flagRepo, envRepo, stateRepo, newFakeRuleRepository(), noOpPublisher{}, noOpAuditRepository{})
	ctx := authCtx("admin-1", domain.RoleAdmin)

	// Seed two environments for the project.
	devEnv := domain.Environment{ID: "env-dev", ProjectID: "proj-1", Slug: "dev", Name: "Dev"}
	prodEnv := domain.Environment{ID: "env-prod", ProjectID: "proj-1", Slug: "prod", Name: "Prod"}
	envRepo.byKey[envKey("proj-1", "dev")] = &devEnv
	envRepo.byID["env-dev"] = &devEnv
	envRepo.byKey[envKey("proj-1", "prod")] = &prodEnv
	envRepo.byID["env-prod"] = &prodEnv

	f := validBoolFlag("proj-1")
	if err := svc.Create(ctx, f); err != nil {
		t.Fatalf("Create: %v", err)
	}

	devState := stateRepo.states[stateKey(f.ID, "env-dev")]
	if devState == nil {
		t.Error("expected state row for dev environment")
	} else if devState.Enabled {
		t.Error("expected state to be disabled for dev")
	}
	prodState := stateRepo.states[stateKey(f.ID, "env-prod")]
	if prodState == nil {
		t.Error("expected state row for prod environment")
	} else if prodState.Enabled {
		t.Error("expected state to be disabled for prod")
	}
}

// Scenario 2: flag creation is atomic — state row failure rolls back the flag.
func TestFlagService_Create_StateRowFailureCompensatesFlag(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	envRepo := newFakeEnvironmentRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	stateRepo.batchErr = errors.New("db: insert failed")
	svc := app.NewFlagService(flagRepo, envRepo, stateRepo, newFakeRuleRepository(), noOpPublisher{}, noOpAuditRepository{})
	ctx := authCtx("admin-1", domain.RoleAdmin)

	// Seed one environment so CreateBatch is actually called.
	env := domain.Environment{ID: "env-dev", ProjectID: "proj-1", Slug: "dev", Name: "Dev"}
	envRepo.byKey[envKey("proj-1", "dev")] = &env
	envRepo.byID["env-dev"] = &env

	f := validBoolFlag("proj-1")
	err := svc.Create(ctx, f)
	if err == nil {
		t.Fatal("expected error from CreateBatch, got nil")
	}

	// Flag must not be persisted.
	_, getErr := flagRepo.GetByKey(context.Background(), "proj-1", "dark-mode")
	if !errors.Is(getErr, domain.ErrNotFound) {
		t.Errorf("flag should have been compensated (deleted), got %v", getErr)
	}
}

// Scenario 3: enabling in one environment does not affect another.
func TestFlagService_SetEnabled_IsolatedPerEnvironment(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	envRepo := newFakeEnvironmentRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	svc := app.NewFlagService(flagRepo, envRepo, stateRepo, newFakeRuleRepository(), noOpPublisher{}, noOpAuditRepository{})
	ctx := authCtx("editor-1", domain.RoleEditor)

	// Seed flag and state rows directly.
	f := validBoolFlag("proj-1")
	f.ID = "flag-1"
	flagRepo.byKey[flagKey("proj-1", "dark-mode")] = f
	flagRepo.byID["flag-1"] = f
	stateRepo.states[stateKey("flag-1", "env-dev")] = &domain.FlagEnvironmentState{FlagID: "flag-1", EnvironmentID: "env-dev", Enabled: false}
	stateRepo.states[stateKey("flag-1", "env-prod")] = &domain.FlagEnvironmentState{FlagID: "flag-1", EnvironmentID: "env-prod", Enabled: false}

	if err := svc.SetEnabled(ctx, app.SetEnabledParams{
		ProjectID: "proj-1", EnvironmentID: "env-dev", FlagKey: "dark-mode",
		Enabled: true, ProjectSlug: "alpha", EnvSlug: "dev",
	}); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}

	if !stateRepo.states[stateKey("flag-1", "env-dev")].Enabled {
		t.Error("expected dev to be enabled")
	}
	if stateRepo.states[stateKey("flag-1", "env-prod")].Enabled {
		t.Error("expected prod to remain disabled")
	}
}

// Scenario 5: list for environment with no flags returns empty (non-nil) slice.
func TestFlagService_ListByEnvironment_NoFlags_ReturnsEmptySlice(t *testing.T) {
	svc := newFlagSvc()
	views, err := svc.ListByEnvironment(context.Background(), "proj-1", "env-staging")
	if err != nil {
		t.Fatalf("ListByEnvironment: %v", err)
	}
	if views == nil {
		t.Error("expected non-nil slice")
	}
	if len(views) != 0 {
		t.Errorf("expected 0 items, got %d", len(views))
	}
}

// Scenario 6: SetEnabled upserts when no state row exists yet.
func TestFlagService_SetEnabled_MissingStateRow_Upserts(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	envRepo := newFakeEnvironmentRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	svc := app.NewFlagService(flagRepo, envRepo, stateRepo, newFakeRuleRepository(), noOpPublisher{}, noOpAuditRepository{})

	f := validBoolFlag("proj-1")
	f.ID = "flag-1"
	flagRepo.byKey[flagKey("proj-1", "dark-mode")] = f
	flagRepo.byID["flag-1"] = f

	err := svc.SetEnabled(authCtx("editor-1", domain.RoleEditor), app.SetEnabledParams{
		ProjectID: "proj-1", EnvironmentID: "env-new", FlagKey: "dark-mode",
		Enabled: true, ProjectSlug: "alpha", EnvSlug: "new",
	})
	if err != nil {
		t.Fatalf("expected upsert to succeed, got %v", err)
	}
	state := stateRepo.states[stateKey("flag-1", "env-new")]
	if state == nil {
		t.Fatal("expected state row to be created")
	}
	if !state.Enabled {
		t.Error("expected state to be enabled")
	}
}

// ── Variant management scenarios ─────────────────────────────────────────────

func seedStringFlag(flagRepo *fakeFlagRepository) *domain.Flag {
	f := &domain.Flag{
		ID:        "flag-theme",
		ProjectID: "proj-1",
		Key:       "theme",
		Name:      "Theme",
		Type:      domain.FlagTypeString,
		Variants: []domain.Variant{
			{Key: "light", Name: "Light"},
			{Key: "dark", Name: "Dark"},
		},
		DefaultVariantKey: "light",
	}
	flagRepo.byKey[flagKey("proj-1", "theme")] = f
	flagRepo.byID["flag-theme"] = f
	return f
}

func seedBoolFlag(flagRepo *fakeFlagRepository) *domain.Flag {
	f := validBoolFlag("proj-1")
	f.ID = "flag-dark-mode"
	flagRepo.byKey[flagKey("proj-1", "dark-mode")] = f
	flagRepo.byID["flag-dark-mode"] = f
	return f
}

// Scenario 1: add variant to string flag succeeds
func TestFlagService_AddVariant_StringFlag_Succeeds(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), newFakeFlagEnvironmentStateRepository(), newFakeRuleRepository(), noOpPublisher{}, noOpAuditRepository{})
	seedStringFlag(flagRepo)
	ctx := authCtx("editor-1", domain.RoleEditor)

	f, err := svc.AddVariant(ctx, "proj-1", "theme", domain.Variant{Key: "system", Name: "System"})
	if err != nil {
		t.Fatalf("AddVariant: %v", err)
	}
	if len(f.Variants) != 3 {
		t.Errorf("expected 3 variants, got %d", len(f.Variants))
	}
}

// Scenario 2: add variant to bool flag returns ErrImmutableVariants
func TestFlagService_AddVariant_BoolFlag_ReturnsErrImmutableVariants(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), newFakeFlagEnvironmentStateRepository(), newFakeRuleRepository(), noOpPublisher{}, noOpAuditRepository{})
	seedBoolFlag(flagRepo)

	_, err := svc.AddVariant(authCtx("editor-1", domain.RoleEditor), "proj-1", "dark-mode", domain.Variant{Key: "maybe", Name: "Maybe"})
	if !errors.Is(err, domain.ErrImmutableVariants) {
		t.Errorf("expected ErrImmutableVariants, got %v", err)
	}
}

// Scenario 3: rename variant updates only the name
func TestFlagService_RenameVariant_UpdatesName(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), newFakeFlagEnvironmentStateRepository(), newFakeRuleRepository(), noOpPublisher{}, noOpAuditRepository{})
	seedStringFlag(flagRepo)
	ctx := authCtx("editor-1", domain.RoleEditor)

	f, err := svc.RenameVariant(ctx, "proj-1", "theme", "light", "Light Mode")
	if err != nil {
		t.Fatalf("RenameVariant: %v", err)
	}
	for _, v := range f.Variants {
		if v.Key == "light" {
			if v.Name != "Light Mode" {
				t.Errorf("name: got %q, want %q", v.Name, "Light Mode")
			}
			return
		}
	}
	t.Error("variant 'light' not found after rename")
}

// Scenario 5: delete non-default variant succeeds
func TestFlagService_DeleteVariant_NonDefault_Succeeds(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), newFakeFlagEnvironmentStateRepository(), newFakeRuleRepository(), noOpPublisher{}, noOpAuditRepository{})
	// Three-variant flag so we can delete one without hitting ErrLastVariant
	f := seedStringFlag(flagRepo)
	f.Variants = append(f.Variants, domain.Variant{Key: "system", Name: "System"})
	ctx := authCtx("editor-1", domain.RoleEditor)

	result, err := svc.DeleteVariant(ctx, "proj-1", "theme", "system", false)
	if err != nil {
		t.Fatalf("DeleteVariant: %v", err)
	}
	if len(result.Variants) != 2 {
		t.Errorf("expected 2 variants remaining, got %d", len(result.Variants))
	}
}

// Scenario 6: delete default variant returns ErrDefaultVariant
func TestFlagService_DeleteVariant_DefaultVariant_ReturnsErrDefaultVariant(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), newFakeFlagEnvironmentStateRepository(), newFakeRuleRepository(), noOpPublisher{}, noOpAuditRepository{})
	seedStringFlag(flagRepo)

	_, err := svc.DeleteVariant(authCtx("editor-1", domain.RoleEditor), "proj-1", "theme", "light", false)
	if !errors.Is(err, domain.ErrDefaultVariant) {
		t.Errorf("expected ErrDefaultVariant, got %v", err)
	}
}

// Scenario 7: delete variant from bool flag returns ErrImmutableVariants
func TestFlagService_DeleteVariant_BoolFlag_ReturnsErrImmutableVariants(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), newFakeFlagEnvironmentStateRepository(), newFakeRuleRepository(), noOpPublisher{}, noOpAuditRepository{})
	seedBoolFlag(flagRepo)

	_, err := svc.DeleteVariant(authCtx("editor-1", domain.RoleEditor), "proj-1", "dark-mode", "true", false)
	if !errors.Is(err, domain.ErrImmutableVariants) {
		t.Errorf("expected ErrImmutableVariants, got %v", err)
	}
}

// Scenario 8: add duplicate variant key returns ErrConflict
func TestFlagService_AddVariant_DuplicateKey_ReturnsErrConflict(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), newFakeFlagEnvironmentStateRepository(), newFakeRuleRepository(), noOpPublisher{}, noOpAuditRepository{})
	seedStringFlag(flagRepo)

	_, err := svc.AddVariant(authCtx("editor-1", domain.RoleEditor), "proj-1", "theme", domain.Variant{Key: "light", Name: "Light Dup"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

// Scenario 9: viewer attempting variant mutation returns ErrForbidden
func TestFlagService_AddVariant_ViewerReturnsForbidden(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), newFakeFlagEnvironmentStateRepository(), newFakeRuleRepository(), noOpPublisher{}, noOpAuditRepository{})
	seedStringFlag(flagRepo)

	_, err := svc.AddVariant(authCtx("viewer-1", domain.RoleViewer), "proj-1", "theme", domain.Variant{Key: "system", Name: "System"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

// ErrLastVariant invariant
func TestFlagService_DeleteVariant_LastVariant_ReturnsErrLastVariant(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), newFakeFlagEnvironmentStateRepository(), newFakeRuleRepository(), noOpPublisher{}, noOpAuditRepository{})
	// Single non-default variant flag
	f := &domain.Flag{
		ID:                "flag-single",
		ProjectID:         "proj-1",
		Key:               "single",
		Name:              "Single",
		Type:              domain.FlagTypeString,
		Variants:          []domain.Variant{{Key: "only", Name: "Only"}, {Key: "default", Name: "Default"}},
		DefaultVariantKey: "default",
	}
	flagRepo.byKey[flagKey("proj-1", "single")] = f
	flagRepo.byID["flag-single"] = f
	// Remove one so only 1 remains
	f.Variants = []domain.Variant{{Key: "only", Name: "Only"}}
	f.DefaultVariantKey = "only"

	_, err := svc.DeleteVariant(authCtx("editor-1", domain.RoleEditor), "proj-1", "single", "only", false)
	// "only" is also the default variant, so ErrDefaultVariant fires first
	if !errors.Is(err, domain.ErrDefaultVariant) {
		t.Errorf("expected ErrDefaultVariant (default check fires before last-variant check), got %v", err)
	}
}

// ── Variant-in-use protection ─────────────────────────────────────────────

func TestFlagService_DeleteVariant_InUseByRule_ReturnsErrVariantInUse(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	ruleRepo := newFakeRuleRepository()
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), newFakeFlagEnvironmentStateRepository(), ruleRepo, noOpPublisher{}, noOpAuditRepository{})
	ctx := authCtx("editor-1", domain.RoleEditor)

	f := seedStringFlag(flagRepo)
	f.Variants = append(f.Variants, domain.Variant{Key: "system", Name: "System"})
	// Create a rule referencing the "system" variant.
	ruleRepo.rules["rule-1"] = &domain.Rule{
		ID: "rule-1", FlagID: f.ID, EnvironmentID: "env-1",
		Conditions: []domain.Condition{{Attribute: "plan", Operator: domain.OperatorEq, Values: []string{"pro"}}},
		VariantKey: "system", Enabled: true,
	}

	_, err := svc.DeleteVariant(ctx, "proj-1", "theme", "system", false)
	if !errors.Is(err, domain.ErrVariantInUse) {
		t.Errorf("expected ErrVariantInUse, got %v", err)
	}
}

func TestFlagService_DeleteVariant_Force_PatchesRuleToDefault(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	ruleRepo := newFakeRuleRepository()
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), newFakeFlagEnvironmentStateRepository(), ruleRepo, noOpPublisher{}, noOpAuditRepository{})
	ctx := authCtx("editor-1", domain.RoleEditor)

	f := seedStringFlag(flagRepo)
	f.Variants = append(f.Variants, domain.Variant{Key: "system", Name: "System"})
	ruleRepo.rules["rule-1"] = &domain.Rule{
		ID: "rule-1", FlagID: f.ID, EnvironmentID: "env-1",
		Conditions: []domain.Condition{{Attribute: "plan", Operator: domain.OperatorEq, Values: []string{"pro"}}},
		VariantKey: "system", Enabled: true,
	}

	_, err := svc.DeleteVariant(ctx, "proj-1", "theme", "system", true)
	if err != nil {
		t.Fatalf("force DeleteVariant: %v", err)
	}
	// Rule should now point to the default variant.
	r := ruleRepo.rules["rule-1"]
	if r.VariantKey != f.DefaultVariantKey {
		t.Errorf("expected rule variant to be %q (default), got %q", f.DefaultVariantKey, r.VariantKey)
	}
}

func TestFlagService_DeleteVariant_Force_PatchesRolloutEntries(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	ruleRepo := newFakeRuleRepository()
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), newFakeFlagEnvironmentStateRepository(), ruleRepo, noOpPublisher{}, noOpAuditRepository{})
	ctx := authCtx("editor-1", domain.RoleEditor)

	f := seedStringFlag(flagRepo)
	f.Variants = append(f.Variants, domain.Variant{Key: "system", Name: "System"})
	ruleRepo.rules["rule-1"] = &domain.Rule{
		ID: "rule-1", FlagID: f.ID, EnvironmentID: "env-1",
		Conditions: []domain.Condition{{Attribute: "plan", Operator: domain.OperatorEq, Values: []string{"pro"}}},
		Rollout: []domain.RolloutEntry{
			{VariantKey: "light", Weight: 40},
			{VariantKey: "system", Weight: 30},
			{VariantKey: "dark", Weight: 30},
		},
		Enabled: true,
	}

	_, err := svc.DeleteVariant(ctx, "proj-1", "theme", "system", true)
	if err != nil {
		t.Fatalf("force DeleteVariant: %v", err)
	}
	r := ruleRepo.rules["rule-1"]
	if len(r.Rollout) != 2 {
		t.Fatalf("expected 2 rollout entries, got %d", len(r.Rollout))
	}
	// First entry should absorb the removed weight.
	if r.Rollout[0].VariantKey != "light" || r.Rollout[0].Weight != 70 {
		t.Errorf("expected light/70, got %s/%d", r.Rollout[0].VariantKey, r.Rollout[0].Weight)
	}
	if r.Rollout[1].VariantKey != "dark" || r.Rollout[1].Weight != 30 {
		t.Errorf("expected dark/30, got %s/%d", r.Rollout[1].VariantKey, r.Rollout[1].Weight)
	}
}

// ── Flag state change propagation scenarios (#48) ────────────────────────────

func seedFlagWithState(flagRepo *fakeFlagRepository, stateRepo *fakeFlagEnvironmentStateRepository) {
	f := validBoolFlag("proj-1")
	f.ID = "flag-1"
	flagRepo.byKey[flagKey("proj-1", "dark-mode")] = f
	flagRepo.byID["flag-1"] = f
	stateRepo.states[stateKey("flag-1", "env-staging")] = &domain.FlagEnvironmentState{
		FlagID: "flag-1", EnvironmentID: "env-staging", Enabled: false,
	}
}

// BDD Scenario 1: Enabling a flag publishes a state changed event
func TestFlagService_SetEnabled_PublishesEventOnEnable(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	spy := &spyPublisher{}
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), stateRepo, newFakeRuleRepository(), spy, noOpAuditRepository{})
	seedFlagWithState(flagRepo, stateRepo)

	err := svc.SetEnabled(authCtx("editor-1", domain.RoleEditor), app.SetEnabledParams{
		ProjectID: "proj-1", EnvironmentID: "env-staging", FlagKey: "dark-mode",
		Enabled: true, ProjectSlug: "alpha", EnvSlug: "staging",
	})
	if err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	if len(spy.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(spy.events))
	}
	evt, ok := spy.events[0].(domain.FlagStateChangedEvent)
	if !ok {
		t.Fatalf("expected FlagStateChangedEvent, got %T", spy.events[0])
	}
	if evt.EventType() != "flag.state_changed" {
		t.Errorf("event type: got %q", evt.EventType())
	}
	if evt.ProjectSlug() != "alpha" {
		t.Errorf("project slug: got %q", evt.ProjectSlug())
	}
	if evt.EnvironmentSlug() != "staging" {
		t.Errorf("environment slug: got %q", evt.EnvironmentSlug())
	}
	if evt.FlagKey() != "dark-mode" {
		t.Errorf("flag key: got %q", evt.FlagKey())
	}
	if !evt.Enabled() {
		t.Error("expected enabled=true")
	}
}

// BDD Scenario 2: Disabling a flag publishes a state changed event
func TestFlagService_SetEnabled_PublishesEventOnDisable(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	spy := &spyPublisher{}
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), stateRepo, newFakeRuleRepository(), spy, noOpAuditRepository{})
	seedFlagWithState(flagRepo, stateRepo)
	// Enable first so we can disable.
	stateRepo.states[stateKey("flag-1", "env-staging")].Enabled = true

	err := svc.SetEnabled(authCtx("editor-1", domain.RoleEditor), app.SetEnabledParams{
		ProjectID: "proj-1", EnvironmentID: "env-staging", FlagKey: "dark-mode",
		Enabled: false, ProjectSlug: "alpha", EnvSlug: "staging",
	})
	if err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	if len(spy.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(spy.events))
	}
	evt := spy.events[0].(domain.FlagStateChangedEvent)
	if evt.Enabled() {
		t.Error("expected enabled=false")
	}
}

// BDD Scenario 3: SetEnabled failure does not publish an event
func TestFlagService_SetEnabled_NoEventOnFailure(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	spy := &spyPublisher{}
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), stateRepo, newFakeRuleRepository(), spy, noOpAuditRepository{})
	// Flag does not exist → SetEnabled returns ErrNotFound before reaching state repo.
	err := svc.SetEnabled(authCtx("editor-1", domain.RoleEditor), app.SetEnabledParams{
		ProjectID: "proj-1", EnvironmentID: "env-1", FlagKey: "ghost",
		Enabled: true, ProjectSlug: "alpha", EnvSlug: "staging",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(spy.events) != 0 {
		t.Errorf("expected 0 events on failure, got %d", len(spy.events))
	}
}

// BDD Scenario 4: Publish failure does not fail the state change
func TestFlagService_SetEnabled_PublishFailureDoesNotFailRequest(t *testing.T) {
	flagRepo := newFakeFlagRepository()
	stateRepo := newFakeFlagEnvironmentStateRepository()
	spy := &spyPublisher{err: errors.New("broker down")}
	svc := app.NewFlagService(flagRepo, newFakeEnvironmentRepository(), stateRepo, newFakeRuleRepository(), spy, noOpAuditRepository{})
	seedFlagWithState(flagRepo, stateRepo)

	err := svc.SetEnabled(authCtx("editor-1", domain.RoleEditor), app.SetEnabledParams{
		ProjectID: "proj-1", EnvironmentID: "env-staging", FlagKey: "dark-mode",
		Enabled: true, ProjectSlug: "alpha", EnvSlug: "staging",
	})
	if err != nil {
		t.Fatalf("SetEnabled should succeed despite publish failure, got %v", err)
	}
	// State should be updated even though publish failed.
	if !stateRepo.states[stateKey("flag-1", "env-staging")].Enabled {
		t.Error("state should be enabled despite publish failure")
	}
}

// BDD Scenario 5: FlagStateChangedEvent satisfies DomainEvent interface
func TestFlagStateChangedEvent_SatisfiesDomainEvent(t *testing.T) {
	evt := domain.NewFlagStateChangedEvent("my-project", "production", "dark-mode", true)
	if evt.EventType() != "flag.state_changed" {
		t.Errorf("EventType: got %q", evt.EventType())
	}
	if evt.OccurredAt().IsZero() {
		t.Error("OccurredAt should not be zero")
	}
}
