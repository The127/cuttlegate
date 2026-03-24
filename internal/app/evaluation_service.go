package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// EvalView is the result of evaluating a flag for a given user context.
type EvalView struct {
	Key      string
	Enabled  bool
	Value    *string // nil for bool flags; deprecated — prefer ValueKey
	ValueKey string  // always present; "true"/"false" for bool flags, variant key for all others
	Reason   domain.EvalReason
	Type     domain.FlagType
}

// EvaluationService orchestrates flag evaluation use cases.
type EvaluationService struct {
	flagRepo    ports.FlagRepository
	stateRepo   ports.FlagEnvironmentStateRepository
	ruleRepo    ports.RuleRepository
	segmentRepo ports.SegmentRepository
	publisher   ports.EvaluationEventPublisher      // nil = no-op
	statsRepo   ports.FlagEvaluationStatsRepository // nil = no-op
	auditRepo   ports.AuditRepository               // nil = no-op
}

// NewEvaluationService constructs an EvaluationService.
func NewEvaluationService(
	flagRepo ports.FlagRepository,
	stateRepo ports.FlagEnvironmentStateRepository,
	ruleRepo ports.RuleRepository,
	segmentRepo ports.SegmentRepository,
	publisher ports.EvaluationEventPublisher,
) *EvaluationService {
	return &EvaluationService{
		flagRepo:    flagRepo,
		stateRepo:   stateRepo,
		ruleRepo:    ruleRepo,
		segmentRepo: segmentRepo,
		publisher:   publisher,
	}
}

// WithStatsRepo attaches a FlagEvaluationStatsRepository so that evaluation
// stats are upserted on each evaluation. The method returns the service to
// allow chaining at construction time.
func (s *EvaluationService) WithStatsRepo(statsRepo ports.FlagEvaluationStatsRepository) *EvaluationService {
	s.statsRepo = statsRepo
	return s
}

// WithAuditRepo attaches an AuditRepository so that an audit event is recorded
// whenever evaluation skips a deleted segment reference. The method returns the
// service to allow chaining at construction time.
func (s *EvaluationService) WithAuditRepo(auditRepo ports.AuditRepository) *EvaluationService {
	s.auditRepo = auditRepo
	return s
}

// publishEvent fires a best-effort evaluation event and stats upsert in a
// goroutine. Errors are logged but never returned — the eval response path must
// not be affected by publish or upsert failures.
func (s *EvaluationService) publishEvent(projectID, environmentID string, flag *domain.Flag, evalCtx domain.EvalContext, result domain.EvalResult, id string, now time.Time) {
	hasWork := s.publisher != nil || s.statsRepo != nil
	if !hasWork {
		return
	}
	// Serialise input context attributes in the calling goroutine (map is safe to read here).
	ctxJSON, err := json.Marshal(evalCtx.Attributes)
	if err != nil {
		ctxJSON = []byte("{}")
	}

	pub := s.publisher
	stats := s.statsRepo
	flagID := flag.ID
	flagKey := flag.Key

	var event *domain.EvaluationEvent
	if pub != nil {
		event = &domain.EvaluationEvent{
			ID:              id,
			FlagKey:         flagKey,
			ProjectID:       projectID,
			EnvironmentID:   environmentID,
			UserID:          evalCtx.UserID,
			InputContext:    string(ctxJSON),
			MatchedRuleID:   result.MatchedRuleID,
			MatchedRuleName: result.MatchedRuleName,
			VariantKey:      result.VariantKey,
			Reason:          result.Reason,
			OccurredAt:      now,
		}
	}

	go func() {
		ctx := context.Background()
		if pub != nil {
			if err := pub.Publish(ctx, event); err != nil {
				slog.Error("evaluation event publish failed", "flag_key", flagKey, "err", err)
			}
		}
		if stats != nil {
			if err := stats.Upsert(ctx, flagID, environmentID, flagKey, now); err != nil {
				slog.Error("evaluation stats upsert failed", "flag_key", flagKey, "err", err)
			}
		}
	}()
}

// Evaluate evaluates a flag for the given user context. Requires at least viewer role.
// Returns ErrNotFound if the flag key does not exist in the project.
func (s *EvaluationService) Evaluate(ctx context.Context, projectID, environmentID, flagKey string, evalCtx domain.EvalContext) (*EvalView, error) {
	if _, err := requireRole(ctx, domain.RoleViewer); err != nil {
		return nil, err
	}

	flag, err := s.flagRepo.GetByKey(ctx, projectID, flagKey)
	if err != nil {
		return nil, err
	}

	state, err := s.stateRepo.GetByFlagAndEnvironment(ctx, flag.ID, environmentID)
	if err != nil {
		return nil, err
	}

	rules, err := s.ruleRepo.ListByFlagEnvironment(ctx, flag.ID, environmentID)
	if err != nil {
		return nil, err
	}

	// Pre-load segment membership for any in_segment / not_in_segment conditions.
	// Extract distinct segment slugs referenced by the rules, then check membership
	// once per slug using evalCtx.UserID as the user key.
	userSegments, err := s.resolveSegments(ctx, projectID, flag.ID, environmentID, rules, evalCtx.UserID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	result := domain.Evaluate(flag, state, rules, evalCtx, userSegments)

	if id, err := newUUID(); err == nil {
		s.publishEvent(projectID, environmentID, flag, evalCtx, result, id, now)
	}

	view := &EvalView{
		Key:      flag.Key,
		Enabled:  result.Reason != domain.ReasonDisabled,
		ValueKey: result.VariantKey,
		Reason:   result.Reason,
		Type:     flag.Type,
	}
	if flag.Type != domain.FlagTypeBool {
		view.Value = &result.VariantKey
	}
	return view, nil
}

// EvaluateAll evaluates all flags in a project+environment for the given user
// context. Returns the list of evaluation results and the timestamp at which
// evaluation was performed. Requires at least viewer role.
func (s *EvaluationService) EvaluateAll(ctx context.Context, projectID, environmentID string, evalCtx domain.EvalContext) ([]EvalView, time.Time, error) {
	if _, err := requireRole(ctx, domain.RoleViewer); err != nil {
		return nil, time.Time{}, err
	}

	evaluatedAt := time.Now().UTC()

	flags, err := s.flagRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, time.Time{}, err
	}

	// Batch-load all states for this environment to avoid N+1 queries.
	allStates, err := s.stateRepo.ListByEnvironment(ctx, environmentID)
	if err != nil {
		return nil, time.Time{}, err
	}
	stateByFlag := make(map[string]*domain.FlagEnvironmentState, len(allStates))
	for _, st := range allStates {
		stateByFlag[st.FlagID] = st
	}

	// Batch-load all rules for this environment to avoid N+1 queries.
	allRules, err := s.ruleRepo.ListByEnvironment(ctx, environmentID)
	if err != nil {
		return nil, time.Time{}, err
	}
	rulesByFlag := make(map[string][]*domain.Rule, len(flags))
	for _, r := range allRules {
		rulesByFlag[r.FlagID] = append(rulesByFlag[r.FlagID], r)
	}

	views := make([]EvalView, 0, len(flags))
	for _, flag := range flags {
		state := stateByFlag[flag.ID] // nil if no state row — treated as disabled
		rules := rulesByFlag[flag.ID] // nil if no rules — treated as empty slice by Evaluate

		userSegments, err := s.resolveSegments(ctx, projectID, flag.ID, environmentID, rules, evalCtx.UserID)
		if err != nil {
			return nil, time.Time{}, err
		}

		result := domain.Evaluate(flag, state, rules, evalCtx, userSegments)

		if id, err := newUUID(); err == nil {
			s.publishEvent(projectID, environmentID, flag, evalCtx, result, id, evaluatedAt)
		}

		view := EvalView{
			Key:      flag.Key,
			Enabled:  result.Reason != domain.ReasonDisabled,
			ValueKey: result.VariantKey,
			Reason:   result.Reason,
			Type:     flag.Type,
		}
		if flag.Type != domain.FlagTypeBool {
			view.Value = &result.VariantKey
		}
		views = append(views, view)
	}

	return views, evaluatedAt, nil
}

// resolveSegments extracts distinct segment slugs referenced by in_segment /
// not_in_segment conditions in rules, resolves each slug to an ID, and returns
// the set of segment slugs the given userKey belongs to.
// Segments that no longer exist are skipped (treated as non-membership) and an
// audit event is recorded if an AuditRepository is configured.
// An empty userKey (unauthenticated caller) always resolves to an empty set —
// IsMember returns false for "", so in_segment conditions always miss.
func (s *EvaluationService) resolveSegments(ctx context.Context, projectID, flagID, environmentID string, rules []*domain.Rule, userKey string) (domain.Set, error) {
	// Collect distinct segment slugs from rule conditions.
	slugs := domain.NewSet()
	for _, rule := range rules {
		for _, c := range rule.Conditions {
			if c.Operator == domain.OperatorInSegment || c.Operator == domain.OperatorNotInSegment {
				if len(c.Values) > 0 {
					slugs.Add(c.Values[0])
				}
			}
		}
	}
	if len(slugs) == 0 {
		return nil, nil
	}

	userSegments := domain.NewSet()
	for slug := range slugs {
		seg, err := s.segmentRepo.GetBySlug(ctx, projectID, slug)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				// Segment was deleted — treat user as non-member and record an audit event
				// so the silent privilege change is visible in the audit log.
				s.recordDeletedSegmentAudit(ctx, flagID, environmentID, slug)
				continue
			}
			return nil, err
		}
		member, err := s.segmentRepo.IsMember(ctx, seg.ID, userKey)
		if err != nil {
			return nil, err
		}
		if member {
			userSegments.Add(slug)
		}
	}
	return userSegments, nil
}

// recordDeletedSegmentAudit emits a best-effort audit event when a rule references
// a segment that no longer exists. Errors are logged but never propagated —
// the evaluation result must not be affected by audit recording failures.
func (s *EvaluationService) recordDeletedSegmentAudit(ctx context.Context, flagID, environmentID, segmentSlug string) {
	if s.auditRepo == nil {
		return
	}
	payload := segmentNotFoundPayload{
		FlagID:        flagID,
		EnvironmentID: environmentID,
		SegmentSlug:   segmentSlug,
		Reason:        "segment_not_found",
	}
	afterState, err := json.Marshal(payload)
	if err != nil {
		slog.Error("deleted segment audit: failed to marshal payload", "segment_slug", segmentSlug, "err", err)
		return
	}
	event := &domain.AuditEvent{
		Action:     "segment_not_found",
		EntityType: "segment",
		EntityKey:  segmentSlug,
		Source:     domain.SourceSystem,
		AfterState: string(afterState),
	}
	if err := s.auditRepo.Record(ctx, event); err != nil {
		slog.Error("deleted segment audit: failed to record event", "segment_slug", segmentSlug, "err", err)
	}
}

// segmentNotFoundPayload is the structured payload stored in AuditEvent.AfterState
// when a rule references a segment that no longer exists at evaluation time.
type segmentNotFoundPayload struct {
	FlagID        string `json:"flag_id"`
	EnvironmentID string `json:"environment_id"`
	SegmentSlug   string `json:"segment_slug"`
	Reason        string `json:"reason"`
}
