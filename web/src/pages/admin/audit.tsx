// Helling WebUI — Audit log page (extracted from pages.jsx during Phase 2A).
//
// Reads audit events from the legacy mocks shim. Phase 3A swaps to a
// real `useAuditQuery` against /api/v1/audit (ADR-019).

import { getAudit } from '../../legacy/mocks';
import { I } from '../../primitives/icon';

export default function PageAudit() {
  const audit = getAudit();
  return (
    <div>
      <div className="toolbar">
        <div className="lft">
          <input
            className="input"
            style={{ width: 260, height: 28, fontSize: 12 }}
            placeholder="Filter by user, action, target…"
          />
          <div className="seg">
            <button type="button" className="on">
              24h
            </button>
            <button type="button">7d</button>
            <button type="button">30d</button>
            <button type="button">Custom</button>
          </div>
        </div>
        <div className="rgt">
          <button type="button" className="btn btn--sm">
            <I n="download" s={13} /> Export CSV
          </button>
        </div>
      </div>
      <table className="tbl">
        <thead>
          <tr>
            <th>Timestamp</th>
            <th>User</th>
            <th>Action</th>
            <th>Target</th>
            <th>Status</th>
            <th>IP</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {audit.map((a, i) => (
            <tr key={`${a.ts}-${i}`}>
              <td className="mono dim">{a.ts}</td>
              <td className="mono">{a.user}</td>
              <td className="mono">{a.action}</td>
              <td className="mono" style={{ color: 'var(--h-accent)' }}>
                {a.target}
              </td>
              <td>
                {a.status === 'ok' ? (
                  <span style={{ color: 'var(--h-success)' }}>✓ ok</span>
                ) : (
                  <span style={{ color: 'var(--h-danger)' }}>✕ fail</span>
                )}
              </td>
              <td className="mono dim">{a.ip}</td>
              <td style={{ textAlign: 'right' }}>
                <button type="button" className="btn btn--sm btn--ghost">
                  <I n="chevron-right" s={13} />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
