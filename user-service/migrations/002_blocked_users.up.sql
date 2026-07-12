CREATE TABLE IF NOT EXISTS blocked_users (
    blocker_id   TEXT NOT NULL,
    blocked_id   TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (blocker_id, blocked_id)
);

CREATE INDEX IF NOT EXISTS idx_blocked_users_blocked_id ON blocked_users (blocked_id);
