CREATE TABLE IF NOT EXISTS app_users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    bio         TEXT DEFAULT '',
    avatar_url  TEXT DEFAULT '',
    photo_urls  TEXT[] DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON app_users(email);
