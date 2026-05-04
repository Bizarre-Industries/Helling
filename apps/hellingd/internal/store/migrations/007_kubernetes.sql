-- 007_kubernetes.sql
-- Kubernetes clusters provisioned via CAPN on Incus VMs.
-- Per docs/spec/architecture.md §Kubernetes.

CREATE TABLE IF NOT EXISTS k8s_clusters (
    id              TEXT    PRIMARY KEY,     -- uuid v7
    user_id         INTEGER NOT NULL REFERENCES users(id),
    name            TEXT    NOT NULL UNIQUE,
    version         TEXT    NOT NULL,        -- K8s version, e.g. "1.32"
    control_planes  INTEGER NOT NULL DEFAULT 1,
    workers         INTEGER NOT NULL DEFAULT 2,
    status          TEXT    NOT NULL DEFAULT 'provisioning',
    kubeconfig      TEXT,                    -- base64-encoded kubeconfig
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_k8s_clusters_user ON k8s_clusters(user_id);
CREATE INDEX IF NOT EXISTS idx_k8s_clusters_status ON k8s_clusters(status);
