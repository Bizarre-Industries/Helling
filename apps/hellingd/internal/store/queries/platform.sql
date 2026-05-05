-- name: ListSchedules :many
SELECT id, user_id, name, kind, target, on_calendar, enabled, last_run_at,
       next_run_at, unit_name, last_status, last_error, created_at, updated_at
FROM schedules
ORDER BY created_at DESC;

-- name: ListWebhooks :many
SELECT id, user_id, name, url, secret_encrypted, events, enabled, created_at,
       updated_at
FROM webhooks
ORDER BY created_at DESC;

-- name: ListRecentEvents :many
SELECT id, type, source, subject, data_json, created_at
FROM events
ORDER BY created_at DESC
LIMIT ?;

-- name: ListFirewallHostRules :many
SELECT id, user_id, direction, action, protocol, source_cidr, destination_cidr,
       destination_port, enabled, comment, nft_comment, nft_handle, created_at,
       updated_at
FROM firewall_host_rules
ORDER BY created_at DESC;
