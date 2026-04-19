# Console

> Console is a latency-sensitive surface. v0.1 uses a SPICE browser client for VM VGA sessions and xterm.js for serial, exec, and logs. Heavy libraries are loaded only when the Console or Logs tab is active.

## Components

- `antd`: `Tabs`, `Segmented`, `Button`, `Space`, `Select`, `Switch`, `Input.Search`, `Tooltip`
- `spice-html5` class browser client (dynamic import) for VM VGA console
- `xterm.js` + `@xterm/addon-fit` + `@xterm/addon-search` (dynamic import) for serial/exec/log streams

## Design Rules

1. **Dynamic imports only.** SPICE browser client and xterm.js must not load in the main bundle.
2. **Console type switcher.**
   - VMs: VGA (SPICE) and Serial (xterm.js)
   - System containers: Serial (xterm.js)
   - App containers: Exec + Logs (xterm.js)
3. **Full-height viewport.** Console fills available page height under toolbar/tab chrome.
4. **VM toolbar actions.** Ctrl+Alt+Del, fullscreen, clipboard, screenshot.
5. **Container exec shell selector.** `bash` default with fallback to `sh`.
6. **Log controls.** Search, timestamp toggle, follow toggle, severity filter.
7. **Reconnect handling.** On WebSocket drop, show inline error with reconnect action and automatic retry.

## Implementation Pattern

### VM VGA Console (SPICE)

- Create SPICE session from dynamic import after tab activation.
- Bind connect/disconnect listeners.
- Enable viewport scaling and resize sync.
- Surface VM actions in toolbar: Ctrl+Alt+Del, screenshot, fullscreen.

### Serial / Exec Console (xterm.js)

- Initialize xterm.js and fit addon after tab activation.
- Open WebSocket to proxy endpoint for selected console type.
- Pipe terminal data bidirectionally for exec/serial sessions.
- Resize terminal on container resize events.

### Log Viewer (xterm.js read-only)

- Use xterm.js in read-only mode for stream display.
- Apply search, severity filter, and timestamp display controls.
- Keep follow mode toggle explicit and user-controlled.

## API Expectations

- VM VGA: `/api/incus/1.0/instances/{name}/console?type=vga`
- VM/CT serial: `/api/incus/1.0/instances/{name}/console?type=console`
- Container exec: `/api/incus/1.0/instances/{name}/exec`
- Container logs: proxied logs WebSocket/stream endpoint

## Cross-References

- ADR-010 (SPICE VM console policy)
- `docs/design/pages/instances.md`
- `docs/spec/compute.md`
