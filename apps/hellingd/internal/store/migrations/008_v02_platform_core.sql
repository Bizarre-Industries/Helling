-- 008_v02_platform_core.sql
-- v0.2 bridge schema repairs and platform-core tables.

-- +goose Up
ALTER TABLE schedules RENAME COLUMN cron_expr TO on_calendar;
ALTER TABLE schedules ADD COLUMN unit_name TEXT;
ALTER TABLE schedules ADD COLUMN last_status TEXT;
ALTER TABLE schedules ADD COLUMN last_error TEXT;

ALTER TABLE webhooks RENAME COLUMN secret TO secret_encrypted;
ALTER TABLE webhook_deliveries ADD COLUMN event_id TEXT;
ALTER TABLE webhook_deliveries ADD COLUMN event_type TEXT;
ALTER TABLE webhook_deliveries ADD COLUMN latency_ms INTEGER;
ALTER TABLE webhook_deliveries ADD COLUMN next_retry_at INTEGER;
ALTER TABLE webhook_deliveries ADD COLUMN delivered_at INTEGER;

ALTER TABLE users ADD COLUMN incus_project TEXT NOT NULL DEFAULT 'default';

CREATE TABLE IF NOT EXISTS events (
    id          TEXT    PRIMARY KEY,
    type        TEXT    NOT NULL,
    source      TEXT    NOT NULL,
    subject     TEXT    NOT NULL,
    data_json   TEXT    NOT NULL DEFAULT '{}',
    created_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);
CREATE INDEX IF NOT EXISTS idx_events_created_id ON events(created_at, id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);

CREATE TABLE IF NOT EXISTS event_outbox (
    id          TEXT    PRIMARY KEY,
    event_id    TEXT    NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    webhook_id  TEXT    REFERENCES webhooks(id) ON DELETE CASCADE,
    status      TEXT    NOT NULL DEFAULT 'pending',
    attempts    INTEGER NOT NULL DEFAULT 0,
    next_run_at INTEGER,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_event_outbox_status ON event_outbox(status, next_run_at);
CREATE INDEX IF NOT EXISTS idx_event_outbox_claim
    ON event_outbox(status, next_run_at, created_at, id);
CREATE INDEX IF NOT EXISTS idx_event_outbox_stale_processing
    ON event_outbox(status, updated_at, id);
CREATE INDEX IF NOT EXISTS idx_event_outbox_event_id ON event_outbox(event_id);

CREATE TABLE IF NOT EXISTS webhook_events (
    webhook_id TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    PRIMARY KEY (webhook_id, event_type)
);
CREATE INDEX IF NOT EXISTS idx_webhook_events_event_type ON webhook_events(event_type);
INSERT OR IGNORE INTO webhook_events (webhook_id, event_type)
SELECT
    w.id,
    trim(je.value) AS event_type
FROM webhooks AS w
CROSS JOIN json_each(w.events) AS je
WHERE trim(je.value) <> '';

CREATE TABLE IF NOT EXISTS firewall_host_rules (
    id               TEXT    PRIMARY KEY,
    user_id          INTEGER NOT NULL REFERENCES users(id),
    direction        TEXT    NOT NULL,
    action           TEXT    NOT NULL,
    protocol         TEXT    NOT NULL,
    source_cidr      TEXT,
    destination_cidr TEXT,
    destination_port INTEGER,
    enabled          INTEGER NOT NULL DEFAULT 1,
    comment          TEXT,
    nft_comment      TEXT    NOT NULL UNIQUE,
    nft_handle       INTEGER,
    created_at       INTEGER NOT NULL,
    updated_at       INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_firewall_host_rules_direction ON firewall_host_rules(direction);

CREATE TABLE IF NOT EXISTS incus_user_certs (
    user_id           INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    encrypted_cert_pem TEXT,
    encrypted_key_pem TEXT,
    fingerprint       TEXT UNIQUE,
    restricted        INTEGER NOT NULL DEFAULT 1,
    project_scope     TEXT NOT NULL DEFAULT 'default',
    expires_at        INTEGER,
    revoked_at        INTEGER,
    created_at        INTEGER NOT NULL,
    updated_at        INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS incus_user_certs;
DROP TABLE IF EXISTS firewall_host_rules;
DROP TABLE IF EXISTS webhook_events;
DROP TABLE IF EXISTS event_outbox;
DROP TABLE IF EXISTS events;

ALTER TABLE users DROP COLUMN incus_project;

ALTER TABLE webhook_deliveries DROP COLUMN delivered_at;
ALTER TABLE webhook_deliveries DROP COLUMN next_retry_at;
ALTER TABLE webhook_deliveries DROP COLUMN latency_ms;
ALTER TABLE webhook_deliveries DROP COLUMN event_type;
ALTER TABLE webhook_deliveries DROP COLUMN event_id;
ALTER TABLE webhooks RENAME COLUMN secret_encrypted TO secret;

ALTER TABLE schedules DROP COLUMN last_error;
ALTER TABLE schedules DROP COLUMN last_status;
ALTER TABLE schedules DROP COLUMN unit_name;
ALTER TABLE schedules RENAME COLUMN on_calendar TO cron_expr;
