# Auth

<!-- markdownlint-disable MD022 MD032 -->

> Status: Draft

Route: `/login` + `/setup`

> **Data source (ADR-014):** Helling API (`/api/v1/*`). Responses in Helling envelope format `{data, meta}`.

---

## Layout

No sidebar, no resource tree. Full-page centered forms. Minimal chrome -- logo, form, footer.

## API Endpoints

- `POST /api/v1/auth/login` -- authenticate (username, password, optional TOTP)
- `POST /api/v1/auth/setup` -- create first admin (requires `/etc/helling/setup-token` while zero users exist)
- `GET /api/v1/auth/setup/status` -- detect whether first-admin setup is still required
- `POST /api/v1/auth/logout` -- invalidate session

> Note: WebAuthn auth flows are deferred to v0.5+ and intentionally excluded from v0.1 UI/API scope.

## Components

### Setup (`/setup`)

- `Card` centered -- "Create the first admin."
- `Form` -- username Input, password Input.Password, confirm Input.Password, setup token Input.Password. One Button: "Create account".
- The setup token is printed by first boot at `/etc/helling/setup-token`; it prevents public first-admin takeover before the real admin is created.
- `/setup` does not configure disks, networking, telemetry, or reboots. The ISO and first-boot service handle host installation before the browser flow starts.

### Login (`/login`)

- `Card` centered -- logo, title
- `ProForm` -- username Input, password Input.Password, optional TOTP Input (shown after first submit if 2FA enabled for user).
- "Remember me" Checkbox (extends JWT expiry)

### First-Load Experience

- After first login, `Tour` component (antd Tour -- dismissable, not blocking) highlights: resource tree, create button, task log, settings.

### Session Expired Modal

- `Modal` overlay (not redirect): "Your session has expired." Password Input + "Re-authenticate" Button. Form data preserved underneath -- never throw away user's work.

## Data Model

- LoginRequest: `username`, `password`
- SetupRequest: `username`, `password`, `setup_token`
- AuthResponse: `token` (JWT), `user{}`, `expires_at`
- Session: `id`, `device`, `ip`, `last_active`

## States

### Empty State

N/A -- these pages always have their forms.

### Loading State

Button shows loading spinner during auth request. Form stays visible.

### Error State

Invalid credentials: inline Alert below form (not a toast). "Invalid username or password." Account locked: "Account locked after 5 failed attempts. Try again in 15 minutes." Server unreachable: "Cannot connect to Helling. Check that the service is running."

## User Actions

- Setup: create first admin account (one-time)
- Login: authenticate with username/password + optional TOTP
- Re-authenticate on session expiry without losing form state
- Dismiss onboarding tour

## Cross-References

- Spec: docs/spec/webui-spec.md (Setup + Login section)
- Identity: docs/design/identity.md (First 5 Minutes, Session Management)
