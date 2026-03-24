// seed-flags inserts N flags into a project for load testing.
//
// Usage:
//
//	go run ./cmd/seed-flags -dsn "$DATABASE_URL" -project-id <uuid> -count 10000
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	dsn := flag.String("dsn", "", "Postgres connection string")
	projectID := flag.String("project-id", "", "Project UUID to seed flags into")
	count := flag.Int("count", 10000, "Number of flags to create")
	flag.Parse()

	if *dsn == "" || *projectID == "" {
		log.Fatal("usage: seed-flags -dsn <dsn> -project-id <uuid> [-count 10000]")
	}

	db, err := sql.Open("postgres", *dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	start := time.Now()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO flags (id, project_id, key, name, type, variants, default_variant_key, created_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, 'bool',
		         '[{"key":"true","name":"On"},{"key":"false","name":"Off"}]'::jsonb,
		         'false', NOW())
		 ON CONFLICT DO NOTHING`)
	if err != nil {
		log.Fatalf("prepare: %v", err)
	}
	defer stmt.Close()

	for i := 0; i < *count; i++ {
		key := fmt.Sprintf("flag-%05d", i)
		name := fmt.Sprintf("Flag %d", i)
		if _, err := stmt.ExecContext(ctx, *projectID, key, name); err != nil {
			tx.Rollback()
			log.Fatalf("insert flag %d: %v", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("commit: %v", err)
	}

	elapsed := time.Since(start)
	fmt.Printf("Seeded %d flags in %s (%.0f flags/sec)\n", *count, elapsed, float64(*count)/elapsed.Seconds())
}
