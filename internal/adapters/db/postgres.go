package dbadapter

import "github.com/lib/pq"

// pgUniqueViolation is the PostgreSQL error code for unique constraint violations.
const pgUniqueViolation = pq.ErrorCode("23505") //nolint:staticcheck
