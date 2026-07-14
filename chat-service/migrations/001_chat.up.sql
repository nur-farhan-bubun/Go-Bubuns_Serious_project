-- PostgreSQL schema for chat service
-- Run via: migrate -path chat-service/migrations -database "$DATABASE_URL" up

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS conversations (
    id         UUID PRIMARY KEY,
    user1_id   TEXT,
    user2_id   TEXT,
    match_id   TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    type       TEXT DEFAULT 'direct',
    name       TEXT
);

CREATE TABLE IF NOT EXISTS messages (
    id              UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       TEXT NOT NULL,
    content         TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(conversation_id, created_at);
CREATE INDEX IF NOT EXISTS idx_conversations_user1 ON conversations(user1_id);
CREATE INDEX IF NOT EXISTS idx_conversations_user2 ON conversations(user2_id);
