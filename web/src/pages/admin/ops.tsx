// Helling WebUI — Operations page (extracted from pages.jsx during Phase 2A).
//
// Reads warnings from the legacy mocks shim and fires toast/confirm via
// the window-globals shim. Phase 2B replaces both shims with proper
// React stores; this file's import paths flip but the body stays.

import { getWarnings } from '../../legacy/mocks';
import { openConfirm, toast } from '../../legacy/window-globals';
import { I } from '../../primitives/icon';

export default function PageOps() {
  const warnings = getWarnings();
  return (
    <div style={{ padding: 20, display: 'grid', gap: 14 }}>
      <div>
        <div className="eyebrow">OPS / SYSTEM HEALTH</div>
        <h1 className="stencil" style={{ fontSize: 22, margin: '6px 0 0' }}>
          Operations
        </h1>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 14 }}>
        <div className="card">
          <header>
            <span className="title">Warnings & failures</span>
          </header>
          <div style={{ padding: 14, display: 'grid', gap: 8 }}>
            {warnings.map((w, i) => (
              <div key={`${w.target}-${i}`} className={`alert alert--${w.sev}`}>
                <I
                  n={
                    w.sev === 'danger' ? 'octagon-x' : w.sev === 'warn' ? 'triangle-alert' : 'info'
                  }
                  s={14}
                />
                <div style={{ flex: 1 }}>
                  <span style={{ fontWeight: 600 }}>{w.msg}</span>
                  <span className="mono dim" style={{ marginLeft: 8, fontSize: 11 }}>
                    · {w.target}
                  </span>
                </div>
                <button
                  type="button"
                  className="btn btn--sm btn--ghost"
                  style={{ color: 'inherit' }}
                >
                  Resolve
                </button>
              </div>
            ))}
          </div>
        </div>
        <div className="card">
          <header>
            <span className="title">Quick actions</span>
          </header>
          <div style={{ padding: 14, display: 'grid', gap: 6 }}>
            <button
              type="button"
              className="btn"
              onClick={() =>
                toast('info', 'helling doctor', 'Checking 42 system invariants — all green')
              }
            >
              <I n="stethoscope" s={13} /> Run `helling doctor`
            </button>
            <button
              type="button"
              className="btn"
              onClick={() =>
                toast('info', 'Verifying backups', 'Running fsck on 14 snapshots — est. 4 min')
              }
            >
              <I n="shield-check" s={13} /> Verify all backups
            </button>
            <button
              type="button"
              className="btn"
              onClick={() =>
                openConfirm({
                  title: 'Restart hellingd?',
                  body: 'Control-plane daemon will briefly restart. Running VMs are not affected.',
                  confirmLabel: 'Restart',
                })
              }
            >
              <I n="rotate-cw" s={13} /> Restart hellingd
            </button>
            <button
              type="button"
              className="btn"
              onClick={() =>
                toast('info', 'Check for updates', 'You are on 0.1.0-rc3 — latest release')
              }
            >
              <I n="download" s={13} /> Check for updates
            </button>
            <button
              type="button"
              className="btn"
              onClick={() =>
                openConfirm({
                  title: 'Drain node-2?',
                  body: 'Migrates all VMs to other nodes, then marks unschedulable. ~3 min.',
                  confirmLabel: 'Drain',
                })
              }
            >
              <I n="power" s={13} /> Drain node-2
            </button>
            <button
              type="button"
              className="btn btn--danger"
              onClick={() =>
                openConfirm({
                  title: 'Shutdown entire cluster?',
                  body: 'All VMs stop. Storage stays intact. Requires out-of-band access to restart. Type SHUTDOWN to confirm.',
                  danger: true,
                  confirmLabel: 'Shutdown',
                  confirmMatch: 'SHUTDOWN',
                })
              }
            >
              <I n="power-off" s={13} /> Shutdown cluster
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
