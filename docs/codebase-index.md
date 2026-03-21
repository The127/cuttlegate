# Codebase Index

_Generated 2026-03-21 02:44 UTC. Read this at session start for orientation._

## Packages
```
cmd/migrate
cmd/server
db
github.com/karo/cuttlegate
internal/adapters/db
internal/adapters/http
internal/app
internal/domain
internal/domain/ports
```

## Domain ports — interfaces (internal/domain/ports/)
```
type DomainEvent interface {
type EnvironmentRepository interface {
type EventPublisher interface {
type FlagEnvironmentStateRepository interface {
type FlagRepository interface {
type ProjectMemberRepository interface {
type ProjectRepository interface {
type RuleRepository interface {
type TokenVerifier interface {
```

## Domain types — structs (internal/domain/)
```
type AuthContext struct {
type Condition struct {
type Environment struct {
type EvalContext struct {
type EvalResult struct {
type FlagEnvironmentState struct {
type Flag struct {
type ProjectMember struct {
type Project struct {
type Rule struct {
type User struct {
type ValidationError struct {
type Variant struct {
```

## App services (internal/app/)
```
type EnvironmentService struct {
type EvaluationService struct {
type EvalView struct {
type FlagEnvironmentView struct {
type FlagService struct {
type ProjectMemberService struct {
type ProjectService struct {
type RuleService struct {
```

## HTTP handlers & middleware (internal/adapters/http/)
```
type EnvironmentHandler struct {
type EvaluationHandler struct {
type FlagEnvironmentHandler struct {
type FlagHandler struct {
type FlagVariantHandler struct {
type OIDCVerifier struct {
type ProjectHandler struct {
type ProjectMemberHandler struct {
type RuleHandler struct {
```

## DB adapters (internal/adapters/db/)
```
type NoOpRuleRepository struct{}
type PostgresEnvironmentRepository struct {
type PostgresFlagEnvironmentStateRepository struct {
type PostgresFlagRepository struct {
type PostgresProjectMemberRepository struct {
type PostgresProjectRepository struct {
type PostgresRuleRepository struct {
```
