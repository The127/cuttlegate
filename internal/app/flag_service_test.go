package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/karo/cuttlegate/internal/app"
	"github.com/karo/cuttlegate/internal/domain"
)

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
	svc := app.NewFlagService(newFakeFlagRepository())
	f := validBoolFlag("proj-1")
	if err := svc.Create(context.Background(), f); err != nil {
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
	svc := app.NewFlagService(newFakeFlagRepository())
	f := &domain.Flag{
		ProjectID: "proj-1",
		Key:       "bad key!",
		Type:      domain.FlagTypeBool,
		Variants:  testVariants,
	}
	if err := svc.Create(context.Background(), f); err == nil {
		t.Error("expected validation error, got nil")
	}
}

func TestFlagService_Create_DuplicateKey_ReturnsErrConflict(t *testing.T) {
	svc := app.NewFlagService(newFakeFlagRepository())
	ctx := context.Background()
	if err := svc.Create(ctx, validBoolFlag("proj-1")); err != nil {
		t.Fatalf("first create: %v", err)
	}
	err := svc.Create(ctx, validBoolFlag("proj-1"))
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestFlagService_GetByKey_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := app.NewFlagService(newFakeFlagRepository())
	_, err := svc.GetByKey(context.Background(), "proj-1", "ghost")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFlagService_ListByProject_EmptyReturnsEmptySlice(t *testing.T) {
	svc := app.NewFlagService(newFakeFlagRepository())
	list, err := svc.ListByProject(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if list == nil {
		t.Error("expected empty slice, got nil")
	}
}

func TestFlagService_Update_Succeeds(t *testing.T) {
	svc := app.NewFlagService(newFakeFlagRepository())
	ctx := context.Background()
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
	svc := app.NewFlagService(newFakeFlagRepository())
	ctx := context.Background()
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
	svc := app.NewFlagService(newFakeFlagRepository())
	err := svc.DeleteByKey(context.Background(), "proj-1", "ghost")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
