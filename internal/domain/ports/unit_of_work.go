package ports

import "context"

// UnitOfWork represents a transactional scope that provides
// repository access within a single atomic operation.
type UnitOfWork interface {
	FlagEnvironmentStateRepository() FlagEnvironmentStateRepository
	RuleRepository() RuleRepository
	AuditRepository() AuditRepository
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// UnitOfWorkFactory begins a new transactional scope.
type UnitOfWorkFactory interface {
	Begin(ctx context.Context) (UnitOfWork, error)
}
