DROP TABLE IF EXISTS flags;

CREATE TABLE flags (
    id                  TEXT        PRIMARY KEY,
    project_id          TEXT        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key                 TEXT        NOT NULL,
    name                TEXT        NOT NULL,
    type                TEXT        NOT NULL,
    variants            JSONB       NOT NULL,
    default_variant_key TEXT        NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL,
    UNIQUE (project_id, key)
);
