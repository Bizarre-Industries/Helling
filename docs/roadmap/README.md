# Helling Roadmap Documentation

This directory contains Helling's implementation roadmap and planning documents.

## Documents Overview

### 📋 [plan.md](./plan.md) - High-Level Roadmap

**Purpose:** Strategic overview of what to build and when
**Audience:** Project leads, stakeholders, new contributors

**Contains:**

- Architecture Decisions (ADRs 001-025)
- Automation & Tooling Index (28 tools)
- Version gates and feature lists for v0.1-v1.0
- Post-v1 feature ideas

**Use this when:** You need to understand the big picture, feature priorities, or ADR rationale.

---

### 📖 [implementation-guide.md](./implementation-guide.md) - Detailed Implementation Steps

**Purpose:** Actionable, code-level implementation instructions
**Audience:** Developers implementing features

**Contains:**

- Step-by-step implementation instructions
- Code examples and file structures
- Configuration snippets
- Verification commands for each step
- Current state assessment

**Use this when:** You're actively implementing a feature and need specific guidance on how to build it.

---

### ✅ [checklist.md](./checklist.md) - Verification Gates

**Purpose:** Verification criteria for each release
**Audience:** QA, release managers, developers

**Contains:**

- Testable verification commands for each version gate
- Build/test/lint requirements
- Code hygiene checks
- Release criteria

**Use this when:** You're validating that a version is ready to ship.

---

## Quick Start for Developers

### Starting v0.1.0-alpha Implementation

1. **Understand the architecture:** Read [plan.md](./plan.md) ADRs section
2. **Follow implementation steps:** Read [implementation-guide.md](./implementation-guide.md) Phase 1
3. **Verify your work:** Use [checklist.md](./checklist.md) v0.1.0-alpha section

### Priority Order for v0.1.0-alpha

Follow this sequence from [implementation-guide.md](./implementation-guide.md):

1. **OpenAPI Spec** (Section 1.1) - CRITICAL
   - Complete `api/openapi.yaml` with all ~40 endpoints
   - Define schemas, pagination, error responses

2. **Code Generation** (Section 1.2) - CRITICAL
   - Set up oapi-codegen for backend + CLI
   - Set up orval for frontend
   - Wire into Makefile

3. **Proxy Middleware** (Section 1.3) - CRITICAL
   - Implement core proxy in `internal/proxy/`
   - Add JWT/RBAC/audit middleware
   - Wire to router

4. **Auth Handlers** (Section 1.4) - HIGH
   - Implement login, setup, JWT, TOTP
   - PAM integration

5. **Frontend Integration** (Section 1.5) - HIGH
   - Create three API clients (Helling, Incus, Podman)
   - Update dashboard pages with real data
   - Replace mocks with actual API calls

6. **Code Cleanup** (Section 1.6) - MEDIUM
   - Delete unused dependencies
   - Clean up TODOs

7. **Verification** (Section 1.7) - CRITICAL
   - Run full checklist from [checklist.md](./checklist.md)
   - Ensure all gates pass

---

## Architecture Quick Reference

**Core Principle:** Proxy-first (ADR-014)

- ~300 lines of proxy code replaces ~150 endpoint handlers
- Incus/Podman APIs exposed natively at `/api/incus/*` and `/api/podman/*`
- Only ~40 Helling-specific endpoints for auth, scheduling, webhooks, etc.

**Key ADRs:**

- ADR-014: Proxy over per-endpoint handlers
- ADR-016: CLI for Helling features only (~15 commands, not ~392)
- ADR-017: systemd timers over in-process cron
- ADR-018: Shell out over Go libraries
- ADR-021: ISO-only deployment

**Dependencies:** Keep backend dependencies minimal and justify each addition against proxy-first architecture.

---

## Version Overview

| Version      | Gate                                    | Focus                                     |
| ------------ | --------------------------------------- | ----------------------------------------- |
| v0.1.0-alpha | Dashboard shows real data from proxies  | Foundation: proxy, auth, OpenAPI spec     |
| v0.1.0-beta  | Create VM -> SPICE VGA console works    | Consoles: WebSocket, SPICE, xterm.js      |
| v0.2.0       | Schedule creates backup, firewall works | Platform: schedules, webhooks, firewall   |
| v0.3.0       | Prometheus scrapes /metrics             | Observability: metrics, notifications     |
| v0.4.0       | K8s cluster created, BMC powers on      | Integration: K8s, BMC, clustering         |
| v0.5.0       | LDAP user logs in, quota enforced       | Enterprise: LDAP, OIDC, WebAuthn          |
| v0.8.0       | All E2E pass, p95 <200ms                | Hardening: fuzzing, performance, security |
| v1.0.0       | ISO boots and installs                  | Release: packaging, SBOM, signing         |

---

## Questions?

- **Architecture questions:** See [docs/spec/architecture.md](../spec/architecture.md)
- **API design:** See [docs/spec/api.md](../spec/api.md)
- **Automation details:** See [docs/design/full-automation-pipeline.md](../design/full-automation-pipeline.md)
- **Contributing:** See [CONTRIBUTING.md](../../CONTRIBUTING.md)

---

**Last Updated:** 2026-04-15
