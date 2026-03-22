CREATE TABLE users (
    id         TEXT        PRIMARY KEY,
    name       TEXT        NOT NULL DEFAULT '',
    email      TEXT        NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL
);
