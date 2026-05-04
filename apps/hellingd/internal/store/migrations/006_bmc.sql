-- 006_bmc.sql
-- BMC (Baseboard Management Controller) endpoints for IPMI/Redfish.
-- Per docs/spec/architecture.md §BMC.

CREATE TABLE IF NOT EXISTS bmc_endpoints (
    id         TEXT    PRIMARY KEY,         -- uuid v7
    user_id    INTEGER NOT NULL REFERENCES users(id),
    name       TEXT    NOT NULL,
    address    TEXT    NOT NULL,            -- IP or hostname
    port       INTEGER NOT NULL DEFAULT 623,
    username   TEXT    NOT NULL,
    password   TEXT    NOT NULL,            -- stored encrypted
    protocol   TEXT    NOT NULL DEFAULT 'ipmi',  -- ipmi, redfish
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_bmc_user ON bmc_endpoints(user_id);
