// Helling WebUI — Networking page (extracted from pages.jsx during Phase 2A).
//
// Reads networks from the legacy mocks shim. Phase 3A replaces with a real
// useNetworksQuery hook (already scaffolded in `web/src/api/queries.ts`); this
// file's import flips, body stays.

import { getNetworks } from '../../legacy/mocks';
import { Copyable } from '../../primitives/copyable';
import { I } from '../../primitives/icon';

export default function PageNetworking() {
  const networks = getNetworks();
  return (
    <div>
      <div className="toolbar">
        <div className="lft">
          <div className="seg">
            <button type="button" className="on">
              Networks
            </button>
            <button type="button">
              DHCP leases{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                14
              </span>
            </button>
            <button type="button">DNS</button>
            <button type="button">
              VPN{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                wg0
              </span>
            </button>
          </div>
        </div>
        <div className="rgt">
          <button type="button" className="btn btn--sm btn--primary">
            <I n="plus" s={13} /> Create network
          </button>
        </div>
      </div>
      <table className="tbl">
        <thead>
          <tr>
            <th>Name</th>
            <th>Type</th>
            <th>CIDR</th>
            <th>DHCP</th>
            <th>Instances</th>
            <th>Gateway</th>
            <th>MTU</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {networks.map((n) => (
            <tr key={n.name}>
              <td className="mono" style={{ fontWeight: 600 }}>
                {n.name}
              </td>
              <td>
                <span className="badge mono">{n.type}</span>
              </td>
              <td className="mono">
                <Copyable text={n.cidr} />
              </td>
              <td>
                {n.dhcp ? (
                  <span style={{ color: 'var(--h-success)' }}>● on</span>
                ) : (
                  <span className="dim">off</span>
                )}
              </td>
              <td className="mono">{n.insts}</td>
              <td className="mono dim">{n.cidr.replace(/\.0\/\d+$/, '.1')}</td>
              <td className="mono">1500</td>
              <td style={{ textAlign: 'right' }}>
                <button type="button" className="btn btn--sm btn--ghost">
                  <I n="pencil" s={13} />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
