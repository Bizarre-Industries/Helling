/* Helling WebUI — page components */
/* eslint-disable */
import { useEffect, useState } from 'react';
import './shell.jsx';
import './infra.jsx';

const {
  I,
  Badge,
  STATUS_BADGE,
  Kbd,
  ProgressBar,
  Copyable,
  TopBar,
  ResourceTree,
  TaskDrawer,
  CommandPalette,
  NODES,
  INSTANCES,
  CONTAINERS,
  CLUSTERS,
  TASKS,
  AUDIT,
  POOLS,
  NETWORKS,
  FW_RULES,
  SCHEDULES,
  USERS,
  TEMPLATES,
  SNAPSHOTS,
  BACKUPS,
  WARNINGS,
  ALERTS,
  ToastBus,
  ToastStack,
  Modal,
  ConfirmModal,
  EmptyState,
  Switch,
  MultiChart,
  FilePath,
} = window;

// ─── DASHBOARD ──────────────────────────────────────────────────
function PageDashboard({ onNav }) {
  useStore();
  const mockRunning = INSTANCES.filter((i) => i.status === 'running').length;
  const mockContainerRunning = CONTAINERS.filter((c) => c.status === 'running').length;
  // Real-data queries via the ADR-014 proxy. Falls back to mock counts when
  // the user is not logged in — preserves the dev-only local experience.
  const counts = window.useDashboardCounts
    ? window.useDashboardCounts(
        INSTANCES.length,
        mockRunning,
        CONTAINERS.length,
        mockContainerRunning,
      )
    : {
        live: false,
        totalInstances: INSTANCES.length,
        runningInstances: mockRunning,
        totalContainers: CONTAINERS.length,
        runningContainers: mockContainerRunning,
        loading: false,
      };
  const running = counts.runningInstances;
  const stopped = counts.totalInstances - counts.runningInstances;
  const cRunning = counts.runningContainers;
  const health = 87;

  return (
    <div style={{ padding: '18px 20px', maxWidth: 1440 }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-end',
          marginBottom: 14,
          gap: 16,
          flexWrap: 'wrap',
        }}
      >
        <div>
          <div className="eyebrow">
            OVERVIEW / {new Date().toISOString().slice(0, 10).toUpperCase()} / NODE-1
          </div>
          <h1
            className="stencil"
            style={{ fontSize: 28, margin: '6px 0 0', letterSpacing: '0.01em' }}
          >
            Good afternoon, admin.
          </h1>
          <div className="muted" style={{ marginTop: 4, fontSize: 13 }}>
            1 warning · 1 failure · last backup 2 hours ago
          </div>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button
            className="btn btn--sm"
            onClick={() => window.toast?.info('Refreshed', 'Cluster state synced 2s ago')}
          >
            <I n="rotate-cw" s={13} /> Refresh
          </button>
          <button className="btn btn--sm" onClick={() => window.openModal?.('create-vm')}>
            <I n="plus" s={13} /> Create instance
          </button>
          <button
            className="btn btn--sm btn--primary"
            onClick={() =>
              window.toast?.info('Shell', 'Opening SSH session to node-01 · helling@10.0.0.11')
            }
          >
            <I n="terminal" s={13} /> Open shell
          </button>
        </div>
      </div>

      {WARNINGS.slice(0, 2).map((w, i) => (
        <div key={i} className={'alert alert--' + w.sev} style={{ marginBottom: 8 }}>
          <I n={w.sev === 'danger' ? 'octagon-x' : 'triangle-alert'} s={14} />
          <div style={{ flex: 1 }}>
            <span style={{ fontWeight: 600 }}>{w.msg}</span>
            <span className="mono dim" style={{ marginLeft: 8, fontSize: 11 }}>
              target · {w.target}
            </span>
          </div>
          <button className="btn btn--sm btn--ghost" style={{ color: 'inherit' }}>
            Fix
          </button>
          <button className="btn btn--sm btn--ghost" style={{ color: 'inherit' }}>
            Dismiss
          </button>
        </div>
      ))}

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(5, 1fr)',
          gap: 10,
          margin: '14px 0',
        }}
      >
        <div className="stat">
          <div className="label">Instances</div>
          <div className="value">
            {running}
            <span className="suf">/ {INSTANCES.length}</span>
          </div>
          <div className="mono dim" style={{ marginTop: 6, fontSize: 11 }}>
            {stopped} stopped · 0 paused
          </div>
        </div>
        <div className="stat">
          <div className="label">Containers</div>
          <div className="value">
            {cRunning}
            <span className="suf">/ {CONTAINERS.length}</span>
          </div>
          <div className="mono dim" style={{ marginTop: 6, fontSize: 11 }}>
            2 updates available
          </div>
        </div>
        <div className="stat">
          <div className="label">CPU · cluster</div>
          <div className="value">
            34<span className="suf">%</span>
          </div>
          <div style={{ marginTop: 8 }}>
            <ProgressBar v={34} />
          </div>
        </div>
        <div className="stat">
          <div className="label">RAM · cluster</div>
          <div className="value">
            52<span className="suf">%</span>
          </div>
          <div style={{ marginTop: 8 }}>
            <ProgressBar v={52} />
          </div>
        </div>
        <div className="stat" style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
          <div className="donut" style={{ ['--p']: health + '%' }}>
            <span className="n">{health}</span>
          </div>
          <div>
            <div className="label">System health</div>
            <div
              className="mono"
              style={{ fontSize: 11, color: 'var(--h-text-2)', marginTop: 8, lineHeight: 1.6 }}
            >
              <div>+ backups current</div>
              <div>+ cluster quorum</div>
              <div style={{ color: 'var(--h-warn)' }}>− storage at 72%</div>
            </div>
          </div>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 10 }}>
        <div className="card">
          <header>
            <span className="title">Nodes</span>
            <a
              className="mono dim link"
              style={{ fontSize: 11, letterSpacing: '0.14em' }}
              href="#"
              onClick={(e) => {
                e.preventDefault();
                onNav('cluster');
              }}
            >
              VIEW ALL →
            </a>
          </header>
          <table className="tbl">
            <thead>
              <tr>
                <th>Node</th>
                <th>Model</th>
                <th>Status</th>
                <th>CPU</th>
                <th>RAM</th>
                <th style={{ textAlign: 'right' }}>Power</th>
              </tr>
            </thead>
            <tbody>
              {NODES.map((n) => (
                <tr key={n.id}>
                  <td className="mono" style={{ fontWeight: 600 }}>
                    {n.name}
                  </td>
                  <td>{n.model}</td>
                  <td>{STATUS_BADGE[n.state]}</td>
                  <td style={{ width: 160 }}>
                    <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                      <ProgressBar v={n.cpuPct} />
                      <span
                        className="mono dim"
                        style={{ width: 34, fontSize: 11, textAlign: 'right' }}
                      >
                        {n.cpuPct}%
                      </span>
                    </div>
                  </td>
                  <td style={{ width: 160 }}>
                    <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                      <ProgressBar v={n.ramPct} />
                      <span
                        className="mono dim"
                        style={{ width: 34, fontSize: 11, textAlign: 'right' }}
                      >
                        {n.ramPct}%
                      </span>
                    </div>
                  </td>
                  <td className="mono" style={{ textAlign: 'right' }}>
                    {n.watts ? n.watts + ' W' : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="card">
          <header>
            <span className="title">Activity</span>
            <a
              className="mono dim link"
              style={{ fontSize: 11, letterSpacing: '0.14em' }}
              href="#"
              onClick={(e) => {
                e.preventDefault();
                onNav('audit');
              }}
            >
              AUDIT LOG →
            </a>
          </header>
          <div>
            {AUDIT.slice(0, 7).map((a, i) => (
              <div
                key={i}
                style={{
                  padding: '8px 14px',
                  display: 'flex',
                  gap: 10,
                  alignItems: 'flex-start',
                  borderBottom: '1px dashed rgba(61,61,61,0.5)',
                }}
              >
                <div style={{ flex: '0 0 6px', marginTop: 7 }}>
                  <span className={'dot dot--' + (a.status === 'ok' ? 'running' : 'stopped')} />
                </div>
                <div style={{ flex: 1, fontSize: 12, lineHeight: 1.45 }}>
                  <div>
                    <span className="mono" style={{ color: 'var(--h-text)' }}>
                      {a.user}
                    </span>{' '}
                    <span className="dim">·</span> <span>{a.action.replace(/\./g, ' ')}</span>{' '}
                    <span className="mono" style={{ color: 'var(--h-accent)' }}>
                      {a.target}
                    </span>
                  </div>
                  <div
                    className="mono dim"
                    style={{ fontSize: 10, marginTop: 2, letterSpacing: '0.08em' }}
                  >
                    {a.ts}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="card" style={{ marginTop: 10 }}>
        <header>
          <span className="title">Top Instances · by CPU</span>
          <div className="seg">
            <button className="on">CPU</button>
            <button>RAM</button>
            <button>Net</button>
            <button>Disk</button>
          </div>
        </header>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)' }}>
          {INSTANCES.filter((i) => i.status === 'running')
            .slice(0, 4)
            .map((i, idx) => (
              <div
                key={i.name}
                style={{ padding: 14, borderRight: idx < 3 ? '1px solid var(--h-border)' : 'none' }}
              >
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'flex-start',
                  }}
                >
                  <div>
                    <div
                      className="mono"
                      style={{ fontSize: 12, color: 'var(--h-text)', fontWeight: 600 }}
                    >
                      {i.name}
                    </div>
                    <div className="mono dim" style={{ fontSize: 10, marginTop: 2 }}>
                      {i.type} · {i.node}
                    </div>
                  </div>
                  {STATUS_BADGE[i.status]}
                </div>
                <div style={{ marginTop: 10, display: 'grid', gap: 6 }}>
                  <div className="meter">
                    <span className="m-label">CPU</span>
                    <div style={{ flex: 1 }}>
                      <ProgressBar v={i.cpuPct} />
                    </div>
                    <span className="m-val">{i.cpuPct}%</span>
                  </div>
                  <div className="meter">
                    <span className="m-label">RAM</span>
                    <div style={{ flex: 1 }}>
                      <ProgressBar v={i.ramPct} variant={i.ramPct > 80 ? 'warn' : undefined} />
                    </div>
                    <span className="m-val">{i.ramPct}%</span>
                  </div>
                </div>
                <div style={{ marginTop: 10, display: 'flex', gap: 4 }}>
                  <button
                    className="btn btn--sm btn--ghost"
                    onClick={() => onNav('instance:' + i.name)}
                  >
                    <I n="arrow-right" s={12} /> Open
                  </button>
                  <button className="btn btn--sm btn--ghost">
                    <I n="terminal" s={12} /> Console
                  </button>
                </div>
              </div>
            ))}
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 10, marginTop: 10 }}>
        <div className="card">
          <header>
            <span className="title">Backups</span>
          </header>
          <div style={{ padding: 14 }}>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginBottom: 10,
              }}
            >
              <div>
                <div className="mono dim" style={{ fontSize: 10, letterSpacing: '0.14em' }}>
                  LAST 24H · SUCCESS
                </div>
                <div className="display" style={{ fontSize: 22, marginTop: 4 }}>
                  24{' '}
                  <span className="dim" style={{ fontSize: 14 }}>
                    / 25
                  </span>
                </div>
              </div>
              <div style={{ textAlign: 'right' }}>
                <div className="mono dim" style={{ fontSize: 10, letterSpacing: '0.14em' }}>
                  VERIFIED
                </div>
                <div
                  className="display"
                  style={{ fontSize: 22, marginTop: 4, color: 'var(--h-success)' }}
                >
                  92
                  <span className="dim" style={{ fontSize: 14 }}>
                    %
                  </span>
                </div>
              </div>
            </div>
            <div
              className="mono"
              style={{
                fontSize: 11,
                color: 'var(--h-text-2)',
                borderTop: '1px solid var(--h-border)',
                paddingTop: 8,
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', padding: '2px 0' }}>
                <span>vm-db-1</span>
                <span className="dim">6h ago · 1.2 GB</span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', padding: '2px 0' }}>
                <span>vm-web-1</span>
                <span className="dim">2h ago · 820 MB</span>
              </div>
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  padding: '2px 0',
                  color: 'var(--h-danger)',
                }}
              >
                <span>vm-old</span>
                <span>failed · pool full</span>
              </div>
            </div>
          </div>
        </div>
        <div className="card">
          <header>
            <span className="title">Storage pools</span>
          </header>
          <div style={{ padding: 14, display: 'grid', gap: 10 }}>
            {POOLS.map((p) => (
              <div key={p.name}>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    fontSize: 12,
                    marginBottom: 4,
                  }}
                >
                  <span>
                    <span className="mono" style={{ fontWeight: 600 }}>
                      {p.name}
                    </span>{' '}
                    <span className="dim">· {p.type}</span>
                  </span>
                  <span className="mono dim">
                    {(p.used / 1024).toFixed(1)}T / {(p.size / 1024).toFixed(1)}T
                  </span>
                </div>
                <ProgressBar
                  v={(p.used / p.size) * 100}
                  variant={p.used / p.size > 0.7 ? 'warn' : undefined}
                />
              </div>
            ))}
          </div>
        </div>
        <div className="card">
          <header>
            <span className="title">Scheduled jobs</span>
          </header>
          <div>
            {SCHEDULES.slice(0, 4).map((s) => (
              <div
                key={s.name}
                style={{
                  padding: '8px 14px',
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  borderBottom: '1px dashed rgba(61,61,61,0.5)',
                }}
              >
                <div>
                  <div className="mono" style={{ fontSize: 12, fontWeight: 600 }}>
                    {s.name}
                  </div>
                  <div className="mono dim" style={{ fontSize: 10, marginTop: 2 }}>
                    {s.cron} · next {s.next}
                  </div>
                </div>
                {s.on ? STATUS_BADGE.running : STATUS_BADGE.stopped}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

// ─── INSTANCES LIST ─────────────────────────────────────────────
function PageInstances({ onNav }) {
  useStore();
  const [view, setView] = useState('table');
  const [filter, setFilter] = useState('all');
  const [sel, setSel] = useState([]);
  const [q, setQ] = useState('');
  const [focus, setFocus] = useState(0);
  const rows = INSTANCES.filter(
    (i) =>
      (filter === 'all' ||
        (filter === 'running' && i.status === 'running') ||
        (filter === 'stopped' && i.status === 'stopped')) &&
      (!q ||
        (i.name + ' ' + i.os + ' ' + i.tags.join(' ') + ' ' + i.ip)
          .toLowerCase()
          .includes(q.toLowerCase())),
  );
  const toggle = (name) =>
    setSel((s) => (s.includes(name) ? s.filter((x) => x !== name) : [...s, name]));

  useEffect(() => {
    const onKey = (e) => {
      const tag = (e.target && e.target.tagName) || '';
      if (tag === 'INPUT' || tag === 'TEXTAREA' || e.metaKey || e.ctrlKey) return;
      if (!rows.length) return;
      if (e.key === 'j' || e.key === 'ArrowDown') {
        e.preventDefault();
        setFocus((f) => Math.min(rows.length - 1, f + 1));
      } else if (e.key === 'k' || e.key === 'ArrowUp') {
        e.preventDefault();
        setFocus((f) => Math.max(0, f - 1));
      } else if (e.key === 'Enter') {
        e.preventDefault();
        const r = rows[focus];
        if (r) onNav('instance:' + r.name);
      } else if (e.key === 'x' || e.key === ' ') {
        e.preventDefault();
        const r = rows[focus];
        if (r) toggle(r.name);
      } else if (e.key === 's') {
        const r = rows[focus];
        if (r && r.status !== 'running') instanceAction(r.name, 'start');
      } else if (e.key === 'g' && !window.__gPressed) {
        window.__gPressed = setTimeout(() => (window.__gPressed = null), 500);
      } else if (e.key === 'G') {
        setFocus(rows.length - 1);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [rows, focus, onNav]);
  useEffect(() => {
    if (focus >= rows.length) setFocus(Math.max(0, rows.length - 1));
  }, [rows.length]);

  return (
    <div>
      <div className="toolbar">
        <div className="lft">
          <div style={{ position: 'relative' }}>
            <I
              n="search"
              s={13}
              style={{
                position: 'absolute',
                left: 10,
                top: 9,
                color: 'var(--h-text-3)',
                zIndex: 1,
              }}
            />
            <input
              className="input"
              value={q}
              onChange={(e) => setQ(e.target.value)}
              style={{ width: 220, paddingLeft: 28, height: 28, fontSize: 12 }}
              placeholder="Search instances…"
            />
          </div>
          <div className="seg">
            <button className={filter === 'all' ? 'on' : ''} onClick={() => setFilter('all')}>
              All{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                {INSTANCES.length}
              </span>
            </button>
            <button
              className={filter === 'running' ? 'on' : ''}
              onClick={() => setFilter('running')}
            >
              Running{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                {INSTANCES.filter((i) => i.status === 'running').length}
              </span>
            </button>
            <button
              className={filter === 'stopped' ? 'on' : ''}
              onClick={() => setFilter('stopped')}
            >
              Stopped{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                {INSTANCES.filter((i) => i.status === 'stopped').length}
              </span>
            </button>
          </div>
          <button className="btn btn--sm btn--ghost">
            <I n="filter" s={13} /> Filter
          </button>
          <span className="dim mono" style={{ fontSize: 11 }}>
            {sel.length} selected
          </span>
          <span
            className="dim mono"
            style={{ fontSize: 10, marginLeft: 4, opacity: 0.6 }}
            title="j/k move · x select · Enter open · s start"
          >
            j k ⏎ · x · s
          </span>
        </div>
        <div className="rgt">
          <div className="seg">
            <button className={view === 'table' ? 'on' : ''} onClick={() => setView('table')}>
              <I n="list" s={13} />
            </button>
            <button className={view === 'grid' ? 'on' : ''} onClick={() => setView('grid')}>
              <I n="layout-grid" s={13} />
            </button>
          </div>
          <button
            className="btn btn--sm"
            disabled={sel.length === 0}
            onClick={() => {
              sel.forEach((n) => instanceAction(n, 'start'));
              setSel([]);
            }}
          >
            <I n="play" s={13} /> Start{sel.length ? ' (' + sel.length + ')' : ''}
          </button>
          <button
            className="btn btn--sm"
            disabled={sel.length === 0}
            onClick={() => {
              window.openModal?.('confirm', {
                title: 'Stop ' + sel.length + ' instance' + (sel.length === 1 ? '' : 's') + '?',
                body:
                  sel.length >= 3
                    ? 'Targets: ' + sel.join(', ') + '. Type STOP to confirm a wide selection.'
                    : sel.join(', '),
                danger: true,
                confirmLabel: 'Stop',
                confirmMatch: sel.length >= 3 ? 'STOP' : undefined,
                onConfirm: () => {
                  sel.forEach((n) => instanceAction(n, 'stop'));
                  setSel([]);
                },
              });
            }}
          >
            <I n="square" s={13} /> Stop{sel.length ? ' (' + sel.length + ')' : ''}
          </button>
          <button className="btn btn--sm btn--primary" onClick={() => onNav('new-instance')}>
            <I n="plus" s={13} /> Create instance
          </button>
        </div>
      </div>

      {rows.length === 0 ? (
        <div style={{ padding: 40 }}>
          <EmptyState
            icon="search-x"
            title="No instances match"
            body={
              q
                ? `Nothing matches "${q}" in ${filter === 'all' ? 'all instances' : filter}. Try broadening your search.`
                : 'No instances in this filter. Switch to All to see every instance.'
            }
            action={
              <button
                className="btn btn--sm"
                onClick={() => {
                  setQ('');
                  setFilter('all');
                }}
              >
                Clear filters
              </button>
            }
          />
        </div>
      ) : view === 'table' ? (
        <table className="tbl">
          <thead>
            <tr>
              <th style={{ width: 28 }}>
                <input
                  type="checkbox"
                  readOnly
                  checked={sel.length === rows.length && rows.length > 0}
                />
              </th>
              <th>Name</th>
              <th>Type</th>
              <th>Node</th>
              <th>Status</th>
              <th>OS</th>
              <th>CPU</th>
              <th>RAM</th>
              <th>IP</th>
              <th>Uptime</th>
              <th>Tags</th>
              <th style={{ textAlign: 'right' }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r, i) => (
              <tr
                key={r.name}
                className={(sel.includes(r.name) ? 'sel' : '') + (i === focus ? ' kbd-focus' : '')}
                onClick={() => onNav('instance:' + r.name)}
                onMouseEnter={() => setFocus(i)}
                style={{ cursor: 'pointer' }}
              >
                <td
                  onClick={(e) => {
                    e.stopPropagation();
                    toggle(r.name);
                  }}
                >
                  <input type="checkbox" readOnly checked={sel.includes(r.name)} />
                </td>
                <td
                  className="mono"
                  style={{
                    fontWeight: 600,
                    color: r.status === 'running' ? 'var(--h-text)' : 'var(--h-text-2)',
                  }}
                >
                  {r.name}
                </td>
                <td>
                  <span className="badge mono">{r.type}</span>
                </td>
                <td className="mono dim">{r.node}</td>
                <td>{STATUS_BADGE[r.status]}</td>
                <td className="dim">{r.os}</td>
                <td style={{ width: 130 }}>
                  {r.status === 'running' ? (
                    <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                      <div style={{ flex: 1 }}>
                        <ProgressBar v={r.cpuPct} />
                      </div>
                      <span
                        className="mono dim"
                        style={{ width: 28, fontSize: 11, textAlign: 'right' }}
                      >
                        {r.cpuPct}
                      </span>
                    </div>
                  ) : (
                    <span className="dim">—</span>
                  )}
                </td>
                <td style={{ width: 130 }}>
                  {r.status === 'running' ? (
                    <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                      <div style={{ flex: 1 }}>
                        <ProgressBar v={r.ramPct} variant={r.ramPct > 80 ? 'warn' : undefined} />
                      </div>
                      <span
                        className="mono dim"
                        style={{ width: 28, fontSize: 11, textAlign: 'right' }}
                      >
                        {r.ramPct}
                      </span>
                    </div>
                  ) : (
                    <span className="dim">—</span>
                  )}
                </td>
                <td className="mono">{r.ip}</td>
                <td className="mono dim">{r.uptime}</td>
                <td>
                  {r.tags.map((t) => (
                    <span
                      key={t}
                      className="badge mono"
                      style={{ marginRight: 3, color: 'var(--h-text-2)' }}
                    >
                      {t}
                    </span>
                  ))}
                </td>
                <td
                  style={{ textAlign: 'right', whiteSpace: 'nowrap' }}
                  onClick={(e) => e.stopPropagation()}
                >
                  <button className="btn btn--sm btn--ghost" title="Console">
                    <I n="terminal" s={13} />
                  </button>
                  <button className="btn btn--sm btn--ghost" title="Restart">
                    <I n="rotate-cw" s={13} />
                  </button>
                  <button className="btn btn--sm btn--ghost" title="More">
                    <I n="more-horizontal" s={13} />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <div
          style={{
            padding: 14,
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
            gap: 10,
          }}
        >
          {rows.map((r) => (
            <div
              key={r.name}
              className="card"
              style={{ cursor: 'pointer' }}
              onClick={() => onNav('instance:' + r.name)}
            >
              <header style={{ background: 'var(--h-surface-2)' }}>
                <div>
                  <div className="mono" style={{ fontSize: 13, fontWeight: 600 }}>
                    {r.name}
                  </div>
                  <div className="mono dim" style={{ fontSize: 10, marginTop: 2 }}>
                    {r.type} · {r.os}
                  </div>
                </div>
                {STATUS_BADGE[r.status]}
              </header>
              <div style={{ padding: 12, display: 'grid', gap: 8 }}>
                <div className="meter">
                  <span className="m-label">CPU</span>
                  <div style={{ flex: 1 }}>
                    <ProgressBar v={r.cpuPct} />
                  </div>
                  <span className="m-val">{r.cpuPct}%</span>
                </div>
                <div className="meter">
                  <span className="m-label">RAM</span>
                  <div style={{ flex: 1 }}>
                    <ProgressBar v={r.ramPct} variant={r.ramPct > 80 ? 'warn' : undefined} />
                  </div>
                  <span className="m-val">{r.ramPct}%</span>
                </div>
                <div className="meter">
                  <span className="m-label">Node</span>
                  <span
                    className="mono"
                    style={{ flex: 1, fontSize: 11, color: 'var(--h-text-2)' }}
                  >
                    {r.node}
                  </span>
                </div>
                <div className="meter">
                  <span className="m-label">IP</span>
                  <span className="mono" style={{ flex: 1, fontSize: 11 }}>
                    {r.ip}
                  </span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ─── INSTANCE DETAIL ────────────────────────────────────────────
function PageInstanceDetail({ name, onNav }) {
  useStore();
  const inst = INSTANCES.find((i) => i.name === name) || INSTANCES[0];
  const [tab, setTab] = useState('overview');
  const TABS = [
    ['overview', 'Overview', 'gauge'],
    ['console', 'Console', 'terminal'],
    ['resources', 'Resources', 'cpu'],
    ['network', 'Network', 'network'],
    ['storage', 'Storage', 'hard-drive'],
    ['snapshots', 'Snapshots', 'camera'],
    ['backups', 'Backups', 'archive'],
    ['tasks', 'Tasks', 'activity'],
  ];

  return (
    <div>
      <div
        style={{
          padding: '14px 20px',
          borderBottom: '1px solid var(--h-border)',
          background: 'var(--h-surface)',
        }}
      >
        <div className="eyebrow" style={{ marginBottom: 4 }}>
          INSTANCE / {inst.type} / {inst.node.toUpperCase()}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 14, flexWrap: 'wrap' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <span
              className={'dot dot--' + (inst.status === 'running' ? 'running' : 'stopped')}
              style={{ width: 10, height: 10 }}
            />
            <h1 className="stencil" style={{ fontSize: 22, margin: 0 }}>
              {inst.name}
            </h1>
            {STATUS_BADGE[inst.status]}
            <span className="badge mono">{inst.os}</span>
            {inst.tags.map((t) => (
              <span key={t} className="badge mono" style={{ color: 'var(--h-text-2)' }}>
                #{t}
              </span>
            ))}
          </div>
          <div style={{ flex: 1 }} />
          <div className="btn-group">
            {inst.status === 'running' ? (
              <>
                <button
                  className="btn btn--sm"
                  onClick={() => instanceAction(inst.name, 'restart')}
                >
                  <I n="rotate-cw" s={13} /> Restart
                </button>
                <button className="btn btn--sm" onClick={() => instanceAction(inst.name, 'pause')}>
                  <I n="pause" s={13} /> Pause
                </button>
                <button
                  className="btn btn--sm btn--danger"
                  onClick={() =>
                    window.openModal?.('confirm', {
                      title: 'Stop ' + inst.name + '?',
                      body: 'Graceful shutdown. Running processes will receive SIGTERM.',
                      danger: true,
                      confirmLabel: 'Stop',
                      onConfirm: () => instanceAction(inst.name, 'stop'),
                    })
                  }
                >
                  <I n="square" s={13} /> Stop
                </button>
              </>
            ) : (
              <button
                className="btn btn--sm btn--primary"
                onClick={() => instanceAction(inst.name, 'start')}
              >
                <I n="play" s={13} /> Start
              </button>
            )}
          </div>
          <div className="btn-group">
            <button className="btn btn--sm" onClick={() => onNav('console:' + inst.name)}>
              <I n="terminal" s={13} /> Console
            </button>
            <button className="btn btn--sm" onClick={() => instanceAction(inst.name, 'snapshot')}>
              <I n="camera" s={13} /> Snapshot
            </button>
            <button className="btn btn--sm" onClick={() => instanceAction(inst.name, 'backup')}>
              <I n="archive" s={13} /> Backup
            </button>
            <button className="btn btn--sm">
              <I n="more-horizontal" s={13} />
            </button>
          </div>
        </div>
        <div className="tabs" style={{ marginTop: 16, marginLeft: -4 }}>
          {TABS.map(([id, label, icon]) => (
            <button
              key={id}
              className={'tab' + (tab === id ? ' on' : '')}
              onClick={() => setTab(id)}
            >
              <I n={icon} s={13} /> {label}
            </button>
          ))}
        </div>
      </div>
      <div style={{ padding: 20 }}>
        {tab === 'overview' && <InstanceOverview inst={inst} />}
        {tab === 'console' && <InstanceConsole inst={inst} />}
        {tab === 'resources' && <InstanceResources inst={inst} />}
        {tab === 'network' && <InstanceNetwork inst={inst} />}
        {tab === 'storage' && <InstanceStorage inst={inst} />}
        {tab === 'snapshots' && <InstanceSnapshots inst={inst} />}
        {tab === 'backups' && <InstanceBackups inst={inst} />}
        {tab === 'tasks' && <InstanceTasks inst={inst} />}
      </div>
    </div>
  );
}

function InstanceOverview({ inst }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 14 }}>
      <div style={{ display: 'grid', gap: 14 }}>
        <div className="card">
          <header>
            <span className="title">Live · last 5 minutes</span>
          </header>
          <div
            style={{ padding: 14, display: 'grid', gridTemplateColumns: 'repeat(4,1fr)', gap: 14 }}
          >
            {[
              ['CPU', inst.cpuPct, '%'],
              ['RAM', inst.ramPct, '%'],
              ['Net', 128, 'KB/s'],
              ['Disk', 4.2, 'MB/s'],
            ].map(([l, v, u]) => (
              <div key={l}>
                <div
                  className="mono dim"
                  style={{ fontSize: 10, letterSpacing: '0.14em', textTransform: 'uppercase' }}
                >
                  {l}
                </div>
                <div
                  className="display"
                  style={{ fontSize: 26, marginTop: 4, fontVariantNumeric: 'tabular-nums' }}
                >
                  {v}
                  <span className="dim" style={{ fontSize: 12 }}>
                    {' '}
                    {u}
                  </span>
                </div>
                <Sparkline />
              </div>
            ))}
          </div>
        </div>
        <div className="card">
          <header>
            <span className="title">Configuration</span>
            <button className="btn btn--sm btn--ghost">
              <I n="pencil" s={13} /> Edit
            </button>
          </header>
          <div style={{ padding: '4px 14px 14px' }}>
            <dl className="desc">
              <dt>Name</dt>
              <dd className="mono">{inst.name}</dd>
              <dt>Type</dt>
              <dd>{inst.type === 'VM' ? 'Virtual Machine' : 'LXC Container'}</dd>
              <dt>Node</dt>
              <dd className="mono">{inst.node}</dd>
              <dt>OS</dt>
              <dd>{inst.os}</dd>
              <dt>vCPU</dt>
              <dd>{inst.cores} cores</dd>
              <dt>Memory</dt>
              <dd>{inst.ram} GB</dd>
              <dt>IP address</dt>
              <dd>
                <Copyable text={inst.ip} />
              </dd>
              <dt>MAC</dt>
              <dd>
                <Copyable text={inst.mac} />
              </dd>
              <dt>Boot disk</dt>
              <dd className="mono">default/{inst.name}-disk-0 · 32 GB</dd>
              <dt>Created</dt>
              <dd>2026-02-11 · by admin</dd>
            </dl>
          </div>
        </div>
      </div>
      <div style={{ display: 'grid', gap: 14 }}>
        <div className="card">
          <header>
            <span className="title">Health</span>
          </header>
          <div style={{ padding: 14, display: 'flex', gap: 14, alignItems: 'center' }}>
            <div className="donut" style={{ ['--p']: inst.health + '%' }}>
              <span className="n">{inst.health}</span>
            </div>
            <div
              className="mono"
              style={{ fontSize: 11, color: 'var(--h-text-2)', lineHeight: 1.7 }}
            >
              <div>
                <span style={{ color: 'var(--h-success)' }}>+</span> Backup verified{' '}
                {inst.backupAge} ago
              </div>
              <div>
                <span style={{ color: 'var(--h-success)' }}>+</span> Guest agent responding
              </div>
              <div>
                <span style={{ color: 'var(--h-success)' }}>+</span> Firewall: 2 rules active
              </div>
              {inst.ramPct > 80 && (
                <div>
                  <span style={{ color: 'var(--h-warn)' }}>−</span> RAM pressure {inst.ramPct}%
                </div>
              )}
            </div>
          </div>
        </div>
        <div className="card">
          <header>
            <span className="title">Backups</span>
          </header>
          <div style={{ padding: 14 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div>
                <div className="mono dim" style={{ fontSize: 10, letterSpacing: '0.14em' }}>
                  LAST SUCCESSFUL
                </div>
                <div className="display" style={{ fontSize: 20, marginTop: 4 }}>
                  {inst.backupAge} ago
                </div>
              </div>
              <button className="btn btn--sm" onClick={() => instanceAction(inst.name, 'backup')}>
                <I n="play" s={13} /> Run now
              </button>
            </div>
            <div
              className="mono"
              style={{
                fontSize: 11,
                color: 'var(--h-text-2)',
                marginTop: 12,
                borderTop: '1px solid var(--h-border)',
                paddingTop: 10,
                lineHeight: 1.8,
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span>Schedule</span>
                <span>daily-backup</span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span>Retention</span>
                <span>7d · 4w · 3m</span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span>Destination</span>
                <span>default</span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span>Verified</span>
                <span style={{ color: 'var(--h-success)' }}>✓ last run</span>
              </div>
            </div>
          </div>
        </div>
        <div className="card">
          <header>
            <span className="title">Recent activity</span>
          </header>
          <div>
            {AUDIT.slice(0, 5).map((a, i) => (
              <div
                key={i}
                style={{
                  padding: '6px 14px',
                  fontSize: 12,
                  borderBottom: '1px dashed rgba(61,61,61,0.4)',
                }}
              >
                <div className="mono" style={{ fontSize: 11 }}>
                  <span style={{ color: 'var(--h-text-2)' }}>{a.user}</span>{' '}
                  <span>{a.action.replace(/\./g, ' ')}</span>
                </div>
                <div className="mono dim" style={{ fontSize: 10, marginTop: 2 }}>
                  {a.ts}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function Sparkline() {
  const pts = [6, 8, 5, 9, 12, 8, 10, 14, 11, 9, 13, 15, 11, 10, 12, 9, 7, 8, 10, 12];
  const w = 140,
    h = 28,
    max = Math.max(...pts),
    min = Math.min(...pts),
    step = w / (pts.length - 1);
  const d = pts
    .map((p, i) => `${i === 0 ? 'M' : 'L'} ${i * step} ${h - ((p - min) / (max - min)) * h}`)
    .join(' ');
  return (
    <svg width={w} height={h} style={{ display: 'block', marginTop: 6, overflow: 'visible' }}>
      <path d={d} stroke="var(--h-accent)" strokeWidth="1.5" fill="none" />
      <path d={`${d} L ${w} ${h} L 0 ${h} Z`} fill="var(--h-accent)" opacity="0.08" />
    </svg>
  );
}

function InstanceConsole({ inst }) {
  return (
    <div className="card">
      <header>
        <span className="title">Console · VNC</span>
        <div style={{ display: 'flex', gap: 6 }}>
          <button className="btn btn--sm">
            <I n="keyboard" s={13} /> Send Ctrl-Alt-Del
          </button>
          <button className="btn btn--sm">
            <I n="maximize-2" s={13} /> Fullscreen
          </button>
          <button className="btn btn--sm">
            <I n="external-link" s={13} /> Open in new tab
          </button>
        </div>
      </header>
      <div style={{ background: '#000', height: 440, position: 'relative' }}>
        <div
          className="term"
          style={{
            margin: 16,
            height: 'calc(100% - 32px)',
            border: 'none',
            background: 'transparent',
            overflow: 'hidden',
          }}
        >
          <div>
            <span className="c-lime">
              {inst.os.includes('Windows')
                ? 'Microsoft Windows [Version 10.0.22631.3007]'
                : 'Debian GNU/Linux 13 ' + inst.name + ' tty1'}
            </span>
          </div>
          <div style={{ marginTop: 8 }}>
            <span className="c-dim">{inst.name} login:</span> admin
          </div>
          <div>
            <span className="c-dim">Password:</span> ●●●●●●●●
          </div>
          <div style={{ marginTop: 8 }}>Last login: Wed Apr 22 10:02:44 UTC 2026 on tty1</div>
          <div style={{ marginTop: 4 }}>
            <span className="c-ok">✓</span> Guest agent connected
          </div>
          <div style={{ marginTop: 12 }}>
            <span className="c-lime">admin@{inst.name}</span>
            <span className="c-dim">:</span>
            <span style={{ color: '#69b7ff' }}>~</span>
            <span className="c-lime">&nbsp;✦&nbsp;</span>
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
        <div
          style={{
            position: 'absolute',
            top: 12,
            right: 20,
            display: 'flex',
            gap: 8,
            alignItems: 'center',
            fontSize: 10,
          }}
          className="mono"
        >
          <span className="dot dot--running" />
          <span className="dim">connected · vnc://10.0.0.1:5901</span>
        </div>
      </div>
    </div>
  );
}

function InstanceResources({ inst }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
      <div className="card">
        <header>
          <span className="title">Hardware</span>
          <button className="btn btn--sm">
            <I n="pencil" s={13} /> Edit
          </button>
        </header>
        <div style={{ padding: 14 }}>
          <dl className="desc">
            <dt>CPU type</dt>
            <dd>host (pass-through)</dd>
            <dt>vCPU</dt>
            <dd>{inst.cores} cores</dd>
            <dt>CPU limit</dt>
            <dd>unlimited</dd>
            <dt>Memory</dt>
            <dd>{inst.ram} GB (balloon on)</dd>
            <dt>BIOS</dt>
            <dd>OVMF / UEFI</dd>
            <dt>Machine</dt>
            <dd className="mono">q35</dd>
            <dt>SCSI controller</dt>
            <dd className="mono">virtio-scsi-single</dd>
            <dt>PCI passthrough</dt>
            <dd className="dim">—</dd>
          </dl>
        </div>
      </div>
      <div className="card">
        <header>
          <span className="title">Options</span>
        </header>
        <div style={{ padding: 14 }}>
          <dl className="desc">
            <dt>Start on boot</dt>
            <dd>
              <span style={{ color: 'var(--h-success)' }}>enabled</span>
            </dd>
            <dt>Boot order</dt>
            <dd className="mono">scsi0, net0</dd>
            <dt>Start delay</dt>
            <dd>0s</dd>
            <dt>Shutdown policy</dt>
            <dd>ACPI shutdown · 180s timeout</dd>
            <dt>QEMU agent</dt>
            <dd>
              <span style={{ color: 'var(--h-success)' }}>connected</span>
            </dd>
            <dt>Cloud-init</dt>
            <dd>user-data.yaml · applied</dd>
            <dt>Protection</dt>
            <dd className="dim">off</dd>
          </dl>
        </div>
      </div>
    </div>
  );
}

function InstanceNetwork({ inst }) {
  return (
    <div className="card">
      <header>
        <span className="title">Network interfaces</span>
        <button className="btn btn--sm">
          <I n="plus" s={13} /> Add interface
        </button>
      </header>
      <table className="tbl">
        <thead>
          <tr>
            <th>ID</th>
            <th>Bridge</th>
            <th>Model</th>
            <th>MAC</th>
            <th>IP</th>
            <th>Firewall</th>
            <th>Rate limit</th>
            <th>MTU</th>
            <th />
          </tr>
        </thead>
        <tbody>
          <tr>
            <td className="mono">net0</td>
            <td className="mono">bridge0</td>
            <td className="mono">virtio</td>
            <td>
              <Copyable text={inst.mac} />
            </td>
            <td>
              <Copyable text={inst.ip} />
            </td>
            <td>{STATUS_BADGE.running}</td>
            <td className="dim">—</td>
            <td className="mono">1500</td>
            <td style={{ textAlign: 'right' }}>
              <button className="btn btn--sm btn--ghost">
                <I n="pencil" s={13} />
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  );
}

function InstanceStorage({ inst }) {
  return (
    <div className="card">
      <header>
        <span className="title">Disks</span>
        <button className="btn btn--sm">
          <I n="plus" s={13} /> Attach disk
        </button>
      </header>
      <table className="tbl">
        <thead>
          <tr>
            <th>ID</th>
            <th>Size</th>
            <th>Pool</th>
            <th>Format</th>
            <th>Cache</th>
            <th>Discard</th>
            <th>SSD emu.</th>
            <th>IOPS</th>
            <th />
          </tr>
        </thead>
        <tbody>
          <tr>
            <td className="mono">scsi0</td>
            <td>32 GB</td>
            <td className="mono">default</td>
            <td className="mono">raw</td>
            <td>writeback</td>
            <td>
              <span style={{ color: 'var(--h-success)' }}>on</span>
            </td>
            <td>
              <span style={{ color: 'var(--h-success)' }}>on</span>
            </td>
            <td className="mono">—</td>
            <td style={{ textAlign: 'right' }}>
              <button className="btn btn--sm btn--ghost">
                <I n="pencil" s={13} />
              </button>
            </td>
          </tr>
          <tr>
            <td className="mono">scsi1</td>
            <td>200 GB</td>
            <td className="mono">fast-pool</td>
            <td className="mono">raw</td>
            <td>none</td>
            <td>
              <span style={{ color: 'var(--h-success)' }}>on</span>
            </td>
            <td>
              <span style={{ color: 'var(--h-success)' }}>on</span>
            </td>
            <td className="mono">5000</td>
            <td style={{ textAlign: 'right' }}>
              <button className="btn btn--sm btn--ghost">
                <I n="pencil" s={13} />
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  );
}

function InstanceSnapshots({ inst }) {
  useStore();
  return (
    <div className="card">
      <header>
        <span className="title">Snapshots</span>
        <button
          className="btn btn--sm btn--primary"
          onClick={() => instanceAction(inst.name, 'snapshot')}
        >
          <I n="camera" s={13} /> Take snapshot
        </button>
      </header>
      <table className="tbl">
        <thead>
          <tr>
            <th style={{ width: 28 }} />
            <th>Name</th>
            <th>Date</th>
            <th>RAM</th>
            <th>Size</th>
            <th>Description</th>
            <th style={{ textAlign: 'right' }}>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>
              <I n="circle-dot" s={12} color="var(--h-accent)" />
            </td>
            <td className="mono" style={{ color: 'var(--h-accent)' }}>
              NOW
            </td>
            <td className="mono dim">current state</td>
            <td className="dim">—</td>
            <td className="dim">—</td>
            <td className="dim">running</td>
            <td />
          </tr>
          {SNAPSHOTS.map((s, i) => (
            <tr key={i}>
              <td>
                <I n="circle" s={12} color="var(--h-text-3)" />
              </td>
              <td className="mono">{s.name}</td>
              <td className="mono dim">{s.date}</td>
              <td>
                {s.ram ? (
                  <span style={{ color: 'var(--h-success)' }}>✓</span>
                ) : (
                  <span className="dim">—</span>
                )}
              </td>
              <td className="mono">{s.size}</td>
              <td className="dim">{s.desc}</td>
              <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                <button className="btn btn--sm btn--ghost">Rollback</button>
                <button className="btn btn--sm btn--ghost">
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

function InstanceBackups({ inst }) {
  const BK = [
    { ts: '2026-04-22 02:00', mode: 'scheduled', size: '1.2 GB', dur: '1m 12s', verified: true },
    { ts: '2026-04-21 02:00', mode: 'scheduled', size: '1.2 GB', dur: '1m 08s', verified: true },
    { ts: '2026-04-20 02:00', mode: 'scheduled', size: '1.1 GB', dur: '1m 10s', verified: true },
    { ts: '2026-04-19 14:18', mode: 'manual', size: '1.1 GB', dur: '1m 04s', verified: false },
  ];
  return (
    <div className="card">
      <header>
        <span className="title">Backups</span>
        <div style={{ display: 'flex', gap: 6 }}>
          <button
            className="btn btn--sm"
            onClick={() =>
              window.toast?.info(
                'Verifying backups',
                'Running fsck on 14 snapshots across 3 pools — est. 4 min',
              )
            }
          >
            <I n="shield-check" s={13} /> Verify all
          </button>
          <button
            className="btn btn--sm btn--primary"
            onClick={() =>
              window.toast?.success('Backup started', 'Snapshotting 8 instances to backup-primary')
            }
          >
            <I n="archive" s={13} /> Run backup
          </button>
        </div>
      </header>
      <table className="tbl">
        <thead>
          <tr>
            <th>Timestamp</th>
            <th>Mode</th>
            <th>Size</th>
            <th>Duration</th>
            <th>Verified</th>
            <th>Destination</th>
            <th style={{ textAlign: 'right' }}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {BK.map((b, i) => (
            <tr key={i}>
              <td className="mono">{b.ts}</td>
              <td>{b.mode}</td>
              <td className="mono">{b.size}</td>
              <td className="mono dim">{b.dur}</td>
              <td>
                {b.verified ? (
                  <span style={{ color: 'var(--h-success)' }}>✓ ok</span>
                ) : (
                  <span className="dim">pending</span>
                )}
              </td>
              <td className="mono dim">default</td>
              <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                <button className="btn btn--sm btn--ghost">Restore</button>
                <button className="btn btn--sm btn--ghost">Verify</button>
                <button className="btn btn--sm btn--ghost">
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

function InstanceTasks({ inst }) {
  const T = TASKS.slice(0, 6);
  return (
    <div className="card">
      <header>
        <span className="title">Tasks</span>
      </header>
      <table className="tbl">
        <thead>
          <tr>
            <th>ID</th>
            <th>Operation</th>
            <th>Status</th>
            <th>Progress</th>
            <th>User</th>
            <th>Started</th>
          </tr>
        </thead>
        <tbody>
          {T.map((t, i) => (
            <tr key={i}>
              <td className="mono dim">{t.id}</td>
              <td className="mono">{t.op}</td>
              <td>{STATUS_BADGE[t.status]}</td>
              <td style={{ width: 200 }}>
                <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                  <div style={{ flex: 1 }}>
                    <ProgressBar v={t.progress} />
                  </div>
                  <span
                    className="mono dim"
                    style={{ width: 34, fontSize: 11, textAlign: 'right' }}
                  >
                    {t.progress}%
                  </span>
                </div>
              </td>
              <td>{t.user}</td>
              <td className="mono dim">{t.started}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ─── CONTAINERS ─────────────────────────────────────────────────
function PageContainers({ onNav }) {
  return (
    <div>
      <div className="toolbar">
        <div className="lft">
          <input
            className="input"
            style={{ width: 220, height: 28, fontSize: 12 }}
            placeholder="Search containers…"
          />
          <div className="seg">
            <button className="on">
              All{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                {CONTAINERS.length}
              </span>
            </button>
            <button>Running</button>
            <button>Exited</button>
            <button>
              Updates{' '}
              <span className="mono" style={{ marginLeft: 4, color: 'var(--h-accent)' }}>
                2
              </span>
            </button>
          </div>
          <button className="btn btn--sm btn--ghost">
            <I n="filter" s={13} /> By stack
          </button>
        </div>
        <div className="rgt">
          <button
            className="btn btn--sm"
            onClick={() =>
              window.toast?.info(
                'Pull image',
                'Enter an image ref (e.g. nginx:alpine) — dialog stub',
              )
            }
          >
            <I n="download" s={13} /> Pull image
          </button>
          <button
            className="btn btn--sm"
            onClick={() =>
              window.toast?.info('From Compose', 'Paste a docker-compose.yml — dialog stub')
            }
          >
            <I n="upload" s={13} /> From Compose
          </button>
          <button
            className="btn btn--sm btn--primary"
            onClick={() => window.openModal?.('create-vm', { initialType: 'Container' })}
          >
            <I n="plus" s={13} /> New container
          </button>
        </div>
      </div>
      <table className="tbl">
        <thead>
          <tr>
            <th style={{ width: 28 }}>
              <input type="checkbox" />
            </th>
            <th>Name</th>
            <th>Image</th>
            <th>Stack</th>
            <th>Status</th>
            <th>Health</th>
            <th>Ports</th>
            <th>CPU</th>
            <th>RAM</th>
            <th>Update</th>
            <th style={{ textAlign: 'right' }}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {CONTAINERS.map((c) => (
            <tr
              key={c.id}
              onClick={() => onNav?.('container:' + c.name)}
              style={{ cursor: 'pointer' }}
            >
              <td onClick={(e) => e.stopPropagation()}>
                <input type="checkbox" />
              </td>
              <td className="mono" style={{ fontWeight: 600, color: 'var(--h-accent)' }}>
                {c.name}
              </td>
              <td className="mono dim">{c.image}</td>
              <td>
                <span className="badge mono" style={{ color: 'var(--h-text-2)' }}>
                  {c.stack}
                </span>
              </td>
              <td>{STATUS_BADGE[c.status]}</td>
              <td>
                {c.health === 'healthy' && (
                  <span style={{ color: 'var(--h-success)' }}>● healthy</span>
                )}
                {c.health === 'degraded' && (
                  <span style={{ color: 'var(--h-warn)' }}>● degraded</span>
                )}
                {c.health === '—' && <span className="dim">—</span>}
              </td>
              <td className="mono">{c.ports}</td>
              <td style={{ width: 120 }}>
                <ProgressBar v={c.cpu} />
              </td>
              <td style={{ width: 120 }}>
                <ProgressBar v={c.ram} />
              </td>
              <td>
                {c.update ? (
                  <span style={{ color: 'var(--h-accent)' }}>● available</span>
                ) : (
                  <span className="dim">current</span>
                )}
              </td>
              <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                <button className="btn btn--sm btn--ghost">
                  <I n="terminal" s={13} />
                </button>
                <button className="btn btn--sm btn--ghost">
                  <I n="scroll-text" s={13} />
                </button>
                <button className="btn btn--sm btn--ghost">
                  <I n="rotate-cw" s={13} />
                </button>
                <button className="btn btn--sm btn--ghost">
                  <I n="more-horizontal" s={13} />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ─── KUBERNETES ─────────────────────────────────────────────────
function PageKubernetes() {
  return (
    <div style={{ padding: 20, display: 'grid', gap: 14 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end' }}>
        <div>
          <div className="eyebrow">CLUSTERS / K3S / APR 2026</div>
          <h1 className="stencil" style={{ fontSize: 22, margin: '6px 0 0' }}>
            Kubernetes
          </h1>
          <div className="muted" style={{ fontSize: 13, marginTop: 2 }}>
            3 clusters · 5 nodes · 42 pods running
          </div>
        </div>
        <button
          className="btn btn--primary"
          onClick={() =>
            window.toast?.info('Create cluster', '3-node control plane wizard — stub for v0.2')
          }
        >
          <I n="plus" s={13} /> Create cluster
        </button>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 14 }}>
        {CLUSTERS.map((c) => (
          <div key={c.name} className="card">
            <header>
              <div>
                <div className="mono" style={{ fontSize: 14, fontWeight: 600 }}>
                  {c.name}
                </div>
                <div className="mono dim" style={{ fontSize: 10, marginTop: 2 }}>
                  {c.flavor} · {c.version}
                </div>
              </div>
              {STATUS_BADGE[c.status]}
            </header>
            <div style={{ padding: 14 }}>
              <dl className="desc" style={{ gridTemplateColumns: '100px 1fr' }}>
                <dt>Nodes</dt>
                <dd>
                  {c.nodes} <span className="dim">({c.workers} workers)</span>
                </dd>
                <dt>CPU</dt>
                <dd>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <div style={{ flex: 1 }}>
                      <ProgressBar v={c.cpu} />
                    </div>
                    <span className="mono dim">{c.cpu}%</span>
                  </div>
                </dd>
                <dt>RAM</dt>
                <dd>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <div style={{ flex: 1 }}>
                      <ProgressBar v={c.ram} variant={c.ram > 70 ? 'warn' : undefined} />
                    </div>
                    <span className="mono dim">{c.ram}%</span>
                  </div>
                </dd>
                <dt>Endpoint</dt>
                <dd>
                  <Copyable text={'https://' + c.name + '.k8s.local:6443'} />
                </dd>
              </dl>
              <div style={{ marginTop: 12, display: 'flex', gap: 6 }}>
                <button className="btn btn--sm">
                  <I n="download" s={13} /> kubeconfig
                </button>
                <button className="btn btn--sm">
                  <I n="terminal" s={13} /> kubectl
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
      <div className="card">
        <header>
          <span className="title">prod-k8s · workloads</span>
          <div className="tabs" style={{ border: 'none', marginBottom: 0 }}>
            <button className="tab on">
              Pods <span className="num">42</span>
            </button>
            <button className="tab">
              Deployments <span className="num">18</span>
            </button>
            <button className="tab">
              Services <span className="num">22</span>
            </button>
            <button className="tab">
              Nodes <span className="num">3</span>
            </button>
            <button className="tab">
              Namespaces <span className="num">8</span>
            </button>
          </div>
        </header>
        <table className="tbl">
          <thead>
            <tr>
              <th>Namespace</th>
              <th>Pod</th>
              <th>Status</th>
              <th>Ready</th>
              <th>Restarts</th>
              <th>Age</th>
              <th>Node</th>
              <th>CPU</th>
              <th>RAM</th>
            </tr>
          </thead>
          <tbody>
            {[
              [
                'kube-system',
                'coredns-5d78c9869d-x8q2p',
                'Running',
                '1/1',
                0,
                '14d',
                'node-1',
                2,
                12,
              ],
              ['default', 'web-5f7b9c-abc12', 'Running', '1/1', 0, '3d', 'node-1', 45, 38],
              ['default', 'web-5f7b9c-def34', 'Running', '1/1', 0, '3d', 'node-2', 42, 41],
              ['monitoring', 'prometheus-0', 'Running', '2/2', 1, '14d', 'node-1', 18, 65],
              ['monitoring', 'grafana-7b8c-xy', 'Running', '1/1', 0, '14d', 'node-1', 4, 22],
              ['ingress', 'traefik-abcde', 'Running', '1/1', 0, '14d', 'node-1', 1, 15],
              ['default', 'worker-queue-xxx', 'Pending', '0/1', 0, '2m', '—', 0, 0],
            ].map((r, i) => (
              <tr key={i}>
                <td className="mono dim">{r[0]}</td>
                <td className="mono">{r[1]}</td>
                <td>{r[2] === 'Running' ? STATUS_BADGE.running : STATUS_BADGE.warn}</td>
                <td className="mono">{r[3]}</td>
                <td className="mono dim">{r[4]}</td>
                <td className="mono dim">{r[5]}</td>
                <td className="mono">{r[6]}</td>
                <td style={{ width: 100 }}>
                  {r[6] === '—' ? <span className="dim">—</span> : <ProgressBar v={r[7]} />}
                </td>
                <td style={{ width: 100 }}>
                  {r[6] === '—' ? <span className="dim">—</span> : <ProgressBar v={r[8]} />}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ─── STORAGE ────────────────────────────────────────────────────
function PageStorage() {
  return (
    <div>
      <div className="toolbar">
        <div className="lft">
          <input
            className="input"
            style={{ width: 220, height: 28, fontSize: 12 }}
            placeholder="Search pools & volumes…"
          />
          <div className="seg">
            <button className="on">
              Pools{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                {POOLS.length}
              </span>
            </button>
            <button>
              Volumes{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                14
              </span>
            </button>
            <button>
              ISO images{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                7
              </span>
            </button>
            <button>
              Snapshots{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                26
              </span>
            </button>
          </div>
        </div>
        <div className="rgt">
          <button
            className="btn btn--sm"
            onClick={() =>
              window.toast?.info('Upload ISO', 'Select a local .iso to stream into pool-iso')
            }
          >
            <I n="upload" s={13} /> Upload ISO
          </button>
          <button
            className="btn btn--sm btn--primary"
            onClick={() =>
              window.toast?.info(
                'Create pool',
                'Pool wizard — choose backend (ZFS / LVM / Directory / Ceph)',
              )
            }
          >
            <I n="plus" s={13} /> Create pool
          </button>
        </div>
      </div>
      <div style={{ padding: 14, display: 'grid', gridTemplateColumns: 'repeat(3,1fr)', gap: 10 }}>
        {POOLS.map((p) => (
          <div key={p.name} className="card">
            <header>
              <div>
                <div className="mono" style={{ fontSize: 14, fontWeight: 600 }}>
                  {p.name}
                </div>
                <div className="mono dim" style={{ fontSize: 10, marginTop: 2 }}>
                  {p.type.toUpperCase()} · {p.inst} instances
                </div>
              </div>
              {STATUS_BADGE.running}
            </header>
            <div style={{ padding: 14 }}>
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  fontSize: 12,
                  marginBottom: 6,
                }}
              >
                <span className="dim mono">USED</span>
                <span className="mono">
                  {(p.used / 1024).toFixed(2)} TB / {(p.size / 1024).toFixed(2)} TB
                </span>
              </div>
              <ProgressBar
                v={(p.used / p.size) * 100}
                variant={p.used / p.size > 0.7 ? 'warn' : undefined}
              />
              <dl className="desc" style={{ gridTemplateColumns: '100px 1fr', marginTop: 14 }}>
                <dt>Compression</dt>
                <dd>lz4</dd>
                <dt>Dedup</dt>
                <dd className="dim">off</dd>
                <dt>Mount</dt>
                <dd className="mono">/mnt/{p.name}</dd>
                <dt>Health</dt>
                <dd>
                  <span style={{ color: 'var(--h-success)' }}>● ONLINE</span>
                </dd>
              </dl>
            </div>
          </div>
        ))}
      </div>
      <div className="card" style={{ margin: 14 }}>
        <header>
          <span className="title">Volumes</span>
        </header>
        <table className="tbl">
          <thead>
            <tr>
              <th>Volume</th>
              <th>Pool</th>
              <th>Size</th>
              <th>Used</th>
              <th>Attached to</th>
              <th>Format</th>
              <th>Created</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {[
              ['vm-web-1-disk-0', 'default', 32, 24, 'vm-web-1', 'raw', '2026-02-11'],
              ['vm-web-1-disk-1', 'fast-pool', 200, 142, 'vm-web-1', 'raw', '2026-02-11'],
              ['vm-db-1-disk-0', 'default', 64, 48, 'vm-db-1', 'raw', '2026-02-01'],
              ['vm-db-1-data', 'fast-pool', 500, 320, 'vm-db-1', 'raw', '2026-02-01'],
              ['ct-gitea-rootfs', 'default', 8, 3.2, 'ct-gitea', 'subvol', '2026-03-10'],
              ['iso-debian-13', 'default', 0.6, 0.6, '—', 'iso', '2026-03-20'],
            ].map((r, i) => (
              <tr key={i}>
                <td className="mono" style={{ fontWeight: 600 }}>
                  {r[0]}
                </td>
                <td className="mono dim">{r[1]}</td>
                <td className="mono">{r[2]} GB</td>
                <td style={{ width: 160 }}>
                  <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                    <div style={{ flex: 1 }}>
                      <ProgressBar v={(r[3] / r[2]) * 100} />
                    </div>
                    <span
                      className="mono dim"
                      style={{ width: 50, fontSize: 11, textAlign: 'right' }}
                    >
                      {r[3]} GB
                    </span>
                  </div>
                </td>
                <td className="mono">{r[4]}</td>
                <td className="mono dim">{r[5]}</td>
                <td className="mono dim">{r[6]}</td>
                <td style={{ textAlign: 'right' }}>
                  <button className="btn btn--sm btn--ghost">
                    <I n="more-horizontal" s={13} />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ─── NETWORKING ─────────────────────────────────────────────────
function PageNetworking() {
  return (
    <div>
      <div className="toolbar">
        <div className="lft">
          <div className="seg">
            <button className="on">Networks</button>
            <button>
              DHCP leases{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                14
              </span>
            </button>
            <button>DNS</button>
            <button>
              VPN{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                wg0
              </span>
            </button>
          </div>
        </div>
        <div className="rgt">
          <button className="btn btn--sm btn--primary">
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
          {NETWORKS.map((n) => (
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
                <button className="btn btn--sm btn--ghost">
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

// ─── FIREWALL ───────────────────────────────────────────────────
function PageFirewall({ onNav }) {
  return (
    <div>
      <div className="toolbar">
        <div className="lft">
          <div className="seg">
            <button className="on">
              Rules{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                {FW_RULES.length}
              </span>
            </button>
            <button>
              Aliases{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                6
              </span>
            </button>
            <button>
              Groups{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                3
              </span>
            </button>
          </div>
          <div className="seg">
            <button className="on">Datacenter</button>
            <button>Per-instance</button>
          </div>
        </div>
        <div className="rgt">
          <button
            className="btn btn--sm"
            onClick={() => onNav?.('firewall-editor')}
            title="Advanced editor with draggable ordering, hit counters, per-rule inspector"
          >
            <I n="sliders" s={13} /> Advanced editor
          </button>
          <button
            className="btn btn--sm"
            onClick={() =>
              window.toast?.info(
                'Test rule',
                'Simulating traffic against current ruleset \u2014 all pass',
              )
            }
          >
            <I n="shield-check" s={13} /> Test rule
          </button>
          <button
            className="btn btn--sm btn--primary"
            onClick={() => window.openModal?.('new-rule')}
          >
            <I n="plus" s={13} /> Add rule
          </button>
        </div>
      </div>
      <table className="tbl">
        <thead>
          <tr>
            <th style={{ width: 36 }}>#</th>
            <th style={{ width: 60 }}>On</th>
            <th>Direction</th>
            <th>Action</th>
            <th>Proto</th>
            <th>Port</th>
            <th>Source</th>
            <th>Destination</th>
            <th>Comment</th>
            <th style={{ textAlign: 'right' }}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {FW_RULES.map((r) => (
            <tr key={r.i}>
              <td className="mono dim">{r.i}</td>
              <td>
                <span
                  style={{
                    display: 'inline-flex',
                    width: 28,
                    height: 14,
                    borderRadius: 999,
                    background: r.on ? 'var(--h-accent)' : 'var(--h-border)',
                    position: 'relative',
                    verticalAlign: 'middle',
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
                      left: r.on ? 16 : 2,
                    }}
                  />
                </span>
              </td>
              <td>
                <span className="badge mono">{r.dir}</span>
              </td>
              <td
                style={{
                  color: r.act === 'accept' ? 'var(--h-success)' : 'var(--h-danger)',
                  fontFamily: 'var(--bzr-font-mono)',
                  textTransform: 'uppercase',
                  fontSize: 11,
                  letterSpacing: '0.08em',
                }}
              >
                {r.act}
              </td>
              <td className="mono">{r.proto}</td>
              <td className="mono">{r.port}</td>
              <td className="mono dim">{r.src}</td>
              <td className="mono">{r.dst}</td>
              <td className="dim">—</td>
              <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                <button className="btn btn--sm btn--ghost">
                  <I n="arrow-up" s={13} />
                </button>
                <button className="btn btn--sm btn--ghost">
                  <I n="arrow-down" s={13} />
                </button>
                <button className="btn btn--sm btn--ghost">
                  <I n="pencil" s={13} />
                </button>
                <button className="btn btn--sm btn--ghost">
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

// ─── IMAGES ─────────────────────────────────────────────────────
function PageImages() {
  const IMGS = [
    {
      n: 'debian-13-genericcloud-amd64.qcow2',
      type: 'cloud',
      size: '560 MB',
      from: 'debian.org',
      date: '2026-03-20',
      used: 4,
    },
    {
      n: 'ubuntu-24.04-server-amd64.iso',
      type: 'iso',
      size: '2.1 GB',
      from: 'ubuntu.com',
      date: '2026-04-10',
      used: 2,
    },
    {
      n: 'alpine-3.20-standard.iso',
      type: 'iso',
      size: '140 MB',
      from: 'alpinelinux.org',
      date: '2026-03-01',
      used: 1,
    },
    {
      n: 'win11-24H2-x64.iso',
      type: 'iso',
      size: '5.8 GB',
      from: 'microsoft.com',
      date: '2026-02-14',
      used: 1,
    },
    {
      n: 'talos-v1.7.0-amd64.iso',
      type: 'cloud',
      size: '80 MB',
      from: 'siderolabs.com',
      date: '2026-03-05',
      used: 3,
    },
  ];
  return (
    <div>
      <div className="toolbar">
        <div className="lft">
          <div className="seg">
            <button className="on">
              All{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                {IMGS.length}
              </span>
            </button>
            <button>ISO</button>
            <button>Cloud images</button>
            <button>VM templates</button>
          </div>
        </div>
        <div className="rgt">
          <button
            className="btn btn--sm"
            onClick={() =>
              window.toast?.info(
                'Download from URL',
                'Paste an image URL — hellingd will fetch and verify checksum',
              )
            }
          >
            <I n="globe" s={13} /> Download from URL
          </button>
          <button
            className="btn btn--sm btn--primary"
            onClick={() =>
              window.toast?.info(
                'Upload image',
                'Drop a .qcow2 / .raw / .iso to upload into pool-primary',
              )
            }
          >
            <I n="upload" s={13} /> Upload image
          </button>
        </div>
      </div>
      <table className="tbl">
        <thead>
          <tr>
            <th>Filename</th>
            <th>Kind</th>
            <th>Size</th>
            <th>Source</th>
            <th>Added</th>
            <th>Used by</th>
            <th style={{ textAlign: 'right' }}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {IMGS.map((i, idx) => (
            <tr key={idx}>
              <td className="mono" style={{ fontWeight: 600 }}>
                {i.n}
              </td>
              <td>
                <span className="badge mono">{i.type}</span>
              </td>
              <td className="mono">{i.size}</td>
              <td className="mono dim">{i.from}</td>
              <td className="mono dim">{i.date}</td>
              <td className="mono">{i.used} instances</td>
              <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                <button
                  className="btn btn--sm btn--ghost"
                  onClick={() => window.openModal?.('create-vm', { initialImage: i.name })}
                >
                  Create VM
                </button>
                <button
                  className="btn btn--sm btn--ghost"
                  title="Download"
                  onClick={() =>
                    window.toast?.info('Download', 'Streaming ' + i.name + ' to your browser')
                  }
                >
                  <I n="download" s={13} />
                </button>
                <button
                  className="btn btn--sm btn--ghost"
                  title="Delete"
                  onClick={() =>
                    window.openModal?.('confirm', {
                      title: 'Delete ' + i.name + '?',
                      body: 'This removes the image from pool-primary. Existing VMs will not be affected. Type the image name to confirm.',
                      danger: true,
                      confirmLabel: 'Delete',
                      confirmMatch: i.name,
                    })
                  }
                >
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

// ─── BACKUPS ────────────────────────────────────────────────────
function PageBackups() {
  useStore();
  const [tab, setTab] = useState('successful');
  const [q, setQ] = useState('');
  const BK = BACKUPS.filter((b) => {
    if (tab === 'successful' && (b.err || !b.ver)) return false;
    if (tab === 'failed' && !b.err) return false;
    if (tab === 'unverified' && (b.ver || b.err)) return false;
    if (q && !(b.inst + ' ' + b.ts).toLowerCase().includes(q.toLowerCase())) return false;
    return true;
  });
  return (
    <div>
      <div style={{ padding: '14px 20px 0' }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4,1fr)', gap: 10 }}>
          <div className="stat">
            <div className="label">Last 24h · successful</div>
            <div className="value">
              24<span className="suf">/ 25</span>
            </div>
          </div>
          <div className="stat">
            <div className="label">Verified</div>
            <div className="value" style={{ color: 'var(--h-success)' }}>
              92<span className="suf">%</span>
            </div>
          </div>
          <div className="stat">
            <div className="label">Storage used</div>
            <div className="value">
              287<span className="suf"> GB</span>
            </div>
          </div>
          <div className="stat">
            <div className="label">Next run</div>
            <div className="value" style={{ fontSize: 22 }}>
              tonight 02:00
            </div>
          </div>
        </div>
      </div>
      <div className="toolbar" style={{ marginTop: 14 }}>
        <div className="lft">
          <input
            className="input"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            style={{ width: 220, height: 28, fontSize: 12 }}
            placeholder="Search backups…"
          />
          <div className="seg">
            <button
              className={tab === 'successful' ? 'on' : ''}
              onClick={() => setTab('successful')}
            >
              Successful
            </button>
            <button className={tab === 'failed' ? 'on' : ''} onClick={() => setTab('failed')}>
              Failed{' '}
              <span className="mono" style={{ marginLeft: 4, color: 'var(--h-danger)' }}>
                1
              </span>
            </button>
            <button
              className={tab === 'unverified' ? 'on' : ''}
              onClick={() => setTab('unverified')}
            >
              Unverified
            </button>
          </div>
        </div>
        <div className="rgt">
          <button
            className="btn btn--sm"
            onClick={() =>
              window.toast?.info('Verifying backups', 'Running fsck on 14 snapshots — est. 4 min')
            }
          >
            <I n="shield-check" s={13} /> Verify all
          </button>
          <button
            className="btn btn--sm btn--primary"
            onClick={() =>
              window.toast?.success(
                'Scheduled backup triggered',
                'daily-all (14 instances) — started now instead of 02:00',
              )
            }
          >
            <I n="play" s={13} /> Run scheduled
          </button>
        </div>
      </div>
      <table className="tbl">
        <thead>
          <tr>
            <th>Timestamp</th>
            <th>Instance</th>
            <th>Mode</th>
            <th>Size</th>
            <th>Duration</th>
            <th>Verified</th>
            <th>Destination</th>
            <th style={{ textAlign: 'right' }}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {BK.length === 0 ? (
            <tr>
              <td colSpan={8} style={{ padding: 40 }}>
                <EmptyState
                  icon={tab === 'failed' ? 'shield-check' : 'archive'}
                  title={
                    tab === 'failed' ? 'No failed backups' : q ? 'No backups match' : 'Nothing here'
                  }
                  body={
                    tab === 'failed'
                      ? 'All scheduled backups completed successfully in this window. Nice.'
                      : q
                        ? `No backups match "${q}" in ${tab}. Try broadening.`
                        : `No ${tab} backups yet. Run a scheduled backup or wait for tonight’s window.`
                  }
                  action={
                    q ? (
                      <button className="btn btn--sm" onClick={() => setQ('')}>
                        Clear search
                      </button>
                    ) : null
                  }
                />
              </td>
            </tr>
          ) : (
            BK.map((b, i) => (
              <tr key={i}>
                <td className="mono">{b.ts}</td>
                <td className="mono" style={{ fontWeight: 600 }}>
                  {b.inst}
                </td>
                <td>{b.mode}</td>
                <td className="mono">{b.size}</td>
                <td className="mono dim">{b.dur}</td>
                <td>
                  {b.err ? (
                    <span style={{ color: 'var(--h-danger)' }}>✕ failed</span>
                  ) : b.ver ? (
                    <span style={{ color: 'var(--h-success)' }}>✓ ok</span>
                  ) : (
                    <span className="dim">pending</span>
                  )}
                </td>
                <td className="mono dim">default</td>
                <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                  <button className="btn btn--sm btn--ghost">Restore</button>
                  <button className="btn btn--sm btn--ghost">Verify</button>
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}

// ─── SCHEDULES ──────────────────────────────────────────────────
function PageSchedules() {
  return (
    <div>
      <div className="toolbar">
        <div className="lft">
          <div className="seg">
            <button className="on">All schedules</button>
            <button>Active</button>
            <button>Disabled</button>
          </div>
        </div>
        <div className="rgt">
          <button className="btn btn--sm btn--primary">
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
          {SCHEDULES.map((s) => (
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
                <button className="btn btn--sm btn--ghost">
                  <I n="play" s={13} /> Run now
                </button>
                <button className="btn btn--sm btn--ghost">
                  <I n="pencil" s={13} />
                </button>
                <button className="btn btn--sm btn--ghost">
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

// ─── TEMPLATES ──────────────────────────────────────────────────
function PageTemplates() {
  const CATS = [
    'all',
    'media',
    'dev',
    'monitor',
    'infra',
    'storage',
    'automation',
    'security',
    'productivity',
  ];
  const [cat, setCat] = useState('all');
  const rows = TEMPLATES.filter((t) => cat === 'all' || t.cat === cat);
  return (
    <div>
      <div className="toolbar">
        <div className="lft">
          <input
            className="input"
            style={{ width: 260, height: 28, fontSize: 12 }}
            placeholder="Search 186 templates…"
          />
          <div className="seg">
            {CATS.map((c) => (
              <button key={c} className={cat === c ? 'on' : ''} onClick={() => setCat(c)}>
                {c}{' '}
                {c === 'all' && (
                  <span className="mono dim" style={{ marginLeft: 4 }}>
                    {TEMPLATES.length}
                  </span>
                )}
              </button>
            ))}
          </div>
        </div>
        <div className="rgt">
          <button
            className="btn btn--sm"
            onClick={() =>
              window.toast?.info('From Compose', 'Paste a docker-compose.yml — dialog stub')
            }
          >
            <I n="git-branch" s={13} /> From Compose
          </button>
        </div>
      </div>
      <div
        style={{
          padding: 14,
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))',
          gap: 10,
        }}
      >
        {rows.map((t) => (
          <div key={t.name} className="card" style={{ cursor: 'pointer' }}>
            <div style={{ padding: 14, display: 'flex', gap: 12, alignItems: 'flex-start' }}>
              <div
                style={{
                  width: 44,
                  height: 44,
                  border: '1px solid var(--h-border)',
                  borderRadius: 'var(--h-radius)',
                  display: 'grid',
                  placeItems: 'center',
                  background: 'var(--h-surface-3)',
                  flex: '0 0 44px',
                }}
              >
                <I n={t.icon} s={22} color="var(--h-accent)" />
              </div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div className="mono" style={{ fontSize: 14, fontWeight: 600 }}>
                  {t.name}
                </div>
                <div
                  className="mono dim"
                  style={{
                    fontSize: 10,
                    marginTop: 2,
                    letterSpacing: '0.14em',
                    textTransform: 'uppercase',
                  }}
                >
                  {t.cat}
                </div>
                <div
                  style={{ fontSize: 12, color: 'var(--h-text-2)', marginTop: 8, lineHeight: 1.45 }}
                >
                  {t.desc}
                </div>
              </div>
            </div>
            <div
              style={{
                padding: '10px 14px',
                borderTop: '1px solid var(--h-border)',
                display: 'flex',
                gap: 6,
                justifyContent: 'space-between',
                alignItems: 'center',
              }}
            >
              <span className="mono dim" style={{ fontSize: 10 }}>
                verified · official
              </span>
              <button className="btn btn--sm btn--primary">Deploy →</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// ─── BMC ────────────────────────────────────────────────────────
function PageBMC() {
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
            {[
              ['node-1', '192.168.2.10 · iDRAC9', 'on', '52°C', '2100 RPM', '21°C', 180],
              ['node-2', '192.168.2.20 · Redfish', 'on', '44°C', '1400 RPM', '22°C', 65],
              ['node-3', '192.168.2.30 · iLO5', 'off', '—', '—', '20°C', 0],
            ].map((r, i) => (
              <tr key={i}>
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
                    className="btn btn--sm btn--ghost"
                    title="Power cycle"
                    onClick={() =>
                      window.openModal?.('confirm', {
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
                    className="btn btn--sm btn--ghost"
                    title="Graceful restart"
                    onClick={() =>
                      window.openModal?.('confirm', {
                        title: 'Restart ' + r[0] + '?',
                        body: 'Drains VMs to other nodes, then reboots. ~4 min.',
                        confirmLabel: 'Restart',
                      })
                    }
                  >
                    <I n="rotate-cw" s={13} />
                  </button>
                  <button
                    className="btn btn--sm btn--ghost"
                    onClick={() =>
                      window.toast?.info(
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

// ─── CLUSTER ────────────────────────────────────────────────────
function PageCluster() {
  return (
    <div style={{ padding: 20, display: 'grid', gap: 14 }}>
      <div>
        <div className="eyebrow">CLUSTER / HELLING-HOME / QUORUM OK</div>
        <h1 className="stencil" style={{ fontSize: 22, margin: '6px 0 0' }}>
          Cluster
        </h1>
        <div className="muted" style={{ fontSize: 13, marginTop: 2 }}>
          3 nodes · quorum 2 of 3 · ceph not configured
        </div>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10 }}>
        {NODES.map((n) => (
          <div key={n.id} className="card">
            <header>
              <div>
                <div className="mono" style={{ fontSize: 14, fontWeight: 600 }}>
                  {n.name}
                </div>
                <div className="mono dim" style={{ fontSize: 10, marginTop: 2 }}>
                  {n.model}
                </div>
              </div>
              {STATUS_BADGE[n.state]}
            </header>
            <div style={{ padding: 14 }}>
              <dl className="desc" style={{ gridTemplateColumns: '90px 1fr' }}>
                <dt>Quorum</dt>
                <dd>
                  <span style={{ color: 'var(--h-success)' }}>● voting</span>
                </dd>
                <dt>CPU</dt>
                <dd>
                  <ProgressBar v={n.cpuPct} />
                </dd>
                <dt>RAM</dt>
                <dd>
                  <ProgressBar v={n.ramPct} />
                </dd>
                <dt>Uptime</dt>
                <dd className="mono">14d 3h</dd>
                <dt>Kernel</dt>
                <dd className="mono">6.8.0-helling</dd>
                <dt>Helling</dt>
                <dd className="mono">v0.1.0</dd>
              </dl>
            </div>
          </div>
        ))}
      </div>
      <div className="card">
        <header>
          <span className="title">High availability</span>
          <button className="btn btn--sm">
            <I n="plus" s={13} /> Add HA group
          </button>
        </header>
        <table className="tbl">
          <thead>
            <tr>
              <th>Group</th>
              <th>Instance</th>
              <th>State</th>
              <th>Preferred</th>
              <th>Current</th>
              <th>Max restart</th>
              <th>Failover policy</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td className="mono">web-ha</td>
              <td className="mono">vm-web-1</td>
              <td>{STATUS_BADGE.running}</td>
              <td className="mono">node-1</td>
              <td className="mono">node-1</td>
              <td className="mono">3</td>
              <td>migrate</td>
            </tr>
            <tr>
              <td className="mono">db-ha</td>
              <td className="mono">vm-db-1</td>
              <td>{STATUS_BADGE.running}</td>
              <td className="mono">node-1</td>
              <td className="mono">node-1</td>
              <td className="mono">1</td>
              <td>migrate</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ─── SETTINGS ───────────────────────────────────────────────────
function PageSettings() {
  const [tab, setTab] = useState('general');
  const TABS = [
    'general',
    'access',
    'networking',
    'storage',
    'updates',
    'notifications',
    'integrations',
    'advanced',
  ];
  return (
    <div style={{ display: 'flex', height: '100%' }}>
      <aside
        style={{
          width: 200,
          flex: '0 0 200px',
          borderRight: '1px solid var(--h-border)',
          background: 'var(--h-surface)',
          padding: '14px 0',
        }}
      >
        {TABS.map((s) => (
          <div
            key={s}
            onClick={() => setTab(s)}
            style={{
              padding: '8px 16px',
              color: tab === s ? 'var(--h-accent)' : 'var(--h-text-2)',
              borderLeft: '2px solid ' + (tab === s ? 'var(--h-accent)' : 'transparent'),
              fontSize: 13,
              cursor: 'pointer',
              background: tab === s ? 'rgba(198,255,36,0.05)' : 'transparent',
              textTransform: 'capitalize',
            }}
          >
            {s}
          </div>
        ))}
      </aside>
      <div style={{ flex: 1, overflow: 'auto', padding: 20 }}>
        <div className="eyebrow">SETTINGS / {tab.toUpperCase()}</div>
        <h2
          className="stencil"
          style={{ fontSize: 22, margin: '6px 0 18px', textTransform: 'capitalize' }}
        >
          {tab}
        </h2>
        {tab === 'general' && (
          <div style={{ display: 'grid', gap: 14, maxWidth: 720 }}>
            <SettingField label="Cluster name" hint="Shown in the top bar and in exports.">
              <input className="input" defaultValue="helling-home" style={{ width: 300 }} />
            </SettingField>
            <SettingField label="Time zone">
              <select className="input" style={{ width: 300 }}>
                <option>Europe/London</option>
                <option>America/New_York</option>
                <option>UTC</option>
              </select>
            </SettingField>
            <SettingField label="Default storage pool">
              <select className="input" style={{ width: 300 }}>
                <option>default</option>
                <option>fast-pool</option>
                <option>archive</option>
              </select>
            </SettingField>
            <SettingField label="Theme" hint="Helling is dark-first by design.">
              <div className="seg">
                <button className="on">Dark</button>
                <button>Light</button>
                <button>System</button>
              </div>
            </SettingField>
            <SettingField label="Density">
              <div className="seg">
                <button>Comfortable</button>
                <button className="on">Compact</button>
                <button>Dense</button>
              </div>
            </SettingField>
            <SettingField label="Telemetry" hint="Send anonymous crash reports. Never PII.">
              <Toggle on={true} />
            </SettingField>
          </div>
        )}
        {tab === 'updates' && (
          <div style={{ maxWidth: 720, display: 'grid', gap: 14 }}>
            <div className="alert alert--info">
              <I n="info" s={14} />
              <div style={{ flex: 1 }}>
                <div>
                  <b>Helling v0.1.1 is available</b> — security patch for CVE-2026-0012.
                </div>
                <div className="mono dim" style={{ fontSize: 11, marginTop: 2 }}>
                  released 2026-04-18 · 42 MB
                </div>
              </div>
              <button className="btn btn--sm btn--primary">Install & restart</button>
            </div>
            <SettingField label="Current version">
              <code className="mono">v0.1.0 (d3a441c)</code>
            </SettingField>
            <SettingField label="Channel">
              <div className="seg">
                <button className="on">stable</button>
                <button>beta</button>
                <button>edge</button>
              </div>
            </SettingField>
            <SettingField label="Automatic updates" hint="Security patches install automatically.">
              <Toggle on={true} />
            </SettingField>
          </div>
        )}
        {tab !== 'general' && tab !== 'updates' && (
          <div
            className="card"
            style={{ padding: '24px 20px', color: 'var(--h-text-2)', textAlign: 'center' }}
          >
            <div
              className="mono dim"
              style={{ letterSpacing: '0.16em', fontSize: 11, textTransform: 'uppercase' }}
            >
              {tab}
            </div>
            <div style={{ marginTop: 8 }}>Settings for this section would appear here.</div>
          </div>
        )}
      </div>
    </div>
  );
}

function SettingField({ label, hint, children }) {
  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: '240px 1fr',
        alignItems: 'flex-start',
        gap: 16,
        borderBottom: '1px dashed rgba(61,61,61,0.4)',
        paddingBottom: 14,
      }}
    >
      <div>
        <div style={{ fontSize: 13, color: 'var(--h-text)', fontWeight: 500 }}>{label}</div>
        {hint && (
          <div className="mono dim" style={{ fontSize: 11, marginTop: 4 }}>
            {hint}
          </div>
        )}
      </div>
      <div>{children}</div>
    </div>
  );
}

function Toggle({ on }) {
  return (
    <span
      style={{
        display: 'inline-flex',
        width: 36,
        height: 18,
        borderRadius: 999,
        background: on ? 'var(--h-accent)' : 'var(--h-border)',
        position: 'relative',
        cursor: 'pointer',
      }}
    >
      <span
        style={{
          position: 'absolute',
          width: 14,
          height: 14,
          borderRadius: '50%',
          background: '#000',
          top: 2,
          left: on ? 20 : 2,
        }}
      />
    </span>
  );
}

Object.assign(window, {
  PageDashboard,
  PageInstances,
  PageInstanceDetail,
  PageContainers,
  PageKubernetes,
  PageStorage,
  PageNetworking,
  PageFirewall,
  PageImages,
  PageBackups,
  PageSchedules,
  PageTemplates,
  PageBMC,
  PageCluster,
  PageSettings,
});
