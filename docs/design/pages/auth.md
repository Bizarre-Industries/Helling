# Auth

> Status: Draft

Route: `/login` + `/setup`

> **Data source (ADR-014):** Helling API (`/api/v1/*`). Responses in Helling envelope format `{data, meta}`.

---

## Layout

No sidebar, no resource tree. Full-page centered forms. Minimal chrome -- logo, form, footer.

## API Endpoints

- `POST /api/v1/auth/login` -- authenticate (username, password, optional TOTP)
- `POST /api/v1/auth/setup` -- create first admin (only works when zero users exist)
- `POST /api/v1/auth/webauthn/begin` -- start WebAuthn challenge
- `POST /api/v1/auth/webauthn/finish` -- complete WebAuthn
- `POST /api/v1/auth/refresh` -- refresh JWT
- `POST /api/v1/auth/logout` -- invalidate session

## Components

### Setup (`/setup`)
- `Card` centered -- "Welcome to Helling. Create your admin account to get started."
- `ProForm` -- username Input, password Input.Password, confirm Input.Password. One Button: "Create Account".
- Three fields only. No email, no org name, no terms checkbox. Respect the user's time.

### Login (`/login`)
- `Card` centered -- logo, title
- `ProForm` -- username Input, password Input.Password, realm Select (if multiple PAM realms configured, hidden if only one), optional TOTP Input (shown after first submit if 2FA enabled for user).
- WebAuthn Button alternative (if user has registered security key)
- "Remember me" Checkbox (extends JWT expiry)

### First-Load Experience
- After first login, `Tour` component (antd Tour -- dismissable, not blocking) highlights: resource tree, create button, task log, settings.

### Session Expired Modal
- `Modal` overlay (not redirect): "Your session has expired." Password Input + "Re-authenticate" Button. Form data preserved underneath -- never throw away user's work.

## Data Model

- LoginRequest: `username`, `password`, `totp_code`, `realm`
- SetupRequest: `username`, `password`
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
- Login: authenticate with username/password + optional TOTP/WebAuthn
- Re-authenticate on session expiry without losing form state
- Dismiss onboarding tour

## Cross-References

- Spec: docs/refs/spec/webui-spec.md (Setup + Login section)
- Identity: docs/design/identity.md (First 5 Minutes, Session Management)
