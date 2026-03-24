# Codebase Index

_Generated 2026-03-24 00:21 UTC. Read this at session start for orientation._

## Packages
```
cmd/migrate
cmd/server
db
github.com/karo/cuttlegate
internal/adapters/db
internal/adapters/http
internal/adapters/mcp
internal/app
internal/domain
internal/domain/ports
sdk/js/node_modules/flatted/golang/pkg/flatted
```

## Domain ports — interfaces (internal/domain/ports/)
```
type APIKeyRepository interface {
type AuditRepository interface {
type DomainEvent interface {
type EnvironmentRepository interface {
type EvaluationEventPublisher interface {
type EvaluationEventRepository interface {
type EventPublisher interface {
type FlagEnvironmentStateRepository interface {
type FlagEvaluationStatsRepository interface {
type FlagRepository interface {
type ProjectMemberRepository interface {
type ProjectRepository interface {
type RuleRepository interface {
type SegmentRepository interface {
type TokenVerifier interface {
type UnitOfWorkFactory interface {
type UnitOfWork interface {
type UserRepository interface {
```

## Domain types — structs (internal/domain/)
```
type APIKey struct {
type AuditEvent struct {
type AuditFilter struct {
type AuthContext struct {
type Condition struct {
type Environment struct {
type EvalContext struct {
type EvalResult struct {
type EvaluationBucket struct {
type EvaluationEvent struct {
type FlagEnvironmentState struct {
type FlagEvaluationStats struct {
type FlagStateChangedEvent struct {
type Flag struct {
type ProjectMember struct {
type Project struct {
type Rule struct {
type Segment struct {
type User struct {
type ValidationError struct {
type Variant struct {
```

## App services (internal/app/)
```
type APIKeyCreateResult struct {
type APIKeyService struct {
type APIKeyView struct {
type AuditService struct {
type BucketView struct {
type EnvironmentService struct {
type EvaluationAuditService struct {
type EvaluationBucketsView struct {
type EvaluationEventView struct {
type EvaluationService struct {
type EvaluationStatsService struct {
type EvalView struct {
type FlagEnvironmentView struct {
type FlagPromotionDiff struct {
type FlagService struct {
type FlagStatsView struct {
type ProjectMemberService struct {
type ProjectMemberView struct {
type ProjectService struct {
type PromotionService struct {
type RuleService struct {
type SegmentService struct {
type SetEnabledParams struct {
```

## HTTP handlers & middleware (internal/adapters/http/)
```
type APIKeyHandler struct {
type APIKeyScope struct {
type AuditHandler struct {
type Broker struct {
type EnvironmentHandler struct {
type EvaluationAuditHandler struct {
type EvaluationHandler struct {
type EvaluationStatsHandler struct {
type FlagEnvironmentHandler struct {
type FlagHandler struct {
type FlagVariantHandler struct {
type OIDCVerifier struct {
type ProjectHandler struct {
type ProjectMemberHandler struct {
type PromotionHandler struct {
type RateLimiter struct {
type RuleHandler struct {
type SegmentHandler struct {
type SSEHandler struct {
```

## DB adapters (internal/adapters/db/)
```
type PostgresAPIKeyRepository struct {
type PostgresAuditRepository struct {
type PostgresEnvironmentRepository struct {
type PostgresEvaluationEventRepository struct {
type PostgresFlagEnvironmentStateRepository struct {
type PostgresFlagEvaluationStatsRepository struct {
type PostgresFlagRepository struct {
type PostgresProjectMemberRepository struct {
type PostgresProjectRepository struct {
type PostgresRuleRepository struct {
type PostgresSegmentRepository struct {
type PostgresUnitOfWorkFactory struct {
type PostgresUnitOfWork struct {
type PostgresUserRepository struct {
```
