# Event Catalog

Helling emits events through two delivery surfaces.

1. SSE stream: `GET /api/v1/events` with `Accept: text/event-stream`
2. JSON snapshot: `GET /api/v1/events?limit=N` with JSON `Accept`
3. Webhooks: configured destinations under `/api/v1/webhooks/*`

The event type namespace is shared across both surfaces.

## Event Envelope

### SSE

SSE frame structure:

```text
event: <event_type>
id: <event_id>
data: {"type":"...","time":"...","source":"...","subject":"...","data":{...}}
```

Rules:

- `id` is an ULID-like monotonic event id used for resume.
- Reconnect resumes via `Last-Event-ID` header.
- `type` in JSON must match SSE `event` line.

### Webhook Body

Webhook payload uses JSON envelope:

```json
{
  "id": "01JZ...",
  "type": "instance.started",
  "time": "2026-04-20T10:00:00Z",
  "source": "incus",
  "subject": "vm-web-1",
  "data": {}
}
```

Headers:

- `X-Helling-Signature`: HMAC-SHA256 signature over raw body.
- `X-Helling-Event-ID`: event id.
- `X-Helling-Event-Type`: event type.

## Sources

| Source  | Description                                             |
| ------- | ------------------------------------------------------- |
| incus   | Events re-emitted from Incus lifecycle/cluster surfaces |
| podman  | Events re-emitted from Podman lifecycle surfaces        |
| helling | Helling-native control plane events                     |

## Event Types

### Incus-derived

- `instance.created`
- `instance.deleted`
- `instance.started`
- `instance.stopped`
- `instance.updated`
- `snapshot.created`
- `snapshot.deleted`
- `cluster.node.joined`
- `cluster.node.left`

### Podman-derived

- `container.created`
- `container.deleted`
- `container.started`
- `container.stopped`

### Helling-native

- `auth.login.ok`
- `auth.login.fail`
- `auth.token.revoked`
- `user.created`
- `user.updated`
- `user.deleted`
- `schedule.created`
- `schedule.updated`
- `schedule.deleted`
- `schedule.started`
- `schedule.completed`
- `schedule.failed`
- `backup.completed`
- `backup.failed`
- `webhook.delivery.ok`
- `webhook.delivery.fail`
- `webhook.created`
- `webhook.updated`
- `webhook.deleted`
- `webhook.test`
- `firewall.rule.created`
- `firewall.rule.deleted`
- `warning.raised`
- `warning.resolved`
- `system.upgrade.started`
- `system.upgrade.completed`
- `system.upgrade.failed`

## Filtering and Replay

SSE query filters (optional):

- `type`: exact or prefix filter (`instance.`)
- `since`: event id lower bound

Replay window is implementation-defined; baseline retention target is 5 minutes in-memory for reconnect continuity.

JSON snapshot mode returns recent events in the Helling envelope shape and is
used by CLI snapshot tests and non-streaming WebUI state hydration.

## Delivery Guarantees

- SSE: best-effort, at-least-once from reconnect boundary.
- Webhooks: at-least-once with retry policy from platform spec.
- Consumers must deduplicate by event id.
