# CLI Specification

`helling` CLI covers Helling-specific features only (ADR-016). For everything else:

- **Incus resources:** `incus` CLI (instances, storage, networks, profiles, projects, cluster, images)
- **Podman resources:** `podman` CLI (containers, pods, images, volumes, networks, secrets, compose)
- **Kubernetes workloads:** `kubectl` (after `helling k8s kubeconfig <name>`)

## Global Flags

```text
--api URL         hellingd API endpoint (default: from config)
--token TOKEN     API token (default: from config)
--output FORMAT   Output format: table (default), json, yaml
--quiet           Minimal output
```

## Commands

### Auth

```bash
helling auth setup --api http+unix:///run/helling/api.sock \
  --setup-token-file /etc/helling/setup-token
                                      # Create first admin with installer setup token
helling auth login                     # Interactive local login, stores JWT
helling auth logout                    # Clear stored tokens
helling auth token create NAME         # Create API token
helling auth token list                # List API tokens
helling auth token revoke ID           # Revoke API token
```

`helling auth setup` never accepts the setup token or first-admin password as
argv values. Interactive runs prompt without echo and ask for password
confirmation. Automation must use `--password-file` and `--setup-token-file`
so secrets do not appear in shell history or process listings.

### Users

```bash
helling user list                      # List users
helling user create USERNAME           # Create local user
helling user get USERNAME              # Get user details (status, 2FA, last login)
helling user update USERNAME           # Update user (2FA status, role)
helling user delete USERNAME           # Delete local user
helling user set-scope USER SCOPE      # Assign user trust scope
```

`helling user 2fa` subcommands are deferred to v0.3 (no `/api/v1/users/{id}/2fa/*` paths in the current OpenAPI contract; see `docs/spec/api.md`). The subcommand shape is specified here for continuity with the v0.3 target:

```bash
helling user 2fa enable USERNAME       # Enroll user in TOTP 2FA (returns provisioning URI)
helling user 2fa disable USERNAME      # Disable 2FA for user
helling user 2fa recovery USERNAME     # Regenerate recovery codes
```

### Schedules

```bash
helling schedule list                  # List backup/snapshot schedules
helling schedule create                # Create schedule (interactive)
helling schedule delete ID             # Delete schedule
helling schedule run ID                # Trigger schedule now
```

### Webhooks

```bash
helling webhook list                   # List webhooks
helling webhook create URL             # Create webhook
helling webhook delete ID              # Delete webhook
helling webhook test ID                # Send test delivery
```

### Audit

```bash
helling audit list                     # List recent audit events
helling audit query [FILTER]           # Query audit events (user, action, target, date-range)
helling audit export [FORMAT]          # Export audit events (csv, json)
```

### Events

```bash
helling events tail                    # Follow system events (SSE) in terminal
helling events list [COUNT]            # List recent events
```

### BMC (v0.4+)

BMC commands are deferred to v0.4 per `docs/spec/platform.md` and the `docs/spec/api.md` deferred domains list. The subcommand shape is specified here for continuity with the v0.4 target:

```bash
helling bmc list                       # List managed BMC endpoints
helling bmc add IP                     # Add BMC endpoint
helling bmc remove ID                  # Remove BMC endpoint
helling bmc power ID on|off|cycle      # Power control
helling bmc sensors ID                 # Read sensor data
```

### Kubernetes

```bash
helling k8s list                       # List K8s clusters
helling k8s create NAME                # Create cluster (k3s via cloud-init wizard)
helling k8s delete NAME                # Delete cluster
helling k8s scale NAME --workers N     # Scale worker pool
helling k8s upgrade NAME --version V   # Rolling upgrade
helling k8s kubeconfig NAME            # Download kubeconfig
```

### System

```bash
helling system info                    # System info (hostname, version, uptime)
helling system health                  # Health check (all services, OK/FAIL)
helling system upgrade                 # Check + apply system upgrade
helling system upgrade --rollback      # Rollback to previous version (requires backup)
helling system config get KEY          # Read config value
helling system config set KEY VALUE    # Set config value
helling system diagnostics             # Self-test
```

### Host Firewall

```bash
helling firewall list                  # List host nftables rules
helling firewall add RULE              # Add host firewall rule
helling firewall remove ID             # Remove host firewall rule
```

### Utilities

```bash
helling version                        # Version, go version, git commit
helling completion bash|zsh|fish       # Shell completions
```

## Shell Completions

```bash
helling completion bash > /etc/bash_completion.d/helling
helling completion zsh > ~/.zfunc/_helling
helling completion fish > ~/.config/fish/completions/helling.fish
```

Installed automatically by the .deb package postinst script.

## Total: ~17 command groups, ~49 subcommands

All commands use the generated Go API client from the Helling OpenAPI spec (~40 endpoints). Man pages generated by Cobra `doc.GenManTree()`.

## API Operation Coverage

Helling OpenAPI operationIds whose CLI command names differ from the naive camelCase variant — listed here so `scripts/check-parity.sh` can resolve coverage without a phase0 exception entry.

- operationId: healthGet → `helling system health`
- operationId: eventsSse → `helling events tail`
- operationId: authSetup → first-boot flow via `helling auth setup`
- operationId: authSetupStatus → WebUI first-boot status probe
- operationId: authMfaComplete → `helling auth login` MFA prompt branch + `helling auth mfa` parent
- operationId: authTotpSetup → `helling auth mfa setup`
- operationId: authTotpVerify → `helling auth mfa verify`
- operationId: authTotpDisable → `helling auth mfa disable`
- operationId: userSetScope → `helling user set-scope`
- operationId: webhookGet → `helling webhook get`
- operationId: webhookUpdate → `helling webhook update`
- operationId: systemHardware → `helling system hardware`
- operationId: systemConfigGet → `helling system config-get`
- operationId: systemConfigPut → `helling system config-set`
- operationId: systemDiagnostics → `helling system diagnostics`

All other operations are covered by the direct command names above (e.g. `helling user list` covers `operationId: userList`).
