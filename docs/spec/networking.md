# Networking Specification

Incus network operations go through the proxy (ADR-014). In v0.1 raw Incus proxy access is admin-only and uses the restricted Incus user socket; ADR-024 per-user TLS certificate identity over loopback HTTPS is deferred until it is wired end to end.

For the full Incus networking API, see [Incus REST API](https://linuxcontainers.org/incus/docs/main/rest-api-spec/).

## What's Available via Proxy

Everything the Incus network API provides:

- Networks: CRUD (bridge, macvlan, sriov, OVN)
- Network forwards: port forwarding rules
- Network peers: peering between networks
- Network ACLs: VM/CT firewalling (ADR-012)
- Network zones: DNS management
- Network leases: DHCP lease information
- Network state: current status, counters

Podman network operations go through the Podman proxy.

## Helling Additions

### Host Firewall (nftables)

For host-level rules and Podman container networking, Helling manages nftables rules by shelling out to `nft --json` (ADR-018). Incus Network ACLs handle VM/CT firewalling (ADR-012).

```text
GET    /api/v1/firewall/host         → nft --json list table inet helling
POST   /api/v1/firewall/host         → nft add rule inet helling ...
DELETE /api/v1/firewall/host/{id}    → nft delete rule inet helling ...
```

### Dashboard Network Topology

The networking page includes an SVG network topology diagram showing: bridges, instances attached to each bridge, VLANs, and external connectivity. Data sourced from the Incus proxy (network state + instance config).
