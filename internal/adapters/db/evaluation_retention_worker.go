package dbadapter

import (
	"context"
	"log/slog"
	"time"
)

// StartEvaluationRetentionWorker runs a background goroutine that deletes
// evaluation_events older than retentionDays on each tick of interval.
// The goroutine exits when ctx is cancelled.
//
// Call this once from main.go after the repository is constructed.
// The caller is responsible for ensuring the context is cancelled on shutdown.
func StartEvaluationRetentionWorker(ctx context.Context, repo *PostgresEvaluationEventRepository, retentionDays int, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
				if err := repo.DeleteOlderThan(ctx, cutoff); err != nil {
					slog.Error("evaluation retention cleanup failed", "err", err)
				}
			}
		}
	}()
}
