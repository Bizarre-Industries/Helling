# Design Note: Operations and Async Work

> Companion to `docs/spec/architecture.md` §3 (request lifecycle) and §4 (data model).
> Status: design, not yet implemented. Tracks decisions for stage 3 of the roadmap.

## Problem

Several Helling operations (instance create, start, stop, delete) take long enough to be unreasonable to block an HTTP request on. Incus's own API exposes them as async operations: a request returns immediately with an `Operation` reference, and the client polls or watches a websocket for completion.

Helling needs to:

1. Surface async operations to its own API consumers cleanly.
2. Not leak Incus internals (URLs, IDs, error shapes) into the Helling API.
3. Survive `hellingd` restarts mid-operation without losing state about the operation.
4. Let users see what's running and what failed.

## Decisions

### A. Helling owns its own operation IDs

API consumers see `helling_op_<uuid>`. Internally we may store the corresponding Incus operation URL; we never expose it.

### B. The `operations` table is the canonical record

When an API call kicks off async work, hellingd:

1. Inserts a row into `operations` with `status='pending'`, returns `202 Accepted` with the new operation in the body.
2. Spawns a background goroutine (rooted in the daemon's `errgroup`) that calls Incus, gets back an Incus `Operation`, updates our row's `status='running'` and stores the Incus operation ID.
3. Calls `op.Wait()` on the Incus operation.
4. Updates our row to `status='success'` or `status='failure'` (with `error` populated) and returns.

If hellingd restarts while an operation is `running`:

- Boot logic queries `operations WHERE status IN ('pending','running')`.
- For each row with an `incus_op_id`, attach to the live Incus operation via the Incus client and resume waiting.
- Rows with no `incus_op_id` (we crashed before submitting to Incus) are marked `failure` with error `"hellingd_restart_before_submit"`.

### C. Polling, not websockets, in v0.1

The v0.1 web UI polls `GET /v1/operations/{id}` every 1.5 seconds while an operation is in `pending` or `running`. Websockets are post-v0.1.

### D. Operation kinds are an enumeration

```text
instance.create
instance.start
instance.stop
instance.delete
```

The `kind` field is part of the API contract. Adding a new kind is a `feat:` change. Renaming or removing one is a breaking change.

### E. Cancellation is not in v0.1

The OpenAPI schema includes `cancelled` as a status only because Incus might surface it; we don't expose a cancellation endpoint. Documented limitation.

### F. Audit is the same table

Every state-changing API call gets an `operations` row, even synchronous ones (e.g. login). Synchronous ones are inserted with `status='success'` directly. This unifies "what did the user do" with "what's currently running".

If audit volume becomes a problem, split `operations` (active work) from `audit` (historical record) in v0.2.

## Operation lifecycle diagram

```text
HTTP POST /v1/instances
  │
  ▼
handler validates input
  │
  ▼
service.CreateInstance(ctx, user, req)
  │
  ├─ INSERT operations (id=<uuid>, status='pending', kind='instance.create', ...)
  │
  ├─ go func() {                                   // background, daemon-scoped
  │     UPDATE operations SET status='running' WHERE id=<uuid>
  │     incusOp, err := incusClient.CreateInstance(...)
  │     UPDATE operations SET incus_op_id=<incusOp.URL>
  │     err := incusOp.Wait()
  │     if err != nil:
  │       UPDATE operations SET status='failure', error=<sanitized>
  │     else:
  │       UPDATE operations SET status='success'
  │  }()
  │
  ▼
return 202 + body { id: <uuid>, status: 'pending', ... }
```

## Open questions

1. **UUID v7 vs v4.** Default to v7 for sortable IDs. Decision in roadmap §"Decisions still owed".
2. **Goroutine lifecycle on shutdown.** Graceful: wait up to 30s for in-flight ops to settle, then force-mark as `failure` with `"hellingd_shutdown"`. Hard: just exit, let restart logic resume on next boot. Default: graceful.
3. **Error sanitization rules.** Incus errors can include filesystem paths and similar. We strip everything except the Incus error code or a fixed-list message before persisting `operations.error`.
4. **What if Incus is unreachable when we try to submit?** Mark `operations.status='failure'` with `error='incus_unavailable'` and return 503 to the API caller, since we never moved past 'pending'.

## Consequences

- The service layer needs an `OperationRunner` abstraction that handlers don't see directly.
- The Incus client wrapper exposes a `WaitForOperationByURL(ctx, url) error` that resume logic uses on restart.
- Background goroutines must be cancellable — pass a context derived from the daemon's lifecycle, not the request.

## Alternatives considered

- **Synchronous everything.** Rejected: Incus operations can take minutes (image download, large filesystem create). Holding HTTP connections that long is a bad shape.
- **Pure event-driven via Incus's websocket.** Rejected for v0.1 complexity. Reconsider in v0.2 if polling load becomes a problem.
- **Drop the operations table; just stream Incus's operation responses through.** Rejected because it leaks Incus IDs and breaks our restart-resume requirement.
