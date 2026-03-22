package dbadapter

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
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

// ListByFlagEnvironment returns all rules for a flag+environment, ordered by priority ascending.
func (r *PostgresRuleRepository) ListByFlagEnvironment(ctx context.Context, flagID, environmentID string) ([]*domain.Rule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, flag_id, environment_id, priority, conditions, variant_key, enabled, created_at
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

// Upsert inserts a new rule or updates an existing one (matched by id).
// The RETURNING clause populates rule.CreatedAt with the DB-stored value;
// on conflict (update) this preserves the original creation timestamp.
func (r *PostgresRuleRepository) Upsert(ctx context.Context, rule *domain.Rule) error {
	conditions, err := marshalConditions(rule.Conditions)
	if err != nil {
		return err
	}
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO rules (id, flag_id, environment_id, priority, conditions, variant_key, enabled, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (id) DO UPDATE
		   SET priority    = EXCLUDED.priority,
		       conditions  = EXCLUDED.conditions,
		       variant_key = EXCLUDED.variant_key,
		       enabled     = EXCLUDED.enabled
		 RETURNING created_at`,
		rule.ID, rule.FlagID, rule.EnvironmentID, rule.Priority,
		conditions, rule.VariantKey, rule.Enabled, rule.CreatedAt,
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
	if err := rows.Scan(
		&rule.ID, &rule.FlagID, &rule.EnvironmentID,
		&rule.Priority, &conditionsJSON,
		&rule.VariantKey, &rule.Enabled, &rule.CreatedAt,
	); err != nil {
		return nil, err
	}
	conditions, err := unmarshalConditions(conditionsJSON)
	if err != nil {
		return nil, err
	}
	rule.Conditions = conditions
	return &rule, nil
}
