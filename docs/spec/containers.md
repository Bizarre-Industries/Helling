# Containers Specification

All Podman container operations go through the proxy (ADR-014). hellingd forwards requests from `/api/podman/*` to the Podman Unix socket at `/run/podman/podman.sock`.

For the full Podman API, see [Podman API documentation](https://docs.podman.io/en/latest/_static/api.html).

## What's Available via Proxy

Everything the Podman libpod API provides:

- Containers: CRUD, lifecycle, logs, exec, stats, top, diff, commit, export, wait
- Pods: CRUD, lifecycle, stats
- Images: list, pull, push, tag, build, prune, search
- Volumes: CRUD
- Networks: CRUD
- Secrets: CRUD
- System: info, prune, disk usage
- Compose: managed via `podman compose` (hellingd shells out)

## Helling Additions

### Exec Terminal

WebSocket exec sessions go through the proxy with WebSocket upgrade support. The dashboard uses xterm.js to render the terminal.

### Compose Stacks

Compose stack management shells out to `podman compose` (ADR-018). The Helling API provides stack tracking:

```
POST /api/v1/stacks          → writes compose file, runs podman compose up
GET  /api/v1/stacks          → lists tracked stacks
DELETE /api/v1/stacks/{name} → runs podman compose down, removes files
```

### App Templates

One-click deployment of common applications as compose stacks. See docs/spec/compute.md for details.

### Host Firewall

Podman container networking uses host nftables rules. These are managed by Helling's host firewall API (ADR-018), not Incus ACLs (which are for VMs/CTs).
