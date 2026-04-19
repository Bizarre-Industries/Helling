# Helling

Helling is a Debian-first virtualization management platform built around a proxy-first control plane for Incus and Podman.

## Status

Pre-alpha. Documentation and architecture are being stabilized before full implementation.

## Why Helling

- Keep upstream APIs native instead of re-wrapping everything.
- Use clear isolation boundaries for auth and operations.
- Prioritize operational simplicity over feature sprawl.

## Scope Snapshot (v0.1)

- PAM + JWT + TOTP auth with per-user Incus TLS trust identity
- Incus and Podman proxy integration
- noVNC for VM browser console
- k3s via cloud-init for Kubernetes provisioning

Deferred from v0.1:

- LDAP/OIDC/WebAuthn
- CAPN controller path
- MicroVM runtime path

## Try It

Not available yet.

## Documentation

- Specs: docs/spec/
- Decisions: docs/decisions/
- Standards: docs/standards/
- Roadmap: docs/roadmap/

## Contributing

See CONTRIBUTING.md.

## License

AGPL-3.0-or-later.
