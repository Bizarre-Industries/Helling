-- +goose Up
-- +goose StatementBegin

-- docs/spec/sqlite-schema.md §1: Identity and Session Tables.

CREATE TABLE users (
    id         TEXT PRIMARY KEY,
    username   TEXT UNIQUE NOT NULL,
    role       TEXT NOT NULL CHECK (role IN ('admin','user','auditor')),
    status     TEXT NOT NULL CHECK (status IN ('active','disabled')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE sessions (
    id                 TEXT PRIMARY KEY,
    user_id            TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash TEXT NOT NULL,
    user_agent         TEXT,
    ip_address         TEXT,
    expires_at         INTEGER NOT NULL,
    revoked_at         INTEGER,
    created_at         INTEGER NOT NULL
);

CREATE INDEX idx_sessions_user_id    ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

CREATE TABLE api_tokens (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    token_hash   TEXT NOT NULL,
    scope        TEXT NOT NULL CHECK (scope IN ('read','write','admin')),
    last_used_at INTEGER,
    expires_at   INTEGER NOT NULL,
    revoked_at   INTEGER,
    created_at   INTEGER NOT NULL
);

CREATE INDEX idx_api_tokens_user_id    ON api_tokens(user_id);
CREATE INDEX idx_api_tokens_expires_at ON api_tokens(expires_at);

-- docs/spec/sqlite-schema.md §2: MFA and Auth Material.

CREATE TABLE totp_secrets (
    user_id          TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    encrypted_secret BLOB NOT NULL,
    enabled          INTEGER NOT NULL CHECK (enabled IN (0,1)),
    created_at       INTEGER NOT NULL,
    updated_at       INTEGER NOT NULL
);

CREATE TABLE recovery_codes (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash  TEXT NOT NULL,
    used_at    INTEGER,
    created_at INTEGER NOT NULL
);

CREATE INDEX idx_recovery_codes_user_id ON recovery_codes(user_id);

CREATE TABLE auth_events (
    id            TEXT PRIMARY KEY,
    user_id       TEXT REFERENCES users(id) ON DELETE SET NULL,
    event_type    TEXT NOT NULL,
    source_ip     TEXT,
    user_agent    TEXT,
    metadata_json TEXT,
    created_at    INTEGER NOT NULL
);

CREATE INDEX idx_auth_events_user_id    ON auth_events(user_id);
CREATE INDEX idx_auth_events_created_at ON auth_events(created_at);

-- docs/spec/sqlite-schema.md §3: Incus Trust and CA Tables.

CREATE TABLE helling_ca (
    id                   TEXT PRIMARY KEY CHECK (id = 'default'),
    cert_pem             TEXT NOT NULL,
    encrypted_key_pem    BLOB NOT NULL,
    not_before           INTEGER NOT NULL,
    not_after            INTEGER NOT NULL,
    rotation_grace_until INTEGER,
    created_at           INTEGER NOT NULL,
    updated_at           INTEGER NOT NULL
);

CREATE TABLE incus_user_certs (
    user_id           TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    cert_pem          TEXT NOT NULL,
    encrypted_key_pem BLOB NOT NULL,
    fingerprint       TEXT UNIQUE NOT NULL,
    restricted        INTEGER NOT NULL CHECK (restricted IN (0,1)),
    project_scope     TEXT NOT NULL,
    expires_at        INTEGER NOT NULL,
    revoked_at        INTEGER,
    created_at        INTEGER NOT NULL,
    updated_at        INTEGER NOT NULL
);

CREATE INDEX idx_incus_user_certs_fingerprint ON incus_user_certs(fingerprint);
CREATE INDEX idx_incus_user_certs_expires_at  ON incus_user_certs(expires_at);

-- docs/spec/sqlite-schema.md §4: Webhooks and Delivery State.

CREATE TABLE webhooks (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    url              TEXT NOT NULL,
    events_json      TEXT NOT NULL,
    secret_encrypted BLOB NOT NULL,
    enabled          INTEGER NOT NULL CHECK (enabled IN (0,1)),
    last_delivery_at INTEGER,
    created_by       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at       INTEGER NOT NULL,
    updated_at       INTEGER NOT NULL
);

CREATE INDEX idx_webhooks_created_by ON webhooks(created_by);
CREATE INDEX idx_webhooks_enabled    ON webhooks(enabled);

CREATE TABLE webhook_deliveries (
    id              TEXT PRIMARY KEY,
    webhook_id      TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event_id        TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    attempt         INTEGER NOT NULL CHECK (attempt >= 1),
    status          TEXT NOT NULL CHECK (status IN ('pending','success','failed')),
    http_status     INTEGER,
    latency_ms      INTEGER,
    next_retry_at   INTEGER,
    error_text      TEXT,
    response_sample TEXT,
    created_at      INTEGER NOT NULL,
    delivered_at    INTEGER
);

CREATE INDEX idx_webhook_deliveries_webhook_id_created_at ON webhook_deliveries(webhook_id, created_at DESC);
CREATE INDEX idx_webhook_deliveries_event_id             ON webhook_deliveries(event_id);

-- docs/spec/sqlite-schema.md §5: Kubernetes Control-Plane Metadata.

CREATE TABLE kubernetes_clusters (
    name                 TEXT PRIMARY KEY,
    state                TEXT NOT NULL CHECK (state IN ('creating','ready','upgrading','deleting','error')),
    k8s_version          TEXT NOT NULL,
    pod_cidr             TEXT,
    service_cidr         TEXT,
    control_plane_count  INTEGER NOT NULL CHECK (control_plane_count >= 1),
    worker_count         INTEGER NOT NULL CHECK (worker_count >= 0),
    kubeconfig_encrypted BLOB NOT NULL,
    last_operation       TEXT,
    last_error           TEXT,
    created_by           TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at           INTEGER NOT NULL,
    updated_at           INTEGER NOT NULL,
    deleted_at           INTEGER
);

CREATE INDEX idx_kubernetes_clusters_state      ON kubernetes_clusters(state);
CREATE INDEX idx_kubernetes_clusters_created_by ON kubernetes_clusters(created_by);

CREATE TABLE kubernetes_nodes (
    id           TEXT PRIMARY KEY,
    cluster_name TEXT NOT NULL REFERENCES kubernetes_clusters(name) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    role         TEXT NOT NULL CHECK (role IN ('control-plane','worker')),
    status       TEXT NOT NULL CHECK (status IN ('provisioning','ready','notready','deleted')),
    instance_ref TEXT,
    ip_address   TEXT,
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL
);

CREATE INDEX idx_kubernetes_nodes_cluster_name            ON kubernetes_nodes(cluster_name);
CREATE UNIQUE INDEX uq_kubernetes_nodes_cluster_name_name ON kubernetes_nodes(cluster_name, name);

-- docs/spec/sqlite-schema.md §6: Firewall and Warning State.

CREATE TABLE firewall_host_rules (
    id         TEXT PRIMARY KEY,
    chain      TEXT NOT NULL,
    position   INTEGER NOT NULL,
    action     TEXT NOT NULL CHECK (action IN ('accept','drop','reject')),
    protocol   TEXT NOT NULL CHECK (protocol IN ('tcp','udp','icmp','any')),
    src_cidr   TEXT,
    dst_cidr   TEXT,
    src_port   TEXT,
    dst_port   TEXT,
    comment    TEXT,
    enabled    INTEGER NOT NULL CHECK (enabled IN (0,1)),
    created_by TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX idx_firewall_host_rules_chain_position ON firewall_host_rules(chain, position);
CREATE INDEX idx_firewall_host_rules_enabled        ON firewall_host_rules(enabled);

CREATE TABLE warnings (
    id              TEXT PRIMARY KEY,
    warning_key     TEXT NOT NULL,
    category        TEXT NOT NULL,
    severity        TEXT NOT NULL CHECK (severity IN ('info','warning','critical')),
    state           TEXT NOT NULL CHECK (state IN ('active','acknowledged','resolved')),
    subject         TEXT,
    message         TEXT NOT NULL,
    details_json    TEXT,
    first_seen_at   INTEGER NOT NULL,
    last_seen_at    INTEGER NOT NULL,
    acknowledged_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    acknowledged_at INTEGER,
    resolved_at     INTEGER
);

CREATE UNIQUE INDEX uq_warnings_warning_key      ON warnings(warning_key);
CREATE INDEX idx_warnings_state_severity         ON warnings(state, severity);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS warnings;
DROP TABLE IF EXISTS firewall_host_rules;
DROP TABLE IF EXISTS kubernetes_nodes;
DROP TABLE IF EXISTS kubernetes_clusters;
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhooks;
DROP TABLE IF EXISTS incus_user_certs;
DROP TABLE IF EXISTS helling_ca;
DROP TABLE IF EXISTS auth_events;
DROP TABLE IF EXISTS recovery_codes;
DROP TABLE IF EXISTS totp_secrets;
DROP TABLE IF EXISTS api_tokens;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;

-- +goose StatementEnd
