// Helling WebUI — Logs page (extracted from pages.jsx during Phase 2A).
//
// Stub implementation: hardcoded recent entries. Phase 3E wires real
// systemd-journal log streaming (ADR-019).

import { I } from '../../primitives/icon';

type LogLevel = 'INFO' | 'WARN' | 'ERROR';
type LogRow = [string, LogLevel, string, string];

const LOG_ROWS: LogRow[] = [
  ['14:22:03', 'INFO', 'hellingd', 'task T-8814 started: snapshot vm-web-1'],
  ['14:21:59', 'INFO', 'api', 'POST /v1/instances/vm-web-1/snapshot (admin)'],
  ['14:21:02', 'INFO', 'hellingd', 'backup verify ok: vm-db-1 (8.2 GB, 6.2s)'],
  ['14:19:45', 'INFO', 'hellingd', 'task T-8813 completed: backup-verify'],
  ['14:15:02', 'INFO', 'podman', 'pulled linuxserver/jellyfin:latest'],
  ['14:12:40', 'WARN', 'storage', 'pool default: 72% full — consider adding capacity'],
  ['13:58:11', 'INFO', 'hellingd', 'migrate vm-build node-1 → node-2 (53s)'],
  ['13:40:00', 'INFO', 'hellingd', 'ct-gitea started'],
  ['13:12:00', 'ERROR', 'backup', 'vm-old failed: pool full (code=STORAGE_FULL, 2)'],
  ['12:08:15', 'INFO', 'auth', 'user.2fa.enable: alice (TOTP)'],
  ['11:44:01', 'INFO', 'firewall', 'rule added: accept tcp 22 → vm-web-1'],
  ['10:02:09', 'INFO', 'hellingd', 'instance.create vm-build (4 vcpu, 8 GB)'],
  ['09:15:00', 'INFO', 'scheduler', 'schedule created: daily-backup (0 2 * * *)'],
];

const levelClass: Record<LogLevel, string> = {
  INFO: 'c-lime',
  WARN: 'c-warn',
  ERROR: 'c-err',
};

export default function PageLogs() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div className="toolbar">
        <div className="lft">
          <select className="input" style={{ height: 28, fontSize: 12, width: 160 }}>
            <option>hellingd</option>
            <option>hellingd-api</option>
            <option>node-1/kernel</option>
            <option>node-1/systemd</option>
          </select>
          <div className="seg">
            <button type="button">All</button>
            <button type="button" className="on">
              Info
            </button>
            <button type="button">
              Warn{' '}
              <span className="mono" style={{ marginLeft: 4, color: 'var(--h-warn)' }}>
                3
              </span>
            </button>
            <button type="button">
              Error{' '}
              <span className="mono" style={{ marginLeft: 4, color: 'var(--h-danger)' }}>
                1
              </span>
            </button>
          </div>
          <input
            className="input"
            style={{ width: 260, height: 28, fontSize: 12 }}
            placeholder="grep…"
          />
        </div>
        <div className="rgt">
          <button type="button" className="btn btn--sm">
            <I n="play" s={13} /> Follow
          </button>
          <button type="button" className="btn btn--sm">
            <I n="download" s={13} /> Export
          </button>
        </div>
      </div>
      <div
        className="term"
        style={{ margin: 14, flex: 1, minHeight: 400, borderRadius: 'var(--h-radius)' }}
      >
        {LOG_ROWS.map(([t, l, c, m], i) => (
          <div key={`${t}-${i}`}>
            <span className="c-dim">{t}</span> <span className={levelClass[l]}>{l.padEnd(5)}</span>{' '}
            <span className="c-dim">[{c}]</span>{' '}
            <span style={{ color: l === 'ERROR' ? 'var(--h-danger)' : '#d8d8d8' }}>{m}</span>
          </div>
        ))}
        <div style={{ marginTop: 8, color: 'var(--h-accent)' }}>● tailing…</div>
      </div>
    </div>
  );
}
