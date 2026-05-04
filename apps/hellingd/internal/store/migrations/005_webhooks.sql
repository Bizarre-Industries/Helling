-- 005_webhooks.sql
-- Outbound webhooks with HMAC-SHA256 signing and delivery log.
-- Per docs/spec/architecture.md §Webhooks.

CREATE TABLE IF NOT EXISTS webhooks (
    id         TEXT    PRIMARY KEY,         -- uuid v7
    user_id    INTEGER NOT NULL REFERENCES users(id),
    name       TEXT    NOT NULL,
    url        TEXT    NOT NULL,
    secret     TEXT    NOT NULL,            -- HMAC secret (stored encrypted)
    events     TEXT    NOT NULL DEFAULT '[]', -- JSON array of event types
    enabled    INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_webhooks_user ON webhooks(user_id);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id           TEXT    PRIMARY KEY,       -- uuid v7
    webhook_id   TEXT    NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event        TEXT    NOT NULL,
    status       TEXT    NOT NULL,          -- pending, success, failure
    status_code  INTEGER,
    response_body TEXT,
    error        TEXT,
    attempt      INTEGER NOT NULL DEFAULT 1,
    created_at   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook ON webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status ON webhook_deliveries(status);
