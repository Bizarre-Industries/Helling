-- +goose Up
-- +goose StatementBegin

-- system_config persists Helling runtime configuration written via
-- /api/v1/system/config/{key}. Keys are dotted (e.g.
-- auth.session_inactivity_timeout). Values are stored as TEXT; callers
-- serialize richer types at the edges.
CREATE TABLE system_config (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at INTEGER NOT NULL,
    updated_by TEXT REFERENCES users(id) ON DELETE SET NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS system_config;

-- +goose StatementEnd
