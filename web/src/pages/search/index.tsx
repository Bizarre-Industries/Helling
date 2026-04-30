// Helling WebUI — Search results page (extracted from pages2.jsx during Phase 2A).
//
// Static RESULTS until Phase 3 swaps in a real cross-resource search query
// hook. Body unchanged from the legacy page.

import { useState } from 'react';

import { I } from '../../primitives/icon';

type SearchResult = {
  type: 'instance' | 'backup' | 'storage' | 'firewall' | 'audit' | 'alert' | 'doc';
  name: string;
  desc: string;
  tags?: string[];
};

type Props = {
  query?: string;
  onNav: (target: string) => void;
};

const RESULTS: SearchResult[] = [
  {
    type: 'instance',
    name: 'db-primary',
    desc: 'KVM · node-1 · running · 8 vCPU · 32 GB',
    tags: ['prod', 'postgres'],
  },
  {
    type: 'instance',
    name: 'db-replica-eu',
    desc: 'KVM · node-2 · running · lag 22s',
    tags: ['prod', 'postgres', 'replica'],
  },
  {
    type: 'instance',
    name: 'db-replica-us',
    desc: 'KVM · node-3 · running · lag 4s',
    tags: ['prod', 'postgres', 'replica'],
  },
  {
    type: 'backup',
    name: 'db-primary · nightly-2026-03-20',
    desc: '612 MB · zstd-9 · retention 30d',
  },
  {
    type: 'backup',
    name: 'db-primary · nightly-2026-03-19',
    desc: '608 MB · zstd-9 · retention 30d',
  },
  {
    type: 'storage',
    name: 'nvme-fast',
    desc: 'Pool · ZFS stripe · 612 GB free · hosts db-primary',
  },
  { type: 'firewall', name: 'Postgres (disabled)', desc: 'Rule #6 · allow 10.0.0.0/24 → 5432' },
  {
    type: 'audit',
    name: 'alice@helling stopped db-replica-eu',
    desc: '2026-03-21 14:22 · from 10.0.0.14',
  },
  { type: 'alert', name: 'Replica lag > 30s · db-replica-eu', desc: 'Firing 22m · warn' },
  { type: 'doc', name: 'How database backups work', desc: 'docs.helling.io/backups/db' },
];

const TYPE_ICON: Record<SearchResult['type'], string> = {
  instance: 'server',
  backup: 'archive',
  storage: 'hard-drive',
  firewall: 'shield',
  audit: 'history',
  alert: 'bell',
  doc: 'book-open',
};

const TYPE_COLOR: Record<SearchResult['type'], string> = {
  instance: 'var(--h-accent)',
  backup: '#8bffd4',
  storage: '#8a9bff',
  firewall: '#ff8aa9',
  audit: 'var(--h-text-3)',
  alert: 'var(--h-warn)',
  doc: 'var(--h-text-2)',
};

export default function PageSearch({ query = 'db', onNav }: Props) {
  const [filter, setFilter] = useState<'all' | SearchResult['type']>('all');
  const filtered = filter === 'all' ? RESULTS : RESULTS.filter((r) => r.type === filter);
  const filterTabs: ('all' | SearchResult['type'])[] = [
    'all',
    'instance',
    'backup',
    'storage',
    'firewall',
    'audit',
    'alert',
    'doc',
  ];
  return (
    <div style={{ maxWidth: 1100, padding: '18px 20px' }}>
      <div style={{ marginBottom: 14 }}>
        <div className="eyebrow">SEARCH</div>
        <h1 className="stencil" style={{ fontSize: 26, margin: '4px 0 0' }}>
          Results for "
          <span className="mono" style={{ color: 'var(--h-accent)' }}>
            {query}
          </span>
          "
        </h1>
        <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>
          {RESULTS.length} matches · ranked by relevance
        </div>
      </div>

      <div style={{ display: 'flex', gap: 6, marginBottom: 14 }}>
        {filterTabs.map((t) => {
          const count = t === 'all' ? RESULTS.length : RESULTS.filter((r) => r.type === t).length;
          return (
            <span
              key={t}
              className={'chip ' + (filter === t ? 'chip--on' : '')}
              onClick={() => setFilter(t)}
              style={{ cursor: 'pointer' }}
            >
              {t}{' '}
              <span className="mono dim" style={{ marginLeft: 4 }}>
                {count}
              </span>
            </span>
          );
        })}
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
        {filtered.map((r, i) => (
          <div
            key={i}
            className="card"
            style={{
              padding: '12px 14px',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: 12,
            }}
            onClick={() => onNav(r.type === 'instance' ? 'instance:' + r.name : r.type)}
          >
            <div
              style={{
                width: 32,
                height: 32,
                background: 'var(--h-bg-2)',
                border: '1px solid var(--h-border)',
                borderRadius: 4,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: TYPE_COLOR[r.type],
              }}
            >
              <I n={TYPE_ICON[r.type]} s={16} />
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 2 }}>
                <span className="mono" style={{ fontWeight: 600, fontSize: 13 }}>
                  {r.name}
                </span>
                <span
                  className="badge"
                  style={{
                    fontSize: 9,
                    color: TYPE_COLOR[r.type],
                    borderColor: TYPE_COLOR[r.type],
                  }}
                >
                  {r.type}
                </span>
                {r.tags?.map((t) => (
                  <span key={t} className="chip mono" style={{ fontSize: 9 }}>
                    {t}
                  </span>
                ))}
              </div>
              <div className="muted" style={{ fontSize: 12 }}>
                {r.desc}
              </div>
            </div>
            <I n="arrow-right" s={14} style={{ color: 'var(--h-text-3)' }} />
          </div>
        ))}
      </div>
    </div>
  );
}
