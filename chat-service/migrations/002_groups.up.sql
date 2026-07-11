-- ScyllaDB schema for group conversations
-- Run via cqlsh or scylla driver

USE app_chat;

-- Group conversations store the name and type
CREATE TABLE IF NOT EXISTS group_conversations (
    id         UUID PRIMARY KEY,
    name       TEXT,
    created_at TIMESTAMP
);

-- Conversation members: one row per member per conversation
-- Allows querying by conversation_id or by user_id
CREATE TABLE IF NOT EXISTS conversation_members (
    conversation_id UUID,
    user_id         TEXT,
    joined_at       TIMESTAMP,
    PRIMARY KEY (conversation_id, user_id)
);

-- Index to look up which conversations a user is in
CREATE INDEX IF NOT EXISTS idx_conversation_members_user ON conversation_members(user_id);

-- Add type and name columns to existing conversations table (for fresh installs)
-- If the columns already exist, this is a no-op.
-- For existing databases, run: ALTER TABLE conversations ADD type TEXT;
-- For existing databases, run: ALTER TABLE conversations ADD name TEXT;
