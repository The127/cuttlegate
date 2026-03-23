//go:build integration

package dbadapter_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
	"github.com/karo/cuttlegate/internal/domain"
)

func TestPostgresAPIKeyRepository(t *testing.T) {
	db := newTestDB(t)
	projRepo := dbadapter.NewPostgresProjectRepository(db)
	envRepo := dbadapter.NewPostgresEnvironmentRepository(db)
	keyRepo := dbadapter.NewPostgresAPIKeyRepository(db)
	ctx := context.Background()

	proj := domain.Project{
		ID:        "aa000000-0000-4000-8000-000000000001",
		Name:      "APIKey Test Project",
		Slug:      "apikey-test-proj",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	if err := projRepo.Create(ctx, proj); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	env := domain.Environment{
		ID:        "bb000000-0000-4000-8000-000000000001",
		ProjectID: proj.ID,
		Name:      "Production",
		Slug:      "production",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	if err := envRepo.Create(ctx, env); err != nil {
		t.Fatalf("seed environment: %v", err)
	}

	env2 := domain.Environment{
		ID:        "bb000000-0000-4000-8000-000000000002",
		ProjectID: proj.ID,
		Name:      "Staging",
		Slug:      "staging",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	if err := envRepo.Create(ctx, env2); err != nil {
		t.Fatalf("seed environment2: %v", err)
	}

	// makeKey builds a deterministic APIKey from a plaintext string.
	makeKey := func(id, name, plaintext string, tier domain.ToolCapabilityTier) *domain.APIKey {
		hash := sha256.Sum256([]byte(plaintext))
		return &domain.APIKey{
			ID:             id,
			ProjectID:      proj.ID,
			EnvironmentID:  env.ID,
			Name:           name,
			KeyHash:        hash,
			DisplayPrefix:  plaintext[:8],
			CapabilityTier: tier,
			CreatedAt:      time.Now().UTC().Truncate(time.Microsecond),
		}
	}

	t.Run("Create happy — stores all fields and GetByHash returns them", func(t *testing.T) {
		key := makeKey("key-0001", "my-key", "cg_testplaintext0001xxxxxxxxxxxx", domain.TierRead)
		if err := keyRepo.Create(ctx, key); err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := keyRepo.GetByHash(ctx, key.KeyHash)
		if err != nil {
			t.Fatalf("GetByHash: %v", err)
		}
		if got.ID != key.ID {
			t.Errorf("ID: got %q, want %q", got.ID, key.ID)
		}
		if got.ProjectID != key.ProjectID {
			t.Errorf("ProjectID: got %q, want %q", got.ProjectID, key.ProjectID)
		}
		if got.EnvironmentID != key.EnvironmentID {
			t.Errorf("EnvironmentID: got %q, want %q", got.EnvironmentID, key.EnvironmentID)
		}
		if got.Name != key.Name {
			t.Errorf("Name: got %q, want %q", got.Name, key.Name)
		}
		if got.DisplayPrefix != key.DisplayPrefix {
			t.Errorf("DisplayPrefix: got %q, want %q", got.DisplayPrefix, key.DisplayPrefix)
		}
		if got.CapabilityTier != key.CapabilityTier {
			t.Errorf("CapabilityTier: got %q, want %q", got.CapabilityTier, key.CapabilityTier)
		}
		if got.KeyHash != key.KeyHash {
			t.Errorf("KeyHash mismatch")
		}
		if got.RevokedAt != nil {
			t.Errorf("RevokedAt: expected nil, got %v", got.RevokedAt)
		}
	})

	t.Run("Create duplicate key_hash returns ErrConflict", func(t *testing.T) {
		key := makeKey("key-0002", "dup-key", "cg_testplaintext0002xxxxxxxxxxxx", domain.TierRead)
		if err := keyRepo.Create(ctx, key); err != nil {
			t.Fatalf("Create original: %v", err)
		}
		dup := &domain.APIKey{
			ID:             "key-0002-dup",
			ProjectID:      proj.ID,
			EnvironmentID:  env.ID,
			Name:           "different-name",
			KeyHash:        key.KeyHash, // same hash → unique violation
			DisplayPrefix:  key.DisplayPrefix,
			CapabilityTier: domain.TierRead,
			CreatedAt:      time.Now().UTC(),
		}
		err := keyRepo.Create(ctx, dup)
		if !errors.Is(err, domain.ErrConflict) {
			t.Errorf("expected ErrConflict, got %v", err)
		}
	})

	t.Run("GetByHash returns ErrNotFound for unknown hash", func(t *testing.T) {
		unknown := sha256.Sum256([]byte("cg_doesnotexistxxxxxxxxxxxxxxxxx"))
		_, err := keyRepo.GetByHash(ctx, unknown)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("GetByHash returns revoked key with RevokedAt set", func(t *testing.T) {
		key := makeKey("key-0003", "revoke-then-get", "cg_testplaintext0003xxxxxxxxxxxx", domain.TierRead)
		if err := keyRepo.Create(ctx, key); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := keyRepo.Revoke(ctx, key.ID); err != nil {
			t.Fatalf("Revoke: %v", err)
		}
		got, err := keyRepo.GetByHash(ctx, key.KeyHash)
		if err != nil {
			t.Fatalf("GetByHash after revoke: %v", err)
		}
		if got.RevokedAt == nil {
			t.Error("expected RevokedAt to be set after Revoke")
		}
	})

	t.Run("ListByEnvironment returns keys for correct (project, env) only", func(t *testing.T) {
		// Two keys for env, one for env2
		k1 := makeKey("key-list-01", "list-key-1", "cg_listplaintext001xxxxxxxxxxxxx", domain.TierRead)
		k1.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		k2 := makeKey("key-list-02", "list-key-2", "cg_listplaintext002xxxxxxxxxxxxx", domain.TierWrite)
		k2.CreatedAt = time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
		k3 := &domain.APIKey{
			ID:             "key-list-03",
			ProjectID:      proj.ID,
			EnvironmentID:  env2.ID,
			Name:           "list-key-3",
			KeyHash:        sha256.Sum256([]byte("cg_listplaintext003xxxxxxxxxxxxx")),
			DisplayPrefix:  "cg_listp",
			CapabilityTier: domain.TierRead,
			CreatedAt:      time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
		}
		for _, k := range []*domain.APIKey{k1, k2, k3} {
			if err := keyRepo.Create(ctx, k); err != nil {
				t.Fatalf("Create %s: %v", k.ID, err)
			}
		}

		list, err := keyRepo.ListByEnvironment(ctx, proj.ID, env.ID)
		if err != nil {
			t.Fatalf("ListByEnvironment: %v", err)
		}
		// Count only our two known IDs (there may be other keys from earlier subtests)
		found := map[string]bool{}
		for _, k := range list {
			if k.ID == k1.ID || k.ID == k2.ID || k.ID == k3.ID {
				found[k.ID] = true
			}
			if k.EnvironmentID != env.ID {
				t.Errorf("got key from wrong environment: %s", k.EnvironmentID)
			}
		}
		if !found[k1.ID] {
			t.Errorf("key-list-01 missing from result")
		}
		if !found[k2.ID] {
			t.Errorf("key-list-02 missing from result")
		}
		if found[k3.ID] {
			t.Errorf("key-list-03 (env2) should not appear in env listing")
		}
	})

	t.Run("ListByEnvironment returns empty slice never nil when no keys exist", func(t *testing.T) {
		// Use a non-existent project/env pair (no FK needed — the WHERE simply finds nothing)
		list, err := keyRepo.ListByEnvironment(ctx, "00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("ListByEnvironment: %v", err)
		}
		if list == nil {
			t.Error("expected non-nil empty slice, got nil")
		}
		if len(list) != 0 {
			t.Errorf("expected 0 keys, got %d", len(list))
		}
	})

	t.Run("ListByEnvironment includes revoked keys", func(t *testing.T) {
		k4 := makeKey("key-list-04", "list-revoked", "cg_listplaintext004xxxxxxxxxxxxx", domain.TierRead)
		if err := keyRepo.Create(ctx, k4); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := keyRepo.Revoke(ctx, k4.ID); err != nil {
			t.Fatalf("Revoke: %v", err)
		}
		list, err := keyRepo.ListByEnvironment(ctx, proj.ID, env.ID)
		if err != nil {
			t.Fatalf("ListByEnvironment: %v", err)
		}
		var found *domain.APIKey
		for _, k := range list {
			if k.ID == k4.ID {
				found = k
				break
			}
		}
		if found == nil {
			t.Fatal("revoked key-list-04 not found in ListByEnvironment")
		}
		if found.RevokedAt == nil {
			t.Error("revoked key should have RevokedAt set")
		}
	})

	t.Run("Revoke sets revoked_at and returns nil", func(t *testing.T) {
		key := makeKey("key-rev-01", "to-revoke", "cg_revokeplaintext01xxxxxxxxxxxx", domain.TierRead)
		if err := keyRepo.Create(ctx, key); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := keyRepo.Revoke(ctx, key.ID); err != nil {
			t.Fatalf("Revoke: %v", err)
		}
		got, err := keyRepo.GetByHash(ctx, key.KeyHash)
		if err != nil {
			t.Fatalf("GetByHash: %v", err)
		}
		if got.RevokedAt == nil {
			t.Error("expected RevokedAt to be non-nil after Revoke")
		}
	})

	t.Run("Revoke returns ErrNotFound for unknown ID", func(t *testing.T) {
		err := keyRepo.Revoke(ctx, "00000000-0000-0000-0000-000000000000")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("Revoke second call on already-revoked key returns ErrNotFound", func(t *testing.T) {
		key := makeKey("key-rev-02", "double-revoke", "cg_revokeplaintext02xxxxxxxxxxxx", domain.TierRead)
		if err := keyRepo.Create(ctx, key); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := keyRepo.Revoke(ctx, key.ID); err != nil {
			t.Fatalf("first Revoke: %v", err)
		}
		err := keyRepo.Revoke(ctx, key.ID)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound on second Revoke, got %v", err)
		}
	})

	t.Run("capability_tier round-trip — all three tier values", func(t *testing.T) {
		tiers := []struct {
			id        string
			plaintext string
			tier      domain.ToolCapabilityTier
		}{
			{"key-tier-read", "cg_tierroundtripread0xxxxxxxxxxx", domain.TierRead},
			{"key-tier-write", "cg_tierroundtripwrite0xxxxxxxxxx", domain.TierWrite},
			{"key-tier-dest", "cg_tierroundtripdest0xxxxxxxxxxx", domain.TierDestructive},
		}
		for _, tc := range tiers {
			key := makeKey(tc.id, "tier-key-"+string(tc.tier), tc.plaintext, tc.tier)
			if err := keyRepo.Create(ctx, key); err != nil {
				t.Fatalf("Create tier %s: %v", tc.tier, err)
			}
			got, err := keyRepo.GetByHash(ctx, key.KeyHash)
			if err != nil {
				t.Fatalf("GetByHash tier %s: %v", tc.tier, err)
			}
			if got.CapabilityTier != tc.tier {
				t.Errorf("tier %s: got %q, want %q", tc.tier, got.CapabilityTier, tc.tier)
			}
		}
	})

	t.Run("capability_tier defaults to read when column default applies", func(t *testing.T) {
		// Insert directly via SQL omitting capability_tier — validates the DEFAULT 'read' clause.
		defaultHash := sha256.Sum256([]byte("cg_defaulttiertestxxxxxxxxxxxxxxxxx"))
		_, err := db.ExecContext(ctx,
			`INSERT INTO api_keys (id, project_id, environment_id, name, key_hash, display_prefix, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			"key-tier-default", proj.ID, env.ID, "default-tier-key", defaultHash[:], "cg_defau", time.Now().UTC(),
		)
		if err != nil {
			t.Fatalf("raw INSERT without capability_tier: %v", err)
		}
		got, err := keyRepo.GetByHash(ctx, defaultHash)
		if err != nil {
			t.Fatalf("GetByHash: %v", err)
		}
		if got.CapabilityTier != domain.TierRead {
			t.Errorf("expected default tier %q, got %q", domain.TierRead, got.CapabilityTier)
		}
	})
}
