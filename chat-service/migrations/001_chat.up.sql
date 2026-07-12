-- ScyllaDB schema for chat service
-- Run via cqlsh or scylla driver

CREATE KEYSPACE IF NOT EXISTS app_chat
    WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

USE app_chat;

-- User IDs are stored as TEXT (not UUID) because auth providers
-- (Clerk, dev-auth) issue non-UUID user identifiers.

CREATE TABLE IF NOT EXISTS conversations (
    id         UUID PRIMARY KEY,
    user1_id   TEXT,
    user2_id   TEXT,
    match_id   TEXT,
    created_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
    conversation_id UUID,
    id              TIMEUUID,
    sender_id       TEXT,
    content         TEXT,
    created_at      TIMESTAMP,
    PRIMARY KEY (conversation_id, id)
) WITH CLUSTERING ORDER BY (id ASC);

CREATE INDEX IF NOT EXISTS idx_conversations_user1 ON conversations(user1_id);
CREATE INDEX IF NOT EXISTS idx_conversations_user2 ON conversations(user2_id);
