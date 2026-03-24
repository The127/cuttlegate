//go:build integration

package dbadapter_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/lib/pq"

	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
)

// newAppRoleDB creates a non-superuser "cg_app" role and returns a *sql.DB
// connected as that role. Superusers bypass RLS, so we need a regular role.
func newAppRoleDB(t *testing.T, superDB *sql.DB, superDSN string) *sql.DB {
	t.Helper()
	ctx := context.Background()

	for _, stmt := range []string{
		`DO $$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'cg_app') THEN CREATE ROLE cg_app LOGIN PASSWORD 'cg_app'; END IF; END $$`,
		`GRANT ALL ON ALL TABLES IN SCHEMA public TO cg_app`,
		`GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO cg_app`,
	} {
		if _, err := superDB.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("setup app role: %v", err)
		}
	}

	appDSN := strings.Replace(superDSN, "cuttlegate:cuttlegate", "cg_app:cg_app", 1)
	appDB, err := sql.Open("postgres", appDSN)
	if err != nil {
		t.Fatalf("open app role db: %v", err)
	}
	t.Cleanup(func() { appDB.Close() })
	return appDB
}

func TestTenantRLS_CorrectProjectIDReturnsFlags(t *testing.T) {
	superDB, dsn := newTestDBWithDSN(t)
	ctx := context.Background()
	appDB := newAppRoleDB(t, superDB, dsn)

	// Seed data as superuser (bypasses RLS).
	superDB.ExecContext(ctx, `INSERT INTO projects (id, name, slug, created_at) VALUES ('proj-aaa', 'A', 'a', NOW())`)
	superDB.ExecContext(ctx, `INSERT INTO projects (id, name, slug, created_at) VALUES ('proj-bbb', 'B', 'b', NOW())`)
	superDB.ExecContext(ctx,
		`INSERT INTO flags (id, project_id, key, name, type, variants, default_variant_key, created_at)
		 VALUES ('f-a1', 'proj-aaa', 'flag-a', 'A', 'bool', '[{"key":"true","name":"On"}]', 'true', NOW())`)
	superDB.ExecContext(ctx,
		`INSERT INTO flags (id, project_id, key, name, type, variants, default_variant_key, created_at)
		 VALUES ('f-b1', 'proj-bbb', 'flag-b', 'B', 'bool', '[{"key":"true","name":"On"}]', 'true', NOW())`)

	// Query as app role with correct project_id.
	tx, err := appDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	if err := dbadapter.SetTenantContext(ctx, tx, "proj-aaa"); err != nil {
		t.Fatal(err)
	}

	var keys []string
	rows, err := tx.QueryContext(ctx, `SELECT key FROM flags`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		rows.Scan(&k)
		keys = append(keys, k)
	}

	if len(keys) != 1 || keys[0] != "flag-a" {
		t.Errorf("expected [flag-a], got %v", keys)
	}
}

func TestTenantRLS_WrongProjectIDReturnsZeroRows(t *testing.T) {
	superDB, dsn := newTestDBWithDSN(t)
	ctx := context.Background()
	appDB := newAppRoleDB(t, superDB, dsn)

	superDB.ExecContext(ctx, `INSERT INTO projects (id, name, slug, created_at) VALUES ('proj-ccc', 'C', 'c', NOW())`)
	superDB.ExecContext(ctx,
		`INSERT INTO flags (id, project_id, key, name, type, variants, default_variant_key, created_at)
		 VALUES ('f-c1', 'proj-ccc', 'flag-c', 'C', 'bool', '[{"key":"true","name":"On"}]', 'true', NOW())`)

	tx, err := appDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	if err := dbadapter.SetTenantContext(ctx, tx, "proj-wrong"); err != nil {
		t.Fatal(err)
	}

	var count int
	tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM flags`).Scan(&count)

	if count != 0 {
		t.Errorf("expected 0 flags for wrong project_id, got %d", count)
	}
}

func TestTenantRLS_OmittingSetTenantContextReturnsZeroRows(t *testing.T) {
	superDB, dsn := newTestDBWithDSN(t)
	ctx := context.Background()
	appDB := newAppRoleDB(t, superDB, dsn)

	superDB.ExecContext(ctx, `INSERT INTO projects (id, name, slug, created_at) VALUES ('proj-eee', 'E', 'e', NOW())`)
	superDB.ExecContext(ctx,
		`INSERT INTO flags (id, project_id, key, name, type, variants, default_variant_key, created_at)
		 VALUES ('f-e1', 'proj-eee', 'flag-e', 'E', 'bool', '[{"key":"true","name":"On"}]', 'true', NOW())`)

	// No SetTenantContext — RLS default-deny should return 0 rows.
	tx, err := appDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	var count int
	tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM flags`).Scan(&count)

	if count != 0 {
		t.Errorf("expected 0 flags without SetTenantContext (RLS default-deny), got %d", count)
	}
}

func TestTenantRLS_EnvironmentsIsolated(t *testing.T) {
	superDB, dsn := newTestDBWithDSN(t)
	ctx := context.Background()
	appDB := newAppRoleDB(t, superDB, dsn)

	superDB.ExecContext(ctx, `INSERT INTO projects (id, name, slug, created_at) VALUES ('proj-g1', 'G1', 'g1', NOW())`)
	superDB.ExecContext(ctx, `INSERT INTO projects (id, name, slug, created_at) VALUES ('proj-g2', 'G2', 'g2', NOW())`)
	superDB.ExecContext(ctx, `INSERT INTO environments (id, project_id, name, slug, created_at) VALUES ('env-g1', 'proj-g1', 'Prod', 'prod', NOW())`)
	superDB.ExecContext(ctx, `INSERT INTO environments (id, project_id, name, slug, created_at) VALUES ('env-g2', 'proj-g2', 'Prod', 'prod', NOW())`)

	tx, err := appDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	if err := dbadapter.SetTenantContext(ctx, tx, "proj-g1"); err != nil {
		t.Fatal(err)
	}

	var count int
	tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM environments`).Scan(&count)

	if count != 1 {
		t.Errorf("expected 1 environment for proj-g1, got %d", count)
	}
}
