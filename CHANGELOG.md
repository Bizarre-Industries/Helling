# Changelog

All notable changes to Helling are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [unreleased]


### Bug fixes

- Stabilize v0.1 backend gates ([`00eb26e`])
- Harden v0.1 auth boundaries ([`f5a6080`])
- Scope local verification gates ([`1b10f51`])


### Build

- Bump github.com/go-jose/go-jose/v4 ([`c0d74b0`])


### CI

- Bump github/codeql-action in the github-official group ([`b4305e4`])
- Bump crate-ci/typos in the linters-and-build group ([`403bad5`])
- Bump DavidAnson/markdownlint-cli2-action from 23.0.0 to 23.1.0 ([`396d58d`])


### Chore

- Tighten agent tooling — slash commands, skills, hook fixes ([`2f82dd2`])


### Features

- Wire Incus instance + operations endpoints (Stage 3) ([`a07f83d`])
## [0.1.0-alpha] — 2026-05-03


### Bug fixes

- Upgrade codeql-action to v4 and guard grype against missing go.mod ([`f2c2424`])
- Improve OpenSSF Scorecard from 5.7 toward 7.0 ([`832a331`])
- Adjust branch formatting in workflow triggers and update Prettier command ([`769690c`])
- Add debug flag to vacuum lint command for enhanced output ([`aefa52c`])
- Satisfy vacuum lint — descriptions and examples ([`a913611`])
- Update vacuum installation process and add SHA256 verification ([`5d7f8aa`])
- Markdownlint in bug_report.md (MD007/MD032) ([`0352080`])
- Move github.com/google/uuid to required dependencies ([`dfd8d36`])
- Correct baseUrl and ensure consistent casing in file names ([`feeecce`])
- Phase 1 audit fix-pass — auth, error boundary, a11y, build ([`1992e3c`])
- Close v0.1 vertical-slice gates for push gate ([`c481147`])
- Align hooks + settings to Helling ([`1df8bd8`])


### Build

- Fix generate-web script + go.work-aware lint ([`95172e6`])


### CI

- Bump the security-scanners group with 2 updates ([`89b08d8`])
- Bump github/codeql-action in the github-official group ([`5272529`])
- Upgrade codeql-action from v3 to v4.35.2 ([`12f0abb`])
- Pass GITLEAKS_LICENSE secret to gitleaks-action ([`18c6e9d`])
- Add GITHUB_TOKEN to gitleaks secret scan job ([`6a64b36`])
- Optimize workflows and add ci reference ([`7185f9a`])
- Run spelling check unconditionally ([`7d1fb47`])
- Bump DavidAnson/markdownlint-cli2-action from 22.0.0 to 23.0.0 ([`155cfc5`])
- Bump crate-ci/typos in the linters-and-build group ([`c61ac98`])
- Bump actions/setup-go in the github-official group ([`02ade1a`])
- Add make-check workflow ([`266c4a3`])
- Pin actions to commit SHAs (ADR-026) ([`68f9aa1`])
- Add bun install + per-module govulncheck ([`372ada9`])
- Bump golangci-lint to v1.65.0 (Go 1.26 support) ([`58c2528`])
- Use golangci-lint v1.64.8 (last v1.x release; matches local toolchain) ([`2da8176`])
- Install golangci-lint via 'go install' so runner Go (1.26) compiles it ([`9c7e00f`])


### Chore

- Remove obsolete checksum file for OpenAPI ([`10aad2c`])
- Add .task/ to .gitignore ([`8ebc5fa`])
- Ignore historical gitleaks false positive ([`561dd19`])
- Update GitHub Actions workflows and documentation ([`b0a1062`])
- Update .gitignore to include Claude cache and plans directories ([`5935d49`])
- Drop helling-proxy/helling-agent references ([`56765ea`])
- Bump vite ([`7916293`])
- Annotate auth exceptions with shipping status (PR H) ([`c441308`])
- Close hygiene + automation tail (Phase A) ([`d18792f`])
- Per-app module layout (hellingd, helling-cli) ([`8f90feb`])
- Ignore .claude/.last-shipped-tag hook artifact ([`ed5bfc8`])


### Documentation

- Implement audit fix pass with ADR-050 and locked decisions ([`36b1bff`])
- Complete ADR amendments and per-doc framework updates ([`b237c5e`])
- Final ADR amendments for audit consistency ([`c64ff96`])
- Close remaining Phase 0 audit contradictions\n\n- ADR-025: switch apt suite example to helling-v1 and align update command\n- Checklist: use unversioned Podman proxy path and remove deferred thumbnail gate\n- Users/WebUI specs: make Roles and Permissions read-only for fixed v0.1 roles\n- Backup/upgrade specs: include /etc/helling key material in backup and rollback\n- Magic feature 8: move changelog storage to journal per ADR-019, defer revert to v0.5+\n- Logs/local-dev/identity: update caddy naming and stale edge-service references\n- Philosophy/tokens/webui/automation docs: normalize @xterm/xterm and current codegen wording ([`817747e`])
- Refine ADR-009 Trivy narrative and strike CH from ADR-023 ([`a76580d`])
- Apply review feedback to ADR-009 and ADR-023 ([`296d5e9`])
- Close api/cli parity gaps (Stacks, user 2fa, BMC v0.4 tag) ([`65d66d8`])
- Address review — move Stacks to deferred, mark 2FA deferred, fix api.md ref in cli.md ([`8c200e1`])
- Magic.md honesty pass + settings.md defer appearance/notifications ([`49e5776`])
- Address reviewer feedback on magic.md and settings.md scope consistency ([`643995d`])
- Stack audit fixes — Incus CVE pin, webui regression, parity sync, k8s datastore, tools rewrite ([`0f4b59e`])
- Close 6 Phase 0 parity gaps with explicit exceptions ([`472f254`])
- Stack audit fixes — Incus CVE pin, webui regression, parity sync, k8s datastore, tools rewrite ([`353d5d5`])
- Add API Operation Coverage footnotes to cli.md + webui-spec.md ([`1269657`])
- Fix dangling refs, stale pam/v2 paths, drop ADR-033 citation ([`ae1c637`])
- Close ADR-019 follow-ups (new audit.md spec + xrefs) ([`00df9ed`])
- Escape underscores in observability.md field-name tokens ([`eb49f7a`])
- Cross-ref sweep for proxies.md and pam.md ([`a74193f`])
- Define auth.session_inactivity_timeout (30m default) ([`0911830`])
- Accept ADR-045 / 046 / 047 / 048 + propagate tool pins ([`852aa3c`])
- Update references for ADR-045, ADR-046, and ADR-047 in various documents ([`0d40de5`])
- Mark CA + WS proxy gates done (PR Q) ([`d70e729`])
- Close per-user mTLS gate (PR R sync) ([`de3c8e6`])
- Reflect CLI completeness (PR X) ([`4b195bb`])
- Capture WebUI audit 2026-04-27 + ADR-051 stack lock ([`5fd90aa`])
- Fix stale error-boundary.jsx → .tsx in README ([`68fb40b`])
- Mark Phase 1 + Phase 2D/2E shipped, add Phase 2 block ([`a3cfd21`])
- Mark Phase 2A auth subfolder complete; chunk Phase 2A by section ([`7a69122`])
- Import full audit, expand checklist Phases 3-6, add F-ID tracking hook ([`27ba17f`])
- Generalize tracking gate to all docs; strip Claude/AI references; add SessionStart sanity hook ([`16ceda0`])
- Annotate WebUI Phase 2/3 F-IDs with scaffold-landed notes ([`fc9fdda`])
- Tick Phase 2A search subfolder (c98ec3f) ([`4409074`])
- Realign architecture/standards/roadmap to v0.1 vertical-slice scope ([`ce0a097`])


### Features

- Update web UI dependencies and enhance console features ([`b55f42f`])
- Revise compliance documentation and standards ([`2f65fac`])
- Remove deprecated CI workflows and security checks ([`60c12db`])
- Update architecture decisions and documentation for v0.1, including Caddy as edge service, SQL-first approach with sqlc and goose, and age for secret management ([`fd720ce`])
- Add Phase 0 API-CLI-WebUI Parity Matrix and error handling specifications ([`201a2c7`])
- Update implementation guide and API versioning policy ([`94ccf49`])
- Add custom OpenAPI linting ruleset and quality assurance standards ([`777e4fb`])
- Accept ADR-043 for Huma integration and update related documentation ([`89da143`])
- Update OpenAPI documentation and migration manifest for Huma integration ([`5853a35`])
- Add build tooling, quality gates, and docs expansion ([`6201e8c`])
- Expand design documentation with new pages and detailed specifications ([`c1a95b2`])
- Refactor ADRs for non-root hellingd and systemd integration ([`25da9bf`])
- Implement Helling API with health check endpoint ([`bda6f66`])
- Add authentication and user management endpoints ([`28615f2`])
- Scaffold helling-cli with version and completion ([`d6d328d`])
- Add auth logout and refresh endpoints ([`e619238`])
- Scaffold auth setup, mfa, totp, and api-token endpoints ([`a9c96ef`])
- Scaffold users crud endpoints ([`c7a52a8`])
- Scaffold schedules domain endpoints ([`3e254fd`])
- Scaffold webhooks domain endpoints ([`ea8b881`])
- Implement Kubernetes management endpoints and tests ([`b71f5a3`])
- Scaffold kubernetes domain endpoints ([`7895388`])
- Scaffold web and port claude design index.html ([`d74893c`])
- Scaffold system, firewall, audit, and events endpoints ([`14a06b8`])
- Refactor code structure for improved readability and maintainability ([`441b2fd`])
- Wire hey-api generated client + tanstack query provider ([`6a3e1f5`])
- Sqlite foundation — goose migrations, pragmas, pool ([`61e89c8`])
- Add hey-api client-fetch and tanstack react-query dependencies ([`34f7986`])
- Wire db.Open into daemon boot + migrate tasks ([`e7abff2`])
- Implement JWT signing and verification with Ed25519 keys ([`05e1578`])
- Implement TOTP-based MFA: Add TOTP enrollment, verification, and recovery code functionality; enhance API token management; and update client configuration for Helling runtime. ([`7bd4cbf`])
- Authenticated reverse-proxy middleware for Incus + Podman ([`b5bb769`])
- Auth + compute subcommands ([`909aa89`])
- Dashboard counts from Incus + Podman via proxy (PR G) ([`0cdbb89`])
- Real userList/Create/Get/Delete/SetScope handlers (PR I) ([`6f63577`])
- Real info/hardware/config/upgrade/diagnostics (PR J) ([`b957b1e`])
- Real CRUD + test delivery (PR K) ([`2abf27c`])
- Internal CA + user cert scaffold (PR O-1) ([`5cea56a`])
- User_certificates schema + repo methods (PR O-2) ([`3544aa9`])
- Wire internal CA bootstrap on startup (PR O-3) ([`4c31a94`])
- WebSocket upgrade passthrough (PR O-4) ([`25becf7`])
- Wire user-cert issuance on userCreate (PR O-5) ([`2576318`])
- Per-user TLS Transport selector (PR R / O-6) ([`e9a6418`])
- Cert renewal worker (PR S) ([`faf5850`])
- User + webhook + system subcommands (PR T) ([`1ee4dcb`])
- Audit + events + system health (PR U) ([`df8b26d`])
- Auth token subcommand (PR V) ([`f1ff973`])
- Auth mfa subcommand (PR W) ([`0d49300`])
- Adopt Parallels Desktop as primary macOS dev/test VM ([`7a7371c`])
- Rate limiter, integration target, frontend hooks (Phase B) ([`5de6f09`])
- Phase C/D foundations — stores, normalizer, QueryStateView, events poll ([`1367cad`])
- Add PageBMC component and integrate into app.jsx ([`30ab4f5`])
- Rewrite openapi.yaml to v0.1 vertical-slice surface ([`4fc4bf1`])
- Add mark-replanned, never-guess, and replan-on-tag scripts for version management ([`e7cede8`])
- Wire login/logout/me + bootstrap admin (Stage 2) ([`237c92d`])


### Performance

- Icon barrel — bundle 1.26MB → 482KB (audit F-30 + F-51) ([`8a08cc5`])
- Lazy-load extracted pages with Suspense (Phase 2C, F-29 partial) ([`5c449e0`])


### Refactor

- Update documentation and specifications for Helling v0.1 ([`03f986f`])
- Simplify markdownlint commands and enhance operation ID variant matching in check-parity script ([`309058c`])
- Standardize quotes and formatting in configuration files ([`d12c61a`])
- Streamline roadmap documentation and remove outdated implementation guide ([`6d997ce`])
- Extract PageLogin to web/src/pages/auth/login.tsx (Phase 2A pilot) ([`ceb9cf4`])
- Extract PageSetup + Switch primitive (Phase 2A continues) ([`1912bfc`])
- Extract PageAudit + legacy/mocks.ts shim (Phase 2A continues) ([`8d1a3bf`])
- Extract PageLogs to web/src/pages/admin/logs.tsx (Phase 2A continues) ([`1892440`])
- Extract PageOps and add warning handling to operations page (Phase 2A continues) ([`14caf91`])
- Extract PageUsers to web/src/pages/admin/users.tsx (Phase 2A continues) ([`19a974e`])
- Extract PageSearch + PageSearchResults to pages/search/ (Phase 2A) ([`c98ec3f`])
- Extract PageNetworking + Copyable primitive (Phase 2A) ([`789c69e`])
- Extract PageSchedules to pages/schedules/ + tick F-29 (Phase 2A) ([`27bc66d`])
- Lowercase module path 'helling' (was mixed Helling/helling) ([`cb5f3d4`])
- Replace huma+goose+jwt stack with chi+sql.DB scaffold ([`564d4b0`])


### Style

- Prettier auto-format 4 files ([`fdfcf4c`])
- Prettier auto-format patch-applied files ([`bc2d3ef`])
- Standardize quotes in hey-api configuration and reorder imports in client.ts ([`f349f71`])
- Re-sort icons.ts imports alphabetically (linter) ([`008d5e0`])


### Tests

- Auth repo + real-handler integration coverage ([`2af11ff`])
- Cover TOTP/recovery/api-token CRUD + close PR C parity ([`ace771c`])
- Vitest scaffold + 3 smoke tests guard Phase 1 (audit F-40) ([`ab83985`])
- Replace typo-flagged 'abd' with 'xyz' in HashToken collision test ([`506db85`])

