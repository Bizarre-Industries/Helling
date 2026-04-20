# Helling WebUI Specification

<!-- markdownlint-disable MD040 -->

## Stack

```text
React 19 + Vite (SPA, no SSR)
antd 6                          → Core components
@ant-design/pro-components      → ProTable, ProForm, StepsForm, ProLayout, Descriptions
@ant-design/charts              → Charts (G2-based, antd theme integrated)
@tanstack/react-query           → Data fetching (via hey-api/openapi-ts generated hooks/options for Helling API)
react-router-dom v7             → Routing
xterm.js                        → Terminal (serial console, exec, logs)
spice-html5                     → VM VGA console (SPICE protocol path, ADR-010)
@uiw/react-codemirror           → YAML editor (cloud-init, compose, hookscripts, dynamic import)
lucide-react                    → Icons
fetch wrapper                   → Shared request wrapper with JWT injection + request ID propagation
```

## Data Flow (ADR-014, ADR-015)

The dashboard uses three API clients:

- **hellingClient** — Helling-specific endpoints (`/api/v1/*`). Uses hey-api/openapi-ts generated SDK/hooks. Envelope format: `{data, meta}`.
- **incusClient** — Incus proxy endpoints (`/api/incus/*`). Typed fetch wrapper. Native Incus response format.
- **podmanClient** — Podman proxy endpoints (`/api/podman/*`). Typed fetch wrapper. Native Podman response format.

All three include the JWT token in the Authorization header via a shared fetch wrapper.

Pages that show Incus resources (instances, storage, networks, images, cluster) fetch from the Incus proxy. Pages that show Podman resources (containers, pods, volumes) fetch from the Podman proxy. Pages for Helling features (users, schedules, webhooks, audit, BMC, K8s, settings) use hey-api/openapi-ts generated SDK/hooks from the Helling API.

## Design Rules

Function over beauty. See docs/design/philosophy.md for full 10 rules.

1. **ProTable for every list.** One component gives sort, filter, paginate, bulk select, density toggle, column visibility, search, loading states. ~30 lines per page.
2. **Descriptions for every detail summary.** Key-value display with copyable text, responsive columns.
3. **StepsForm for every create wizard.** Multi-step with validation, step persistence.
4. **Tables, not cards.** Cards ONLY for: app templates gallery, workspace templates, storage pool overview.
5. **Compact theme.** Override antd token: small border radius, tight padding, 13px body font in tables.
6. **No animations.** ConfigProvider: motion=false. Progress bars and toasts only.
7. **Two-click max.** Any action reachable in ≤2 clicks from resource tree.

## Theme

```tsx
<ConfigProvider
  theme={{
    token: {
      borderRadius: 4,
      fontSize: 13,
      controlHeight: 32,
      motion: false,
    },
    components: {
      Table: { cellPaddingBlockSM: 4, cellPaddingInlineSM: 8, fontSize: 13 },
      Tabs: { horizontalItemPaddingSM: '8px 12px' },
      Badge: { fontSize: 11 },
    },
  }}
>
```

## Layout (ProLayout)

```text
┌──────────────────────────────────────────────────────────────┐
│ Top: Logo │ Global Search (Cmd+K) │ Warnings Badge │ User   │
├────────────┬─────────────────────────────────────────────────┤
│ Resource   │              Content Area                       │
│ Tree       │  ┌───────────────────────────────────────────┐  │
│ (antd Tree │  │ Breadcrumb                                │  │
│  with      │  │ Tabs (antd Tabs)                          │  │
│  search,   │  ├───────────────────────────────────────────┤  │
│  icons,    │  │ ProTable / Descriptions / Form / Console  │  │
│  drag for  │  │                                           │  │
│  migration)│  │ Dense, data-rich content                  │  │
│            │  └───────────────────────────────────────────┘  │
│ ★ Pinned   │                                                 │
│ 🕐 Recent  │                                                 │
│ ──────     │                                                 │
│ Dashboard  │                                                 │
│ 🖥 node-1  │                                                 │
│  💻 VMs    │                                                 │
│  📦 CTs    │                                                 │
│  🐳 Podman │                                                 │
│  ☸ K8s     │                                                 │
│ ──────     │                                                 │
│ Templates  │                                                 │
│ Workspaces │                                                 │
│ Storage    │                                                 │
│ Network    │                                                 │
│ Firewall   │                                                 │
│ Backups    │                                                 │
│ BMC        │                                                 │
│ Users      │                                                 │
│ Settings   │                                                 │
├────────────┴─────────────────────────────────────────────────┤
│ Task Log: antd Table (compact) with Progress bars, SSE       │
└──────────────────────────────────────────────────────────────┘
```

Resource tree: `<Tree>` with `showIcon`, `searchValue`, draggable (for migration), `onRightClick` (context menu via `<Dropdown>`). SSE updates tree data.

Task log: bottom drawer (`<Drawer>` placement=bottom, collapsible). Compact `<Table>` with real-time SSE rows. `<Progress>` bars for long operations.

## Pages

### / Dashboard

```tsx
<Row gutter={[16, 16]}>
  <Col span={6}><Statistic title="VMs" value={running} suffix={`/ ${total}`} /></Col>
  <Col span={6}><Statistic title="Containers" value={...} /></Col>
  <Col span={6}><Statistic title="CPU" value={cpuPercent} suffix="%" /></Col>
  <Col span={6}><Statistic title="RAM" value={ramUsed} suffix={`/ ${ramTotal} GB`} /></Col>
</Row>
// Top consumers ProTable (compact), active warnings Alert list,
// recent tasks Timeline, storage pool Progress bars, quick launch buttons.
```

### /instances — Instance List

```tsx
<ProTable
  columns={[status, name, type, cpu%, ram%, IPs, node, tags, actions]}
  request={fetchInstances}
  rowSelection={{}}           // Bulk select
  search={{ filterType: 'light' }}
  options={{ density: true }}
  toolBarRender={() => [<Button type="primary">Create Instance</Button>]}
/>
```

Inline row actions: Start/Stop/Console (no drill-down needed). Bulk actions bar on selection.

### /instances/:name — Instance Detail

8 `<Tabs>` items:

**Summary:** `<Descriptions>` (status, uptime, CPU, RAM, disk, IPs, MACs, config, tags, notes). `<Progress>` gauges. Quick action `<Button.Group>`. ALL on one screen, no scrolling on 1080p.

**Console:** `<SpiceConsole>` (`spice-html5`, dynamic import, ADR-010) for VMs: Ctrl+Alt+Del `<Button>`, clipboard, screenshot, fullscreen. `<SerialConsole>` (xterm.js) for CTs.

**Hardware:** `<ProTable>` of devices (CPU, RAM, disks, NICs, USB, PCI, GPU). Add/Edit/Detach actions. Disk resize `<Slider>`. GPU passthrough with IOMMU group display.

**Snapshots:** `<ProTable>` (name, date, RAM, size). Take Snapshot `<ModalForm>`. Rollback/Delete per row. Optional `<Timeline>` visualization.

**Backup:** `<ProTable>` of backups for this instance. Backup Now `<ModalForm>`. Restore/Verify. Schedule link.

**Firewall:** `<ProTable>` of per-instance rules. Add Rule `<ModalForm>`. Security Group `<Select>`. Enable/disable `<Switch>`.

**Guest:** `<Descriptions>` (filesystems, disk usage from virt-df). Reset Password / Inject SSH Key / Sysprep `<Button>`s. Only rendered when libguestfs available.

**Options:** Boot order (antd `<List>` with drag-and-drop via `dnd-kit`). Autostart `<Switch>`. Protection `<Switch>`. Cloud-init `<CloudInitForm>` with YAML toggle (CodeMirror dynamic import). Profiles `<Select mode="multiple">`. Hookscript assignment per lifecycle phase.

### /containers — Container List

Three views via `<Segmented>`: Containers | Stacks | Pods.

**Containers:** `<ProTable>` with status, name, image, ports (as `<a>` links), CPU%, RAM%, health.

**Stacks:** Grouped by compose stack. `<Collapse>` panels per stack. Stack actions: Start/Stop All, View YAML, Edit + Redeploy, Combined Logs, Save as Template.

**Pods:** Grouped by pod. Same pattern.

Image update detection: `<Badge dot>` on container row when newer digest available.

### /containers/:id — Container Detail

6 tabs:

**Summary:** `<Descriptions>` (status, image, ports as links, env vars, volumes, limits, health). Quick actions.

**Logs:** xterm.js with search, timestamps toggle, follow toggle. Severity filter `<Select>`.

**Exec:** xterm.js terminal. Shell selector `<Select>` (bash, sh, zsh).

**Stats:** `<Area>` charts (CPU, RAM, net I/O, disk I/O) from `@ant-design/charts`. Timeframe `<Segmented>`.

**Files:** `<Tree>` filesystem browser. Click file → view/edit (CodeMirror). Upload `<Upload dragger>`. Download per file.

**Config:** `<Descriptions>` of full container config. `<Typography.Text copyable>` on all values.

### /templates — App Template Gallery

```tsx
<ProList
  grid={{ column: 4 }}
  dataSource={templates}
  renderItem={(t) => (
    <Card cover={<img src={t.logo} />} actions={[<Button type="primary">Deploy</Button>]}>
      <Card.Meta title={t.title} description={t.description} />
      <Tag>{t.category}</Tag>
    </Card>
  )}
/>
```

Deploy → `<ModalForm>` with customizable env vars. Advanced toggle → CodeMirror YAML editor.

~50 built-in templates. Custom template repo URL in settings.

### /kubernetes — Multi-Cluster View

`<ProTable>` of clusters: name, flavor, nodes, K8s version, status `<Badge>`, resource usage `<Progress>`.

Create wizard: `<StepsForm>` with 6 steps (Flavor cards, Control Plane sliders, Worker Pools repeatable section with labels/taints, Networking with CNI select + CIDR inputs, Add-ons checkboxes, Review `<Descriptions>`).

### /kubernetes/:id — Cluster Detail

8 tabs:

**Overview:** `<Descriptions>` (version, endpoint, status, etcd health, nodes, pods). Resource `<Progress>` bars. Kubeconfig `<Button>` download.

**Nodes:** `<ProTable>` (name, role, status, version, CPU/RAM, pods, conditions). Actions: Cordon, Drain, Uncordon, Delete, Maintenance Mode.

**Workloads:** Sub-tabs: Deployments | StatefulSets | DaemonSets | Jobs | Pods. Each a `<ProTable>`. Click → detail `<Drawer>` with YAML, events, logs.

**Services:** `<ProTable>` for Services, Ingresses, NetworkPolicies.

**Storage:** `<ProTable>` for PVCs, StorageClasses, PVs.

**Config:** `<ProTable>` for ConfigMaps, Secrets (values hidden).

**Events:** Real-time K8s events `<Table>` via SSE. Filter by namespace, type `<Select>`.

**kubectl:** Embedded xterm.js with pre-loaded kubeconfig. Autocomplete.

Lifecycle actions in toolbar: Upgrade `<ModalForm>` (version picker, rolling progress), Scale `<ModalForm>`, etcd Snapshot/Restore.

Helm tab (or separate /kubernetes/:id/helm): Repo management, chart search `<ProTable>`, install `<StepsForm>` with values editor, release management (upgrade, rollback, uninstall).

### /workspaces

`<ProList grid>` of template cards with "Launch" `<Button>`. Active sessions `<ProTable>` (name, template, uptime, user, Destroy button). Launch → ephemeral instance + auto-open console. Idle timeout `<Statistic.Countdown>`.

### /storage

Pool cards (`<Card>` with `<Progress>` usage bars, segments colored per instance). Type `<Tag>`. Create Pool `<StepsForm>`. Click pool → `<ProTable>` of volumes with resize, clone, snapshot, delete.

**Disks tab:** `<ProTable>` of physical disks with SMART health `<Badge>`. Click → `<Descriptions>` with full SMART attributes. Wipe `<Button danger>`.

### /networking

`<ProTable>` of networks. Type `<Tag>`. Create `<ModalForm>`. Click → config `<Descriptions>`. Forwards `<ProTable>`. Leases `<ProTable>`.

**Topology tab:** SVG/Canvas network diagram (instances as nodes, networks as zones).

### /firewall

4 `<Tabs>`: Rules `<ProTable>` (drag-to-reorder), Security Groups, IP Sets, Macros. Add Rule `<ModalForm>`. Assign group via `<Select>`.

### /images

3 `<Tabs>`: Local `<ProTable>` (Incus + Podman), Remote (Incus image server browser, search), Templates (profiles). Upload `<Upload.Dragger>`. Pull `<ModalForm>`.

### /backups

2 `<Tabs>`: Backups `<ProTable>` (instance, date, size, status, restore/verify actions), Schedules `<ProTable>` (cron, retention, target, enable `<Switch>`, execution history).

### /bmc

Server cards with health. Detail: 5 tabs (Power, Sensors with charts, Event Log, KVM Console proxy, Virtual Media).

### /cluster

Node cards with `<Progress>` CPU/RAM. Quorum `<Badge>`. Join `<ModalForm>`. Evacuate/Remove per node.

### /users

4 `<Tabs>`: Users `<ProTable>` (role, 2FA, last login), Roles, Permissions (ACL editor), API Tokens. 2FA setup: TOTP QR (`<QRCode>`), recovery codes.

### /audit

`<ProTable>` with filters: user, action, target, date range, status, source IP. Export CSV `<Button>`.

### /settings

5 `<Tabs>`: General, Certificates, Notifications, Updates, Registries. Each uses `<ProForm>` or `<Descriptions>`.

### /logs

Source `<Segmented>`. Severity `<Select>`. Search `<Input.Search>`. Time range `<DatePicker.RangePicker>`. Auto-scroll `<Switch>`. Instance filter `<Select>`. Download `<Button>`.

### /schedules

`<ProTable>` (instance, action, cron, next run, last status, enable `<Switch>`). Create `<ModalForm>`.

### /setup + /login

Setup: create first admin. Login: username + password + optional TOTP. First-login onboarding `<Tour>` (antd Tour component — dismissable).
