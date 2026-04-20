# Magic Touches

<!-- markdownlint-disable MD040 MD032 MD060 -->

Features that make people stop and say "no other hypervisor does this." Not more endpoints. Not more pages. Things that change how people think about managing infrastructure.

---

## 1. Automatic Snapshot Before Every Destructive Operation

Like Time Machine. Before any risky change, Helling auto-snapshots with no user action.

```text
User clicks "Resize disk from 30GB → 50GB"
  → hellingd auto-creates snapshot "pre-resize-2026-04-13-14:22"
  → Performs resize
  → If fails → auto-rollback to snapshot
  → If succeeds → snapshot retained for 24h, then auto-pruned

Applies to:
  - Config changes (CPU, RAM, NIC, boot order)
  - Disk resize
  - Migration
  - K8s upgrades
  - Profile changes
  - Rebuild
  - ANY operation tagged as "destructive" in the API spec

Dashboard shows:
  ⟲ "Auto-snapshot created. Undo this change?" [Rollback] [Dismiss]
  Toast with one-click rollback for 60 seconds after every destructive op.
```

No other hypervisor does this by default. Proxmox requires manual snapshots. VMware requires manual checkpoints. This is a zero-effort safety net.

Configurable: `auto_snapshot.enabled: true`, `auto_snapshot.retention: 24h`, `auto_snapshot.exclude_tags: ["ephemeral"]`

---

## 2. VM Screenshot Thumbnails (Deferred from v0.1)

Status: deferred from v0.1 scope; keep as post-v0.1 design direction.

The resource tree and instance list show a tiny screenshot of what's currently on each VM's display. You can SEE which VM has the Windows login screen, which one is running htop, which one is showing an error — without opening any console.

```text
Resource tree:
  💻 VMs
    🟢 windows-dev    [🖼 tiny screenshot showing desktop]
    🟢 ubuntu-server   [🖼 tiny screenshot showing terminal]
    🔴 kali            [grayed out — stopped]

Instance list:
  | Screen | Status | Name | CPU | RAM |
  | [📸]   | 🟢     | win  | 23% | 4GB |
  | [📸]   | 🟢     | ubu  | 5%  | 2GB |
```

Implementation: Incus API provides VGA console WebSocket. Take a single frame every 30 seconds, encode as 120x90 JPEG thumbnail, cache in hellingd memory. ~3KB per thumbnail. 100 VMs = ~300KB total.

Configurable: `thumbnails.enabled: true`, `thumbnails.interval: 30s`, `thumbnails.quality: 60`

No other hypervisor does this. vSphere has console previews but they require the full vSphere client. Proxmox doesn't do it at all.

---

## 3. Config Change Preview (terraform plan for everything)

Before any config change is applied, show exactly what will change and what the consequences are.

```text
User changes VM config: CPU 2→4, RAM 4GB→8GB

Dashboard shows diff BEFORE applying:
┌──────────────────────────────────────────────────┐
│ Config Change Preview                             │
├──────────────────────────────────────────────────┤
│  cpu.cores:  2 → 4                                │
│  memory:     4GB → 8GB                            │
│                                                   │
│  ⚠ Requires restart: Yes (CPU hotplug not         │
│    supported on this instance type)               │
│  ⚠ Host impact: node-1 will be at 87% RAM        │
│    after this change                              │
│  ✓ Auto-snapshot will be created before applying  │
│                                                   │
│  [Apply Changes]  [Cancel]                        │
└──────────────────────────────────────────────────┘

For network changes:
  "Changing network from bridge0 to bridge1 will change IP address.
   Current IP: 192.168.1.50. New IP: will be assigned by DHCP on bridge1."

For storage changes:
  "Growing disk from 30GB → 50GB. This cannot be undone (shrinking not supported).
   Pool 'default' will be at 78% after this change."

For firewall changes:
  "Adding rule: allow TCP 22 from 0.0.0.0/0. This exposes SSH to ALL networks."
```

Every change shows: what changes, what stays the same, side effects, warnings, whether restart is needed, host resource impact. Like `terraform plan` but for every single operation.

---

## 4. Backup Verification (Prove Your Backups Work)

Don't just back up. Periodically prove the backup is restorable.

```text
Scheduled (weekly by default):
  1. Pick latest backup of each instance tagged "verify"
  2. Restore to temporary instance (name: "verify-{original}-{timestamp}")
  3. Boot the temporary instance
  4. Wait for guest agent to report "running" (timeout: 5 min)
  5. Run health checks:
     - Guest agent responds? ✓
     - Network interface up? ✓
     - SSH port open? ✓ (if configured)
     - HTTP port responds? ✓ (if configured)
  6. Record result in backup verification log
  7. Delete temporary instance
  8. Notify: "Backup verification: 12/14 passed, 2 failed (vm-old, ct-legacy)"

Dashboard shows:
  Backup status per instance:
  ✓ vm-web-1: Last backup 2h ago, last verified 3d ago (PASS)
  ✓ vm-db-1:  Last backup 6h ago, last verified 3d ago (PASS)
  ⚠ vm-old:   Last backup 12d ago, last verified 3d ago (FAIL: boot timeout)
  ✗ ct-dev:   No backup configured
```

No other hypervisor does automatic backup verification. Proxmox can back up. Veeam can verify. Nobody auto-verifies on a schedule and reports results in the dashboard.

---

## 5. Infrastructure Health Score

Every resource gets a 0-100 health score. Dashboard shows aggregate score.

```text
vm-web-1: Health 92/100
  ✓ Running (uptime: 14d)         +20
  ✓ Backup: 2 hours ago           +20
  ✓ Backup verified: 3 days ago   +15
  ✓ CPU usage: 45% (healthy)      +15
  ✓ RAM usage: 62% (healthy)      +15
  ⚠ No firewall rules             -5
  ⚠ Guest agent not installed     -3
  ✗ Disk 85% full                 -5

vm-old: Health 31/100
  ✓ Running                       +20
  ✗ No backup (never)             -30
  ✗ CPU overprovisioned (8→avg 0.3) -10
  ✗ Stopped for 90+ days          -20
  ⚠ No tags                       -2
  ⚠ No description                -2
  ✗ Guest agent not responding     -5

Dashboard:
  Infrastructure Health: 78/100
  ████████████████████░░░░░ 78%

  12 healthy │ 3 warnings │ 1 critical
```

Criteria (configurable weights):

- Running/responsive (+20)
- Recent backup (+20), verified backup (+15)
- CPU utilization healthy (+15), RAM healthy (+15)
- Firewall configured (+5)
- Guest agent installed (+5)
- Tags assigned (+2), description set (+2)
- Penalties: no backup (-30), overprovisioned (-10), stale snapshots (-5), disk full (-10)

---

## 6. Infrastructure Blueprints

Define a multi-resource environment as YAML. Deploy everything at once. Tear down with one command. Like Docker Compose but for VMs + CTs + networks + firewall + storage.

```yaml
# blueprint: dev-environment.yaml
name: dev-environment
description: "Full development stack: app server + database + redis + monitoring"

networks:
  - name: dev-net
    type: bridge
    ipv4.address: 10.99.0.1/24
    ipv4.dhcp: true

instances:
  - name: dev-app
    type: container
    image: images:ubuntu/24.04
    profiles: [default, dev-app]
    cpu: 2
    memory: 4GB
    disk: 20GB
    network: dev-net
    cloud-init:
      packages: [nginx, nodejs, npm]

  - name: dev-db
    type: container
    image: images:ubuntu/24.04
    cpu: 2
    memory: 8GB
    disk: 50GB
    network: dev-net
    cloud-init:
      packages: [postgresql-16]

containers:
  - name: dev-redis
    image: redis:alpine
    network: dev-net
    ports: ["6379:6379"]

firewall:
  - direction: in
    action: accept
    protocol: tcp
    dport: "80,443"
    source: 0.0.0.0/0
    target: dev-app
```

```bash
helling blueprint deploy dev-environment.yaml      # Create everything
helling blueprint status dev-environment            # Show status of all resources
helling blueprint destroy dev-environment           # Tear down everything
helling blueprint clone dev-environment staging     # Clone entire environment
helling blueprint export dev-environment > env.yaml # Export current state as blueprint
```

Dashboard: /blueprints page with deploy/status/destroy/clone. "Save current selection as blueprint" button in instance list (select VMs → save as blueprint).

---

## 7. Wake-on-LAN Integration

For physical servers in a cluster. Server is powered off (saves electricity) → user needs a VM → Helling wakes the server → server boots → VM starts.

```text
Cluster node status:
  node-1: 🟢 Online (4 VMs running)
  node-2: 🟢 Online (2 VMs running)
  node-3: 💤 Sleeping (WoL available, MAC: xx:xx:xx:xx:xx:xx)

User starts a VM assigned to node-3:
  → hellingd sends WoL magic packet to node-3's MAC
  → Dashboard shows: "Waking node-3... (estimated 90s)"
  → node-3 boots → incusd starts → joins cluster
  → VM starts on node-3
  → Dashboard shows: "node-3 online, vm-build started"

Auto-sleep policy:
  "If node has no running instances for 30 minutes, send IPMI shutdown"
  (Configurable per node)
```

Energy-efficient homelabs. Dell PowerEdges idle at 80-150W each. Sleeping nodes = $0 electricity. Wake only when needed.

Implementation: WoL via `bmclib` or raw magic packet. Sleep via IPMI shutdown. Requires BMC or WoL-capable NIC.

---

## 8. Infrastructure Changelog

Git-like history of every config change to every resource. Diff between any two points in time.

```bash
helling changelog vm-web-1
  2026-04-13 14:22  admin   config  cpu.cores: 2 → 4, memory: 4GB → 8GB
  2026-04-12 09:15  admin   start   instance started
  2026-04-11 23:00  system  backup  automatic backup (daily)
  2026-04-10 16:30  admin   config  cloud-init: added SSH key
  2026-04-08 11:00  admin   create  created from image ubuntu/24.04

helling changelog vm-web-1 --diff 2026-04-08..2026-04-13
  + cpu.cores: 4 (was 2)
  + memory: 8GB (was 4GB)
  + cloud-init.ssh_keys: [added 1 key]
  ~ uptime: 5d 3h
  ~ snapshots: 2 (was 0)
  ~ backups: 3 (was 0)

Dashboard:
  Instance Options tab → "History" sub-tab
  Timeline of all changes with diffs
  "Revert to this point" button per entry
```

Implementation: every config change stored in SQLite audit table with before/after JSON. Diff computed on read.

---

## 9. Migration Assistant

Import from other platforms. Not just disk images — configs, networks, firewall rules.

```bash
helling import proxmox --host 192.168.1.100 --user root --token xxx
  Scanning Proxmox server...
  Found: 12 VMs, 5 CTs, 3 storage pools, 4 networks

  Select resources to import:
  [x] vm-100 (ubuntu-web, 4 cores, 8GB, 50GB)
  [x] vm-101 (windows-dev, 8 cores, 16GB, 100GB)
  [ ] vm-102 (old-test, stopped for 200 days)
  [x] ct-200 (dns-server, 1 core, 512MB)

  Import plan:
  - Download vm-100 disk (50GB) → convert to Incus volume
  - Download vm-101 disk (100GB) → convert to Incus volume
  - Download ct-200 rootfs → convert to Incus container
  - Import network configs (bridge0, bridge1)
  - Import firewall rules (12 rules)

  Estimated time: ~45 minutes
  [Start Import]

Dashboard: /settings → "Import" tab → connect to source platform → browse → select → import
```

Sources: Proxmox (API), VMware (OVA/OVF), Docker (compose stacks), raw disk images (qcow2/vmdk/raw), Incus remote server.

---

## 10. Smart Command Palette

Cmd+K opens a command palette that understands natural language AND structured commands.

```text
Cmd+K: "show vms using more than 4gb ram"
  → Filters instance list: RAM > 4GB

Cmd+K: "create ubuntu vm 4 cores 8gb"
  → Opens create wizard pre-filled: Ubuntu 24.04, 4 cores, 8GB

Cmd+K: "why is vm-db slow"
  → Shows: CPU 95%, RAM 89%, disk I/O 120MB/s write
  → Suggests: "vm-db CPU is at 95%. Consider adding cores or migrating to a less loaded node."

Cmd+K: "stop all dev vms"
  → Finds all VMs tagged "dev" → confirms → stops them

Cmd+K: "backup everything"
  → Lists all instances without recent backup → offers to back them all up

Cmd+K: "what changed today"
  → Shows infrastructure changelog for today

Structured commands also work:
  Cmd+K: "go instances" → navigates to /instances
  Cmd+K: "go vm-web-1 console" → opens console for vm-web-1
  Cmd+K: "start vm-web-1" → starts the VM
```

Implementation: antd Command palette (cmdk-style). Natural language parsed locally with simple keyword matching (not LLM — must work offline and instant). Structured commands are direct actions.

---

## 11. Power Consumption Tracking

Estimate per-VM power consumption based on CPU/RAM usage. Track total infrastructure power.

```text
Dashboard widget:
  ⚡ Power Consumption
  Total: ~340W (estimated)

  node-1 (Dell R730): ~180W
    vm-web-1:    ~25W (2 cores @ 45%)
    vm-db-1:     ~40W (4 cores @ 80%)
    ct-build:    ~15W (2 cores @ 30%)
    host overhead: ~100W

  node-2 (Minisforum MS-01): ~65W
    vm-dev:      ~10W (2 cores @ 15%)
    host overhead: ~55W

  node-3 (sleeping): 0W 💤

Monthly estimate: ~245 kWh (~$35/month @ $0.14/kWh)

Recommendations:
  "node-2 has 1 VM using 15% CPU. Consider migrating to node-1 and sleeping node-2."
  "Estimated savings: ~47 kWh/month (~$7/month)"
```

Implementation: BMC reports actual power draw (via IPMI/Redfish sensor data). For nodes without BMC, estimate from TDP × utilization. Per-VM estimate = host power × (VM CPU/RAM share of total).

---

## 12. One-Click Environment Cloning

Clone not just one VM but an entire environment: VMs, containers, networks, firewall rules, volumes.

```sql
Select multiple resources → "Clone as Environment":
  Source: production-stack (vm-web, vm-db, container-redis, net-prod, 3 firewall rules)

  Clone settings:
    Prefix: staging-
    Network: Create new isolated network (10.100.0.0/24)
    DNS: Update to staging hostnames

  Result:
    staging-vm-web (clone of vm-web, connected to staging-net)
    staging-vm-db (clone of vm-db, connected to staging-net)
    staging-redis (clone of container-redis, connected to staging-net)
    staging-net (new bridge, 10.100.0.0/24)
    3 firewall rules (cloned, scoped to staging resources)

  Total: cloned in 45 seconds (LXC) or 3 minutes (VM full copy)
```

Creates a complete isolated copy of a multi-resource environment. For dev/staging/testing. Destroy with one click when done.

---

## Summary

| #   | Magic Touch                          | Why it's magic                                       | Effort |
| --- | ------------------------------------ | ---------------------------------------------------- | ------ |
| 1   | Auto-snapshot before destructive ops | Zero-effort undo. No other hypervisor.               | Low    |
| 2   | VM screenshot thumbnails             | See what's on screen without opening console.        | Medium |
| 3   | Config change preview                | terraform plan for every operation.                  | Medium |
| 4   | Backup verification                  | Prove backups work. Not just "backup succeeded."     | Medium |
| 5   | Infrastructure health score          | 0-100 per resource. Overall infrastructure grade.    | Low    |
| 6   | Infrastructure blueprints            | Compose for VMs. Deploy/destroy entire environments. | High   |
| 7   | Wake-on-LAN                          | Energy-efficient homelab. Sleep unused nodes.        | Low    |
| 8   | Infrastructure changelog             | Git-like history + diff for every config change.     | Medium |
| 9   | Migration assistant                  | Import from Proxmox/VMware with configs + networks.  | High   |
| 10  | Smart command palette                | Natural language + structured commands.              | Medium |
| 11  | Power consumption tracking           | Per-VM watts, monthly cost, savings recommendations. | Low    |
| 12  | Environment cloning                  | Clone entire multi-VM stacks for dev/staging.        | Medium |

These are the features that get blog posts written, Reddit threads started, and YouTube videos made. Not because they're technically complex, but because they solve problems nobody else bothered to solve.
