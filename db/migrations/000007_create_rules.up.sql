CREATE TABLE rules (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    flag_id        TEXT        NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    environment_id TEXT        NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    priority       INT         NOT NULL DEFAULT 0,
    conditions     JSONB       NOT NULL DEFAULT '[]',
    variant_key    TEXT        NOT NULL,
    enabled        BOOLEAN     NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX rules_flag_env_idx ON rules (flag_id, environment_id);
