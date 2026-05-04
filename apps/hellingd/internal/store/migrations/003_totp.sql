-- 003_totp.sql
-- TOTP secrets and recovery codes per docs/spec/architecture.md §5.

CREATE TABLE IF NOT EXISTS totp_secrets (
    user_id        INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    secret         TEXT    NOT NULL,        -- base32-encoded TOTP secret
    enabled        INTEGER NOT NULL DEFAULT 0,
    created_at     INTEGER NOT NULL,
    updated_at     INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS totp_recovery_codes (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash TEXT   NOT NULL,             -- bcrypt hash of a 16-char recovery code
    used     INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_totp_recovery_user ON totp_recovery_codes(user_id);
