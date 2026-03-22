CREATE TABLE evaluation_events (
    id               TEXT        NOT NULL PRIMARY KEY,
    flag_key         TEXT        NOT NULL,
    project_id       TEXT        NOT NULL,
    environment_id   TEXT        NOT NULL,
    user_id          TEXT        NOT NULL DEFAULT '',
    input_context    JSONB       NOT NULL DEFAULT '{}',
    matched_rule_id  TEXT        NOT NULL DEFAULT '',
    matched_rule_name TEXT       NOT NULL DEFAULT '',
    variant_key      TEXT        NOT NULL,
    reason           TEXT        NOT NULL,
    occurred_at      TIMESTAMPTZ NOT NULL
);

-- Primary access pattern: newest-first for a specific flag in a project+environment.
CREATE INDEX evaluation_events_project_env_flag_idx
    ON evaluation_events (project_id, environment_id, flag_key, occurred_at DESC);
