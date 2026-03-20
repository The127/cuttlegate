DROP TABLE IF EXISTS flags;

CREATE TABLE flags (
    id             TEXT        PRIMARY KEY,
    project_id     TEXT        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment_id TEXT        NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    key            TEXT        NOT NULL,
    enabled        BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL,
    UNIQUE (environment_id, key)
);
