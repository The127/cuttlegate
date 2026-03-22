//go:build integration

package dbadapter_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
	"github.com/karo/cuttlegate/internal/domain"
)

func TestUnitOfWork_CommitPersistsBothRepos(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	const (
		projID = "ffffffff-a001-4001-8001-000000000001"
		envID  = "ffffffff-a001-4001-8001-000000000002"
		flagID = "ffffffff-a001-4001-8001-000000000003"
	)

	seedProjectEnvFlag(t, db, ctx, projID, envID, flagID)

	factory := dbadapter.NewPostgresUnitOfWorkFactory(db)

	uow, err := factory.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	// Seed a state row within the UoW.
	stateRepo := uow.FlagEnvironmentStateRepository()
	if err := stateRepo.CreateBatch(ctx, []*domain.FlagEnvironmentState{
		{FlagID: flagID, EnvironmentID: envID, Enabled: true},
	}); err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	// Write an audit event within the UoW.
	auditRepo := uow.AuditRepository()
	if err := auditRepo.Record(ctx, &domain.AuditEvent{
		ID:         "ffffffff-a001-4001-8001-000000000010",
		ProjectID:  projID,
		ActorID:    "actor-1",
		Action:     "flag.enabled",
		EntityType: "flag_environment_state",
		EntityID:   flagID,
		EntityKey:  "uow-test-flag",
		OccurredAt: time.Now().UTC().Truncate(time.Microsecond),
	}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	if err := uow.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Verify state was persisted.
	directStateRepo := dbadapter.NewPostgresFlagEnvironmentStateRepository(db)
	state, err := directStateRepo.GetByFlagAndEnvironment(ctx, flagID, envID)
	if err != nil {
		t.Fatalf("verify state: %v", err)
	}
	if state == nil || !state.Enabled {
		t.Error("expected flag to be enabled after commit")
	}

	// Verify audit event was persisted.
	directAuditRepo := dbadapter.NewPostgresAuditRepository(db)
	events, err := directAuditRepo.ListByProject(ctx, projID, domain.AuditFilter{})
	if err != nil {
		t.Fatalf("verify audit: %v", err)
	}
	if len(events) == 0 {
		t.Error("expected audit event after commit, got none")
	}
}

func TestUnitOfWork_RollbackDiscardsWrites(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	const (
		projID = "ffffffff-b001-4001-8001-000000000001"
		envID  = "ffffffff-b001-4001-8001-000000000002"
		flagID = "ffffffff-b001-4001-8001-000000000003"
	)

	seedProjectEnvFlag(t, db, ctx, projID, envID, flagID)

	factory := dbadapter.NewPostgresUnitOfWorkFactory(db)

	uow, err := factory.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	// Write state within UoW.
	stateRepo := uow.FlagEnvironmentStateRepository()
	if err := stateRepo.CreateBatch(ctx, []*domain.FlagEnvironmentState{
		{FlagID: flagID, EnvironmentID: envID, Enabled: true},
	}); err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	// Write audit within UoW.
	auditRepo := uow.AuditRepository()
	if err := auditRepo.Record(ctx, &domain.AuditEvent{
		ID:         "ffffffff-b001-4001-8001-000000000010",
		ProjectID:  projID,
		ActorID:    "actor-1",
		Action:     "flag.enabled",
		EntityType: "flag_environment_state",
		EntityID:   flagID,
		EntityKey:  "uow-rollback-flag",
		OccurredAt: time.Now().UTC().Truncate(time.Microsecond),
	}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	if err := uow.Rollback(ctx); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	// Verify no state row was persisted.
	directStateRepo := dbadapter.NewPostgresFlagEnvironmentStateRepository(db)
	state, err := directStateRepo.GetByFlagAndEnvironment(ctx, flagID, envID)
	if err != nil {
		t.Fatalf("verify state: %v", err)
	}
	if state != nil {
		t.Error("expected no state row after rollback")
	}

	// Verify no audit event persisted.
	directAuditRepo := dbadapter.NewPostgresAuditRepository(db)
	events, err := directAuditRepo.ListByProject(ctx, projID, domain.AuditFilter{})
	if err != nil {
		t.Fatalf("verify audit: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 audit events after rollback, got %d", len(events))
	}
}

func TestUnitOfWork_RollbackAfterCommitIsNoop(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	factory := dbadapter.NewPostgresUnitOfWorkFactory(db)

	uow, err := factory.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	if err := uow.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	if err := uow.Rollback(ctx); err != nil {
		t.Errorf("expected Rollback after Commit to be no-op, got: %v", err)
	}
}

func TestUnitOfWork_CommitAfterRollbackIsError(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	factory := dbadapter.NewPostgresUnitOfWorkFactory(db)

	uow, err := factory.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	if err := uow.Rollback(ctx); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	if err := uow.Commit(ctx); err == nil {
		t.Error("expected Commit after Rollback to return an error")
	}
}

// seedProjectEnvFlag creates the minimum scaffolding (project, environment, flag)
// needed to test repository writes that reference these entities via foreign keys.
func seedProjectEnvFlag(t *testing.T, db *sql.DB, ctx context.Context, projID, envID, flagID string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)

	projRepo := dbadapter.NewPostgresProjectRepository(db)
	if err := projRepo.Create(ctx, domain.Project{
		ID: projID, Name: "UoW Test Project", Slug: "uow-test-" + projID[:8],
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	envRepo := dbadapter.NewPostgresEnvironmentRepository(db)
	if err := envRepo.Create(ctx, domain.Environment{
		ID: envID, ProjectID: projID, Name: "Test Env", Slug: "test-env",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed environment: %v", err)
	}

	flagRepo := dbadapter.NewPostgresFlagRepository(db)
	if err := flagRepo.Create(ctx, &domain.Flag{
		ID: flagID, ProjectID: projID, Key: "uow-flag",
		Name: "UoW Flag", Type: domain.FlagTypeBool,
		Variants:          []domain.Variant{{Key: "true", Name: "On"}, {Key: "false", Name: "Off"}},
		DefaultVariantKey: "false",
		CreatedAt:         now,
	}); err != nil {
		t.Fatalf("seed flag: %v", err)
	}
}
