package dbadapter

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/The127/cuttlegate/internal/domain"
	"github.com/The127/cuttlegate/internal/domain/ports"
)

// PostgresRuleRepository implements ports.RuleRepository using PostgreSQL.
type PostgresRuleRepository struct {
	db DBTX
}

// NewPostgresRuleRepository constructs a PostgresRuleRepository.
func NewPostgresRuleRepository(db DBTX) *PostgresRuleRepository {
	return &PostgresRuleRepository{db: db}
}

var _ ports.RuleRepository = (*PostgresRuleRepository)(nil)

// dbCondition is the JSON-serialisable form of domain.Condition stored in the conditions JSONB column.
type dbCondition struct {
	Attribute string   `json:"attribute"`
	Operator  string   `json:"operator"`
	Values    []string `json:"values"`
}

func marshalConditions(conditions []domain.Condition) ([]byte, error) {
	dc := make([]dbCondition, len(conditions))
	for i, c := range conditions {
		dc[i] = dbCondition{
			Attribute: c.Attribute,
			Operator:  string(c.Operator),
			Values:    c.Values,
		}
	}
	return json.Marshal(dc)
}

// dbRolloutEntry is the JSON-serialisable form of domain.RolloutEntry stored in the rollout JSONB column.
type dbRolloutEntry struct {
	VariantKey string `json:"variant_key"`
	Weight     int    `json:"weight"`
}

func marshalRollout(entries []domain.RolloutEntry) ([]byte, error) {
	if len(entries) == 0 {
		return nil, nil // NULL in DB
	}
	de := make([]dbRolloutEntry, len(entries))
	for i, e := range entries {
		de[i] = dbRolloutEntry{VariantKey: e.VariantKey, Weight: e.Weight}
	}
	return json.Marshal(de)
}

func unmarshalRollout(data []byte) ([]domain.RolloutEntry, error) {
	if data == nil {
		return nil, nil
	}
	var de []dbRolloutEntry
	if err := json.Unmarshal(data, &de); err != nil {
		return nil, err
	}
	entries := make([]domain.RolloutEntry, len(de))
	for i, e := range de {
		entries[i] = domain.RolloutEntry{VariantKey: e.VariantKey, Weight: e.Weight}
	}
	return entries, nil
}

func unmarshalConditions(data []byte) ([]domain.Condition, error) {
	var dc []dbCondition
	if err := json.Unmarshal(data, &dc); err != nil {
		return nil, err
	}
	conditions := make([]domain.Condition, len(dc))
	for i, c := range dc {
		conditions[i] = domain.Condition{
			Attribute: c.Attribute,
			Operator:  domain.Operator(c.Operator),
			Values:    c.Values,
		}
	}
	return conditions, nil
}

// ListByFlag returns all rules for a flag across all environments.
func (r *PostgresRuleRepository) ListByFlag(ctx context.Context, flagID string) ([]*domain.Rule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, flag_id, environment_id, name, priority, conditions, variant_key, rollout, enabled, created_at
		 FROM rules
		 WHERE flag_id = $1
		 ORDER BY environment_id, priority ASC`,
		flagID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]*domain.Rule, 0)
	for rows.Next() {
		rule, err := scanRuleRow(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

// ListByFlagEnvironment returns all rules for a flag+environment, ordered by priority ascending.
func (r *PostgresRuleRepository) ListByFlagEnvironment(ctx context.Context, flagID, environmentID string) ([]*domain.Rule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, flag_id, environment_id, name, priority, conditions, variant_key, rollout, enabled, created_at
		 FROM rules
		 WHERE flag_id = $1 AND environment_id = $2
		 ORDER BY priority ASC`,
		flagID, environmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]*domain.Rule, 0)
	for rows.Next() {
		rule, err := scanRuleRow(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

// ListByEnvironment returns all rules for an environment, ordered by flag_id then priority ascending.
// Intended for batch loading in EvaluateAll to avoid N+1 queries.
func (r *PostgresRuleRepository) ListByEnvironment(ctx context.Context, environmentID string) ([]*domain.Rule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, flag_id, environment_id, name, priority, conditions, variant_key, rollout, enabled, created_at
		 FROM rules
		 WHERE environment_id = $1
		 ORDER BY flag_id, priority ASC`,
		environmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]*domain.Rule, 0)
	for rows.Next() {
		rule, err := scanRuleRow(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

// Upsert inserts a new rule or updates an existing one (matched by id).
// The RETURNING clause populates rule.CreatedAt with the DB-stored value;
// on conflict (update) this preserves the original creation timestamp.
func (r *PostgresRuleRepository) Upsert(ctx context.Context, rule *domain.Rule) error {
	conditions, err := marshalConditions(rule.Conditions)
	if err != nil {
		return err
	}
	rollout, err := marshalRollout(rule.Rollout)
	if err != nil {
		return err
	}
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO rules (id, flag_id, environment_id, name, priority, conditions, variant_key, rollout, enabled, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (id) DO UPDATE
		   SET name        = EXCLUDED.name,
		       priority    = EXCLUDED.priority,
		       conditions  = EXCLUDED.conditions,
		       variant_key = EXCLUDED.variant_key,
		       rollout     = EXCLUDED.rollout,
		       enabled     = EXCLUDED.enabled
		 RETURNING created_at`,
		rule.ID, rule.FlagID, rule.EnvironmentID, rule.Name, rule.Priority,
		conditions, rule.VariantKey, rollout, rule.Enabled, rule.CreatedAt,
	)
	return row.Scan(&rule.CreatedAt)
}

// DeleteByFlagEnvironment removes all rules for a flag+environment pair in one statement.
func (r *PostgresRuleRepository) DeleteByFlagEnvironment(ctx context.Context, flagID, environmentID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM rules WHERE flag_id = $1 AND environment_id = $2`,
		flagID, environmentID,
	)
	return err
}

// Delete removes a rule by ID. Returns ErrNotFound if no row was deleted.
func (r *PostgresRuleRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM rules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanRuleRow(rows *sql.Rows) (*domain.Rule, error) {
	var rule domain.Rule
	var conditionsJSON []byte
	var rolloutJSON []byte
	if err := rows.Scan(
		&rule.ID, &rule.FlagID, &rule.EnvironmentID,
		&rule.Name, &rule.Priority, &conditionsJSON,
		&rule.VariantKey, &rolloutJSON, &rule.Enabled, &rule.CreatedAt,
	); err != nil {
		return nil, err
	}
	conditions, err := unmarshalConditions(conditionsJSON)
	if err != nil {
		return nil, err
	}
	rule.Conditions = conditions
	rollout, err := unmarshalRollout(rolloutJSON)
	if err != nil {
		return nil, err
	}
	rule.Rollout = rollout
	return &rule, nil
}
