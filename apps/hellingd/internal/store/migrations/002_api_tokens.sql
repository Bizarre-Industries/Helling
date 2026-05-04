-- 002_api_tokens.sql
-- API tokens for non-interactive auth (CLI, scripts, CI).
-- Tokens are stored as SHA-256 hashes; the raw token is shown once at creation.

CREATE TABLE IF NOT EXISTS api_tokens (
    id          TEXT    PRIMARY KEY,        -- uuid v7
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT    NOT NULL,           -- human label, e.g. "CI deployer"
    token_hash  TEXT    NOT NULL UNIQUE,    -- sha256(helling_<random 40>)
    scopes      TEXT    NOT NULL DEFAULT 'read',  -- read, write, admin
    created_at  INTEGER NOT NULL,
    expires_at  INTEGER,                    -- NULL = never
    last_used_at INTEGER
);
CREATE INDEX IF NOT EXISTS idx_api_tokens_user ON api_tokens(user_id);
