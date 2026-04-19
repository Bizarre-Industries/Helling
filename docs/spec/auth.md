# Authentication & Authorization Specification

> Status: Draft

Authentication (PAM + JWT + TOTP + WebAuthn), authorization (RBAC via Incus project mapping), identity federation (LDAP/AD/OIDC), API tokens, projects/multi-tenancy via Incus projects, secrets management, and internal CA.

**Proxy integration (ADR-014):** When the proxy forwards requests to Incus, it maps the authenticated Helling user to an Incus project via the `?project=` query parameter. Non-admin users only see resources in their assigned project. Admin users see all projects.

---

## 1. Authentication

### 1.1 PAM Backend

Linux PAM provides the local authentication backend. hellingd authenticates users against the host PAM stack, which supports `/etc/shadow`, SSSD, and any pluggable module installed on the Debian 13 host.

- hellingd calls PAM via `pam_authenticate()` for login requests.
- User existence is verified against PAM before any JWT is issued.
- PAM groups (`helling-admin`, `helling-user`, `helling-audit`) are read at login time and embedded as JWT claims.
- Failed attempts are rate-limited: 5 per 15 minutes per IP + per username.
- PAM configuration lives in `/etc/pam.d/helling`.

### 1.2 JWT Tokens

> See `docs/refs/standards/security-standards.md` section 1 (Application Security > Authentication) for algorithm, expiry, storage, and signing-key rotation requirements.

Summary of normative constraints (defined in security-standards.md):

| Parameter | Requirement |
|---|---|
| Algorithm | EdDSA (Ed25519) or ES256. Never HS256 with weak secret. |
| Access token expiry | 15 minutes |
| Refresh token expiry | 7 days, stored server-side, revocable |
| Access token storage | Memory only (not localStorage) |
| Refresh token storage | httpOnly, Secure, SameSite=Strict cookie |
| Claims | user ID, username, roles, issued_at, expiry, jti |
| Key rotation | Documented procedure, JWKS-based rollover |

Token lifecycle:

1. **Login** -- PAM authenticates credentials, hellingd issues access + refresh token pair.
2. **Refresh** -- Client sends refresh token cookie to `/api/auth/refresh`, receives new access token. Old refresh token is rotated (one-time use).
3. **Revocation** -- Logout invalidates access token (add jti to denylist with TTL), deletes refresh token from server-side store, clears cookies.
4. **Admin force-logout** -- Admin can terminate any user's sessions via `/api/users/{id}/sessions`.

### 1.3 TOTP 2FA

RFC 6238 TOTP with the following parameters:

- Digits: 6
- Period: 30 seconds
- Algorithm: SHA-1 (compatibility with all authenticator apps)
- Issuer: `Helling`

**Setup flow:**

1. User navigates to `/settings/security` and clicks "Enable 2FA".
2. Server generates TOTP secret, returns QR code (otpauth:// URI) and base32 secret for manual entry.
3. User scans QR code with authenticator app.
4. User enters current TOTP code to confirm setup.
5. Server verifies code, enables 2FA, generates and displays 10 recovery codes.
6. User must acknowledge they have saved recovery codes before dialog closes.

**Verify flow:**

1. User submits username + password (PAM auth succeeds).
2. Server detects 2FA is enabled, returns `HTTP 202` with `mfa_required: true` and a short-lived MFA challenge token.
3. Client presents TOTP input field.
4. User enters 6-digit code from authenticator app.
5. Server validates code (allowing +/- 1 time step for clock skew).
6. On success, server issues full JWT access + refresh token pair.
7. On failure, increment attempt counter. After 5 failed TOTP attempts, require recovery code.

> See `docs/refs/standards/security-standards.md` section 1 (2FA) for lockout thresholds and recovery requirements.

### 1.4 WebAuthn / Passkeys

WebAuthn Level 2 with discoverable credentials (passkeys).

- User verification: preferred (biometric/PIN when available).
- Attestation: none (we don't verify authenticator make/model).
- Resident keys: required (discoverable credentials for passwordless login).
- Multiple credentials per user supported.
- Credential metadata stored: credential ID, public key, sign count, transports, created_at, last_used_at, user-assigned name.

**Registration flow:**

1. User navigates to `/settings/security` and clicks "Add Passkey".
2. Server generates registration challenge with user info and relying party ID.
3. Browser prompts authenticator (platform or roaming).
4. User completes gesture (biometric, PIN, or touch).
5. Client sends attestation response to server.
6. Server validates response, stores credential.

**Authentication flow:**

1. User clicks "Login with Passkey" on login page (or enters username first).
2. Server generates authentication challenge, optionally with allowCredentials list.
3. Browser prompts authenticator.
4. User completes gesture.
5. Client sends assertion response to server.
6. Server validates signature, checks sign count, issues JWT.

### 1.5 Recovery Codes

- 10 codes generated when 2FA is first enabled.
- Each code: 16 alphanumeric characters, grouped as `XXXX-XXXX-XXXX-XXXX` for readability.
- Codes are bcrypt-hashed before storage. Plaintext shown only once at generation time.
- Single-use: each code is deleted after successful use.
- Using a recovery code bypasses TOTP/WebAuthn for that single login.
- Users can regenerate all codes (invalidates previous set) from `/settings/security`.
- When fewer than 3 codes remain, show a warning in the dashboard.

### 1.6 API Tokens

> See `docs/refs/standards/security-standards.md` section 1 (API Tokens) for format, hashing, scope, and expiry requirements.

- Format: `helling_<random 40 chars>` (identifiable prefix for secret scanning tools).
- Storage: SHA-256 hash only. The raw token is displayed once at creation time and never again.
- Scopes: `read`, `write`, `admin` -- granular per resource type (e.g., `instances:read`, `storage:write`).
- Expiry: configurable per token, default 90 days, maximum 365 days.
- Revocation: immediate, checked on every request against a revocation list.
- Metadata: description, created_at, last_used_at, last_used_ip, expires_at.
- Management: `/settings/api-tokens` -- list, create, revoke. Bulk revoke supported.

---

## 2. Authentication Realms

### 2.1 LDAP / Active Directory

External identity sources configured under `/settings/authentication`.

**LDAP configuration fields:**

| Field | Description |
|---|---|
| Server | `ldap://ldap.example.com` or `ldaps://ldap.example.com` |
| Port | 389 (LDAP), 636 (LDAPS) |
| Base DN | `dc=example,dc=com` |
| User Attribute | `uid` (LDAP) or `sAMAccountName` (AD) |
| Bind DN | `cn=admin,dc=example,dc=com` |
| Bind Password | Stored encrypted in hellingd SQLite |
| User Filter | `(&(objectClass=person)(memberOf=cn=helling-users,...))` |
| Group Filter | `(&(objectClass=group)(cn=helling-*))` |
| TLS Mode | None, StartTLS, or LDAPS |
| CA Certificate | Upload custom CA cert for LDAPS/StartTLS verification |

**Features:**

- **Test Connection** button validates connectivity and bind credentials before saving.
- **Sync Users Now** performs immediate user/group synchronization.
- **Scheduled Sync** configurable interval (default: daily) to import users and group memberships.
- Synced users appear in `/users` with a realm badge (e.g., `alice@ldap`).
- Group memberships from LDAP are mapped to Helling roles: LDAP groups matching `helling-*` pattern automatically map to corresponding Helling roles.
- Users can exist in multiple realms. Login requires selecting the realm.
- If LDAP server is unreachable, previously synced users can still authenticate using cached credentials (optional, configurable).

### 2.2 OpenID Connect

**OIDC configuration fields:**

| Field | Description |
|---|---|
| Issuer URL | `https://auth.example.com` (must support `.well-known/openid-configuration`) |
| Client ID | Registered client identifier |
| Client Secret | Stored encrypted in hellingd SQLite |
| Scopes | `openid profile email groups` |
| Username Claim | `preferred_username` (configurable) |
| Groups Claim | `groups` (configurable) |
| Autocreate Users | Toggle -- create Helling user on first OIDC login |
| Default Role | Role assigned to auto-created users (default: `helling-user`) |

**Features:**

- **Test Connection** validates issuer URL and client credentials.
- Authorization Code Flow with PKCE for browser-based login.
- Groups claim is mapped to Helling roles using a configurable mapping table.
- If the OIDC provider returns a `groups` claim, those groups are synchronized to Helling roles at each login.
- Token refresh handled by hellingd -- the OIDC id_token is validated once at login, then hellingd issues its own JWT.
- Multiple OIDC providers can be configured simultaneously.
- Provider-specific logout URL support (back-channel logout optional post-v1).

### 2.3 Login Page Realm Selector

The login page adapts based on configured realms:

- **Single realm (PAM only):** Standard username/password form, no selector.
- **Multiple realms:** Dropdown selector showing all configured realms (e.g., PAM, LDAP, corp-oidc).
- **OIDC realms:** Display a "Login with SSO" button per OIDC provider that redirects to the provider's authorization endpoint.
- Realm selection is remembered per browser (localStorage) for convenience.
- URL parameter `?realm=ldap` can pre-select the realm for bookmarking/SSO direct links.

---

## 3. Authorization (RBAC)

### 3.1 Roles

Roles are derived from PAM groups and stored as JWT claims at login time.

**Built-in roles:**

| Role | PAM Group | Permissions |
|---|---|---|
| Administrator | `helling-admin` | Full access to all resources and settings. Can manage users, realms, clusters, storage, networks. |
| User | `helling-user` | Manage own resources within assigned projects. Cannot access cluster settings, storage pool management, or other users' resources. |
| Auditor | `helling-audit` | Read-only access to all resources plus audit logs. Cannot create, modify, or delete anything. |

**Custom roles:**

- Administrators can create custom roles with granular permission sets.
- Custom roles are stored in hellingd SQLite (not PAM groups).
- A user's effective permissions are the union of all assigned roles.
- Custom roles can be assigned per-project for project-scoped access.

### 3.2 Per-Resource ACLs

Fine-grained access control at the individual resource level:

- Any resource (instance, container, storage volume, network, image) can have ACL entries.
- ACL entry: `(user_or_group, resource_id, permission_set)`.
- Permission sets: `view`, `use` (console/exec), `manage` (start/stop/snapshot), `admin` (delete/modify config).
- ACLs override role-based defaults -- a user with `helling-user` role can be granted `admin` on a specific VM.
- ACLs are evaluated after role check: role grants baseline, ACL can expand or restrict.

### 3.3 Permission Model

The authorization chain for any API request:

1. **Authenticate** -- Validate JWT or API token. Extract user ID, roles, project membership.
2. **Role check** -- Does any of the user's roles grant the requested action on this resource type?
3. **Project check** -- Is the resource in a project the user has access to?
4. **ACL check** -- Is there a resource-specific ACL entry for this user?
5. **Quota check** -- For create operations, does the user/project have remaining quota?
6. **Decision** -- Allow if any of steps 2-4 grants access. Deny with structured error otherwise.

Denied requests return:

```json
{
  "error": "Permission denied: cannot delete instance 'vm-web-1'",
  "code": "FORBIDDEN",
  "action": "Check your role permissions or request access from an administrator",
  "doc_link": "/docs/authorization"
}
```

### 3.4 IP-Based Restrictions

Per-user and per-realm source IP restrictions:

- **Allow list:** Only permit login from specified IP addresses or CIDR ranges.
- **Deny list:** Block login from specified IP addresses or CIDR ranges.
- Configured under `/settings/authentication` per realm, or under `/users/{id}/security` per user.
- IP restrictions are evaluated before PAM/LDAP/OIDC authentication (fail fast).
- API token requests are also subject to IP restrictions if configured for the token's owner.
- Supports IPv4 and IPv6 addresses and ranges.

### 3.5 Password Policies

Configurable password policy under `/settings/authentication`:

| Policy | Default | Range |
|---|---|---|
| Minimum length | 8 characters | 8-128 |
| Maximum length | 72 bytes (bcrypt limit) | -- |
| Complexity rules | None (NIST 800-63B compliant) | Optional: require uppercase/number/symbol |
| Breached password check | Enabled (offline HaveIBeenPwned top 100K list) | Toggle |
| Expiry | Disabled | 0-365 days |
| History | 5 previous passwords | 0-24 |
| Rate limiting | 5 attempts per 15 min per IP + username | Configurable |

> See `docs/refs/standards/security-standards.md` section 1 (Passwords) for hashing algorithm requirements (bcrypt or argon2id).

---

## 4. Session Management

### 4.1 Active Sessions

Users and administrators can view and manage active sessions:

- **User view** (`/settings/security/sessions`): List of own active sessions showing IP address, User-Agent, location (GeoIP approximate), created_at, last_active_at.
- **Admin view** (`/users/{id}/sessions`): Same view for any user.
- **Revoke single session:** Invalidates the session's refresh token and adds access token jti to denylist.
- **Revoke all sessions:** Terminates all sessions except the current one. "Revoke all including current" also available.
- Session entries are cleaned up automatically after refresh token expiry (7 days).

### 4.2 Session Timeout

> See `docs/refs/standards/security-standards.md` section 1 (Session Security) for timeout and binding requirements.

- **Inactivity timeout:** 30 minutes (configurable per realm).
- On timeout, the client detects the expired access token and attempts refresh. If the refresh token is also expired or revoked, redirect to login.
- **Form data preservation:** When re-authentication is required, the dashboard saves form state to sessionStorage so users don't lose in-progress work (e.g., a half-completed VM creation wizard).
- **Session binding:** Sessions are tied to IP + User-Agent. A change in either triggers a warning event in the audit log but does not block the session (configurable to block).

### 4.3 Concurrent Session Limit

- Default: unlimited concurrent sessions (but all are viewable and individually revocable).
- Configurable per-realm: set maximum concurrent sessions per user.
- When limit is reached, the oldest session is terminated on new login (FIFO), or login is blocked (configurable behavior).
- Administrators are exempt from concurrent session limits.

---

## 5. Projects / Multi-Tenancy

Projects provide resource isolation and namespace-scoped access within a single Helling instance. Projects map directly to Incus projects.

### 5.1 Project Structure

| Field | Description |
|---|---|
| Name | Unique identifier, used as Incus project name |
| Owner | Primary administrator of the project |
| Members | Users with access, each with a project-scoped role |
| Quotas | Resource limits for the project (see section 7) |
| Allowed Networks | Which host networks this project can use |
| Allowed Storage Pools | Which storage pools this project can use |
| Allowed Templates | Which instance templates this project can access |

### 5.2 Project Management

Admin view at `/admin/projects`:

| Project | Owner | Instances | CPU Used/Quota | RAM Used/Quota |
|---|---|---|---|---|
| web-team | alice | 4 | 8/16 | 16/32 GB |
| data-team | bob | 6 | 12/32 | 48/64 GB |
| personal | carol | 2 | 2/4 | 4/8 GB |

**Create Project** wizard:

1. Name and description.
2. Assign owner and initial members with roles.
3. Set resource quotas (instances, CPU, RAM, storage, K8s clusters).
4. Select permitted networks and storage pools.
5. Select permitted templates.

### 5.3 Incus Project Mapping

- Each Helling project creates a corresponding Incus project.
- Incus project features enabled: `features.images`, `features.profiles`, `features.storage.volumes`, `features.networks` (as configured).
- Resource isolation is enforced at the Incus level -- a user in project A cannot see or interact with resources in project B.
- The `default` Incus project is reserved for administrators and shared infrastructure.
- Project deletion requires all resources within the project to be removed first (relationship check).

---

## 6. Self-Service Portal

Non-administrator users with the `helling-user` role get a simplified dashboard scoped to their project(s).

### 6.1 Self-Service Capabilities

| Allowed | Not Allowed |
|---|---|
| View own resources (instances, containers, volumes) | Manage storage pools |
| Create instances/containers within quota | Manage host networks |
| Access consoles for own instances (VNC, serial, exec) | Cluster management |
| Manage own SSH keys | User management |
| View own resource usage and quota | Realm/auth configuration |
| Create/restore own snapshots | Global settings |
| Manage own API tokens | Firewall rules (host-level) |

### 6.2 Simplified Instance Creation

The self-service "Create Instance" wizard is simpler than the admin version:

- **Exposed fields:** Name, flavor (predefined CPU/RAM/disk combinations), image, SSH keys, network selection (from project-allowed networks).
- **Hidden fields:** NUMA topology, CPU pinning, disk cache modes, raw QEMU options, device passthrough.
- **Flavor-based creation:** Administrators define flavors (e.g., "Small: 1 vCPU, 2 GB RAM, 20 GB disk") that self-service users select instead of specifying raw resources.

### 6.3 Quota Enforcement

- The dashboard displays current usage against quotas prominently: "You have used 3/5 instances, 8/16 CPU cores, 16/32 GB RAM".
- Creation is blocked when any quota would be exceeded, with a clear message explaining which quota is exhausted.
- Quota warnings appear at 80% utilization.
- Users can request quota increases from project owners/admins via an in-dashboard request flow (notification sent to admin).

---

## 7. Resource Quotas

Quotas are enforced at three levels: per-user, per-project (team), and per-storage-pool.

### 7.1 Quota Types

| Quota | Unit | Scope |
|---|---|---|
| Max instances (VMs) | Count | User, Project |
| Max system containers (LXC) | Count | User, Project |
| Max app containers (Podman) | Count | User, Project |
| Max CPU cores | vCPUs | User, Project |
| Max RAM | GB | User, Project |
| Max storage | GB | User, Project, Pool |
| Max K8s clusters | Count | User, Project |
| Max snapshots per instance | Count | User, Project |
| Max backup retention | Days | User, Project |

### 7.2 Quota Enforcement

- Quotas are checked before every create operation. If the operation would exceed any quota, it is rejected with a structured error.
- Quota usage is calculated in real-time from Incus and Podman state (not cached counters that can drift).
- Storage quotas map to Incus project limits (`limits.disk`).
- CPU and RAM quotas map to Incus project limits (`limits.cpu`, `limits.memory`).
- Administrators can set quotas at `/admin/projects/{name}/quotas` and `/admin/users/{id}/quotas`.
- Quota overrides: administrators can temporarily exceed quotas for emergency operations.

---

## 8. Secrets Management

Encrypted key-value store for sensitive configuration, accessible at `/security/secrets`.

### 8.1 Secret Types

| Type | Example |
|---|---|
| Password | Database credentials, service passwords |
| API Key | Stripe keys, cloud provider tokens |
| TLS Certificate | Wildcard certs, service TLS pairs |
| SSH Key | Deploy keys, machine authentication |
| Generic | Any arbitrary sensitive string |

### 8.2 Secret Storage

- Secrets are encrypted at rest in hellingd SQLite using AES-256-GCM.
- Encryption key derived from `HELLING_MASTER_KEY` environment variable or `/etc/helling/master.key` (file permissions 0400).
- Secret values are never returned in API list responses -- only metadata (name, type, used_by, created_at, rotated_at).
- Secret values are retrievable only via explicit GET with audit logging.

### 8.3 Injection Methods

Secrets can be delivered to workloads through multiple mechanisms:

| Method | Workload Type | How |
|---|---|---|
| Environment variable | Podman containers | Injected at container start via Podman secrets API |
| File mount | VMs | Written via guest agent to a specified path |
| Cloud-init reference | VMs, system containers | Referenced in cloud-init user-data template |
| K8s secret sync | K8s clusters | Synced as a Kubernetes Secret object in the target namespace |

### 8.4 Secret Rotation

- **Auto-rotate:** Configurable interval per secret (e.g., 30 days, 90 days).
- **Rotation hook:** An optional script or webhook to execute after rotation (e.g., restart a service, update application config).
- **Notification:** Notify secret owner and configured contacts on rotation.
- **Rotation log:** Full history of rotations with timestamps and trigger (manual/auto).
- **Forced rotation:** Admin can force-rotate any secret immediately.

---

## 9. Internal Certificate Authority

Helling runs its own CA for internal TLS and mTLS, accessible at `/security/certificate-authority`.

### 9.1 Root CA

- Self-signed root CA generated on first boot ("Helling Internal CA").
- Validity: 10 years.
- Key type: ECDSA P-256 (or Ed25519, configurable).
- **Download CA Certificate** button allows clients and browsers to trust the internal CA.
- CA private key stored encrypted on disk, protected by master key.

### 9.2 Issued Certificates

Management table at `/security/certificate-authority`:

| Subject | Type | SANs | Expires | Status |
|---|---|---|---|---|
| `*.helling.local` | Server | DNS wildcard | 2027-04-13 | Valid |
| `db.internal` | Server | IP: 10.0.1.5 | 2027-04-13 | Valid |
| `api-client` | Client | -- | 2027-04-13 | Valid |

**Issue Certificate** wizard:

1. Subject (common name).
2. Subject Alternative Names (DNS names, IP addresses).
3. Type: Server (TLS) or Client (mTLS).
4. Validity period (default: 1 year, max: 5 years).
5. Key type: ECDSA P-256 or Ed25519.

### 9.3 Use Cases

| Use Case | Description |
|---|---|
| Dashboard TLS | Default HTTPS certificate for helling-proxy |
| mTLS between services | Mutual TLS for hellingd-to-Incus or inter-node communication |
| Client certificate auth | API clients can authenticate with client certs instead of tokens |
| K8s admission webhooks | TLS certificates for webhook endpoints in managed K8s clusters |
| Internal service TLS | Certificates for databases, caches, and other internal services |

### 9.4 Certificate Lifecycle

- **Expiry warnings:** Certificates expiring within 30 days generate a warning in the warning engine (checked every 5 minutes).
- **Auto-renewal:** Optional auto-renewal for certificates issued by the internal CA (re-issue with same SANs and subject before expiry).
- **Revocation:** Certificates can be revoked, which adds them to an internal CRL. The CRL is served at `/api/ca/crl` for clients that check revocation.
- **ACME integration (post-v1):** Support for Let's Encrypt / ACME for publicly-facing certificates.
