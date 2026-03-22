//go:build integration

package dbadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

func TestRuleService_PriorityCollision_Integration(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	projRepo := dbadapter.NewPostgresProjectRepository(db)
	envRepo := dbadapter.NewPostgresEnvironmentRepository(db)
	flagRepo := dbadapter.NewPostgresFlagRepository(db)
	ruleRepo := dbadapter.NewPostgresRuleRepository(db)
	svc := app.NewRuleService(ruleRepo)

	const (
		projID = "ffffffff-f001-4001-8001-000000000001"
		envID  = "ffffffff-f001-4001-8001-000000000002"
		flagID = "ffffffff-f001-4001-8001-000000000003"
	)

	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM rules WHERE flag_id = $1`, flagID)
		db.ExecContext(ctx, `DELETE FROM flags WHERE id = $1`, flagID)
		db.ExecContext(ctx, `DELETE FROM environments WHERE id = $1`, envID)
		db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, projID)
	})

	if err := projRepo.Create(ctx, domain.Project{
		ID:        projID,
		Name:      "Rule Test Project",
		Slug:      "rule-test-proj",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if err := envRepo.Create(ctx, domain.Environment{
		ID:        envID,
		ProjectID: projID,
		Name:      "Production",
		Slug:      "production",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}); err != nil {
		t.Fatalf("seed environment: %v", err)
	}
	if err := flagRepo.Create(ctx, &domain.Flag{
		ID:                flagID,
		ProjectID:         projID,
		Key:               "rule-test-flag",
		Name:              "Rule Test Flag",
		Type:              domain.FlagTypeBool,
		Variants:          []domain.Variant{{Key: "true", Name: "On"}, {Key: "false", Name: "Off"}},
		DefaultVariantKey: "false",
		CreatedAt:         time.Now().UTC().Truncate(time.Microsecond),
	}); err != nil {
		t.Fatalf("seed flag: %v", err)
	}

	authCtxEditor := domain.NewAuthContext(ctx, domain.AuthContext{UserID: "editor-1", Role: domain.RoleEditor})
	conditions := []domain.Condition{
		{Attribute: "plan", Operator: domain.OperatorEq, Values: []string{"pro"}},
	}

	t.Run("first rule at priority 5 succeeds", func(t *testing.T) {
		_, err := svc.Create(authCtxEditor, flagID, envID, 5, conditions, "true", "")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
	})

	t.Run("second rule at same priority returns ErrPriorityConflict", func(t *testing.T) {
		_, err := svc.Create(authCtxEditor, flagID, envID, 5, conditions, "false", "")
		if !errors.Is(err, domain.ErrPriorityConflict) {
			t.Errorf("expected ErrPriorityConflict, got %v", err)
		}
	})

	t.Run("rule at different priority succeeds", func(t *testing.T) {
		_, err := svc.Create(authCtxEditor, flagID, envID, 10, conditions, "false", "")
		if err != nil {
			t.Errorf("expected success for different priority, got %v", err)
		}
	})
}
