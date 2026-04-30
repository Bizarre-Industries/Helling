// Helling WebUI — BMC page (extracted from pages.jsx during Phase 2A).
//
// Inline mock node table until Phase 3 swaps in a real BMC query feeding
// /api/v1/bmc/* endpoints (v0.4.0 backend work).

import { openConfirm, toast } from '../../legacy/window-globals';
import { Copyable } from '../../primitives/copyable';
import { I } from '../../primitives/icon';

type NodeRow = [
  name: string,
  bmc: string,
  power: 'on' | 'off',
  cpuTemp: string,
  fans: string,
  inlet: string,
  watts: number,
];

const NODES: NodeRow[] = [
  ['node-1', '192.168.2.10 · iDRAC9', 'on', '52°C', '2100 RPM', '21°C', 180],
  ['node-2', '192.168.2.20 · Redfish', 'on', '44°C', '1400 RPM', '22°C', 65],
  ['node-3', '192.168.2.30 · iLO5', 'off', '—', '—', '20°C', 0],
];

export default function PageBMC() {
  return (
    <div style={{ padding: 20, display: 'grid', gap: 14 }}>
      <div>
        <div className="eyebrow">BMC / IPMI / POWER</div>
        <h1 className="stencil" style={{ fontSize: 22, margin: '6px 0 0' }}>
          Out-of-band management
        </h1>
      </div>
      <div className="card">
        <header>
          <span className="title">Nodes</span>
        </header>
        <table className="tbl">
          <thead>
            <tr>
              <th>Node</th>
              <th>BMC</th>
              <th>Power</th>
              <th>CPU temp</th>
              <th>Fans</th>
              <th>Inlet</th>
              <th>Watts</th>
              <th style={{ textAlign: 'right' }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {NODES.map((r) => (
              <tr key={r[0]}>
                <td className="mono" style={{ fontWeight: 600 }}>
                  {r[0]}
                </td>
                <td className="mono">
                  <Copyable text={r[1]} />
                </td>
                <td>
                  {r[2] === 'on' ? (
                    <span style={{ color: 'var(--h-success)' }}>● ON</span>
                  ) : (
                    <span className="dim">● OFF</span>
                  )}
                </td>
                <td className="mono">{r[3]}</td>
                <td className="mono">{r[4]}</td>
                <td className="mono">{r[5]}</td>
                <td className="mono">{r[6]} W</td>
                <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                  <button
                    type="button"
                    className="btn btn--sm btn--ghost"
                    title="Power cycle"
                    onClick={() =>
                      openConfirm({
                        title: 'Power cycle ' + r[0] + '?',
                        body: 'Hard reset via BMC. VMs will be killed ungracefully. Prefer Drain + Restart.',
                        danger: true,
                        confirmLabel: 'Power cycle',
                      })
                    }
                  >
                    <I n="power" s={13} />
                  </button>
                  <button
                    type="button"
                    className="btn btn--sm btn--ghost"
                    title="Graceful restart"
                    onClick={() =>
                      openConfirm({
                        title: 'Restart ' + r[0] + '?',
                        body: 'Drains VMs to other nodes, then reboots. ~4 min.',
                        confirmLabel: 'Restart',
                      })
                    }
                  >
                    <I n="rotate-cw" s={13} />
                  </button>
                  <button
                    type="button"
                    className="btn btn--sm btn--ghost"
                    onClick={() =>
                      toast(
                        'info',
                        'Serial-over-LAN',
                        'Attaching to ' + r[0] + ' BMC — scroll to remote console below',
                      )
                    }
                  >
                    <I n="terminal" s={13} /> Serial
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="card">
        <header>
          <span className="title">Remote console · node-1 · Serial-over-LAN</span>
        </header>
        <div className="term" style={{ margin: 14, border: 'none' }}>
          <div>
            <span className="c-dim">[ 0.000000]</span> Linux version 6.8.0-helling-amd64
          </div>
          <div>
            <span className="c-dim">[ 0.124012]</span> ACPI: Core revision 20230628
          </div>
          <div>
            <span className="c-dim">[ 0.552001]</span> PCI: Using configuration type 1 for base
            access
          </div>
          <div>
            <span className="c-dim">[ 1.002148]</span> systemd[1]: Starting Load Kernel Modules...
          </div>
          <div>
            <span className="c-dim">[ 1.104712]</span> <span className="c-ok">[ OK ]</span> Started
            Load Kernel Modules.
          </div>
          <div>
            <span className="c-dim">[ 1.210044]</span> <span className="c-ok">[ OK ]</span> Reached
            target Local File Systems.
          </div>
          <div>
            <span className="c-dim">[ 2.440008]</span> <span className="c-ok">[ OK ]</span> Started
            hellingd — Helling supervisor.
          </div>
          <div style={{ marginTop: 8 }}>
            <span className="c-lime">node-1 login:</span>{' '}
            <span
              style={{
                background: 'var(--h-accent)',
                color: '#000',
                width: 8,
                display: 'inline-block',
              }}
            >
              &nbsp;
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
