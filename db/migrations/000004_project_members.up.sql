CREATE TABLE project_members (
    -- user_id is the OIDC JWT sub claim; no FK to a users table (identity is authoritative from the IdP)
    project_id  text        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id     text        NOT NULL,
    role        text        NOT NULL,
    created_at  timestamptz NOT NULL,
    PRIMARY KEY (project_id, user_id)
);
