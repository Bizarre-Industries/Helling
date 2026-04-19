# ADR-008: Function over beauty in UI

> Status: Accepted

## Context

Proxmox is ugly but manages 200 VMs effortlessly. Xen Orchestra looks modern but frustrates users with hidden functionality and excessive whitespace. Beautiful consumer UIs optimize for first impressions; admin UIs optimize for daily use.

## Decision

Tables not cards. Information density. No animations. Two-click maximum. Every pixel earns its place by showing data or enabling action. Desktop-first (admin dashboards are used on monitors, not phones).

## Consequences

- Higher information density = faster daily workflows
- No animations = instant page transitions
- Tables by default = scannable data for 100+ resources
- Less visually appealing to first-time visitors
- Power users are more productive
- Design rules codified in docs/design/philosophy.md
