# Benchmarks

## Flag Pagination at 10k Flags

**Test:** `TestFlagPaginationAt10k` (`internal/adapters/db/flag_pagination_benchmark_test.go`)

**Setup:** 10,000 boolean flags in a single project, Postgres 16 (testcontainers), 20 iterations per query after 1 warm-up.

**Threshold:** p95 < 200ms

| Query | p50 | p95 | p99 |
|---|---|---|---|
| First page (50 rows, default sort) | 3.4ms | 4.2ms | 4.2ms |
| Last page (page 200, default sort) | 4.8ms | 5.6ms | 5.6ms |
| Search substring (`flag-099`) | 15.1ms | 17.2ms | 17.2ms |
| Sort by name desc | 4.4ms | 5.7ms | 5.7ms |
| Search + sort (`flag-05`, key asc) | 13.8ms | 14.9ms | 14.9ms |

**Seed time:** ~2.9s for 10k flags (single-row inserts, no batching).

**Result:** All queries pass — no optimization needed. Search queries are ~3x slower due to `ILIKE` scans but still well within threshold.

_Run: 2026-03-24, Postgres 16-alpine, Linux 6.19.8 / Fedora 43_
