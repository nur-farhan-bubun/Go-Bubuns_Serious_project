-- PostgreSQL schema for group conversations
-- Run via: migrate -path chat-service/migrations -database "$DATABASE_URL" up

CREATE TABLE IF NOT EXISTS group_conversations (
    id         UUID PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS conversation_members (
    conversation_id UUID NOT NULL REFERENCES group_conversations(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL,
    joined_at       TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_conversation_members_user ON conversation_members(user_id);
