# Detail Views

> Every resource detail page follows the same layout: summary header with status + quick actions, then tabbed content using Descriptions for key-value data. The Summary tab shows everything without scrolling on 1080p. Learn the pattern once, use it for every resource type.

## Ant Design Components

- `@ant-design/pro-components`: `ProDescriptions`
- `antd`: `Descriptions`, `Tabs`, `Badge`, `Button`, `Space`, `Progress`, `Tag`, `Typography.Text`, `Statistic`
- `lucide-react`: action button icons

## Design Rules

1. **Descriptions for every key-value view.** CPU config, network details, storage info, user properties. No custom CSS grids.
2. **Summary tab shows everything.** Status, uptime, CPU, RAM, disk, IPs, MACs, config summary, tags, notes, quick actions -- ALL on one screen, no scrolling on 1080p (>=14 rows visible).
3. **Tabbed layout.** Every resource detail has tabs. Instance: Summary, Console, Hardware, Snapshots, Backup, Firewall, Guest, Options. Consistent tab order across resource types.
4. **Summary header.** Top of detail page shows: resource name, status badge, type tag, and quick action buttons (Start/Stop/Restart/Console/Snapshot/Backup). Always visible regardless of active tab.
5. **Copyable technical values.** IPs, MACs, UUIDs, fingerprints, connection strings use `<Typography.Text copyable>` with `code` style.
6. **Responsive columns.** `<Descriptions column={{ xxl: 4, xl: 3, lg: 2, md: 1 }}>`. Adapts to screen width without breaking layout.
7. **Progress gauges inline.** CPU and RAM usage shown as `<Progress percent={} size="small" />` directly in Descriptions items. Not separate chart widgets.
8. **Bordered layout for dense data.** Use `bordered` prop on Descriptions for hardware config, storage details, network config. Unbounded for summary overview.
9. **Consistent label width.** Labels are brief: "CPU", "RAM", "Disk", "Status", "Uptime", "IPs". Never verbose: not "Current CPU Usage Percentage".
10. **Two-click access.** Click resource in tree -> detail page loads with Summary tab. All info visible immediately.

## Code Pattern

### Instance Detail Page

```tsx
import { Descriptions, Tabs, Badge, Button, Space, Progress, Tag, Typography } from "antd";
import { Play, Square, RotateCcw, Terminal, Camera, Download } from "lucide-react";

export function InstanceDetail({ instance }: { instance: Instance }) {
  return (
    <>
      {/* Summary Header -- always visible */}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 16
        }}
      >
        <Space size="middle">
          <Typography.Title level={4} style={{ margin: 0 }}>
            {instance.name}
          </Typography.Title>
          <Badge
            status={instance.status === "running" ? "success" : "error"}
            text={instance.status}
          />
          <Tag>{instance.type === "vm" ? "VM" : "CT"}</Tag>
        </Space>
        <Space>
          {instance.status === "running" ? (
            <>
              <Button icon={<Square size={14} />} onClick={() => stopInstance(instance.name)}>
                Stop
              </Button>
              <Button icon={<RotateCcw size={14} />} onClick={() => restartInstance(instance.name)}>
                Restart
              </Button>
              <Button icon={<Terminal size={14} />} onClick={() => openConsole(instance.name)}>
                Console
              </Button>
            </>
          ) : (
            <Button
              type="primary"
              icon={<Play size={14} />}
              onClick={() => startInstance(instance.name)}
            >
              Start
            </Button>
          )}
          <Button icon={<Camera size={14} />} onClick={() => snapshot(instance.name)}>
            Snapshot
          </Button>
          <Button icon={<Download size={14} />} onClick={() => backup(instance.name)}>
            Backup
          </Button>
        </Space>
      </div>

      {/* Tabbed Content */}
      <Tabs
        defaultActiveKey="summary"
        items={[
          { key: "summary", label: "Summary", children: <InstanceSummary instance={instance} /> },
          { key: "console", label: "Console", children: <ConsoleTab instance={instance} /> },
          { key: "hardware", label: "Hardware", children: <HardwareTab instance={instance} /> },
          { key: "snapshots", label: "Snapshots", children: <SnapshotsTab instance={instance} /> },
          { key: "backup", label: "Backup", children: <BackupTab instance={instance} /> },
          { key: "firewall", label: "Firewall", children: <FirewallTab instance={instance} /> },
          { key: "guest", label: "Guest", children: <GuestTab instance={instance} /> },
          { key: "options", label: "Options", children: <OptionsTab instance={instance} /> }
        ]}
      />
    </>
  );
}
```

### Summary Tab (Descriptions)

```tsx
function InstanceSummary({ instance }: { instance: Instance }) {
  return (
    <Descriptions column={{ xxl: 4, xl: 3, lg: 2, md: 1 }} size="small">
      <Descriptions.Item label="Status">
        <Badge
          status={instance.status === "running" ? "success" : "error"}
          text={instance.status}
        />
      </Descriptions.Item>
      <Descriptions.Item label="Uptime">{instance.uptime}</Descriptions.Item>
      <Descriptions.Item label="Node">{instance.node}</Descriptions.Item>
      <Descriptions.Item label="Architecture">{instance.arch}</Descriptions.Item>

      <Descriptions.Item label="CPU">
        {instance.cpuCores} cores
        <Progress
          percent={instance.cpuPercent}
          size="small"
          style={{ width: 100, marginLeft: 8 }}
        />
      </Descriptions.Item>
      <Descriptions.Item label="RAM">
        {instance.ramUsed} / {instance.ramTotal}
        <Progress
          percent={instance.ramPercent}
          size="small"
          style={{ width: 100, marginLeft: 8 }}
        />
      </Descriptions.Item>
      <Descriptions.Item label="Disk" span={2}>
        {instance.disks.map((d) => (
          <div key={d.name}>
            {d.name}: {d.used} / {d.size}
            <Progress percent={d.usedPercent} size="small" style={{ width: 100, marginLeft: 8 }} />
          </div>
        ))}
      </Descriptions.Item>

      <Descriptions.Item label="IPs" span={2}>
        <Space direction="vertical" size={0}>
          {instance.ips.map((ip) => (
            <Typography.Text key={ip} copyable code style={{ fontSize: 12 }}>
              {ip}
            </Typography.Text>
          ))}
        </Space>
      </Descriptions.Item>
      <Descriptions.Item label="MACs" span={2}>
        <Space direction="vertical" size={0}>
          {instance.macs.map((mac) => (
            <Typography.Text key={mac} copyable code style={{ fontSize: 12 }}>
              {mac}
            </Typography.Text>
          ))}
        </Space>
      </Descriptions.Item>

      <Descriptions.Item label="OS">{instance.os}</Descriptions.Item>
      <Descriptions.Item label="Boot">{instance.bootOrder}</Descriptions.Item>
      <Descriptions.Item label="Profiles">{instance.profiles.join(", ")}</Descriptions.Item>
      <Descriptions.Item label="Created">{instance.createdAt}</Descriptions.Item>

      <Descriptions.Item label="Tags" span={4}>
        {instance.tags?.map((t) => (
          <Tag key={t}>{t}</Tag>
        ))}
        <Button type="dashed" size="small" style={{ marginLeft: 4 }}>
          + Add
        </Button>
      </Descriptions.Item>
    </Descriptions>
  );
}
```

### Bordered Descriptions (Hardware Config)

```tsx
function HardwareConfig({ instance }: { instance: Instance }) {
  return (
    <Descriptions bordered column={2} size="small" title="CPU Configuration">
      <Descriptions.Item label="Cores">{instance.cpuCores}</Descriptions.Item>
      <Descriptions.Item label="Sockets">{instance.cpuSockets}</Descriptions.Item>
      <Descriptions.Item label="Type">{instance.cpuType}</Descriptions.Item>
      <Descriptions.Item label="NUMA">{instance.numa ? "Enabled" : "Disabled"}</Descriptions.Item>
      <Descriptions.Item label="Affinity">
        <Typography.Text code>{instance.cpuAffinity || "None"}</Typography.Text>
      </Descriptions.Item>
      <Descriptions.Item label="Priority">{instance.cpuPriority}</Descriptions.Item>
    </Descriptions>
  );
}
```

## Standards References

- `docs/design/philosophy.md` -- Rule 6 (Summary tab shows everything), Rule 7 (two-click max)
- `docs/spec/webui-spec.md` -- 8 tabs per instance, Descriptions for summary, ProTable for sub-lists
- `CLAUDE.md` -- Descriptions replaces PropertyGrid, zero custom CSS

## Pages Using This Pattern

- `/instances/:name` -- 8-tab detail (Summary, Console, Hardware, Snapshots, Backup, Firewall, Guest, Options)
- `/containers/:id` -- 6-tab detail (Summary, Logs, Exec, Stats, Files, Config)
- `/kubernetes/:id` -- 8-tab detail (Overview, Nodes, Workloads, Services, Storage, Config, Events, kubectl)
- `/storage/:pool` -- pool detail with Descriptions + volume ProTable
- `/networking/:name` -- network config Descriptions + forwards/leases ProTable
- `/bmc/:id` -- 5-tab detail (Power, Sensors, Event Log, KVM Console, Virtual Media)
- `/users/:name` -- user detail Descriptions + permissions
