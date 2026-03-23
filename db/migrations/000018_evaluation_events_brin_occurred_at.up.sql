-- BRIN index on occurred_at supports the retention DELETE pattern:
-- DELETE FROM evaluation_events WHERE occurred_at < $1
-- BRIN is appropriate for this append-only time-series table — the physical
-- ordering of rows correlates with occurred_at, making BRIN highly selective
-- at a fraction of the storage cost of a B-tree index.
CREATE INDEX evaluation_events_occurred_at_brin
    ON evaluation_events USING BRIN (occurred_at);
