# ADR-028: Unix socket between proxy and daemon

> Status: Accepted (2026-04-19)

## Context

Local inter-process communication between `helling-proxy` and `hellingd` needs low overhead and strong local identity signals.

## Decision

Use a Unix domain socket for proxy-to-daemon communication, with peer credential validation (`SO_PEERCRED` or platform equivalent) where supported.

## Consequences

- No exposed localhost TCP control plane by default
- Simple file-permission based access control
- Easier service hardening with systemd and filesystem ACLs
