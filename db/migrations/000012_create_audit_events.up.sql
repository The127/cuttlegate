CREATE TABLE audit_events (
    id           TEXT        PRIMARY KEY,
    project_id   TEXT        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    actor_id     TEXT        NOT NULL,
    action       TEXT        NOT NULL,
    entity_type  TEXT        NOT NULL,
    entity_id    TEXT        NOT NULL,
    entity_key   TEXT        NOT NULL DEFAULT '',
    before_state TEXT        NOT NULL DEFAULT '',
    after_state  TEXT        NOT NULL DEFAULT '',
    occurred_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_audit_events_project_time ON audit_events (project_id, occurred_at DESC);
CREATE INDEX idx_audit_events_project_entity_key_time ON audit_events (project_id, entity_key, occurred_at DESC);
