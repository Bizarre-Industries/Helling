// Helling WebUI — Schedules page (extracted from pages.jsx during Phase 2A).
//
// Reads schedules from the legacy mocks shim. Phase 3A swaps to a real
// schedules-query hook fed by /api/v1/schedules.

import { getSchedules } from '../../legacy/mocks';
import { I } from '../../primitives/icon';

export default function PageSchedules() {
  const schedules = getSchedules();
  return (
    <div>
      <div className="toolbar">
        <div className="lft">
          <div className="seg">
            <button type="button" className="on">
              All schedules
            </button>
            <button type="button">Active</button>
            <button type="button">Disabled</button>
          </div>
        </div>
        <div className="rgt">
          <button type="button" className="btn btn--sm btn--primary">
            <I n="plus" s={13} /> New schedule
          </button>
        </div>
      </div>
      <table className="tbl">
        <thead>
          <tr>
            <th style={{ width: 60 }}>Enabled</th>
            <th>Name</th>
            <th>Target</th>
            <th>Action</th>
            <th>Cron</th>
            <th>Next run</th>
            <th>Last run</th>
            <th style={{ textAlign: 'right' }}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {schedules.map((s) => (
            <tr key={s.name}>
              <td>
                <span
                  style={{
                    display: 'inline-flex',
                    width: 28,
                    height: 14,
                    borderRadius: 999,
                    background: s.on ? 'var(--h-accent)' : 'var(--h-border)',
                    position: 'relative',
                  }}
                >
                  <span
                    style={{
                      position: 'absolute',
                      width: 10,
                      height: 10,
                      borderRadius: '50%',
                      background: '#000',
                      top: 2,
                      left: s.on ? 16 : 2,
                    }}
                  />
                </span>
              </td>
              <td className="mono" style={{ fontWeight: 600 }}>
                {s.name}
              </td>
              <td className="mono dim">{s.target}</td>
              <td>
                <span className="badge mono">{s.action}</span>
              </td>
              <td className="mono">{s.cron}</td>
              <td className="mono dim">{s.next}</td>
              <td>
                {s.last === 'ok' ? (
                  <span style={{ color: 'var(--h-success)' }}>✓ ok</span>
                ) : (
                  <span className="dim">—</span>
                )}
              </td>
              <td style={{ textAlign: 'right' }}>
                <button type="button" className="btn btn--sm btn--ghost">
                  <I n="play" s={13} /> Run now
                </button>
                <button type="button" className="btn btn--sm btn--ghost">
                  <I n="pencil" s={13} />
                </button>
                <button type="button" className="btn btn--sm btn--ghost">
                  <I n="trash-2" s={13} />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
