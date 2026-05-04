# Security Policy

## Reporting a vulnerability

**Don't open a public GitHub issue.** Helling is alpha software. Even minor bugs can have security implications because the daemon talks to Incus.

Instead:

1. Email **<security@bizarre.industries>** with the details. PGP key fingerprint: TBD (will be published before v0.1.0).
2. Include: affected version, reproduction steps, expected vs actual behavior, and an estimate of impact.
3. We'll acknowledge within 72 hours.

## Supported versions

Helling is pre-1.0. Only the most recent tagged release is supported. Once v1.0 ships, we'll publish a versioned support window here.

| Version | Supported         |
| ------- | ----------------- |
| < 0.1   | No (prerelease)   |
| 0.1.x   | Yes (latest only) |

## Disclosure timeline

- **Day 0:** Report received.
- **Day 1–3:** Acknowledgement sent. Initial triage.
- **Day 1–14:** Investigation and fix development.
- **Day 14–30:** Coordinated disclosure window. Patch tested.
- **Day 30 or earlier:** Patch released, advisory published.

We'll work with reporters on timing if active exploitation is suspected or if upstream coordination (e.g. with Incus) is needed.

## What's in scope

- The `hellingd` daemon
- The Caddy edge service and Helling's shipped Caddy configuration
- The `helling-cli` client
- The web dashboard (`web/`)
- Generated API client code (Go and TypeScript)
- Build pipeline tampering / supply-chain issues affecting our published artifacts

## What's out of scope

- Vulnerabilities in Incus, Podman, the kernel, or systemd themselves. Report those upstream.
- Issues that require root on the host (root already wins).
- Issues that require access to the `incus-admin` group (privilege escalation by definition).
- Denial of service via unauthenticated traffic flood; we expect operators to put rate limiting in front of Caddy for public deployments.
- Best-practices nags without an actual exploit path (we run our own scans).

## Bounty

No formal bounty program. We'll publicly credit reporters in advisories unless they ask not to be.

## See also

- [docs/standards/security.md](docs/standards/security.md) — security model and guarantees
- [docs/spec/architecture.md](docs/spec/architecture.md) — design overview, threat model boundaries
