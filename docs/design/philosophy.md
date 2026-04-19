# WebUI Design Philosophy

<!-- markdownlint-disable MD040 -->

Function over beauty. Information density over whitespace. Speed over animation. Every pixel earns its place by showing data or enabling action.

---

## Why Proxmox Works (Despite Looking Like 2008)

Proxmox is ugly. ExtJS, cramped fonts, grey everything. But admins who manage 200+ VMs across 16 nodes love it because:

1. **Everything visible without clicking.** The summary tab shows CPU, RAM, disk, IPs, uptime, config — all at once. No "show more" buttons, no collapsed sections, no cards that hide data.
2. **Tables, not cards.** Tables show 50 VMs on screen. Cards show 8. When you manage 200 VMs, you need tables.
3. **No wasted whitespace.** Every pixel shows data. No hero sections, no decorative gradients, no padding-64px.
4. **Instant response.** Click a VM → detail loads instantly. No skeleton screens, no loading spinners, no 300ms fade-in animations. Data appears NOW.
5. **Copy-paste friendly.** IPs, MAC addresses, fingerprints are plain selectable text. Not styled badges you can't copy.
6. **Consistent patterns.** Every resource type: list (table) → detail (tabs) → config (key-value). Learn it once, use it everywhere.
7. **Status at a glance.** Green dot = running. Red dot = stopped. You don't need to hover to see what a color means. The word "Running" is right there next to the dot.
8. **Keyboard works.** Tab, arrow keys, Enter. Power users never touch the mouse.
9. **No tutorial mode.** It assumes you know what a VM is. It doesn't hand-hold. The UI respects your intelligence.
10. **Fast on bad connections.** Works over SSH tunnel to a remote server on hotel WiFi. No heavy JS bundles, no lazy-loaded chunks that fail to load.

## Why Xen Orchestra / Cockpit / Fancy UIs Fail

1. **Cards everywhere.** Pretty cards with rounded corners and shadows. You see 6 VMs before scrolling. Where are the other 194?
2. **Whitespace addiction.** Padding and margins everywhere. The "summary" section shows 3 facts in the space Proxmox shows 30.
3. **Animations.** Slide-in panels, fade transitions, skeleton loading. Feels "modern." Feels SLOW when you're trying to restart a crashed production VM at 3 AM.
4. **Click depth.** Dashboard → Hosts → Host → VMs → VM → Tab → Section → Expand → finally see the IP address. Proxmox: click VM in tree, IP is right there.
5. **Style over substance.** Beautiful empty dashboards with 4 circles showing CPU/RAM. Where's the actual information?
6. **Slow on bad connections.** 2MB JS bundle, lazy-loaded chunks, WebSocket reconnection dance. Unusable over VPN.

## Helling's Design Rules

### Rule 1: Tables by Default, Cards by Exception

```
Tables for:
  - Instance list (VMs, CTs, containers)
  - Snapshot list
  - Backup list
  - Firewall rules
  - Users
  - Tasks
  - Logs
  - Volumes
  - Network list
  - Cluster nodes
  - Everything that could have >10 items

Cards ONLY for:
  - App template gallery (visual browsing)
  - Workspace template gallery
  - Storage pool overview (when ≤6 pools)
  - Dashboard widgets
  - BMC server cards (when ≤8 servers)
```

### Rule 2: Information Density

```
DO:
  - Show 20+ rows per table without scrolling (compact row height)
  - Show key stats inline in list rows (CPU%, RAM%, IP, status)
  - Show complete config in one scroll on detail pages
  - Use abbreviations where unambiguous (GB, vCPU, NIC)
  - Use monospace for technical values (IPs, MACs, fingerprints, UUIDs)

DON'T:
  - Hide information behind "Show more" unless there are genuinely 50+ items
  - Use large padding between sections
  - Put one piece of information per "card"
  - Use hero sections or large headers
  - Add decorative elements that don't convey data
```

### Rule 3: Zero Unnecessary Animation

```
ALLOWED animations:
  - Progress bars (backup progress, migration progress)
  - Spinner on buttons while action is processing (subtle, not full-page)
  - SSE-driven status icon updates (instant color change)
  - Toast notifications (slide in/out, <200ms)
  - Modal open/close (<150ms, CSS only)

BANNED animations:
  - Skeleton loading screens (show data or show nothing)
  - Page transition animations
  - Staggered list item reveal
  - Parallax scrolling
  - Fade-in on scroll
  - Animated counters (just show the number)
  - Hover-to-reveal information (show it always)
```

### Rule 4: Everything Selectable and Copyable

```
All technical values must be plain text, selectable:
  - IP addresses
  - MAC addresses
  - UUIDs
  - Fingerprints
  - Console output
  - Log entries
  - Config values
  - Error messages

Add a tiny copy button (📋) next to values users frequently copy:
  - IP addresses
  - Kubeconfig content
  - API tokens
  - SSH commands
  - Connection strings
```

### Rule 5: Data Loads Instantly or Shows Why It Can't

```
Target: <100ms for any page transition with data visible

How:
  - TanStack Query with staleTime: Infinity, refetchOnWindowFocus
  - Cache aggressively. Show cached data immediately, refresh in background.
  - SSE pushes updates. Don't poll.
  - If data truly isn't available yet: show "Loading..." text inline, not a full-page spinner
  - If API is down: show last known data with "(stale)" indicator, not a blank page

NEVER:
  - Show a full-page loading spinner
  - Show a skeleton screen for >500ms
  - Block the entire UI while one API call completes
  - Show an empty page with "No data" when data is loading
```

### Rule 6: Summary Tab Shows Everything

```
Instance detail → Summary tab must show (without scrolling on 1080p):
  - Status badge + uptime
  - CPU: allocated / usage gauge
  - RAM: allocated / usage gauge
  - Disk: each disk with size + usage
  - Network: each NIC with IP, MAC, traffic counters
  - Config summary: CPU cores, RAM, OS, architecture, boot order
  - Tags (inline, editable)
  - Notes preview (first 2 lines, click to expand)
  - Quick action buttons: Start/Stop/Restart/Console/Snapshot/Backup

All on ONE SCREEN. No scrolling for basic info.
```

### Rule 7: Two-Click Maximum for Any Action

```
Action                          Clicks from any page
─────────────────────────────── ─────────────────────
Start a VM                      2 (click VM in tree → click Start)
Open VM console                 2 (click VM → click Console tab)
See VM's IP address              1 (click VM → visible in summary)
Create a snapshot               3 (click VM → Snapshots tab → Take Snapshot)
Check backup status             2 (click VM → Backup tab)
Add a firewall rule             3 (click VM → Firewall tab → Add Rule)
See storage pool usage          1 (click Storage in tree)
View container logs             2 (click container → Logs tab)
Deploy an app template          2 (click template → Deploy)
Download kubeconfig             2 (click cluster → Download button)

NEVER: more than 4 clicks for any operation
```

### Rule 8: Responsive ≠ Mobile-First

```
Design for 1920x1080 first. This is a server management tool.
Most users sit at a desk with a real monitor.

Responsive breakpoints exist for emergency use from a phone,
not because "mobile-first" is trendy. Tablet and phone layouts
sacrifice information density for touch targets — that's fine
for emergency operations, but desktop is the primary experience.

Desktop (≥1280px): Full three-panel layout, maximum density
Tablet (768-1279px): Tree collapses to icons, two-panel
Phone (<768px): Single-panel, navigation drawer, big action buttons
```

### Rule 9: No JavaScript Framework Bloat

```
Current: 340KB main chunk (86KB gzipped). This is the ceiling, not the floor.

Every new dependency must justify its bundle size:
  - TanStack Table: YES (complex data tables are hard)
  - TanStack Query: YES (caching + deduplication)
  - xterm.js: YES (terminal emulation is hard)
  - spice-html5: YES (VM VGA browser console, ADR-010)
  - @uiw/react-codemirror: MAYBE (load only on pages that need raw YAML/config editing)
  - D3: MAYBE (only for network topology visualization)
  - Three.js: NO (why would we need 3D?)
  - Framer Motion: NO (see Rule 3)
  - React-Beautiful-DnD equivalent: NO (HTML5 drag API works fine)

Dynamic imports for heavy components:
  import('@uiw/react-codemirror') // Only on cloud-init/config editor
  import('spice-html5')       // Only on VM VGA console tab (ADR-010)
  import('xterm')          // Only on terminal tab
```

### Rule 10: Consistent Visual Language

```
Status colors (same everywhere):
  Green  (#22c55e) = Running, Healthy, Success, Connected
  Red    (#ef4444) = Stopped, Error, Failed, Disconnected
  Yellow (#eab308) = Warning, Degraded, Paused
  Blue   (#3b82f6) = Info, Creating, Migrating
  Gray   (#6b7280) = Unknown, N/A, Disabled

Status always = icon + color + text:
  🟢 Running    (never just a green dot without the word)
  🔴 Stopped    (never rely on color alone — accessibility)

Action button colors:
  Primary (blue): Start, Create, Save, Deploy
  Danger (red): Delete, Stop, Destroy, Wipe
  Neutral (gray): Cancel, Close, Back
  Warning (yellow): Restart, Rebuild, Force Stop

Typography:
  Monospace for: IPs, MACs, UUIDs, paths, commands, config values, hashes
  Regular for: everything else
  No serif fonts anywhere (this isn't a blog)

Table density:
  Row height: 36px (compact), 44px (comfortable — default)
  Font size in tables: 13px (not 16px — information density)
  Padding in cells: 8px horizontal, 4px vertical
```

---

## What This Means for Implementation

The dashboard uses React 19 + antd 6 + @tanstack/react-query (via orval-generated hooks). antd's compact theme (configured in `theme.ts`) provides information density. Generated hooks and a shared fetch wrapper handle data fetching, pagination, and auth token injection. Override antd defaults for admin-density:

```css
/* antd compact theme overrides for information-dense admin UI */

/* Tighter tables (configured via antd theme tokens) */
.table th,
.table td {
  padding: 0.4rem 0.6rem;
  font-size: 0.8125rem;
}

/* Compact form controls */
.input,
.select,
.btn {
  height: 2rem;
  min-height: 2rem;
  font-size: 0.8125rem;
}

/* Less rounded everything */
.btn,
.card,
.input,
.select,
.badge {
  border-radius: 0.25rem; /* not 0.5rem */
}

/* Smaller badges */
.badge {
  font-size: 0.6875rem;
  padding: 0.1rem 0.4rem;
  height: auto;
}

/* Compact tabs */
.tabs .tab {
  height: 2rem;
  min-height: 2rem;
  font-size: 0.8125rem;
}

/* No card shadow by default */
.card {
  box-shadow: none;
  border: 1px solid var(--border-color);
}
```

This gives you antd's component architecture with admin-density overrides. Dense, fast, functional.
