# Data Tables

> Every list of resources in Helling uses ProTable. Tables show 50 VMs on screen; cards show 8. When you manage 200 VMs, you need tables. Cards are only for templates, workspaces, and storage pool overview.

## Ant Design Components

- `@ant-design/pro-components`: `ProTable` (primary), `ProList` (card grids only)
- `antd`: `Tag`, `Badge`, `Button`, `Space`, `Dropdown`, `Popconfirm`, `Typography.Text`, `Progress`
- `lucide-react`: row action icons

## Design Rules

1. **Tables by default.** Every resource with >10 possible items uses ProTable. No exceptions.
2. **Cards only for:** app template gallery, workspace template gallery, storage pool overview (when <=6 pools), dashboard widgets, BMC server cards (when <=8 servers).
3. **Inline key stats.** Every table row shows CPU%, RAM%, IP, status without drill-down. Users scan the list, not individual detail pages.
4. **Inline row actions.** Start/Stop/Console as icon buttons directly in the row. No drill-down required for common ops.
5. **Bulk action bar.** Appears on row selection. Actions: Start, Stop, Snapshot, Backup, Delete, Tag.
6. **Compact row height.** 44px default (comfortable). 36px available via density toggle. Show 20+ rows without scrolling on 1080p.
7. **Monospace for technical values.** IPs, MACs, UUIDs, paths render in `fontFamily: monospace` with `<Typography.Text copyable>`.
8. **Filter placement.** Use `search={{ filterType: 'light' }}` for inline column filters. Never a separate filter panel.
9. **Status = icon + color + text.** `<Badge status="success" text="Running" />`. Never rely on color alone.
10. **Pagination.** Default 50 per page. Show total count. Use `pagination={{ defaultPageSize: 50, showTotal: (total) => \`${total} items\` }}`.

## Code Pattern

```tsx
import { ProTable, type ProColumns } from '@ant-design/pro-components';
import { Badge, Button, Dropdown, Popconfirm, Space, Tag, Typography } from 'antd';
import { Play, Square, Terminal, MoreHorizontal, Trash2 } from 'lucide-react';

const columns: ProColumns<Instance>[] = [
  {
    title: 'Status',
    dataIndex: 'status',
    width: 100,
    filters: true,
    valueEnum: {
      running: { text: 'Running', status: 'Success' },
      stopped: { text: 'Stopped', status: 'Error' },
      frozen: { text: 'Frozen', status: 'Warning' },
    },
  },
  {
    title: 'Name',
    dataIndex: 'name',
    sorter: true,
    render: (_, record) => (
      <a href={`/instances/${record.name}`}>{record.name}</a>
    ),
  },
  {
    title: 'Type',
    dataIndex: 'type',
    width: 80,
    filters: true,
    valueEnum: { vm: { text: 'VM' }, container: { text: 'CT' } },
  },
  {
    title: 'CPU',
    dataIndex: 'cpuPercent',
    width: 80,
    sorter: true,
    render: (val) => `${val}%`,
  },
  {
    title: 'RAM',
    dataIndex: 'ramPercent',
    width: 80,
    sorter: true,
    render: (val) => `${val}%`,
  },
  {
    title: 'IPs',
    dataIndex: 'ips',
    render: (_, record) => (
      <Space direction="vertical" size={0}>
        {record.ips.map((ip) => (
          <Typography.Text key={ip} copyable code style={{ fontSize: 12 }}>
            {ip}
          </Typography.Text>
        ))}
      </Space>
    ),
  },
  {
    title: 'Node',
    dataIndex: 'node',
    width: 100,
    filters: true,
  },
  {
    title: 'Tags',
    dataIndex: 'tags',
    render: (_, record) =>
      record.tags?.map((t) => <Tag key={t}>{t}</Tag>),
  },
  {
    title: 'Actions',
    width: 120,
    valueType: 'option',
    render: (_, record) => [
      <Button
        key="start"
        type="text"
        size="small"
        icon={record.status === 'running' ? <Square size={14} /> : <Play size={14} />}
        onClick={() => toggleInstance(record)}
      />,
      <Button
        key="console"
        type="text"
        size="small"
        icon={<Terminal size={14} />}
        onClick={() => openConsole(record)}
      />,
      <Dropdown
        key="more"
        menu={{
          items: [
            { key: 'snapshot', label: 'Snapshot' },
            { key: 'backup', label: 'Backup' },
            { key: 'migrate', label: 'Migrate' },
            { type: 'divider' },
            { key: 'delete', label: 'Delete', danger: true },
          ],
        }}
      >
        <Button type="text" size="small" icon={<MoreHorizontal size={14} />} />
      </Dropdown>,
    ],
  },
];

export function InstanceList() {
  return (
    <ProTable<Instance>
      columns={columns}
      request={fetchInstances}
      rowKey="name"
      rowSelection={{}}
      search={{ filterType: 'light' }}
      options={{ density: true, setting: true }}
      pagination={{ defaultPageSize: 50, showTotal: (total) => `${total} instances` }}
      toolBarRender={(_, { selectedRowKeys }) => [
        selectedRowKeys?.length ? (
          <Space key="bulk">
            <Button size="small">Start Selected</Button>
            <Button size="small">Stop Selected</Button>
            <Button size="small">Snapshot Selected</Button>
            <Popconfirm title="Delete selected instances?">
              <Button size="small" danger>Delete Selected</Button>
            </Popconfirm>
          </Space>
        ) : null,
        <Button key="create" type="primary">Create Instance</Button>,
      ]}
      tableAlertOptionRender={({ selectedRowKeys, onCleanSelected }) => (
        <Space>
          <span>{selectedRowKeys.length} selected</span>
          <a onClick={onCleanSelected}>Clear</a>
        </Space>
      )}
    />
  );
}
```

## Standards References

- `docs/design/philosophy.md` -- Rule 1 (tables by default), Rule 2 (information density), Rule 7 (two-click max)
- `docs/refs/spec/webui-spec.md` -- per-page ProTable column definitions
- `CLAUDE.md` -- ProTable replaces custom tables, ~30 lines per page

## Pages Using This Pattern

- `/instances` -- VMs + system containers
- `/containers` -- app containers, stacks, pods (segmented view, each segment is a ProTable)
- `/kubernetes` -- cluster list
- `/kubernetes/:id` -- nodes, workloads, services, storage, config, events (sub-tables per tab)
- `/storage` -- volumes per pool
- `/networking` -- networks, forwards, leases
- `/firewall` -- rules, security groups, IP sets, macros
- `/images` -- local images, remote browser, templates
- `/backups` -- backups list, schedules list
- `/users` -- users, roles, permissions, API tokens
- `/audit` -- audit log with filters
- `/schedules` -- scheduled operations
- `/logs` -- system logs
- `/bmc` -- event log tab
