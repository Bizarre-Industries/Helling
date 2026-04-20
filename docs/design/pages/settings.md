# Settings

> Status: Draft

Route: `/settings`

> **Data source (ADR-014):** Helling API (`/api/v1/*`). Responses in Helling envelope format `{data, meta}`.

---

## Layout

Sidebar: "Settings" selected (admin-only). Main panel: 5 Tabs. Each tab uses ProForm or Descriptions.

## API Endpoints

- `GET /api/v1/settings` -- all settings
- `PUT /api/v1/settings` -- update settings
- `GET /api/v1/certificates` -- TLS certificates
- `POST /api/v1/certificates/acme` -- request ACME cert
- `POST /api/v1/certificates/upload` -- upload cert
- `GET /api/v1/notifications/channels` -- notification channels
- `POST /api/v1/notifications/channels` -- add channel
- `POST /api/v1/notifications/test` -- test notification
- `GET /api/v1/system/updates` -- update check
- `POST /api/v1/system/updates/apply` -- apply update
- `GET /api/v1/registries` -- container registries
- `POST /api/v1/registries` -- add registry

## Components

- `Tabs` -- General | Certificates | Notifications | Updates | Registries | Keyboard

**General tab:** `ProForm` -- hostname, DNS domain, timezone Select, backup defaults, appearance (logo Upload, accent color ColorPicker, login message TextArea, favicon Upload). Preset theme Segmented (Default, Homelab Green, Enterprise Gray).

**Certificates tab:** `Descriptions` (current cert: expiry, issuer, fingerprint copyable). `Button`: "Upload Certificate" or "Generate from ACME". ACME `ProForm` (domain, email, provider Select: Let's Encrypt / ZeroSSL). Cert viewer.

**Notifications tab:** `ProTable` of channels (name, type Tag: Discord/Slack/Telegram/Ntfy/Email/Gotify/Webhook, status Badge). `ModalForm` to add channel. Test notification `Button` per channel. Event routing config: `ProForm` with severity-to-channel mapping. Quiet hours `TimePicker.RangePicker`.

**Updates tab:** `Descriptions` (current version, latest version, release date). [View Changes] [Upgrade Now] Buttons. Security patch banner when applicable.

**Registries tab:** `ProTable` of container registries (name, URL, auth status Badge). `ModalForm` to add registry (URL, username, password/token). Test connection Button.

**Keyboard tab:** Per-user keymap editor. See docs/design/keyboard.md for the full architecture; this is the UI surface.

- Top bar: `Switch` "Vim mode" (toggles `vim_mode` preference; off by default per ADR-049). `Switch` "Use local overrides on this device" (localStorage layer).
- Conflict banner: `Alert type="error"` when two actions in overlapping contexts share the same binding; rows involved highlight.
- Main table: `ProTable` of every action registered in `web/src/keyboard/registry.ts`:
  - Columns: Action ID (monospace, copyable), Label, Category filter, Context (monospace tag), Default binding (`Kbd` chips, read-only), Current binding (`Kbd` chips + inline "Record keystroke" button), Reset-to-default row action.
  - Search by label or action ID.
  - Filter by category, vim-only, overridden-only.
- "Record keystroke" input: opens a popover that listens for a chord; escape cancels; enter confirms. Validates against the reserved-shortcut list (`Cmd+T`, `Cmd+W`, `Cmd+N`, `Cmd+Tab`, `F5`, `Cmd+L`, `Cmd+S`) and warns; `Cmd+P` and `Cmd+Shift+P` are allowed per ADR-049.
- Bulk actions: Export as JSON, Import from JSON, Reset all to default.

## Data Model

- Settings: `hostname`, `domain`, `timezone`, `backup_defaults{}`, `appearance{}`
- Certificate: `subject`, `issuer`, `expiry`, `fingerprint`, `type` (self-signed/acme/uploaded)
- NotificationChannel: `id`, `name`, `type`, `config{}`, `enabled`
- UpdateInfo: `current_version`, `latest_version`, `release_notes`, `security_patch`
- Registry: `id`, `name`, `url`, `auth_configured`

## States

### Empty State

Settings always have defaults. Notifications tab: "No notification channels. [Add Channel] to get alerts for critical events."

### Loading State

Settings cached. Update check runs in background.

### Error State

ACME failure: inline Alert with error. Notification test failure: toast with reason.

## User Actions

- Edit general settings (hostname, DNS, timezone, appearance)
- Upload or generate TLS certificates
- Add/edit/delete/test notification channels
- Configure event-to-channel routing and quiet hours
- Check for and apply updates
- Add/remove container registries
- Toggle vim mode, re-bind any keyboard shortcut, export/import keymap as JSON

## Cross-References

- Spec: docs/spec/webui-spec.md (Settings section)
- Identity: docs/design/identity.md (Branding, Update Experience, Notification Architecture)
- Keyboard: docs/design/keyboard.md (architecture, default keymap, vim mode)
- ADR-049 (vim mode and keymap surfacing)
