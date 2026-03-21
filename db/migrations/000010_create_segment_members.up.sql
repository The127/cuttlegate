CREATE TABLE segment_members (
    segment_id UUID        NOT NULL REFERENCES segments(id) ON DELETE CASCADE,
    user_key   TEXT        NOT NULL,
    added_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (segment_id, user_key)
);
