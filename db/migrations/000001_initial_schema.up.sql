CREATE TABLE projects (
    id         TEXT        PRIMARY KEY,
    name       TEXT        NOT NULL,
    slug       TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE environments (
    id         TEXT        PRIMARY KEY,
    project_id TEXT        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    slug       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE (project_id, slug)
);

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
