-- 001_initial.sql
-- Creates users, sessions, and operations tables per docs/spec/architecture.md §4.

CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL,
    created_at    INTEGER NOT NULL,
    is_admin      INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sessions (
    id            TEXT    PRIMARY KEY,        -- sha256 of the random session token
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at    INTEGER NOT NULL,
    expires_at    INTEGER NOT NULL,
    last_seen_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user    ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS operations (
    id            TEXT    PRIMARY KEY,        -- uuid v7
    user_id       INTEGER NOT NULL REFERENCES users(id),
    kind          TEXT    NOT NULL,
    target        TEXT    NOT NULL,
    incus_op_id   TEXT,
    status        TEXT    NOT NULL,
    error         TEXT,
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL,
    metadata      BLOB
);
CREATE INDEX IF NOT EXISTS idx_operations_user   ON operations(user_id);
CREATE INDEX IF NOT EXISTS idx_operations_status ON operations(status);
