# Graph Report - Helling  (2026-07-19)

## Corpus Check
- 441 files · ~334,552 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3683 nodes · 7803 edges · 231 communities (177 shown, 54 thin omitted)
- Extraction: 90% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 752 edges (avg confidence: 0.78)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `e21148b5`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- server.gen.go
- index.ts
- ResponseWriter
- react-query.gen.ts
- useTempConfigDir
- UserTrustService
- UnauthorizedJSONResponse
- OperationHandle
- New
- firewall_handlers.go
- schemas.gen.ts
- devcontainer.json
- trust_helper.go
- shell.jsx
- Context
- RenderScheduleUnits
- ForbiddenJSONResponse
- app.jsx
- .handleLogin
- New
- UUID
- newTestStore
- events_handlers.go
- Hash
- models.go
- Context
- UserFromContext
- mocks.ts
- writeError
- bodySerializer.gen.ts
- pages.jsx
- compilerOptions
- BadRequestJSONResponse
- Open
- helling-debian.pkr.hcl
- utils.gen.ts
- user.go
- .handleCreateSchedule
- server.go
- ui-store.ts
- auth.go
- proxy.go
- QueryStateView.tsx
- audit_handlers.go
- run
- .handleInstanceStateChange
- loginCookie
- compilerOptions
- .handleCreateUser
- .auditForActorWithPolicyReason
- Store
- writeJSON
- queries.ts
- index.ts
- window-globals.ts
- Client
- Context
- newTestServer
- RateLimiter
- auth_audit_test.go
- AGENTS.md
- devDependencies
- Config
- helling-first-boot.sh
- build-iso.sh
- install-tools.sh
- setup.tsx
- system-store.ts
- JWTSigner
- main.tsx
- Operation
- Context
- dependencies
- types.gen.ts
- icon.tsx
- ConflictJSONResponse
- newRootCmd
- params.gen.ts
- infra.jsx
- NewSystemCmd
- New
- Record
- auth_ext_handlers.go
- newSetupTokenServer
- Context
- run
- suspicious
- biome.json
- rules
- userClient
- Context
- 008_v02_platform_core.sql
- Context
- ignore
- check-parity.sh
- scripts
- login.tsx
- normalize.ts
- NewFirewallCmd
- DialEventsWithClient
- ToInstance
- doGet
- BMCEndpoint
- fix_file
- AuthLogin200JSONResponse
- .ExportAuditEvents
- ListAuditEventsParams
- .ListInstances
- .ListOperations
- .ListSchedules
- NewEventsCmd
- .newReverseProxy
- NotFoundJSONResponse
- parallels-vm-bootstrap.sh
- parallels-vm-deploy-rsync.sh
- sync-agent-tooling.sh
- use-events-stream.ts
- index.tsx
- FirewallRuleCreateRequest
- unit_linker_test.go
- logs.tsx
- .mcp.json
- parallels-vm-build-dev.sh
- parallels-vm-deploy-deb.sh
- plan-next-version.sh
- Interceptors
- queryKeySerializer.gen.ts
- index.tsx
- audit.tsx
- .ListEvents
- newAuthMfaCmd
- newAuthTokenCmd
- config_test.go
- NewWebhookCmd
- provision-helling-dev.sh
- formatter
- formatter
- ExportAuditEvents200ApplicationxNdjsonResponse
- loadOrCreateAgeIdentity
- events.go
- a11y
- noExcessiveCognitiveComplexity
- package.json
- events.gen.ts
- index.tsx
- client.gen.ts
- TestPodmanRequestAllowed
- kubernetes_handlers.go
- 001_initial.sql
- .repairV02Schema
- never-guess.sh
- correctness
- index.tsx
- vite-env.d.ts
- main.go
- main.go
- main.go
- bmc_handlers.go
- .DBSizeBytes
- 003_totp.sql
- 005_webhooks.sql
- config
- check-coverage.sh
- run-peer-agent.sh
- tsconfig.json
- notification_handlers.go
- 002_api_tokens.sql
- 004_schedules.sql
- 006_bmc.sql
- 007_kubernetes.sql
- mark-replanned.sh
- never-guess.sh
- replan-on-tag.sh
- mark-replanned.sh
- never-guess.sh
- replan-on-tag.sh
- clean
- mark-replanned.sh
- post-bash-version-shipped.sh
- replan-on-tag.sh
- check-docs-sync.sh
- check-iso-config.sh
- post-bash-version-shipped.sh
- docs-snapshot.sh
- fakeOpHandle
- @testing-library/user-event
- typescript
- vitest
- github.com/Bizarre-Industries/helling/apps/helling-cli
- github.com/Bizarre-Industries/helling/apps/hellingd
- UpdateUserScope200JSONResponse
- @types/react-dom
- Section 07 — Accessibility
- Section 09 — Operator ergonomics
- Section 10 — Security & supply chain
- UnescapedCookieParamError
- TestEventsSSEStreamsNewEvents
- ResponseWriter
- Time
- Client
- Cookie
- Response
- Store
- T
- Config
- .ServeHTTP
- clientIP
- Context
- auth_ext_handlers.go
- newSetupTokenServer
- DialEventsWithClient
- config_test.go
- totpAt
- loadOrCreateAgeIdentity
- Duration
- Mutex
- Time
- Context
- FileMode
- X25519Identity

## God Nodes (most connected - your core abstractions)
1. `writeError()` - 76 edges
2. `useTempConfigDir()` - 63 edges
3. `Handler()` - 57 edges
4. `seedProfile()` - 54 edges
5. `ServerInterfaceWrapper` - 45 edges
6. `strictHandler` - 44 edges
7. `Unimplemented` - 41 edges
8. `writeJSON()` - 40 edges
9. `userClient()` - 39 edges
10. `UnauthorizedJSONResponse` - 38 edges

## Surprising Connections (you probably didn't know these)
- `New()` --calls--> `NewRateLimiter()`  [INFERRED]
  apps/hellingd/internal/server/server.go → apps/hellingd/internal/auth/ratelimit.go
- `TestLoadOrCreateCertificateAuthorityPersistsEncryptedKey()` --calls--> `Open()`  [INFERRED]
  apps/hellingd/internal/incus/certificates_test.go → apps/hellingd/internal/store/store.go
- `readLockedDownJSONFile()` --calls--> `Open()`  [INFERRED]
  apps/hellingd/internal/incus/trust_helper.go → apps/hellingd/internal/store/store.go
- `newIncusTrustTestStore()` --calls--> `Open()`  [INFERRED]
  apps/hellingd/internal/incus/user_trust_test.go → apps/hellingd/internal/store/store.go
- `newTestStore()` --calls--> `Open()`  [INFERRED]
  apps/hellingd/internal/poller/poller_test.go → apps/hellingd/internal/store/store.go

## Import Cycles
- None detected.

## Communities (231 total, 54 thin omitted)

### Community 0 - "server.gen.go"
Cohesion: 0.01
Nodes (188): AuditEvent, AuditEventListEnvelope, AuditEventOutcome, AuthLogin202JSONResponse, AuthLogin204Response, AuthLogin204ResponseHeaders, AuthLogin429JSONResponse, AuthLoginRequestObject (+180 more)

### Community 1 - "index.ts"
Cohesion: 0.03
Nodes (190): RFC-3339, deleteHostFirewallRule(), deleteSchedule(), deleteWebhook(), listHostFirewallRules(), listSchedules(), Options, testWebhook() (+182 more)

### Community 2 - "ResponseWriter"
Cohesion: 0.04
Nodes (29): ChiServerOptions, CreateInstance409JSONResponse, CreateUser409JSONResponse, GetHealth200JSONResponse, GetInstance404JSONResponse, GetOperation404JSONResponse, GetVersion200JSONResponse, ListInstances401JSONResponse (+21 more)

### Community 3 - "react-query.gen.ts"
Cohesion: 0.02
Nodes (140): authLogin(), authMfaComplete(), createHostFirewallRule(), createInstance(), createSchedule(), createUser(), createWebhook(), deleteInstance() (+132 more)

### Community 4 - "useTempConfigDir"
Cohesion: 0.05
Nodes (112): T, runAudit(), TestAuditExport_CSVFlag(), TestAuditExport_JSONDefault(), TestAuditQuery_ForwardsFilters(), fakeHellingd(), Server, T (+104 more)

### Community 5 - "UserTrustService"
Cohesion: 0.09
Nodes (33): certificateSerial(), createAndStoreCertificateAuthority(), Certificate, PrivateKey, Time, IssueUserCertificate(), IssueUserCertificateWithCA(), LoadOrCreateCertificateAuthority() (+25 more)

### Community 6 - "UnauthorizedJSONResponse"
Cohesion: 0.03
Nodes (37): AuthLogin401JSONResponse, AuthMfaComplete401JSONResponse, AuthSetup401JSONResponse, CreateHostFirewallRule401JSONResponse, CreateInstance401JSONResponse, CreateSchedule401JSONResponse, CreateUser401JSONResponse, CreateWebhook401JSONResponse (+29 more)

### Community 7 - "OperationHandle"
Cohesion: 0.06
Nodes (46): Connect(), decodeResponse(), Context, Instance, Request, Response, Transport, newRequest() (+38 more)

### Community 8 - "New"
Cohesion: 0.08
Nodes (23): Context, NullString, Store, Time, nullableInt(), nullableString(), ptrString(), scanFirewallRule() (+15 more)

### Community 9 - "firewall_handlers.go"
Cohesion: 0.08
Nodes (45): Context, Writer, T, TestHelperDeleteRequiresHellingOwnedComment(), TestValidateNFTArgsAllowsHellingRuleShape(), TestValidateNFTArgsRejectsArbitraryNFT(), validAction(), validateAddArgs() (+37 more)

### Community 10 - "schemas.gen.ts"
Cohesion: 0.04
Nodes (46): AuditEventListEnvelopeSchema, AuditEventSchema, AuthTokenEnvelopeSchema, AuthTokenResponseSchema, ErrorSchema, EventEnvelopeSchema, EventListEnvelopeSchema, EventSchema (+38 more)

### Community 11 - "devcontainer.json"
Cohesion: 0.04
Nodes (44): label, onAutoForward, label, onAutoForward, label, onAutoForward, containerEnv, GOFLAGS (+36 more)

### Community 12 - "trust_helper.go"
Cohesion: 0.12
Nodes (31): bytesTrimSpace(), fileUID(), Context, parseTrustRequestName(), policyAllowsAction(), policyAllowsProject(), readLockedDownJSONFile(), assertTrustCommandCalls() (+23 more)

### Community 13 - "shell.jsx"
Cohesion: 0.07
Nodes (33): InstanceSnapshots(), InstanceTasks(), PageBackups(), PageInstanceDetail(), PageInstances(), ALERTS, BACKUPS, CLUSTERS (+25 more)

### Community 14 - "Context"
Cohesion: 0.11
Nodes (19): Context, Event, Server, Webhook, webhookSubscribesToEvent(), claimEventOutbox(), claimOutboxEventIDs(), Context (+11 more)

### Community 15 - "RenderScheduleUnits"
Cohesion: 0.12
Nodes (23): containsControl(), containsControlOrSpace(), Context, RenderScheduleUnits(), T, TestRenderScheduleUnits(), TestRenderScheduleUnitsDefaultsToPackagedCLIPath(), TestRenderScheduleUnitsRejectsUnitInjection() (+15 more)

### Community 16 - "ForbiddenJSONResponse"
Cohesion: 0.05
Nodes (22): CreateHostFirewallRule403JSONResponse, CreateSchedule403JSONResponse, CreateUser403JSONResponse, CreateWebhook403JSONResponse, DeleteHostFirewallRule403JSONResponse, DeleteSchedule403JSONResponse, DeleteUser403JSONResponse, DeleteWebhook403JSONResponse (+14 more)

### Community 17 - "app.jsx"
Cohesion: 0.09
Nodes (30): ADR-0031, ADR-0037, clearAccessToken(), emitChange(), isAuthenticated(), setAccessToken(), subscribeAuthChange(), LogoutRequest (+22 more)

### Community 19 - ".handleLogin"
Cohesion: 0.25
Nodes (6): Request, User, wantsAuthJSON(), ResponseWriter, mfaChallenge, Server

### Community 20 - "New"
Cohesion: 0.05
Nodes (55): escapeMetricLabel(), Duration, Mutex, Request, ResponseWriter, Server, metricLabels(), newMetricsRegistry() (+47 more)

### Community 21 - "UUID"
Cohesion: 0.09
Nodes (19): DeleteHostFirewallRuleRequestObject, DeleteScheduleRequestObject, DeleteWebhookRequestObject, FirewallRule, FirewallRuleAction, FirewallRuleDirection, FirewallRuleEnvelope, FirewallRuleListEnvelope (+11 more)

### Community 22 - "newTestStore"
Cohesion: 0.08
Nodes (40): Store, T, Webhook, seedWebhookForEvents(), TestClaimPendingOutboxEventsPreventsDuplicateClaim(), TestClaimPendingOutboxEventsReclaimsStaleProcessing(), TestCreateEventQueuesOnlyMatchingWebhooks(), TestListEventsSinceReturnsAscendingAfterID() (+32 more)

### Community 23 - "events_handlers.go"
Cohesion: 0.18
Nodes (20): eventMatchesFilter(), eventToResponse(), flushSSE(), Context, Event, Mutex, Request, ResponseWriter (+12 more)

### Community 24 - "Hash"
Cohesion: 0.20
Nodes (16): decodeArgon2idHash(), DefaultArgon2Params(), Hash(), fastParams(), T, TestHashRefusesEmptyPassword(), TestHashVerifyRoundtrip(), TestVerifyRejectsMalformedHash() (+8 more)

### Community 25 - "models.go"
Cohesion: 0.09
Nodes (25): NullInt64, NullString, Context, Event, NullInt64, NullString, Queries, Webhook (+17 more)

### Community 26 - "Context"
Cohesion: 0.13
Nodes (12): Context, Rows, Store, Time, scanSchedules(), defaultString(), Context, NullString (+4 more)

### Community 27 - "UserFromContext"
Cohesion: 0.23
Nodes (10): auditID(), Context, Mutex, Request, ResponseWriter, Server, User, UserFromContext() (+2 more)

### Community 28 - "mocks.ts"
Cohesion: 0.11
Nodes (27): getAlerts(), getAudit(), getBackups(), getClusters(), getContainers(), getFirewallRules(), getInstances(), getNodes() (+19 more)

### Community 29 - "writeError"
Cohesion: 0.27
Nodes (7): Request, ResponseWriter, Server, Request, ResponseWriter, Server, writeError()

### Community 30 - "bodySerializer.gen.ts"
Cohesion: 0.13
Nodes (21): BodySerializer, formDataBodySerializer, jsonBodySerializer, QuerySerializerOptionsObject, urlSearchParamsBodySerializer, ArraySeparatorStyle, ArrayStyle, MatrixStyle (+13 more)

### Community 31 - "pages.jsx"
Cohesion: 0.09
Nodes (18): getPageContent(), PageAlerts(), PageFirewallEditor(), PageMarketplace(), PageMetrics(), PageRBAC(), InstanceOverview(), ADR-0014 (+10 more)

### Community 32 - "compilerOptions"
Cohesion: 0.08
Nodes (25): DOM, DOM.Iterable, ES2022, src, compilerOptions, allowImportingTsExtensions, erasableSyntaxOnly, forceConsistentCasingInFileNames (+17 more)

### Community 33 - "BadRequestJSONResponse"
Cohesion: 0.08
Nodes (13): AuthLogin400JSONResponse, AuthMfaComplete400JSONResponse, AuthSetup400JSONResponse, BadRequestJSONResponse, CreateHostFirewallRule400JSONResponse, CreateInstance400JSONResponse, CreateSchedule400JSONResponse, CreateUser400JSONResponse (+5 more)

### Community 34 - "Open"
Cohesion: 0.25
Nodes (22): Request, User, newJSONRequest(), postJSON(), seedRegularUser(), TestAdminJWTBearerCanAccessAdminAPI(), TestAdminNonAdminScopeAPITokenCannotReadRawIncusProxy(), TestAdminWriteAPITokenCannotMutateAdminIncusProxy() (+14 more)

### Community 35 - "helling-debian.pkr.hcl"
Cohesion: 0.11
Nodes (24): local.debian_134_checksums, local.default_iso_checksum, local.default_iso_url, local.iso_checksum, local.iso_url, local.parallels_tools_flavor, local.preseed_installer_locale, local.ssh_public_key_b64 (+16 more)

### Community 36 - "utils.gen.ts"
Cohesion: 0.16
Nodes (22): createClient(), TODO: we probably want to return error and improve types, ReqInit, buildUrl(), checkForExistence(), createConfig(), createInterceptors(), createQuerySerializer() (+14 more)

### Community 37 - "user.go"
Cohesion: 0.24
Nodes (22): Client, Command, Context, Reader, NewUserCmd(), newUserCreateCmd(), newUserDeleteCmd(), newUserGetCmd() (+14 more)

### Community 38 - ".handleCreateSchedule"
Cohesion: 0.22
Nodes (13): Context, Request, ResponseWriter, Server, Time, scheduleArtifactName(), scheduleToResponse(), scheduleUnitSpec() (+5 more)

### Community 39 - "server.go"
Cohesion: 0.18
Nodes (17): Client, Duration, Store, Time, incusProxyMutation(), requestStartedFromContext(), timeoutExceptSSE(), AuditEmitFunc (+9 more)

### Community 40 - "ui-store.ts"
Cohesion: 0.13
Nodes (21): closeModal(), Density, dismissToast(), emit(), getSnapshot(), listeners, Modal, openModal() (+13 more)

### Community 41 - "auth.go"
Cohesion: 0.23
Nodes (21): decodeJWTClaims(), Client, Command, Context, Reader, Writer, NewAuthCmd(), newAuthLoginCmd() (+13 more)

### Community 42 - "proxy.go"
Cohesion: 0.17
Nodes (22): Certificate, Logger, Store, Transport, loadCertPool(), MustParseURL(), NewDelegatedIncusProxy(), NewIncusProxy() (+14 more)

### Community 43 - "QueryStateView.tsx"
Cohesion: 0.13
Nodes (17): ADR-0051, deleteWebhookMutation(), listWebhooksOptions(), listWebhooksQueryKey(), testWebhookMutation(), cardStyle, describeQueryError(), ErrorEnvelope (+9 more)

### Community 44 - "audit_handlers.go"
Cohesion: 0.20
Nodes (18): auditFromJournal(), auditJournalArgs(), auditLimit(), Context, Request, ResponseWriter, Server, Time (+10 more)

### Community 45 - "run"
Cohesion: 0.08
Nodes (23): Bootstrap Process, By Helling (Proxy), By Incus, CA Certificate, CA Key Encryption, CA Key Management, CA Rotation, Certificate Lifecycle (+15 more)

### Community 46 - ".handleInstanceStateChange"
Cohesion: 0.24
Nodes (12): Client, Request, ResponseWriter, Server, isAlreadyExists(), isConflict(), isNotFound(), toOperationResponse() (+4 more)

### Community 47 - "loginCookie"
Cohesion: 0.25
Nodes (18): Client, Request, Server, Store, T, mustRequest(), newServerWithIncus(), TestCreateInstanceCreatesOperationRow() (+10 more)

### Community 48 - "compilerOptions"
Cohesion: 0.10
Nodes (20): ES2023, hey-api.config.ts, vite.config.ts, compilerOptions, allowImportingTsExtensions, erasableSyntaxOnly, lib, module (+12 more)

### Community 49 - ".handleCreateUser"
Cohesion: 0.25
Nodes (10): ValidTrustProjectName(), Context, Request, ResponseWriter, Server, User, normalizeIncusProject(), createUserRequest (+2 more)

### Community 50 - ".auditForActorWithPolicyReason"
Cohesion: 0.28
Nodes (7): auditDurationMS(), auditStatusForOutcome(), Request, Server, User, Logger, jwtIDFromContext()

### Community 51 - "Store"
Cohesion: 0.23
Nodes (13): Context, Store, T, User, seedAdminUser(), TestCreateScheduleInstallsSystemdUnits(), TestCreateScheduleRejectsOnCalendarInjectionBeforeInstall(), TestDeleteScheduleRemovesSystemdUnits() (+5 more)

### Community 52 - "writeJSON"
Cohesion: 0.20
Nodes (10): Request, ResponseWriter, Server, writeJSON(), boolToStatus(), errToString(), Request, ResponseWriter (+2 more)

### Community 53 - "queries.ts"
Cohesion: 0.24
Nodes (17): getAccessToken(), authedFetch(), fetchIncusInstances(), fetchIncusList(), fetchPodmanContainers(), IncusImage, IncusNetwork, IncusOperation (+9 more)

### Community 54 - "index.ts"
Cohesion: 0.19
Nodes (16): BuildUrlFn, Client, ClientOptions, Config, CreateClientConfig, MethodFn, OmitKeys, Options (+8 more)

### Community 55 - "window-globals.ts"
Cohesion: 0.19
Nodes (16): getWarnings(), closeModal(), ConfirmModalProps, ModalKind, navigate(), openConfirm(), openModal(), toast() (+8 more)

### Community 56 - "Client"
Cohesion: 0.18
Nodes (9): decodeAPIError(), encodeRequestBody(), Context, Reader, Response, Transport, unixTransport(), APIError (+1 more)

### Community 57 - "Context"
Cohesion: 0.19
Nodes (12): HashAPIToken(), NewAPIToken(), ValidScopes(), apiTokenScopesFromContext(), Context, Request, ResponseWriter, Server (+4 more)

### Community 58 - "newTestServer"
Cohesion: 0.36
Nodes (14): Config, Server, Store, T, newTestServer(), newTestServerWithConfig(), seedUser(), TestHealthAndVersion() (+6 more)

### Community 59 - "RateLimiter"
Cohesion: 0.19
Nodes (12): NewRateLimiter(), T, TestRateLimiterAllowsUpToLimit(), TestRateLimiterIsolatesKeys(), TestRateLimiterKeyCap(), TestRateLimiterReset(), TestRateLimiterWindowExpiry(), RateLimiter (+4 more)

### Community 60 - "auth_audit_test.go"
Cohesion: 0.29
Nodes (14): Mutex, Server, T, Time, newAuditedTestServer(), TestAdminPolicyDenyEmitsAuditReason(), TestAuthFailedLoginEmitsAuditRecord(), TestAuthLoginLogoutEmitAuditRecords() (+6 more)

### Community 61 - "AGENTS.md"
Cohesion: 0.08
Nodes (22): Agent-specific guidance, Agent Tooling, Build and release, Code generation (mandatory), Code style and architecture rules, Council Subagents (`.codex/agents/` and `.claude/agents/`), Development workflow, graphify (+14 more)

### Community 62 - "devDependencies"
Cohesion: 0.12
Nodes (17): @biomejs/biome, @hey-api/openapi-ts, jsdom, @testing-library/jest-dom, @testing-library/react, @types/react, vite, @vitejs/plugin-react (+9 more)

### Community 63 - "Config"
Cohesion: 0.13
Nodes (20): Context, ScrapeMetrics(), argon2ParamsFromConfig(), buildDelegatedIncusProxy(), connectIncusClient(), Client, Config, FileMode (+12 more)

### Community 64 - "helling-first-boot.sh"
Cohesion: 0.26
Nodes (15): add_user_to_group_if_present(), configure_services(), configure_zabbly_incus_repo(), create_identities_and_paths(), ensure_group(), ensure_packages(), ensure_user(), fail() (+7 more)

### Community 65 - "build-iso.sh"
Cohesion: 0.31
Nodes (14): build_iso(), detect_arch(), done_(), fail(), goarch_from_deb(), log(), main(), prepare_workdir() (+6 more)

### Community 66 - "install-tools.sh"
Cohesion: 0.54
Nodes (15): done_(), fail(), have(), install_frontend_tools(), install_gitleaks_binary(), install_go_tools(), install_node_tools(), install_python_tools() (+7 more)

### Community 67 - "setup.tsx"
Cohesion: 0.17
Nodes (15): authSetup(), authSetupStatus(), authSetupMutation(), authSetupStatusOptions(), authSetupStatusQueryKey(), Error, getToast(), initialForm (+7 more)

### Community 68 - "system-store.ts"
Cohesion: 0.19
Nodes (15): bumpTick(), clearAlerts(), emit(), getSnapshot(), listeners, pushAlert(), pushTask(), setState() (+7 more)

### Community 69 - "JWTSigner"
Cohesion: 0.13
Nodes (14): By area, By severity, F-36 — implementation walked away from the documented stack 🔴 BLOCKER · NEW · ✅ ADR-051, ✦ Helling WebUI — Combined Audit Report, If you do nothing else from this report, Notes, Section 01 — Stack drift, the headline, Section 11 — All findings index (+6 more)

### Community 70 - "main.tsx"
Cohesion: 0.15
Nodes (5): ErrorBoundary, ErrorBoundaryProps, ErrorBoundaryState, container, queryClient

### Community 71 - "Operation"
Cohesion: 0.34
Nodes (7): Context, Rows, Store, Time, scanOperationRow(), Operation, OperationStatus

### Community 72 - "Context"
Cohesion: 0.20
Nodes (9): Context, Logger, Mutex, Request, ResponseWriter, Router, Server, loggerMiddleware() (+1 more)

### Community 73 - "dependencies"
Cohesion: 0.15
Nodes (13): @hey-api/client-fetch, lucide-react, react, react-dom, @scalar/api-reference-react, @tanstack/react-query, dependencies, @hey-api/client-fetch (+5 more)

### Community 74 - "types.gen.ts"
Cohesion: 0.20
Nodes (12): Auth, AuthToken, QuerySerializer, QuerySerializerOptions, ServerSentEventsOptions, ServerSentEventsResult, StreamEvent, Client (+4 more)

### Community 75 - "icon.tsx"
Cohesion: 0.28
Nodes (8): IconName, ICONS, getNetworks(), PageNetworking(), Copyable(), Props, I(), IconProps

### Community 76 - "ConflictJSONResponse"
Cohesion: 0.20
Nodes (6): AuthSetup409JSONResponse, ConflictJSONResponse, CreateSchedule409JSONResponse, DeleteInstance409JSONResponse, StartInstance409JSONResponse, StopInstance409JSONResponse

### Community 77 - "newRootCmd"
Cohesion: 0.30
Nodes (10): Command, Writer, main(), newRootCmd(), newVersionCmd(), run(), T, TestCompletionSubcommandExists() (+2 more)

### Community 78 - "params.gen.ts"
Cohesion: 0.21
Nodes (11): buildClientParams(), buildKeyMap(), extraPrefixes, extraPrefixesMap, Field, Fields, FieldsConfig, KeyMap (+3 more)

### Community 79 - "infra.jsx"
Cohesion: 0.18
Nodes (3): ToastBus, Switch(), SwitchProps

### Community 80 - "NewSystemCmd"
Cohesion: 0.55
Nodes (10): Command, NewSystemCmd(), newSystemConfigGetCmd(), newSystemConfigPutCmd(), newSystemDiagnosticsCmd(), newSystemHardwareCmd(), newSystemHealthCmd(), newSystemInfoCmd() (+2 more)

### Community 81 - "New"
Cohesion: 0.44
Nodes (10): New(), T, TestClient_DoBearerAndRefreshCookieRotation(), TestClient_ErrorEnvelopeIsAPIError(), TestClient_ExplicitBaseURLOverridesProfile(), TestClient_HellingErrorEnvelopeIsAPIError(), TestClient_HTTPUnixEndpoint(), TestClient_NewRequiresAPI() (+2 more)

### Community 82 - "Record"
Cohesion: 0.27
Nodes (9): Emit(), Logger, T, TestTruncateJournalFieldLimitsBytes(), TestTruncateJournalFieldPreservesUTF8(), truncateJournalField(), defaultAuditEmit(), Logger (+1 more)

### Community 83 - "auth_ext_handlers.go"
Cohesion: 0.18
Nodes (8): DeleteUserRequestObject, GetUser404JSONResponse, GetUserRequestObject, UpdateUserRequestObject, UpdateUserScopeRequestObject, UpdateUserJSONRequestBody, UpdateUserScopeJSONRequestBody, UserID

### Community 84 - "newSetupTokenServer"
Cohesion: 0.20
Nodes (9): Ant Design Components, Bordered Descriptions (Hardware Config), Code Pattern, Design Rules, Detail Views, Instance Detail Page, Pages Using This Pattern, Standards References (+1 more)

### Community 85 - "Context"
Cohesion: 0.31
Nodes (4): Context, Store, Time, K8sCluster

### Community 86 - "run"
Cohesion: 0.27
Nodes (8): errWriter, Writer, main(), run(), T, TestMainUsesInjectedWritersAndExit(), TestRunReturnsErrorForFailingWriter(), TestRunWritesOpenAPIDocument()

### Community 87 - "suspicious"
Cohesion: 0.18
Nodes (11): error, info, warn, level, options, allow, suspicious, noConsole (+3 more)

### Community 88 - "biome.json"
Cohesion: 0.18
Nodes (10): files, linter, enabled, organizeImports, enabled, $schema, vcs, clientKind (+2 more)

### Community 89 - "rules"
Cohesion: 0.18
Nodes (11): rules, noDelete, performance, recommended, security, style, noDangerouslySetInnerHtml, noNonNullAssertion (+3 more)

### Community 90 - "userClient"
Cohesion: 0.51
Nodes (9): Command, NewScheduleCmd(), newScheduleCreateCmd(), newScheduleDeleteCmd(), newScheduleGetCmd(), newScheduleListCmd(), newScheduleRunCmd(), newScheduleUpdateCmd() (+1 more)

### Community 91 - "Context"
Cohesion: 0.36
Nodes (4): Context, Store, Time, APIToken

### Community 92 - "008_v02_platform_core.sql"
Cohesion: 0.31
Nodes (9): event_outbox, events, firewall_host_rules, incus_user_certs, schedules, users, webhook_deliveries, webhook_events (+1 more)

### Community 93 - "Context"
Cohesion: 0.29
Nodes (5): Context, Duration, Store, Time, Session

### Community 94 - "ignore"
Cohesion: 0.20
Nodes (10): build, coverage, dist, node_modules, src/api/generated, src/infra.jsx, src/pages2.jsx, src/pages.jsx (+2 more)

### Community 95 - "check-parity.sh"
Cohesion: 0.33
Nodes (6): have_py(), in_cli_spec(), in_spec(), in_webui_spec(), is_exempt(), check-parity.sh script

### Community 96 - "scripts"
Cohesion: 0.20
Nodes (10): scripts, build, check, dev, fmt, gen:api, prepare, preview (+2 more)

### Community 97 - "login.tsx"
Cohesion: 0.40
Nodes (5): ApiErrorLike, PageLogin(), PageLoginProps, responseError(), Stage

### Community 98 - "normalize.ts"
Cohesion: 0.27
Nodes (10): canonicalizeStatus(), CanonicalStatus, Container, Instance, normalizeIncusInstance(), normalizeIncusInstances(), normalizePodmanContainer(), normalizePodmanContainers() (+2 more)

### Community 99 - "NewFirewallCmd"
Cohesion: 0.39
Nodes (7): confirmDestructiveAction(), Command, Command, newFirewallAddCmd(), NewFirewallCmd(), newFirewallDeleteCmd(), newFirewallListCmd()

### Community 100 - "DialEventsWithClient"
Cohesion: 0.22
Nodes (9): F-05 — `pages.jsx` + `pages2.jsx` are 8,599 lines in two files 🟠 MAJOR, F-06 — 783 inline-style objects across pages 🟠 MAJOR, F-07 — `window.*` coupling between modules 🟡 MINOR, F-08 — `eslint-disable` banner per file, biome a11y rules enabled 🟡 MINOR, F-09 — mixed JSX / TS extensions 🟡 MINOR, F-39 — zero error boundaries 🟠 MAJOR · NEW · ✅ Phase 1, F-40 — zero tests, zero test deps 🟠 MAJOR · NEW · ✅ Phase 2, F-41 — fresh-clone build broken 🟠 MAJOR · NEW · ✅ Phase 1 (+1 more)

### Community 101 - "ToInstance"
Cohesion: 0.36
Nodes (7): Time, T, TestToInstance(), TestToInstanceFingerprintFallback(), TestToInstanceNil(), ToInstance(), Instance

### Community 102 - "doGet"
Cohesion: 0.56
Nodes (8): doGet(), Client, Response, T, TestMetricsEndpointExposesPrometheusBaseline(), TestMetricsIncludesProxiedIncusMetrics(), TestMetricsRecordsFiveHundredErrors(), TestMetricsScrapeFailureKeepsHellingMetricsAvailable()

### Community 103 - "BMCEndpoint"
Cohesion: 0.36
Nodes (4): Context, Store, Time, BMCEndpoint

### Community 104 - "fix_file"
Cohesion: 0.33
Nodes (8): classify(), find_target_files(), fix_file(), main(), Scan file, label bare fences. Returns (count_changed, list_of_changes)., Return the list of files to process., Inspect block content and return a best-guess language label., Path

### Community 105 - "AuthLogin200JSONResponse"
Cohesion: 0.25
Nodes (6): AuthLogin200JSONResponse, AuthLogin200ResponseHeaders, AuthMfaComplete200JSONResponse, AuthMfaComplete200ResponseHeaders, AuthTokenEnvelope, AuthTokenResponse

### Community 106 - ".ExportAuditEvents"
Cohesion: 0.25
Nodes (8): F-14 — wizard has no validation, no preview 🟠 MAJOR, F-15 — confirm modals don't require typing target name 🟠 MAJOR · ✅ Phase 1, F-16 — bulk actions limited, no aggregate progress 🟠 MAJOR, F-17 — keyboard story missing on detail pages 🟡 MINOR, F-18 — toast bus has no inbox 🟡 MINOR, F-19 — toast-only buttons are landmines 🟡 MINOR, F-38 — login is theatre 🔴 BLOCKER · NEW · ✅ Phase 1, Section 05 — Interaction & flow

### Community 107 - "ListAuditEventsParams"
Cohesion: 0.25
Nodes (8): F-20 — light mode is structural inversion 🟠 MAJOR, F-21 — 87 hardcoded color literals 🟡 MINOR, F-22 — viewport locked to 1440 🟡 MINOR · ✅ Phase 1 (R-03 part) · Phase 6 (responsive policy), F-23 — page headers drift across pages 🟡 MINOR, F-24 — status colors not in DS palette 🔵 NIT, F-44 — no `prefers-*` media queries 🟡 MINOR · NEW · ✅ Phase 1, F-50 — theme persists, density doesn't 🔵 NIT · NEW · ✅ Phase 1, Section 06 — Visual design & brand fit

### Community 108 - ".ListInstances"
Cohesion: 0.50
Nodes (3): ListInstancesParams, ListInstancesParamsStatus, ListInstancesRequestObject

### Community 109 - ".ListOperations"
Cohesion: 0.43
Nodes (5): HashToken(), NewToken(), T, TestHashTokenDeterministic(), TestNewTokenUniqueAndHashStable()

### Community 110 - ".ListSchedules"
Cohesion: 0.25
Nodes (4): ListSchedules403JSONResponse, ListSchedulesParams, ListSchedulesParamsKind, ListSchedulesRequestObject

### Community 111 - "NewEventsCmd"
Cohesion: 0.43
Nodes (7): copySSEData(), Command, Reader, Writer, NewEventsCmd(), newEventsListCmd(), newEventsTailCmd()

### Community 112 - ".newReverseProxy"
Cohesion: 0.57
Nodes (3): Request, ResponseWriter, Server

### Community 113 - "NotFoundJSONResponse"
Cohesion: 0.07
Nodes (15): DeleteHostFirewallRule404JSONResponse, DeleteInstance404JSONResponse, DeleteSchedule404JSONResponse, DeleteUser404JSONResponse, DeleteWebhook404JSONResponse, GetSchedule404JSONResponse, GetWebhook404JSONResponse, NotFoundJSONResponse (+7 more)

### Community 114 - "parallels-vm-bootstrap.sh"
Cohesion: 0.46
Nodes (7): done_(), fail(), have(), log(), parallels-vm-bootstrap.sh script, skip(), SSH()

### Community 115 - "parallels-vm-deploy-rsync.sh"
Cohesion: 0.46
Nodes (6): done_(), fail(), log(), remote_install_payload(), parallels-vm-deploy-rsync.sh script, SSH()

### Community 116 - "sync-agent-tooling.sh"
Cohesion: 0.57
Nodes (7): check_json(), check_mcp_json(), check_toml_dir(), fail(), require_executable(), require_file(), sync-agent-tooling.sh script

### Community 117 - "use-events-stream.ts"
Cohesion: 0.32
Nodes (6): dispatchKeys(), EventsResponse, fetchEventsBatch(), HellingEvent, invalidateEventQueries(), trimSeenEvents()

### Community 118 - "index.tsx"
Cohesion: 0.25
Nodes (5): Props, RESULTS, SearchResult, TYPE_COLOR, TYPE_ICON

### Community 119 - "FirewallRuleCreateRequest"
Cohesion: 0.29
Nodes (4): FirewallRuleCreateRequest, FirewallRuleCreateRequestAction, FirewallRuleCreateRequestDirection, FirewallRuleCreateRequestProtocol

### Community 120 - "unit_linker_test.go"
Cohesion: 0.57
Nodes (6): assertCommandCalls(), T, TestUnitLinkerInstallTimerLinksAndEnables(), TestUnitLinkerRejectsUnsafeArgs(), TestUnitLinkerRemoveTimerDisablesAndUnlinks(), commandCall

### Community 121 - "logs.tsx"
Cohesion: 0.29
Nodes (5): ADR-0019, levelClass, LOG_ROWS, LogLevel, LogRow

### Community 122 - ".mcp.json"
Cohesion: 0.29
Nodes (6): bash, npx, codex-peer, context7, openaiDeveloperDocs, @upstash/context7-mcp

### Community 123 - "parallels-vm-build-dev.sh"
Cohesion: 0.67
Nodes (6): done_(), fail(), have(), log(), register_pvm(), parallels-vm-build-dev.sh script

### Community 124 - "parallels-vm-deploy-deb.sh"
Cohesion: 0.57
Nodes (6): done_(), fail(), have(), log(), parallels-vm-deploy-deb.sh script, skip()

### Community 125 - "plan-next-version.sh"
Cohesion: 0.48
Nodes (5): done_(), fail(), log(), plan-next-version.sh script, version_gt()

### Community 127 - "queryKeySerializer.gen.ts"
Cohesion: 0.48
Nodes (6): isPlainObject(), JsonValue, queryKeyJsonReplacer(), serializeQueryKeyValue(), serializeSearchParams(), stringifyToJsonValue()

### Community 128 - "index.tsx"
Cohesion: 0.52
Nodes (6): deleteScheduleMutation(), listSchedulesOptions(), listSchedulesQueryKey(), runScheduleMutation(), PageSchedules(), ScheduleFilter

### Community 129 - "audit.tsx"
Cohesion: 0.43
Nodes (6): auditExportHeaders(), auditExportParams(), AuditOutcomeFilter, AuditQueryParams, downloadBlob(), PageAudit()

### Community 130 - ".ListEvents"
Cohesion: 0.29
Nodes (7): F-01 — components own mock state directly 🔴 BLOCKER, F-02 — mock data shape doesn't match real API shape 🔴 BLOCKER, F-03 — 1.5s mock tick mutates global state 🟠 MAJOR, F-04 — errors and loading states are decorative 🟠 MAJOR, F-42 — OpenAPI ships `eventsSse` but no SSE consumer 🟠 MAJOR · NEW, F-43 — mock fidelity gap is shape-level 🟠 MAJOR · NEW, Section 02 — Data layer & API integration

### Community 131 - "newAuthMfaCmd"
Cohesion: 0.73
Nodes (5): Command, newAuthMfaCmd(), newAuthMfaDisableCmd(), newAuthMfaSetupCmd(), newAuthMfaVerifyCmd()

### Community 132 - "newAuthTokenCmd"
Cohesion: 0.73
Nodes (5): Command, newAuthTokenCmd(), newAuthTokenCreateCmd(), newAuthTokenListCmd(), newAuthTokenRevokeCmd()

### Community 133 - "config_test.go"
Cohesion: 0.29
Nodes (7): F-29 — no code splitting 🔴 BLOCKER, F-30 — Lucide icons block tree-shake 🟡 MINOR · ✅ Phase 2, F-31 — `useStore()` re-renders every subscriber 🟡 MINOR, F-45 — `refetchOnWindowFocus: false` default wrong 🟡 MINOR · NEW · ✅ Phase 1, F-48 — `setInterval`s leak across HMR 🟡 MINOR · NEW, F-51 — Lucide stroke width drift outside `I` wrapper 🔵 NIT · NEW · ✅ Phase 2 (with F-30), Section 08 — Performance & bundle

### Community 134 - "NewWebhookCmd"
Cohesion: 0.33
Nodes (5): `.claude/agents/` — Helling agent roster, Council deliberation flow, When the council theater alarm trips, Why no test-runner / linter agents, Why not agent teams (the heavier multi-Claude-instance pattern)

### Community 135 - "provision-helling-dev.sh"
Cohesion: 0.53
Nodes (5): DEBIAN_FRONTEND, fail(), log(), provision-helling-dev.sh script, warn()

### Community 136 - "formatter"
Cohesion: 0.33
Nodes (6): formatter, enabled, indentStyle, indentWidth, lineEnding, lineWidth

### Community 137 - "formatter"
Cohesion: 0.33
Nodes (6): jsxQuoteStyle, quoteStyle, semicolons, trailingCommas, javascript, formatter

### Community 138 - "ExportAuditEvents200ApplicationxNdjsonResponse"
Cohesion: 0.40
Nodes (3): ExportAuditEvents200ApplicationxNdjsonResponse, ListEvents200TexteventStreamResponse, Reader

### Community 139 - "loadOrCreateAgeIdentity"
Cohesion: 0.33
Nodes (6): R-01 → F-03 — SSE is the right answer, not `refetchInterval`, R-02 → F-06, F-14, F-23, F-25, F-26 — primitive-extraction findings assume the current stack stays, R-03 → F-22 — viewport-meta is actively wrong, not just restrictive, R-04 → F-29 — bundles two unrelated wins, one needs verification, R-05 → F-04 — right but undersells how broken auth feedback is, Section 00 — Reframings: where v0.1 points at the wrong fix

### Community 140 - "events.go"
Cohesion: 0.50
Nodes (4): Time, Event, IncusLifecycleEvent, Source

### Community 141 - "a11y"
Cohesion: 0.40
Nodes (5): noAutofocus, recommended, useButtonType, useValidAnchor, a11y

### Community 142 - "noExcessiveCognitiveComplexity"
Cohesion: 0.40
Nodes (5): noExcessiveCognitiveComplexity, level, options, maxAllowedComplexity, complexity

### Community 143 - "package.json"
Cohesion: 0.40
Nodes (4): name, private, type, version

### Community 144 - "events.gen.ts"
Cohesion: 0.40
Nodes (4): Event, EventData, IncusLifecycleEvent, Source

### Community 145 - "index.tsx"
Cohesion: 0.80
Nodes (4): deleteHostFirewallRuleMutation(), listHostFirewallRulesOptions(), listHostFirewallRulesQueryKey(), PageFirewall()

### Community 146 - "client.gen.ts"
Cohesion: 0.50
Nodes (3): client, CreateClientConfig, ClientOptions

### Community 147 - "TestPodmanRequestAllowed"
Cohesion: 0.80
Nodes (4): Command, NewAuditCmd(), newAuditExportCmd(), newAuditQueryCmd()

### Community 148 - "kubernetes_handlers.go"
Cohesion: 0.50
Nodes (3): createK8sRequest, scaleK8sRequest, upgradeK8sRequest

### Community 149 - "001_initial.sql"
Cohesion: 0.83
Nodes (3): operations, sessions, users

### Community 151 - "never-guess.sh"
Cohesion: 0.50
Nodes (3): never-guess.sh script, watch_patterns, watch_tools

### Community 152 - "correctness"
Cohesion: 0.50
Nodes (4): noUnusedImports, noUnusedVariables, useExhaustiveDependencies, correctness

### Community 154 - "vite-env.d.ts"
Cohesion: 0.50
Nodes (3): *.css, *.jsx, Window

### Community 187 - "fakeOpHandle"
Cohesion: 0.18
Nodes (11): Duration, PrivateKey, LoadOrCreateJWTSigner(), mustRandomHex(), NewJWTSigner(), NewJWTSignerFromSeed(), trimSpaceBytes(), JWTClaims (+3 more)

### Community 201 - "UpdateUserScope200JSONResponse"
Cohesion: 0.40
Nodes (5): F-10 — sidebar grows linearly forever 🟠 MAJOR, F-11 — crumb-to-route mapping ad-hoc 🟠 MAJOR, F-12 — Firewall + FirewallEditor split unclear 🟡 MINOR, F-13 — Marketplace + Templates overlap 🟡 MINOR, Section 04 — Information architecture & navigation

### Community 203 - "Section 07 — Accessibility"
Cohesion: 0.40
Nodes (5): F-25 — modals don't trap focus 🟠 MAJOR, F-26 — tables aren't tables 🟠 MAJOR, F-27 — status communicated by color only 🟠 MAJOR, F-28 — toast notifications aren't announced 🟡 MINOR, Section 07 — Accessibility

### Community 204 - "Section 09 — Operator ergonomics"
Cohesion: 0.40
Nodes (5): F-32 — no global "what changed?" view 🟠 MAJOR, F-33 — backup story doesn't surface failure or recency 🟠 MAJOR, F-34 — no compare/diff anywhere 🟡 MINOR, F-35 — shell glyph ✦ unused 🟡 MINOR, Section 09 — Operator ergonomics

### Community 205 - "Section 10 — Security & supply chain"
Cohesion: 0.40
Nodes (5): F-37 — `auth-store.ts` violates `docs/spec/auth.md` §2.2 🔴 BLOCKER · NEW · ✅ Phase 1, F-46 — no `<noscript>`, no CSP meta, no SRI 🟡 MINOR · NEW, F-47 — global `ResizeObserver` warning suppression 🔵 NIT · NEW · ✅ Phase 1, F-49 — no Renovate/Dependabot config 🔵 NIT · NEW, Section 10 — Security & supply chain

### Community 215 - "Config"
Cohesion: 0.27
Nodes (12): applyAuthEnv(), applyBaseEnv(), applyEnv(), applyIncusEnv(), Defaults(), FileMode, Load(), AuthConfig (+4 more)

### Community 216 - ".ServeHTTP"
Cohesion: 0.29
Nodes (8): Request, ResponseWriter, podmanRequestAllowed(), T, TestPodmanRequestAllowed(), TestProxyDirectorStripsHellingCredentials(), writeProxyError(), Header

### Community 217 - "clientIP"
Cohesion: 0.26
Nodes (10): clientIP(), T, TestClientIP(), isParsableIP(), normalizedRemoteHost(), parseForwardedFor(), parseIP(), writeAuthData() (+2 more)

### Community 218 - "Context"
Cohesion: 0.26
Nodes (4): Context, Store, Time, TOTPSecret

### Community 219 - "auth_ext_handlers.go"
Cohesion: 0.20
Nodes (10): Store, Time, newFirstAdminSetupService(), createTokenRequest, createTokenResponse, firstAdminSetupService, mfaCompleteRequest, setupStatusResponse (+2 more)

### Community 220 - "newSetupTokenServer"
Cohesion: 0.35
Nodes (10): Server, Store, T, Writer, newSetupTokenServer(), TestRetireSetupTokenTruncatesWhenDirectoryCannotUnlink(), TestSetupCreatesAdminWithInstallerTokenAndLogsAuditEvent(), TestSetupRejectsOversizedBody() (+2 more)

### Community 221 - "DialEventsWithClient"
Cohesion: 0.39
Nodes (8): DialEvents(), DialEventsWithClient(), Client, Conn, Context, MapLifecycleEvent(), WatchLifecycleEvents(), LifecycleEvent

### Community 222 - "config_test.go"
Cohesion: 0.67
Nodes (5): T, TestLoadAcceptsDefaultConfig(), TestLoadRejectsInvalidAuthConfig(), TestLoadRejectsInvalidSocketConfig(), writeConfig()

### Community 223 - "totpAt"
Cohesion: 0.83
Nodes (3): decodeBase32(), totpAt(), ValidateTOTP()

### Community 224 - "loadOrCreateAgeIdentity"
Cohesion: 0.67
Nodes (3): ageIdentityPath(), X25519Identity, loadOrCreateAgeIdentity()

## Knowledge Gaps
- **597 isolated node(s):** `Project overview`, `Monorepo layout`, `Required toolchain`, `Setup commands`, `Code generation (mandatory)` (+592 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **54 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Handler()` connect `ResponseWriter` to `server.gen.go`, `server.go`, `Context`, `.ListSchedules`, `auth_ext_handlers.go`, `Context`?**
  _High betweenness centrality (0.091) - this node is a cross-community bridge._
- **Why does `writeError()` connect `writeError` to `ResponseWriter`, `.handleCreateSchedule`, `server.go`, `Context`, `firewall_handlers.go`, `audit_handlers.go`, `.handleInstanceStateChange`, `.newReverseProxy`, `.handleCreateUser`, `writeJSON`, `New`, `events_handlers.go`, `Context`, `UserFromContext`?**
  _High betweenness centrality (0.066) - this node is a cross-community bridge._
- **Why does `New()` connect `New` to `doGet`, `server.go`, `Context`, `firewall_handlers.go`, `OperationHandle`, `UserFromContext`, `auth_ext_handlers.go`, `loginCookie`, `newSetupTokenServer`, `Store`, `events_handlers.go`, `newTestServer`, `RateLimiter`, `auth_audit_test.go`?**
  _High betweenness centrality (0.051) - this node is a cross-community bridge._
- **Are the 99 inferred relationships involving `HandlerFunc` (e.g. with `.AuthLogin()` and `.AuthMfaComplete()`) actually correct?**
  _`HandlerFunc` has 99 INFERRED edges - model-reasoned connections that need verification._
- **Are the 70 inferred relationships involving `writeError()` (e.g. with `streamAuditJournal()` and `.adminMiddleware()`) actually correct?**
  _`writeError()` has 70 INFERRED edges - model-reasoned connections that need verification._
- **Are the 47 inferred relationships involving `useTempConfigDir()` (e.g. with `TestAuditExport_CSVFlag()` and `TestAuditExport_JSONDefault()`) actually correct?**
  _`useTempConfigDir()` has 47 INFERRED edges - model-reasoned connections that need verification._
- **What connects `Project overview`, `Monorepo layout`, `Required toolchain` to the rest of the system?**
  _597 weakly-connected nodes found - possible documentation gaps or missing edges._