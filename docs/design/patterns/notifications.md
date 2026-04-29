# Notifications

> Notifications tell users about events as they happen (toasts), collect persistent alerts (bell icon), and reach users outside the dashboard (Discord, Slack, email). Destructive operations get an "Undo" toast. Errors get actionable toasts with links.

## Ant Design Components

- `antd`: `message`, `notification`, `Badge`, `Button`, `Alert`, `Dropdown`, `List`, `Typography.Text`, `Space`, `Switch`
- `@ant-design/pro-components`: `ProForm`, `ProFormSelect` (for notification settings)

## Design Rules

1. **Toast hierarchy.** Use `message` for quick confirmations (instance started, snapshot created). Use `notification` for events that need detail or action (backup failed, storage full). Never use `Modal` for notifications.
2. **Undo for destructive ops.** When a destructive operation completes (delete, stop, destroy), show an "Undo" toast for 10 seconds. Backed by auto-snapshot: clicking Undo rolls back to the pre-operation snapshot.
3. **Error toasts are persistent.** `notification.error` with `duration: 0` (stays until dismissed). Must include: what failed, why, and an action button (Retry, View Logs, View Storage).
4. **Success toasts are brief.** `message.success` with `duration: 3` (3 seconds, auto-dismiss). Just the confirmation text. No extra detail needed.
5. **Warning toasts for side effects.** `notification.warning` when an operation succeeded but has consequences. Example: "Instance started. Note: no firewall rules configured."
6. **Bell icon badge.** Persistent notification center in the top bar. `<Badge count={unreadCount}>` on bell icon. Click opens dropdown with recent notifications as `<List>` items. Mark all read button.
7. **SSE-driven.** All real-time notifications come via SSE. The EventSource handler dispatches to both the toast system and the notification center.
8. **External channels.** 7 configurable channels: Dashboard (always on), Discord webhook, Slack webhook, Telegram bot, Ntfy, Email (SMTP), Gotify, generic webhook. Configured in Settings > Notifications.
9. **Event routing.** Users map event severity to channels: Critical (instance crashed, backup failed) -> Discord + Email. Warning (storage >85%) -> Discord. Info (instance started) -> Dashboard only.
10. **Quiet hours.** "Don't send external notifications between 11 PM and 7 AM unless Critical." Configurable in settings. Dashboard notifications always delivered.

## Code Pattern

### Toast Patterns

```tsx
import { message, notification, Button, Space } from "antd";

// Success: brief, auto-dismiss
function onInstanceStarted(name: string) {
  message.success(`${name} started`);
}

// Success with undo: destructive ops
function onInstanceDeleted(name: string, snapshotId: string) {
  const key = `undo-${name}`;
  notification.success({
    key,
    message: `${name} deleted`,
    description: "Auto-snapshot preserved for 60 seconds.",
    duration: 10,
    btn: (
      <Button
        type="primary"
        size="small"
        onClick={() => {
          rollbackToSnapshot(name, snapshotId);
          notification.destroy(key);
          message.info(`Restoring ${name} from snapshot...`);
        }}
      >
        Undo
      </Button>
    )
  });
}

// Error: persistent, actionable
function onBackupFailed(instanceName: string, reason: string, poolPath?: string) {
  notification.error({
    message: `Backup of ${instanceName} failed`,
    description: (
      <Space direction="vertical" size={4}>
        <span>{reason}</span>
        <Space>
          {poolPath && (
            <Button size="small" type="link" href={poolPath}>
              View Storage
            </Button>
          )}
          <Button size="small" type="link" onClick={() => retryBackup(instanceName)}>
            Retry Backup
          </Button>
        </Space>
      </Space>
    ),
    duration: 0 // persistent until dismissed
  });
}

// Warning: operation succeeded but with side effects
function onInstanceStartedWithWarning(name: string, warnings: string[]) {
  notification.warning({
    message: `${name} started`,
    description: (
      <Space direction="vertical" size={4}>
        {warnings.map((w, i) => (
          <span key={i}>{w}</span>
        ))}
      </Space>
    ),
    duration: 8
  });
}
```

### SSE Event Handler

```tsx
import { message, notification } from "antd";
import { queryClient } from "./queryClient";

interface HellingEvent {
  type: "instance" | "container" | "backup" | "storage" | "cluster" | "system";
  action: string;
  severity: "info" | "warning" | "error" | "critical";
  resource: string;
  message: string;
  detail?: string;
}

export function initSSENotifications() {
  const es = new EventSource("/api/v1/events");

  es.onmessage = (event) => {
    const data: HellingEvent = JSON.parse(event.data);

    // Invalidate relevant React Query cache
    queryClient.invalidateQueries({ queryKey: [data.type] });
    if (data.resource) {
      queryClient.invalidateQueries({ queryKey: [data.type, data.resource] });
    }

    // Update notification center (bell icon)
    addToNotificationCenter(data);

    // Show toast based on severity
    switch (data.severity) {
      case "info":
        message.success(data.message, 3);
        break;
      case "warning":
        notification.warning({
          message: data.message,
          description: data.detail,
          duration: 8
        });
        break;
      case "error":
      case "critical":
        notification.error({
          message: data.message,
          description: data.detail,
          duration: 0
        });
        break;
    }
  };

  return es;
}
```

### Notification Center (Bell Icon)

```tsx
import { Badge, Button, Dropdown, List, Typography, Space } from "antd";
import { Bell, Check } from "lucide-react";
import { useState } from "react";

const { Text } = Typography;

interface NotificationItem {
  id: string;
  message: string;
  detail?: string;
  severity: "info" | "warning" | "error" | "critical";
  timestamp: string;
  read: boolean;
}

export function NotificationBell() {
  const [notifications, setNotifications] = useState<NotificationItem[]>([]);

  const unreadCount = notifications.filter((n) => !n.read).length;

  const markAllRead = () => {
    setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));
  };

  const severityColor = {
    info: undefined,
    warning: "orange",
    error: "red",
    critical: "red"
  };

  return (
    <Dropdown
      trigger={["click"]}
      dropdownRender={() => (
        <div
          style={{
            width: 400,
            background: "var(--ant-color-bg-elevated)",
            border: "1px solid var(--ant-color-border)",
            borderRadius: 4,
            boxShadow: "0 2px 8px rgba(0,0,0,0.15)"
          }}
        >
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              padding: "8px 12px",
              borderBottom: "1px solid var(--ant-color-border)"
            }}
          >
            <Text strong>Notifications</Text>
            {unreadCount > 0 && (
              <Button type="link" size="small" icon={<Check size={14} />} onClick={markAllRead}>
                Mark all read
              </Button>
            )}
          </div>
          <List
            dataSource={notifications.slice(0, 20)}
            style={{ maxHeight: 400, overflow: "auto" }}
            renderItem={(item) => (
              <List.Item
                style={{
                  padding: "8px 12px",
                  background: item.read ? undefined : "var(--ant-color-bg-text-hover)"
                }}
              >
                <List.Item.Meta
                  title={
                    <Text style={{ color: severityColor[item.severity], fontSize: 13 }}>
                      {item.message}
                    </Text>
                  }
                  description={
                    <Space direction="vertical" size={0}>
                      {item.detail && (
                        <Text type="secondary" style={{ fontSize: 12 }}>
                          {item.detail}
                        </Text>
                      )}
                      <Text type="secondary" style={{ fontSize: 11 }}>
                        {item.timestamp}
                      </Text>
                    </Space>
                  }
                />
              </List.Item>
            )}
            locale={{ emptyText: "No notifications" }}
          />
        </div>
      )}
    >
      <Badge count={unreadCount} size="small" offset={[-2, 2]}>
        <Button type="text" icon={<Bell size={16} />} />
      </Badge>
    </Dropdown>
  );
}
```

### Notification Settings (Event Routing)

```tsx
import { ProForm, ProFormSelect } from "@ant-design/pro-components";
import { Descriptions, Switch, TimePicker, Typography } from "antd";

export function NotificationSettings() {
  return (
    <ProForm onFinish={saveNotificationSettings} layout="vertical">
      {/* Channel configuration */}
      <Typography.Title level={5}>Channels</Typography.Title>
      <Descriptions bordered column={1} size="small">
        <Descriptions.Item label="Dashboard">Always enabled</Descriptions.Item>
        <Descriptions.Item label="Discord">
          <ProFormText name="discordWebhook" placeholder="Webhook URL" />
        </Descriptions.Item>
        <Descriptions.Item label="Slack">
          <ProFormText name="slackWebhook" placeholder="Webhook URL" />
        </Descriptions.Item>
        <Descriptions.Item label="Email">
          <ProFormText name="smtpServer" placeholder="SMTP server" />
        </Descriptions.Item>
        <Descriptions.Item label="Ntfy">
          <ProFormText name="ntfyUrl" placeholder="Ntfy server URL" />
        </Descriptions.Item>
        <Descriptions.Item label="Telegram">
          <ProFormText name="telegramToken" placeholder="Bot token" />
        </Descriptions.Item>
        <Descriptions.Item label="Gotify">
          <ProFormText name="gotifyUrl" placeholder="Server URL" />
        </Descriptions.Item>
      </Descriptions>

      {/* Event routing */}
      <Typography.Title level={5} style={{ marginTop: 24 }}>
        Event Routing
      </Typography.Title>
      <ProFormSelect
        name="criticalChannels"
        label="Critical events (crash, backup fail, disk failing, cert expired)"
        mode="multiple"
        options={channelOptions}
        initialValue={["dashboard", "discord", "email"]}
      />
      <ProFormSelect
        name="warningChannels"
        label="Warnings (storage >85%, backup stale, cert expiring)"
        mode="multiple"
        options={channelOptions}
        initialValue={["dashboard", "discord"]}
      />
      <ProFormSelect
        name="infoChannels"
        label="Info (started, stopped, backup complete, snapshot created)"
        mode="multiple"
        options={channelOptions}
        initialValue={["dashboard"]}
      />

      {/* Quiet hours */}
      <Typography.Title level={5} style={{ marginTop: 24 }}>
        Quiet Hours
      </Typography.Title>
      <Space>
        <Switch checkedChildren="On" unCheckedChildren="Off" />
        <TimePicker.RangePicker format="HH:mm" />
        <Typography.Text type="secondary">
          Critical events still delivered during quiet hours
        </Typography.Text>
      </Space>
    </ProForm>
  );
}
```

## Standards References

- `docs/design/identity.md` -- lines 219-256: notification channels, event routing, quiet hours
- `docs/design/magic.md` -- auto-snapshot enables Undo toast (magic touch #1)
- `docs/design/philosophy.md` -- Rule 3 (toast slide in/out <200ms, no other animation)
- `docs/spec/events.md` -- Real-time = SSE
- `docs/spec/audit.md` -- audit everything

## Pages Using This Pattern

- **All pages** -- SSE-driven toasts for real-time events, connection-lost banner
- **Top bar** -- bell icon notification center (global component)
- `/instances/:name` -- undo toast after delete/stop, warning toast for missing firewall
- `/backups` -- error toast when backup fails, success toast when backup completes
- `/firewall` -- warning toast when creating overly permissive rules
- `/kubernetes` -- progress toasts during cluster operations (create, upgrade, scale)
- `/settings/notifications` -- channel config, event routing, quiet hours
- `/bmc` -- critical toast when BMC detects hardware failure
