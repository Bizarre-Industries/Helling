# Console

> Console is the most latency-sensitive component in the dashboard. SPICE (spice-js or spice-html5) for VM VGA consoles (see ADR-010), xterm.js for serial consoles, container exec, and log streaming. All heavy libraries loaded via dynamic import. Console must work over slow connections and SSH tunnels.

## Ant Design Components

- `antd`: `Tabs`, `Segmented`, `Button`, `Space`, `Select`, `Switch`, `Input.Search`, `Tooltip`
- External: `spice-js` or `spice-html5` (dynamic import, for VGA console per ADR-010), `xterm.js` + `@xterm/addon-fit` + `@xterm/addon-search` (dynamic import)
- `monaco-editor`: not used in console, but adjacent in the Options tab for cloud-init YAML

## Design Rules

1. **Dynamic imports only.** SPICE client (~150KB) and xterm.js (~80KB) are loaded only when the Console or Logs tab is opened. Never in the main bundle.
2. **Console type switcher.** VMs offer VGA (SPICE client) and Serial (xterm.js). CTs offer Shell (xterm.js). Containers offer Exec (xterm.js) and Logs (xterm.js read-only). Use `<Segmented>` to switch.
3. **Full-height layout.** Console fills the entire content area below the tab bar. No padding, no margins, no wasted space. `height: calc(100vh - headerHeight - tabHeight)`.
4. **Ctrl+Alt+Del button.** For VMs only. Sends the key combo via SPICE client API. Placed in toolbar above the console, not inside it.
5. **Clipboard integration.** Copy/paste between local machine and VM console via SPICE client clipboard helper. Button to open clipboard panel.
6. **Fullscreen toggle.** Button to go fullscreen. Uses browser Fullscreen API. Essential for VGA consoles.
7. **Screenshot button.** For VMs. Captures current VGA frame via SPICE client canvas. Downloads as PNG.
8. **Shell selector.** For container exec: `<Select>` with bash, sh, zsh, ash. Default: bash with fallback to sh.
9. **Log controls.** For container logs: search (`<Input.Search>`), timestamp toggle (`<Switch>`), follow/auto-scroll toggle (`<Switch>`), severity filter (`<Select>`).
10. **Reconnect on disconnect.** If WebSocket drops, show inline message "Console disconnected" with [Reconnect] button. Auto-reconnect after 3s.

## Code Pattern

### VNC Console (SPICE client for VMs)

```tsx
import { useEffect, useRef, useState, lazy, Suspense } from 'react';
import { Button, Space, Segmented, Spin, Alert, Tooltip } from 'antd';
import { Maximize, Clipboard, Camera, Keyboard } from 'lucide-react';

// Dynamic import -- only loaded when Console tab opens
const loadNoVNC = () => import('spice-js/core/rfb');

interface VncConsoleProps {
  instanceName: string;
  wsUrl: string; // ws:///api/v1/instances/{name}/console?type=vga
}

export function VncConsole({ instanceName, wsUrl }: VncConsoleProps) {
  const canvasRef = useRef<HTMLDivElement>(null);
  const rfbRef = useRef<any>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let rfb: any;

    loadNoVNC().then(({ default: RFB }) => {
      if (!canvasRef.current) return;

      rfb = new RFB(canvasRef.current, wsUrl, {
        credentials: { password: '' },
      });

      rfb.scaleViewport = true;
      rfb.resizeSession = true;

      rfb.addEventListener('connect', () => setConnected(true));
      rfb.addEventListener('disconnect', (e: any) => {
        setConnected(false);
        if (!e.detail.clean) {
          setError('Console disconnected unexpectedly');
          // Auto-reconnect after 3s
          setTimeout(() => {
            setError(null);
            rfb.disconnect();
            // Re-initialize
          }, 3000);
        }
      });

      rfbRef.current = rfb;
    });

    return () => {
      rfb?.disconnect();
    };
  }, [wsUrl]);

  const sendCtrlAltDel = () => rfbRef.current?.sendCtrlAltDel();
  const toggleFullscreen = () => canvasRef.current?.requestFullscreen();
  const takeScreenshot = () => {
    const canvas = canvasRef.current?.querySelector('canvas');
    if (!canvas) return;
    const link = document.createElement('a');
    link.download = `${instanceName}-screenshot.png`;
    link.href = canvas.toDataURL('image/png');
    link.click();
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Toolbar */}
      <Space style={{ padding: '4px 8px', borderBottom: '1px solid var(--ant-color-border)' }}>
        <Tooltip title="Send Ctrl+Alt+Del">
          <Button size="small" icon={<Keyboard size={14} />} onClick={sendCtrlAltDel}>
            Ctrl+Alt+Del
          </Button>
        </Tooltip>
        <Tooltip title="Clipboard">
          <Button size="small" icon={<Clipboard size={14} />} />
        </Tooltip>
        <Tooltip title="Screenshot">
          <Button size="small" icon={<Camera size={14} />} onClick={takeScreenshot} />
        </Tooltip>
        <Tooltip title="Fullscreen">
          <Button size="small" icon={<Maximize size={14} />} onClick={toggleFullscreen} />
        </Tooltip>
      </Space>

      {/* Console area */}
      {error && (
        <Alert
          message={error}
          type="error"
          showIcon
          closable
          action={<Button size="small" onClick={() => setError(null)}>Reconnect</Button>}
        />
      )}
      <div ref={canvasRef} style={{ flex: 1, background: '#000' }}>
        {!connected && !error && <Spin tip="Connecting to console..." style={{ margin: '20% auto', display: 'block' }} />}
      </div>
    </div>
  );
}
```

### Serial/Exec Console (xterm.js)

```tsx
import { useEffect, useRef, useState } from 'react';
import { Select, Space, Button, Alert } from 'antd';

// Dynamic import
const loadXterm = async () => {
  const [{ Terminal }, { FitAddon }, { SearchAddon }] = await Promise.all([
    import('xterm'),
    import('@xterm/addon-fit'),
    import('@xterm/addon-search'),
  ]);
  return { Terminal, FitAddon, SearchAddon };
};

interface ExecConsoleProps {
  wsUrl: string; // ws:///api/v1/containers/{id}/exec
}

export function ExecConsole({ wsUrl }: ExecConsoleProps) {
  const termRef = useRef<HTMLDivElement>(null);
  const [shell, setShell] = useState('bash');

  useEffect(() => {
    let terminal: any;
    let ws: WebSocket;

    loadXterm().then(({ Terminal, FitAddon, SearchAddon }) => {
      if (!termRef.current) return;

      terminal = new Terminal({
        fontSize: 13,
        fontFamily: 'JetBrains Mono, Menlo, Monaco, Courier New, monospace',
        theme: {
          background: '#1e1e1e',
          foreground: '#d4d4d4',
          cursor: '#d4d4d4',
        },
        cursorBlink: true,
      });

      const fitAddon = new FitAddon();
      const searchAddon = new SearchAddon();
      terminal.loadAddon(fitAddon);
      terminal.loadAddon(searchAddon);
      terminal.open(termRef.current);
      fitAddon.fit();

      // WebSocket connection
      ws = new WebSocket(`${wsUrl}?shell=${shell}`);
      ws.onmessage = (e) => terminal.write(e.data);
      terminal.onData((data: string) => ws.send(data));

      // Resize handling
      const resizeObserver = new ResizeObserver(() => fitAddon.fit());
      resizeObserver.observe(termRef.current);

      return () => resizeObserver.disconnect();
    });

    return () => {
      terminal?.dispose();
      ws?.close();
    };
  }, [wsUrl, shell]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Space style={{ padding: '4px 8px', borderBottom: '1px solid var(--ant-color-border)' }}>
        <Select
          size="small"
          value={shell}
          onChange={setShell}
          options={[
            { label: 'bash', value: 'bash' },
            { label: 'sh', value: 'sh' },
            { label: 'zsh', value: 'zsh' },
            { label: 'ash', value: 'ash' },
          ]}
          style={{ width: 100 }}
        />
      </Space>
      <div ref={termRef} style={{ flex: 1, background: '#1e1e1e' }} />
    </div>
  );
}
```

### Container Log Viewer

```tsx
import { useEffect, useRef, useState } from 'react';
import { Space, Input, Switch, Select, Typography } from 'antd';

export function LogViewer({ wsUrl }: { wsUrl: string }) {
  const termRef = useRef<HTMLDivElement>(null);
  const [follow, setFollow] = useState(true);
  const [timestamps, setTimestamps] = useState(true);
  const [searchValue, setSearchValue] = useState('');

  // xterm.js in read-only mode for log streaming
  // Similar setup to ExecConsole but terminal.options.disableStdin = true

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Space style={{ padding: '4px 8px', borderBottom: '1px solid var(--ant-color-border)' }}>
        <Input.Search
          size="small"
          placeholder="Search logs..."
          value={searchValue}
          onChange={(e) => setSearchValue(e.target.value)}
          style={{ width: 200 }}
        />
        <Typography.Text type="secondary">Timestamps:</Typography.Text>
        <Switch size="small" checked={timestamps} onChange={setTimestamps} />
        <Typography.Text type="secondary">Follow:</Typography.Text>
        <Switch size="small" checked={follow} onChange={setFollow} />
        <Select
          size="small"
          placeholder="Severity"
          allowClear
          options={[
            { label: 'Error', value: 'error' },
            { label: 'Warning', value: 'warning' },
            { label: 'Info', value: 'info' },
            { label: 'Debug', value: 'debug' },
          ]}
          style={{ width: 120 }}
        />
      </Space>
      <div ref={termRef} style={{ flex: 1, background: '#1e1e1e' }} />
    </div>
  );
}
```

### Console Type Switcher

```tsx
import { Segmented } from 'antd';
import { useState } from 'react';

export function InstanceConsole({ instance }: { instance: Instance }) {
  const isVM = instance.type === 'vm';
  const [consoleType, setConsoleType] = useState<string>(isVM ? 'vga' : 'shell');

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {isVM && (
        <Segmented
          options={[
            { label: 'VGA Console', value: 'vga' },
            { label: 'Serial Console', value: 'serial' },
          ]}
          value={consoleType}
          onChange={(v) => setConsoleType(v as string)}
          style={{ marginBottom: 8 }}
        />
      )}
      <div style={{ flex: 1 }}>
        {consoleType === 'vga' && (
          <VncConsole
            instanceName={instance.name}
            wsUrl={`/api/v1/instances/${instance.name}/console?type=vga`}
          />
        )}
        {consoleType === 'serial' && (
          <ExecConsole wsUrl={`/api/v1/instances/${instance.name}/console?type=serial`} />
        )}
        {consoleType === 'shell' && (
          <ExecConsole wsUrl={`/api/v1/instances/${instance.name}/exec`} />
        )}
      </div>
    </div>
  );
}
```

## Standards References

- `docs/design/philosophy.md` -- Rule 9 (no framework bloat, dynamic imports for heavy components)
- `docs/refs/spec/webui-spec.md` -- Console tab: VNC + serial for VMs, exec + logs for containers
- `CLAUDE.md` -- xterm.js for terminal, spice-js for VM VGA console
- `lessons.md` -- SPICE client was installed but never imported; must actually build VncConsole

## Pages Using This Pattern

- `/instances/:name` Console tab -- VGA (SPICE client) + serial (xterm.js) for VMs, shell (xterm.js) for CTs
- `/containers/:id` Logs tab -- xterm.js read-only log streaming
- `/containers/:id` Exec tab -- xterm.js interactive shell with shell selector
- `/kubernetes/:id` kubectl tab -- xterm.js with pre-loaded kubeconfig
- `/bmc/:id` KVM Console tab -- proxied KVM-over-IP via xterm.js or iframe
