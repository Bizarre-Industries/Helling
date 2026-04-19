# What Makes Helling, Helling

Not features. Not endpoints. Not pages. The identity, the product layer, and the hundred small decisions that prove someone gave a shit.

---

## Identity

**Helling** (Dutch: "slipway") — the structure where ships are built and launched.

**One sentence:** Helling is the hypervisor that treats your infrastructure like it matters.

**The belief:** Your homelab runs your family photos, your media, your dev environments, your learning. It deserves the same care as production infrastructure — automated backups, proactive warnings, verified restores, zero-effort safety nets — without requiring a platform team to operate.

**The anti-belief:** "It's just a homelab, it doesn't need to be reliable." That attitude is why people lose data. Helling assumes everything you run matters to someone.

**What Helling is NOT:**
- Not a NAS OS (TrueNAS exists)
- Not a container orchestrator (Portainer exists)
- Not a cloud platform (OpenStack exists)
- Not an enterprise product (Proxmox/VMware exist)

**What Helling IS:**
- The hypervisor for people who care about their infrastructure but don't have a team to manage it
- Auto-snapshots before changes because your time is worth more than disk space
- Backup verification because "backup succeeded" means nothing if restore fails
- Health scores because you shouldn't have to audit your own infrastructure manually
- Power tracking because your electricity bill matters
- Wake-on-LAN because servers shouldn't idle at 150W when nobody needs them

---

## The First 5 Minutes

This is the entire product experience compressed. If these 5 minutes are bad, nothing else matters.

### Minute 0: Discovery
User finds Helling on GitHub or bizarre.industries. README shows:
- One-line description
- Screenshot of the dashboard (the REAL dashboard, not a mockup)
- ISO download link + flash instructions
- "What is this?" section: 3 bullet points
- "How is this different from Proxmox?" section: honest comparison table

### Minute 1: First Run (ISO boot)
User boots the Helling ISO on hardware or a VM. The setup wizard runs:
```
Welcome to Helling.

Hostname:  [helling         ]
Disk:      [sda - 500GB     ] (auto-detected, selectable)
Admin password: [••••••••••      ]
Confirm:        [••••••••••      ]

[Install →]
```
One screen. A few fields. No email, no organization name, no "agree to terms." Respect the user's time.

### Minute 2: Setup Complete
Installation finishes, system reboots. User opens https://<hostname>:8006 and logs in.

### Minute 3: Dashboard (first load)
Dashboard shows REAL data from the host:
```
System: 8 cores, 32GB RAM, 500GB disk (42% used)
Instances: 0 running
Containers: 0 running
```

### Minute 4: First Action
The dashboard empty state for "no user-created containers" shows:
```
No containers yet.

[🚀 Deploy from Template]    [📦 Create Container]

Popular templates: Jellyfin · Gitea · Uptime Kuma · Pi-hole
```

User clicks "Jellyfin" → template form with 3 fields (name, port, data path) → Deploy → container starts in 2 seconds → port link clickable → Jellyfin is running.

### Minute 5: "Aha" Moment
User sees the health score widget appear on dashboard:
```
Infrastructure Health: 65/100
  ⚠ jellyfin: No backup configured (-30)
  ⚠ jellyfin: No description (-2)
  ✓ jellyfin: Running, healthy (+20)
```

User thinks: "Wait, it's telling me I should back this up? And it's tracking my infrastructure's health?" THAT is the moment they decide to install on bare metal.

---

## Runtime Environment

Helling ships as an ISO and installs directly onto hardware (ADR-021). Both Incus and Podman run on the host.

- Incus: VMs, system containers, storage pools, clustering, networking, K8s, workspaces
- Podman: application containers, pods, compose stacks, images
- Both available from first boot. No feature gating or mode switching.

---

## Every State is Designed

### Empty States (per page)

```
/instances (zero instances):
  "No virtual machines or system containers yet."
  [Create Instance]  [Deploy from Template]
  "New to Helling? Create your first virtual machine in 60 seconds. [Quick Start →]"

/containers (zero containers):
  "No application containers yet."
  [Create Container]  [Deploy from Template]  [Import Compose File]

/kubernetes (zero clusters):
  "No Kubernetes clusters yet."
  [Create Cluster]
  "Helling creates K8s clusters from Incus VMs with full lifecycle management."

/backups (zero backups):
  "⚠ No backups configured. Your data is not protected."
  [Configure Backup Schedule]
  "Helling can automatically back up all your instances on a schedule."

/firewall (zero rules):
  "No firewall rules. All traffic is allowed to all instances."
  [Create Rule]  [Apply Default Policy]
  "Recommended: start with deny-all and add specific allow rules."

/templates (zero custom templates):
  "No custom templates. Helling ships with ~50 built-in app templates."
  "You can also [add a custom template repository] or [convert a running instance to a template]."
```

Every empty state has: what this page is for, primary action button, secondary action or help link. Never just "No data."

### Loading States
```
Rule: Show cached data immediately. Refresh in background via SSE/React Query.

First load (no cache): Show skeleton for <500ms. If data arrives within 500ms, skip skeleton entirely.
Subsequent loads: Show cached data instantly. Subtle refresh indicator in corner.
SSE push: Data updates in-place. No loading indicator needed.

NEVER: Full-page spinner. NEVER: "Loading..." text for more than 500ms.
```

### Error States
```
API unreachable:
  Banner (not modal): "Connection lost. Reconnecting..."
  Data: show last cached data with "(stale)" indicator
  Auto-retry every 5s. Remove banner when reconnected.

Permission denied:
  Inline message where content would be: "You don't have permission to view this resource."
  [Request Access] button if RBAC supports it.

Incus unavailable (Incus crashed):
  Banner: "Incus service is unavailable. VM and container management is offline."
  [View System Logs] [Restart Incus Service]
  Container (Podman) features still work.

Operation failed:
  Toast with: what failed, why, what to do.
  "Backup of vm-web-1 failed: storage pool 'default' is full (98%).
   [View Storage] [Retry Backup]"
```

### Offline Behavior
```
Browser loses connection to hellingd:
  - Banner: "Connection lost. Showing cached data."
  - All data shows "(stale)" timestamp
  - Action buttons disabled with tooltip: "Reconnect to perform actions"
  - Auto-reconnect with exponential backoff
  - When reconnected: banner clears, data refreshes, actions re-enable
```

---

## Notification Architecture

Not just warnings in the dashboard. A complete notification system.

### Channels
```
Built-in:
  Dashboard notifications (bell icon with badge count)
  SSE-pushed toasts for real-time events

Configurable external:
  Discord webhook
  Slack webhook
  Telegram bot
  Ntfy (self-hosted push notifications)
  Email (SMTP)
  Gotify
  Generic webhook (any URL)
```

### Event → Channel Routing
```
Settings → Notifications:
  "Critical events" → Discord + Email
  "Warnings" → Discord only
  "Info" → Dashboard only

Event categories:
  Critical:  Instance crashed, Backup failed, Disk failing (SMART), Node offline, Cert expired
  Warning:   Storage >85%, Backup >7d old, Cert <30d, Resource overprovisioned
  Info:      Instance started/stopped, Backup completed, Snapshot created
```

### Quiet Hours
```
"Don't send external notifications between 11 PM and 7 AM unless Critical"
```

---

## Self-Diagnostics

### `helling doctor`

```bash
$ helling doctor
Checking Helling installation...

System:
  ✓ Debian 13 (Trixie)
  ✓ Kernel 6.12.3 (KVM modules loaded)
  ✓ 48 CPU cores, 128GB RAM
  ✓ Disk: 2TB available on /var/lib/incus

Services:
  ✓ incusd: running (v6.23.0)
  ✓ podman: available (v5.3.0, socket activated)
  ✓ hellingd: running (v0.1.0)
  ✓ helling-proxy: running (TLS on :8006)
  ✓ smartd: running
  ✓ chronyd: running (synced)

Configuration:
  ✓ helling.yaml: valid
  ✓ SQLite database: healthy (12.4MB, schema v3)
  ✓ TLS certificate: valid (expires in 342 days)
  ⚠ Audit log: 2.1GB (consider rotation)
  ⚠ No backup of helling.db in last 24h

Network:
  ✓ API accessible on :8080
  ✓ Dashboard accessible on :8006
  ✓ Incus Unix socket: connected
  ✓ Podman socket: connected

Security:
  ✓ AppArmor: enforcing
  ✓ No known CVEs in installed Go modules
  ⚠ 3 instances have no firewall rules

Summary: 14 passed, 3 warnings, 0 errors
```

### Support Bundle

```bash
$ helling support-bundle
Generating support bundle...
  ✓ System info (OS, kernel, hardware)
  ✓ Service status (incusd, podman, hellingd)
  ✓ Configuration (helling.yaml, sanitized — passwords redacted)
  ✓ Recent logs (last 1000 lines each)
  ✓ Instance list (names, status, config — no user data)
  ✓ Storage pool status
  ✓ Network configuration
  ✓ Firewall rules
  ✓ Recent audit entries (last 100)
  ✓ Doctor output

Bundle saved: /tmp/helling-support-2026-04-13.tar.gz (2.4MB)
Upload this file when filing a GitHub issue.
No user data, passwords, or private keys are included.
```

---

## Self-Service DNS

Instances automatically get DNS names resolvable from the host and from other instances.

```
Instance "web-server" created:
  → Automatically registered: web-server.helling.local
  → Resolvable from: host, all other instances, all containers

DNS entries:
  web-server.helling.local          → 10.0.0.50
  db.helling.local                  → 10.0.0.51
  redis.container.helling.local     → 172.17.0.3
  my-cluster.k8s.helling.local      → 10.0.0.100 (K8s API)
```

Implementation: Incus provides DNS for instances via dnsmasq/CoreDNS. For Podman containers, hellingd writes to /etc/hosts or runs a lightweight CoreDNS. Domain suffix configurable in helling.yaml.

Users can access VMs by name instead of remembering IPs. `ssh ubuntu@web-server.helling.local` instead of `ssh ubuntu@10.0.0.50`.

---

## Remote Access (Built-in VPN)

Access your instances from anywhere without port forwarding or exposing services to the internet.

```
Settings → Remote Access:
  [Enable Tailscale Integration]
  OR
  [Enable Built-in WireGuard]

Tailscale:
  - hellingd registers as a Tailscale node
  - All instances accessible via Tailscale network
  - Dashboard accessible from phone/laptop anywhere
  - Zero configuration port forwarding

Built-in WireGuard:
  - hellingd generates WireGuard config
  - User downloads config or scans QR code
  - Phone/laptop connects → full access to all instances
  - Dashboard shows connected clients
```

This solves the #1 homelab problem: "How do I access my stuff from outside my house?"

---

## Update Experience

### In-Dashboard Update Notification
```
After upgrade, first login shows:
┌──────────────────────────────────────────────────────────────┐
│ 🎉 Helling v0.2.0                                            │
│                                                               │
│ What's new:                                                   │
│ • K8s cluster management with rolling upgrades                │
│ • Helm chart deployment from dashboard                        │
│ • Wake-on-LAN for energy-efficient homelabs                   │
│ • 47 bug fixes                                                │
│                                                               │
│ [Full Changelog →]  [Dismiss]                                 │
└──────────────────────────────────────────────────────────────┘
```

### Update Check
```
hellingd checks bizarre.industries/api/latest daily (opt-out in helling.yaml).
Settings → Updates shows:
  Current: v0.1.0
  Latest:  v0.2.0
  [View Changes]  [Upgrade Now]

Security patches show a banner on ALL pages:
  "⚠ Security update available: v0.1.1 fixes CVE-2026-XXXXX. [Upgrade Now]"
```

---

## Session Management

```
Settings → Security → Active Sessions:
  | Device | IP | Location | Last Active | |
  | Chrome/Mac | 192.168.1.10 | Local | Now (current) | |
  | Safari/iOS | 100.64.0.3 | Tailscale | 2h ago | [Revoke] |
  | API Token: deploy | 10.0.0.5 | Local | 5h ago | [Revoke] |

  [Revoke All Other Sessions]
```

### Session Timeout
```
JWT expires → modal overlay (not redirect):
  "Your session has expired."
  [Password: ________] [Re-authenticate]

If user was mid-form: form data preserved. Re-auth → form still there.
NEVER throw away user's work because of session timeout.
```

---

## Dashboard Branding

```
Settings → Appearance:
  Logo: [Upload]  (replaces Helling logo in top bar)
  Accent color: [Color picker]  (primary button color, active tab, badges)
  Login message: [Custom text]  (shown on login page)
  Favicon: [Upload]

Presets:
  Helling Default (dark blue)
  Homelab Green
  Enterprise Gray
```

Companies/power users can brand the dashboard without forking the code.

---

## Weekly Digest Email

```
Opt-in: Settings → Notifications → "Weekly infrastructure digest"

Subject: "Helling Weekly: 12 instances healthy, 2 warnings"

Body:
  Infrastructure Health: 78/100 (↑3 from last week)
  
  Running: 12 instances, 8 containers
  Stopped: 2 instances (vm-old: 95 days, ct-legacy: 12 days)
  
  Backups: 10/14 instances backed up this week
  ⚠ Missing: vm-dev, ct-test, container-redis, container-temp
  
  Storage: default 72% (↑5%), fast-pool 45% (↓2%)
  
  Top event: vm-build-server restarted 3 times (check logs)
  
  Power: ~340W average, ~245 kWh estimated, ~$35/month
```

You don't need to log in to know your infrastructure is healthy.

---

## What This Adds Up To

These aren't features. They're the product layer that separates "a dashboard that wraps Incus" from "a product that manages my infrastructure."

| Category | What it does | Why it matters |
|----------|-------------|---------------|
| First 5 minutes | Discovery → Docker run → Setup → First container → Aha moment | Users decide in 5 minutes |
| Runtime environment | Incus + Podman from first boot, no mode gating | No confusion about what's available |
| Empty states | Every zero-data page guides, not blocks | New users never feel lost |
| Loading/error states | Cached data, graceful degradation, actionable errors | Never a blank page |
| Notification system | 7 channels, event routing, quiet hours | Problems reach you, not just the dashboard |
| Self-diagnostics | `helling doctor`, support bundle | Self-service troubleshooting |
| Self-service DNS | instance.helling.local | Stop memorizing IPs |
| Remote access | Built-in VPN (Tailscale/WireGuard) | Access from anywhere |
| Update experience | In-dashboard changelog, security banners | Users stay current |
| Session management | Active sessions, revoke, graceful timeout | Security without frustration |
| Branding | Logo, colors, login message | Make it yours |
| Weekly digest | Email summary of infrastructure health | Peace of mind without logging in |
