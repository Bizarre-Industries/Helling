# ADR-007: AGPL-3.0-or-later License

> Status: Accepted

## Context

Needed an open source license that prevents proprietary forks while allowing community use. MIT/Apache allow closed-source forks. GPL-3.0 covers binaries but not SaaS. AGPL-3.0 closes the SaaS loophole.

## Decision

License Helling under AGPL-3.0-or-later with DCO (Developer Certificate of Origin) for contributions.

## Consequences

- Anyone hosting Helling as a service must publish their modifications
- Community contributions under DCO (not CLA)
- Compatible with: MIT, BSD, Apache-2.0, LGPL dependencies
- Incompatible with: GPL-2.0-only, SSPL, BSL, proprietary dependencies
- Enterprise users who can't accept AGPL need a commercial license (future consideration)
