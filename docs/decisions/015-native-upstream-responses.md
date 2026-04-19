# ADR-015: Native Upstream Response Formats for Proxied Endpoints

> Status: Accepted

## Context

With ADR-014 (proxy architecture), hellingd forwards requests to Incus and Podman sockets. The previous architecture wrapped all upstream responses in a Helling envelope (`{data, meta}` for success, `{error, code, action, doc_link}` for errors). This required deserializing every upstream response, re-wrapping it, and re-serializing it — adding latency, complexity, and a maintenance burden every time upstream response formats changed.

## Decision

Proxied Incus and Podman responses pass through in their native format. No re-enveloping, no transformation, no deserialization.

- **Incus responses:** Recursive metadata format per the [Incus REST API spec](https://linuxcontainers.org/incus/docs/main/rest-api-spec/)
- **Podman responses:** Libpod format per the [Podman API spec](https://docs.podman.io/en/latest/_static/api.html)
- **Helling responses:** `{data, meta}` envelope for success, `{error, code, action, doc_link}` for errors — only on the ~25 Helling-specific endpoints

The frontend maintains separate API clients:
- `hellingClient` — speaks Helling envelope format, used for auth/users/schedules/etc.
- `incusClient` — speaks Incus native format, used for instances/storage/networks/etc.
- `podmanClient` — speaks Podman native format, used for containers/images/volumes/etc.

## Consequences

**Easier:**
- Zero overhead on proxied requests (pass-through)
- No type synchronization between Helling and upstream response schemas
- Upstream API documentation directly applicable to Helling proxy endpoints
- Proxy implementation is trivial (~200 lines)

**Harder:**
- Frontend handles three response formats (requires three typed API clients)
- Can't add metadata to proxied responses (e.g., Helling tags on Incus instances must be fetched separately or read from Incus `user.*` config keys)
- Error format differs between Helling endpoints and proxied endpoints
