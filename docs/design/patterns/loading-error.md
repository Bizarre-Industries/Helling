# Loading and Error States

> Show cached data immediately. Refresh in background. Never show a full-page spinner. Never show an empty page when data is loading. Every error tells the user what happened, why, and what to do next.

## Ant Design Components

- `antd`: `Alert`, `Spin`, `Skeleton`, `Result`, `Button`, `Typography.Text`, `message`, `notification`
- `@tanstack/react-query`: `useQuery` with `staleTime`, `gcTime`, SSE-driven invalidation

## Design Rules

1. **500ms skeleton rule.** First load with no cache: show `<Skeleton>` for up to 500ms. If data arrives within 500ms, skip skeleton entirely (never flash it). After 500ms, show skeleton until data arrives.
2. **Stale-while-revalidate.** Subsequent loads show cached data instantly with a subtle refresh indicator (small `<Spin>` in corner). Background refetch updates in-place.
3. **SSE push updates.** Data updates from SSE events trigger React Query cache invalidation. No polling. No loading indicator needed for SSE-pushed data.
4. **Connection-lost banner.** When WebSocket/SSE disconnects or API is unreachable, show a non-blocking `<Alert>` banner at the top of the content area. NOT a modal. NOT a full-page error. Show last cached data with "(stale)" indicator below.
5. **Auto-reconnect.** Exponential backoff: 1s, 2s, 4s, 8s, 16s, max 30s. Remove banner when reconnected. Refresh data on reconnect.
6. **Incus-unavailable banner.** If Incus service is down, show `<Alert type="error">` banner. Podman features still work. Show which features are affected.
7. **Never full-page spinner.** No `<Spin size="large">` wrapping the entire page. If needed, use inline `<Spin>` within the specific component that is loading.
8. **Actionable errors.** Every error tells the user: what failed, why (if known), and what to do. Include action buttons: [Retry], [View Logs], [View Storage].
9. **Offline behavior.** Browser loses connection: banner, stale timestamps, action buttons disabled with tooltip "Reconnect to perform actions". Re-enable on reconnect.

## Code Pattern

### React Query Configuration

```tsx
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000, // 30s before considered stale
      gcTime: 5 * 60_000, // 5min cache retention
      refetchOnWindowFocus: true,
      retry: 3,
      retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 30000)
    }
  }
});

// SSE event handler invalidates relevant queries
function useSSEInvalidation() {
  useEffect(() => {
    const es = new EventSource("/api/v1/events");
    es.onmessage = (event) => {
      const data = JSON.parse(event.data);
      switch (data.type) {
        case "instance":
          queryClient.invalidateQueries({ queryKey: ["instances"] });
          queryClient.invalidateQueries({ queryKey: ["instance", data.name] });
          break;
        case "operation":
          queryClient.invalidateQueries({ queryKey: ["tasks"] });
          break;
        case "storage":
          queryClient.invalidateQueries({ queryKey: ["storage"] });
          break;
      }
    };
    return () => es.close();
  }, []);
}
```

### Connection Status Banner

```tsx
import { Alert, Button, Typography } from "antd";
import { useEffect, useState, useRef } from "react";

const { Text } = Typography;

export function ConnectionBanner() {
  const [connected, setConnected] = useState(true);
  const [lastConnected, setLastConnected] = useState<Date | null>(null);
  const retryRef = useRef(1000);

  useEffect(() => {
    let timeout: ReturnType<typeof setTimeout>;
    const es = new EventSource("/api/v1/events");

    es.onopen = () => {
      setConnected(true);
      setLastConnected(new Date());
      retryRef.current = 1000; // reset backoff
    };

    es.onerror = () => {
      setConnected(false);
      // Exponential backoff reconnect
      timeout = setTimeout(() => {
        retryRef.current = Math.min(retryRef.current * 2, 30000);
      }, retryRef.current);
    };

    return () => {
      es.close();
      clearTimeout(timeout);
    };
  }, []);

  if (connected) return null;

  return (
    <Alert
      message="Connection lost. Reconnecting..."
      description={
        lastConnected && (
          <Text type="secondary">
            Last connected: {lastConnected.toLocaleTimeString()}. Showing cached data.
          </Text>
        )
      }
      type="warning"
      showIcon
      banner
      action={
        <Button size="small" onClick={() => window.location.reload()}>
          Reload
        </Button>
      }
    />
  );
}
```

### Incus Unavailable Banner

```tsx
function IncusStatusBanner({ incusAvailable }: { incusAvailable: boolean }) {
  if (incusAvailable) return null;

  return (
    <Alert
      message="Incus service is unavailable"
      description="VM and container management is offline. Podman containers still work."
      type="error"
      showIcon
      banner
      action={
        <Space>
          <Button size="small" href="/logs?source=incus">
            View System Logs
          </Button>
          <Button size="small" onClick={() => restartIncus()}>
            Restart Incus
          </Button>
        </Space>
      }
    />
  );
}
```

### Data Fetching with Graceful Loading

```tsx
import { useQuery } from "@tanstack/react-query";
import { Skeleton, Alert, Button } from "antd";

function useInstances() {
  return useQuery({
    queryKey: ["instances"],
    queryFn: fetchInstances,
    staleTime: 30_000,
    // Return empty array instead of error when Incus is down
    placeholderData: []
  });
}

export function InstanceList() {
  const { data, isLoading, error, refetch, isFetching, dataUpdatedAt } = useInstances();

  // First load, no cache
  if (isLoading && !data?.length) {
    return <Skeleton active paragraph={{ rows: 10 }} />;
  }

  // API error with cached data available
  if (error && data?.length) {
    return (
      <>
        <Alert
          message="Failed to refresh data"
          description={`Showing data from ${new Date(dataUpdatedAt).toLocaleTimeString()}`}
          type="warning"
          showIcon
          closable
          action={
            <Button size="small" onClick={() => refetch()}>
              Retry
            </Button>
          }
          style={{ marginBottom: 16 }}
        />
        <InstanceTable data={data} stale />
      </>
    );
  }

  // API error with no cached data
  if (error && !data?.length) {
    return (
      <Alert
        message="Failed to load instances"
        description={error.message}
        type="error"
        showIcon
        action={
          <Button size="small" onClick={() => refetch()}>
            Retry
          </Button>
        }
      />
    );
  }

  return (
    <InstanceTable
      data={data}
      loading={isFetching} // subtle spinner in table, not blocking
    />
  );
}
```

### Actionable Error Toast

```tsx
import { notification, Button, Space } from "antd";

function showOperationError(error: OperationError) {
  notification.error({
    message: error.message,
    description: (
      <Space direction="vertical" size={4}>
        <span>{error.detail}</span>
        <Space>
          {error.action === "storage_full" && (
            <Button size="small" type="link" href="/storage">
              View Storage
            </Button>
          )}
          <Button size="small" type="link" onClick={() => retryOperation(error.operationId)}>
            Retry
          </Button>
        </Space>
      </Space>
    ),
    duration: 0 // stay until dismissed for errors
  });
}

// Example call:
// showOperationError({
//   message: 'Backup of vm-web-1 failed',
//   detail: "Storage pool 'default' is full (98%).",
//   action: 'storage_full',
//   operationId: 'op-123',
// });
```

### Disabled Actions While Offline

```tsx
import { Button, Tooltip } from "antd";

function ActionButton({ connected, onClick, children, ...props }: ActionButtonProps) {
  if (!connected) {
    return (
      <Tooltip title="Reconnect to perform actions">
        <Button {...props} disabled>
          {children}
        </Button>
      </Tooltip>
    );
  }
  return (
    <Button {...props} onClick={onClick}>
      {children}
    </Button>
  );
}
```

## Standards References

- `docs/design/identity.md` -- lines 169-215: loading, error, offline behavior rules
- `docs/design/philosophy.md` -- Rule 5 (data loads instantly or shows why)
- `docs/spec/errors.md` -- Graceful degradation: Incus down = empty arrays, not 404/500
- `docs/spec/events.md` -- Real-time = SSE, no polling

## Pages Using This Pattern

- **All pages** -- React Query stale-while-revalidate, connection banner
- `/` Dashboard -- cached stats with background refresh, SSE-driven counter updates
- `/instances` -- empty array when Incus down (not error), inline loading for table refresh
- `/containers` -- Podman socket activation handling (connection refused on first request is normal)
- `/kubernetes` -- cluster status polling with SSE, offline cluster shows last-known state
- `/backups` -- operation progress via SSE, failure toast with actionable links
- `/bmc` -- BMC unreachable shows "offline" status per server, not page-level error
