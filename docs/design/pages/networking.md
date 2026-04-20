# Networking

> Status: Draft

Route: `/networking`

> **Data source (ADR-014):** Incus proxy (`/api/incus/1.0/*`). Responses in native Incus format. Podman networks via Podman proxy (`/api/podman/v5.0/libpod/*`).

---

## Layout

Sidebar: "Network" selected. Main panel: Tabs for network types + topology view. ProTable per tab.

## API Endpoints

- `GET /api/incus/1.0/networks` -- list all Incus networks
- `GET /api/incus/1.0/networks/:name` -- network detail
- `POST /api/incus/1.0/networks` -- create network
- `DELETE /api/incus/1.0/networks/:name` -- delete network
- `GET /api/incus/1.0/networks/:name/leases` -- DHCP leases
- `GET /api/incus/1.0/networks/:name/forwards` -- port forwards
- `GET /api/podman/v5.0/libpod/networks/json` -- Podman networks

## Components

- `Tabs` -- Bridges | VLANs | Podman Networks | Topology
- `ProTable` -- network list per tab (name, type Tag, subnet, gateway, instances connected, status Badge)
- `ModalForm` -- create network wizard (type: bridge/macvlan/OVN, subnet, DHCP range, DNS)
- `Descriptions` -- network detail on click (config, DHCP range, DNS config, NAT settings)
- `ProTable` -- port forwarding rules per network
- `ProTable` -- DHCP leases per network (IP, MAC, hostname, expiry)
- **Topology tab:** SVG/Canvas network diagram (instances as nodes, networks as zones, connections as edges)

## Data Model

- Network: `name`, `type` (bridge/macvlan/ovn/podman), `status`, `config{}` (ipv4.address, ipv4.dhcp, dns.domain)
- Lease: `hostname`, `address`, `mac`, `expiry`
- Forward: `listen_address`, `listen_ports`, `target_address`, `target_ports`, `protocol`

## States

### Empty State

"No custom networks. Instances use the default bridge." [Create Network]. "Create isolated networks for different workloads."

### Loading State

Cached network list shown. Lease/forward tables load on network selection.

### Error State

Network interface down: status Badge shows error. Connected instances listed with warning.

## User Actions

- Create network via ModalForm (type, subnet, DHCP, DNS)
- View connected instances per network
- Manage port forwarding rules per network
- View DHCP leases
- Topology diagram for visual overview
- Delete network (with dependency check: refuses if instances attached)

## Cross-References

- Spec: docs/spec/webui-spec.md (Networking section)
