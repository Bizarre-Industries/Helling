# BMC

> Status: Draft

Route: `/bmc` (list) + `/bmc/:id` (detail)

> **Data source (ADR-014):** Helling API (`/api/v1/*`). Responses in Helling envelope format `{data, meta}`. BMC communication via bmclib (IPMI/Redfish).

---

## Layout

Sidebar: "BMC" selected. List view: server cards (<=8 servers) or ProTable. Detail view: breadcrumb + 5 Tabs.

## API Endpoints

- `GET /api/v1/bmc` -- list BMC endpoints
- `GET /api/v1/bmc/:id` -- BMC detail
- `POST /api/v1/bmc` -- add BMC
- `DELETE /api/v1/bmc/:id` -- remove BMC
- `GET /api/v1/bmc/:id/sensors` -- sensor readings
- `POST /api/v1/bmc/:id/power` -- power action (on/off/reset/cycle)
- `GET /api/v1/bmc/:id/events` -- SEL event log
- `GET /api/v1/bmc/:id/console` -- KVM console WebSocket
- `POST /api/v1/bmc/:id/media` -- mount virtual media

## Components

### List (`/bmc`)
- `Card` (per server) -- hostname, model, power state Badge, health Badge (temps, fans). Quick power buttons. Connection status indicator.
- `ModalForm` -- Add BMC (IP/hostname, username, password, protocol Select: IPMI/Redfish/auto)

### Detail (`/bmc/:id`) -- 5 Tabs

**Sensors tab:** `ProTable` (name, type Tag, value, unit, status Badge, thresholds). Auto-refresh. Historical sensor `Area` charts (@ant-design/charts) for temps and power draw.

**Power tab:** Status `Descriptions` (current state, last changed). `Button.Group`: Power On (primary), Power Off (danger), Reset (warning), Power Cycle (warning). Confirmation Modal for destructive actions.

**Event Log tab:** `ProTable` of SEL entries (timestamp, severity Badge, sensor, event, description). Sortable, filterable. "Clear SEL" Button (danger).

**Console tab:** KVM console proxy via WebSocket. Similar to VNC console pattern. Fullscreen Button.

**Virtual Media tab:** `ModalForm` to mount ISO from URL. Current mounts `Descriptions`. Unmount Button.

## Data Model

- BMC: `id`, `hostname`, `address`, `protocol`, `model`, `firmware`, `power_state`, `health`
- Sensor: `name`, `type`, `value`, `unit`, `status`, `lower_critical`, `upper_critical`
- SELEntry: `id`, `timestamp`, `severity`, `sensor`, `event`, `description`

## States

### Empty State
"No BMC endpoints configured." [Add BMC]. "Connect to server baseboard management controllers for out-of-band management."

### Loading State
Server cards cached. Sensor data refreshes on interval (30s).

### Error State
BMC unreachable: connection status Badge red. Sensors show last known values with "(stale)".

## User Actions

- Add/remove BMC endpoints
- Power on/off/reset/cycle servers
- Monitor sensor readings (temps, fans, voltages, power)
- Browse SEL event log
- Open KVM console
- Mount/unmount virtual media (ISO)

## Cross-References

- Spec: docs/refs/spec/webui-spec.md (BMC sections)
