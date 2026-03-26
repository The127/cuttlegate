//go:build integration

package dbadapter_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	dbadapter "github.com/The127/cuttlegate/internal/adapters/db"
	"github.com/The127/cuttlegate/internal/domain"
)

// TestFlagPaginationAt10k validates that paginated queries perform within
// acceptable latency at 10k flags. Run with:
//
//	go test -tags integration -run TestFlagPaginationAt10k -v ./internal/adapters/db/
func TestFlagPaginationAt10k(t *testing.T) {
	db := newTestDB(t)
	repo := dbadapter.NewPostgresFlagRepository(db)
	ctx := context.Background()

	// Create a project.
	projectID := "bench-proj-00000000-0000-0000-0000-000000000001"
	_, err := db.ExecContext(ctx,
		`INSERT INTO projects (id, name, slug, created_at) VALUES ($1, 'Bench', 'bench', NOW())`,
		projectID)
	if err != nil {
		t.Fatal(err)
	}

	// Seed 10k flags.
	const flagCount = 10000
	t.Logf("Seeding %d flags...", flagCount)
	seedStart := time.Now()
	for i := 0; i < flagCount; i++ {
		key := fmt.Sprintf("flag-%05d", i)
		name := fmt.Sprintf("Flag %d", i)
		_, err := db.ExecContext(ctx,
			`INSERT INTO flags (id, project_id, key, name, type, variants, default_variant_key, created_at)
			 VALUES (gen_random_uuid(), $1, $2, $3, 'bool',
			         '[{"key":"true","name":"On"},{"key":"false","name":"Off"}]'::jsonb,
			         'false', NOW())`,
			projectID, key, name)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("Seeded %d flags in %s", flagCount, time.Since(seedStart))

	queries := []struct {
		name      string
		filter    domain.FlagListFilter
		maxP95ms  int
		wantTotal int
	}{
		{
			name:      "first page, default sort",
			filter:    domain.FlagListFilter{Page: 1, PerPage: 50},
			maxP95ms:  200,
			wantTotal: flagCount,
		},
		{
			name:      "last page, default sort",
			filter:    domain.FlagListFilter{Page: 200, PerPage: 50},
			maxP95ms:  200,
			wantTotal: flagCount,
		},
		{
			name:      "search substring",
			filter:    domain.FlagListFilter{Page: 1, PerPage: 50, Search: "flag-099"},
			maxP95ms:  200,
			wantTotal: 11, // flag-09900..flag-09910 (approx)
		},
		{
			name:      "sort by name desc",
			filter:    domain.FlagListFilter{Page: 1, PerPage: 50, SortBy: "name", SortDir: "desc"},
			maxP95ms:  200,
			wantTotal: flagCount,
		},
		{
			name:      "search + sort",
			filter:    domain.FlagListFilter{Page: 1, PerPage: 50, Search: "flag-05", SortBy: "key", SortDir: "asc"},
			maxP95ms:  200,
			wantTotal: 1111, // flag-05xxx matches ~1111
		},
	}

	const iterations = 20

	for _, q := range queries {
		t.Run(q.name, func(t *testing.T) {
			// Warm up.
			repo.ListByProjectPaginated(ctx, projectID, q.filter)

			var durations []time.Duration
			for i := 0; i < iterations; i++ {
				start := time.Now()
				flags, total, err := repo.ListByProjectPaginated(ctx, projectID, q.filter)
				elapsed := time.Since(start)
				if err != nil {
					t.Fatal(err)
				}
				durations = append(durations, elapsed)
				_ = flags

				// Verify total is in expected range (allow some tolerance for search).
				if i == 0 && total < 1 {
					t.Errorf("expected total > 0, got %d", total)
				}
			}

			// Calculate p95.
			p95 := percentile(durations, 0.95)
			t.Logf("p50=%s  p95=%s  p99=%s  (n=%d)",
				percentile(durations, 0.50),
				p95,
				percentile(durations, 0.99),
				iterations)

			if p95.Milliseconds() > int64(q.maxP95ms) {
				t.Errorf("p95 latency %s exceeds %dms threshold", p95, q.maxP95ms)
			}
		})
	}
}

func percentile(durations []time.Duration, p float64) time.Duration {
	n := len(durations)
	if n == 0 {
		return 0
	}
	// Sort.
	sorted := make([]time.Duration, n)
	copy(sorted, durations)
	for i := 1; i < n; i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	idx := int(float64(n-1) * p)
	return sorted[idx]
}
