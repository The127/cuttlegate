CREATE TABLE flag_evaluation_stats (
    flag_id           TEXT        NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    environment_id    TEXT        NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    flag_key          TEXT        NOT NULL,
    evaluation_count  BIGINT      NOT NULL DEFAULT 0,
    last_evaluated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (flag_id, environment_id)
);
