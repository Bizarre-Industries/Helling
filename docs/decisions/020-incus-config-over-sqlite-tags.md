# ADR-020: Incus Config Keys Over SQLite for Resource Tags

> Status: Accepted

## Context

The previous architecture stored tags in SQLite tables (`tags`, `tag_resources`) and attempted to synchronize them with Incus resources. This created:

- Two sources of truth for the same data
- Sync bugs when Incus state changed outside of Helling (e.g., via `incus` CLI)
- Custom Go code for tag CRUD, sync, and conflict resolution
- Tags didn't survive Incus instance migration or backup/restore (SQLite was separate)

Incus natively supports arbitrary key-value metadata on every resource via `user.*` config keys. These keys:

- Travel with the resource during migration, backup, restore, and cluster operations
- Are queryable via the Incus API
- Are visible in `incus config show`
- Sync automatically across cluster nodes (Incus handles this)

## Decision

Tags are stored as `user.tag.<name>=<color>` config keys on Incus resources. No SQLite tag tables.

Tag a resource:
```bash
incus config set vm-web-1 user.tag.production=blue
incus config set vm-web-1 user.tag.critical=red
```

List tags on a resource: read `user.tag.*` keys from the resource config (available in the proxied response).

List all unique tags: query Incus for all resources with `user.tag.*` keys (available via Incus API filtering).

For Podman resources: Podman supports labels on containers, images, volumes. Tags map to labels with a `helling.tag.` prefix:
```bash
podman container label add mycontainer helling.tag.production=blue
```

The dashboard reads tags from the proxied response (Incus `config` or Podman `labels`). No separate tag API needed — the data is already in the resource response.

Helling-specific tag operations (bulk tagging, tag management UI) send requests through the proxy to set Incus config keys or Podman labels.

## Consequences

**Easier:**
- One source of truth (the resource itself carries its tags)
- Tags survive migration, backup, restore, cluster operations
- No sync bugs between SQLite and Incus
- No SQLite tag tables, no tag CRUD endpoints
- `incus` CLI users can tag resources directly
- Cluster-wide tag consistency guaranteed by Incus

**Harder:**
- Querying "all resources with tag X" requires filtering Incus API responses (Incus supports `?filter=config.user.tag.X` but it's not as fast as a SQL index)
- Tag colors/metadata are limited to what fits in the config value string
- Podman label API is different from Incus config API (frontend handles both)
