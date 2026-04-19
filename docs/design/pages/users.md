# Users

<!-- markdownlint-disable MD022 MD032 -->

> Status: Draft

Route: `/users`

> **Data source (ADR-014):** Helling API (`/api/v1/*`). Responses in Helling envelope format `{data, meta}`.

---

## Layout

Sidebar: "Users" selected (admin-only section). Main panel: 4 Tabs.

## API Endpoints

- `GET /api/v1/users` -- user list
- `GET /api/v1/users/:id` -- user detail
- `POST /api/v1/users` -- create user
- `PUT /api/v1/users/:id` -- update user
- `DELETE /api/v1/users/:id` -- delete user
- `GET /api/v1/users/:id/tokens` -- API tokens
- `POST /api/v1/users/:id/tokens` -- create token
- `DELETE /api/v1/users/:id/tokens/:tid` -- revoke token
- `POST /api/v1/users/:id/2fa/enable` -- enable 2FA
- `GET /api/v1/auth/sessions` -- active sessions
- `DELETE /api/v1/auth/sessions/:sid` -- revoke session
- `GET /api/v1/auth/roles` -- role definitions
- `GET /api/v1/auth/permissions` -- permission matrix

## Components

- `Tabs` -- Users | Roles | Permissions | API Tokens

**Users tab:** `ProTable` (username, role Tag, 2FA status Badge, last login, created_at). Actions: Edit, Delete, Enable 2FA. `ModalForm` for Create User (username, password, group/role Select).

**Roles tab:** `ProTable` of PAM group roles (name, description, user count, permissions summary).

**Permissions tab:** ACL editor -- `ProTable` with user rows, resource columns, permission Select per cell (none/read/admin). Or simpler: per-user permission Descriptions.

**API Tokens tab:** `ProTable` (name, scope Tags, created, expires, last used). Create via `ModalForm` (name, scope checkboxes, expiry DatePicker). `Typography.Text copyable` for token value (shown once).

**2FA setup:** `ModalForm` with `QRCode` (antd QRCode component) for TOTP and recovery codes list with copy/download.

## Data Model

- User: `id`, `username`, `role`, `groups[]`, `twofa_enabled`, `twofa_type`, `last_login`, `created_at`
- Token: `id`, `name`, `scope[]`, `created_at`, `expires_at`, `last_used`
- Session: `id`, `device`, `ip`, `location`, `last_active`, `is_current`
- Role: `name`, `description`, `permissions[]`

## States

### Empty State

Only shown if zero additional users beyond admin. "You're the only user. [Create User] to share access."

### Loading State

Cached user list. Token creation returns value immediately.

### Error State

PAM unavailable: banner with link to system logs. Users shown as cached.

## User Actions

- Create/edit/delete users with role assignment
- Enable/disable 2FA (TOTP QR code, recovery codes)
- Create/revoke API tokens with scope and expiry
- View and revoke active sessions
- Manage role definitions and permission assignments

## Cross-References

- Spec: docs/spec/webui-spec.md (Users section)
- Identity: docs/design/identity.md (Session Management)
