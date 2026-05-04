# Helling

A self-hosted, single-node, Debian-first system container and VM management platform built on top of [Incus](https://linuxcontainers.org/incus/) and [Podman](https://podman.io/).

> **Status:** v0.1 in active development. APIs and behavior are not yet stable. Don't run this in production.

## What it is

Helling gives you a unified web UI, REST API, and CLI for managing containers and VMs on a single Debian host. It sits on top of Incus (which does the actual virtualization) and adds:

- A clean HTTP API generated from a single OpenAPI contract
- Local user authentication with sessions (no PAM, no LDAP needed)
- A React 19 + Ant Design 6 dashboard
- A typed Go and TypeScript client, both generated from the same spec
- Audit logging and async operation tracking

## What it isn't

- Not Kubernetes. Not trying to be.
- Not multi-node. Single host only in v0.1.
- Not a Docker replacement. Use Podman or Incus's application containers if you want OCI workloads.
- Not a managed cloud. You run it on your own hardware.

## Repository layout

```text
helling/
├── api/                    # OpenAPI spec — single source of truth
├── apps/
│   ├── hellingd/           # Backend daemon
│   ├── helling-cli/        # CLI client
├── web/                    # React 19 + Vite + antd dashboard
├── deploy/                 # Installer ISO profile, Caddy config, systemd units, packaging
├── docs/
│   ├── spec/               # Architecture, source-of-truth specs
│   ├── design/             # Design notes per feature
│   ├── standards/          # Coding and security standards
│   └── roadmap/            # Versioned roadmaps
├── go.work                 # Go workspace
├── Makefile                # All build/test/lint commands
├── AGENTS.md               # Guide for AI agents working in this repo
└── LICENSE                 # AGPL-3.0
```

## Required toolchain

- Go 1.26+ (1.26.2 recommended)
- Bun (frontend tooling and Hey API generation)
- Incus 6.0 LTS or later (for runtime; not required to build)
- golangci-lint, gofumpt, goimports

Install everything else with:

```bash
make dev-setup
```

Installer ISO usage is documented in [`docs/install.md`](docs/install.md).

## Quick start (development)

```bash
git clone https://github.com/Bizarre-Industries/helling.git
cd helling
make dev-setup
make generate         # OpenAPI → Go server, Go client, TS client
make build            # produces ./bin/hellingd and ./bin/helling
./bin/hellingd        # listens on /run/helling/api.sock by default
```

In another terminal:

```bash
make web-dev          # Vite dev server, proxies /v1 to hellingd
```

## Architecture

See [docs/spec/architecture.md](docs/spec/architecture.md) for the canonical design. Short version:

- `hellingd` is the backend. Listens on a Unix socket. Talks to Incus over its socket.
- Caddy terminates TLS and serves the web bundle. Forwards API calls to `hellingd`.
- `helling-cli` is a CLI that hits the same socket.
- SQLite for Helling's own state (users, sessions, operations). Incus is the source of truth for instance state.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). The TL;DR:

1. Read [docs/spec/architecture.md](docs/spec/architecture.md) before changing behavior.
2. Spec changes go through `api/openapi.yaml` first, then `make generate`.
3. `make check` must pass before opening a PR.

## License

[AGPL-3.0](LICENSE). If you run a modified version of Helling and let other people interact with it over a network, you must offer them the source code.

If AGPL doesn't work for your use case, get in touch.

## Project status and roadmap

- v0.1: see [docs/v0.1.md](docs/v0.1.md). Minimum viable platform — auth + container CRUD + dashboard shell.
- Beyond v0.1: VM support, storage volumes, network management, multi-user RBAC tied to Incus projects, OIDC SSO. Not committed; will be tracked in versioned roadmap docs.

## Naming

Helling is named after a [submarine launching ramp](https://en.wikipedia.org/wiki/Slipway) — the structure that gets boats from dry land into water. Same idea: get workloads from definition to running.
