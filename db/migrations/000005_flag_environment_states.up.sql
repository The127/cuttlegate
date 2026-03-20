CREATE TABLE flag_environment_states (
    flag_id        text    NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    environment_id text    NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    enabled        boolean NOT NULL DEFAULT false,
    PRIMARY KEY (flag_id, environment_id)
);
