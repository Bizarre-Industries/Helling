-- 004_schedules.sql
-- systemd timer-backed schedules for backups, snapshots, and custom actions.
-- Per ADR-017: schedules write .timer + .service units to /etc/systemd/system/.

CREATE TABLE IF NOT EXISTS schedules (
    id          TEXT    PRIMARY KEY,        -- uuid v7
    user_id     INTEGER NOT NULL REFERENCES users(id),
    name        TEXT    NOT NULL,
    kind        TEXT    NOT NULL,           -- backup, snapshot, custom
    target      TEXT    NOT NULL,           -- instance or volume name
    cron_expr   TEXT    NOT NULL,           -- systemd OnCalendar expression
    enabled     INTEGER NOT NULL DEFAULT 1,
    last_run_at INTEGER,
    next_run_at INTEGER,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_schedules_user ON schedules(user_id);
CREATE INDEX IF NOT EXISTS idx_schedules_target ON schedules(target);
