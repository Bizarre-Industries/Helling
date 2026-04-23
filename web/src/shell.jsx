import * as LucideIcons from 'lucide-react';
/* Helling WebUI — shared data & small primitives */
/* eslint-disable no-unused-vars */
import React, { useState, useEffect, useRef, useMemo, useCallback } from 'react';

// ─── LIVE TICK — 1s heartbeat shared across the app ──────────
const __tickListeners = new Set();
let __tickStarted = false;
function __startTick() {
  if (__tickStarted) return;
  __tickStarted = true;
  setInterval(() => {
    const t = Date.now();
    __tickListeners.forEach((fn) => {
      try {
        fn(t);
      } catch (e) {}
    });
  }, 1000);
}
function useTick(ms = 1000) {
  const [t, setT] = useState(0);
  useEffect(() => {
    __startTick();
    const fn = () => setT((x) => x + 1);
    __tickListeners.add(fn);
    return () => __tickListeners.delete(fn);
  }, []);
  return t;
}
// deterministic-ish jitter: walk a value by ±amplitude, clamped
function __jitter(base, amp, seed) {
  const n = (Math.sin((seed || 0) * 99.7 + Date.now() / 4000) + 1) / 2;
  return Math.max(0, Math.min(100, base + (n - 0.5) * 2 * amp));
}
window.useTick = useTick;
window.__jitter = __jitter;

// ─── MUTABLE STORE HELPERS ────────────────────────────────────
// Mutates shared mock arrays in-place; listeners force a re-render.
const __storeListeners = new Set();
function __notifyStore() {
  __storeListeners.forEach((fn) => {
    try {
      fn();
    } catch (e) {}
  });
}
function useStore() {
  const [, setN] = useState(0);
  useEffect(() => {
    const fn = () => setN((x) => x + 1);
    __storeListeners.add(fn);
    return () => __storeListeners.delete(fn);
  }, []);
}
let __taskSeq = 8820;
function pushTask({ op, target, user = 'admin', status = 'running', progress = 0, err }) {
  const id = 'T-' + __taskSeq++;
  const now = new Date();
  const started =
    String(now.getHours()).padStart(2, '0') +
    ':' +
    String(now.getMinutes()).padStart(2, '0') +
    ':' +
    String(now.getSeconds()).padStart(2, '0');
  TASKS.unshift({ id, op, target, user, status, progress, started, err });
  if (TASKS.length > 20) TASKS.length = 20;
  __notifyStore();
  return id;
}
function instanceAction(name, action) {
  const inst = INSTANCES.find((i) => i.name === name);
  if (!inst) return;
  const prev = inst.status;
  if (action === 'start' || action === 'restart') {
    inst.status = 'running';
    if (action === 'restart') {
      inst.uptime = '0m';
    }
    pushTask({ op: 'instance.' + action, target: name });
    window.toast?.success(
      (action === 'start' ? 'Starting' : 'Restarting') + ' ' + name,
      'Queued — tracking in task log',
    );
  } else if (action === 'stop' || action === 'pause') {
    const prevCpu = inst.cpuPct,
      prevRam = inst.ramPct,
      prevUp = inst.uptime;
    inst.status = 'stopped';
    inst.cpuPct = 0;
    inst.ramPct = action === 'pause' ? inst.ramPct : 0;
    pushTask({ op: 'instance.' + action, target: name });
    window.toast?.info((action === 'pause' ? 'Pausing' : 'Stopping') + ' ' + name, 'Task queued', {
      ttl: 6000,
      action: {
        label: 'Undo',
        run: () => {
          inst.status = prev;
          inst.cpuPct = prevCpu;
          inst.ramPct = prevRam;
          inst.uptime = prevUp;
          pushTask({ op: 'instance.start', target: name });
          window.toast?.success('Restarted ' + name, 'Action reverted');
        },
      },
    });
  } else if (action === 'delete') {
    const idx = INSTANCES.indexOf(inst);
    const snap = { ...inst };
    INSTANCES.splice(idx, 1);
    pushTask({ op: 'instance.delete', target: name });
    window.toast?.warn('Deleted ' + name, 'Disks marked for garbage collection', {
      ttl: 8000,
      action: {
        label: 'Undo',
        run: () => {
          INSTANCES.splice(idx, 0, snap);
          __notifyStore();
          window.toast?.success('Restored ' + name, 'Instance and its disks recovered');
        },
      },
    });
  } else if (action === 'snapshot') {
    const stamp = new Date();
    const pad = (n) => String(n).padStart(2, '0');
    SNAPSHOTS.unshift({
      name: 'snap-' + pad(stamp.getHours()) + pad(stamp.getMinutes()) + pad(stamp.getSeconds()),
      date: '2026-04-24 ' + pad(stamp.getHours()) + ':' + pad(stamp.getMinutes()),
      ram: true,
      size: (Math.random() * 2 + 0.8).toFixed(1) + ' GB',
      desc: 'manual (' + name + ')',
    });
    pushTask({ op: 'instance.snapshot', target: name });
    window.toast?.info('Snapshotting ' + name, 'Consistent RAM+disk snapshot in progress');
  } else if (action === 'backup') {
    const stamp = new Date();
    const pad = (n) => String(n).padStart(2, '0');
    BACKUPS.unshift({
      ts: '2026-04-24 ' + pad(stamp.getHours()) + ':' + pad(stamp.getMinutes()),
      inst: name,
      size: (Math.random() * 4 + 0.5).toFixed(1) + ' GB',
      dur: '0m 0' + Math.floor(Math.random() * 9) + 's',
      ver: false,
      mode: 'manual',
    });
    pushTask({ op: 'backup', target: name });
    window.toast?.info('Backup started for ' + name, 'Writing to pool backup-primary');
  }
  __notifyStore();
}
window.pushTask = pushTask;
window.instanceAction = instanceAction;
window.useStore = useStore;

// ─── LIVE TICK ────────────────────────────────────────────────
// Single setInterval that: drifts running-instance metrics, advances in-flight tasks,
// completes them, and updates footer latency. All mutations trigger __notifyStore().
let __latency = 18;
window.__getLatency = () => __latency;
setInterval(() => {
  let touched = false;
  // instance metric drift
  INSTANCES.forEach((i) => {
    if (i.status !== 'running') return;
    const drift = (pct) => Math.max(1, Math.min(99, pct + Math.round((Math.random() - 0.5) * 6)));
    i.cpuPct = drift(i.cpuPct);
    i.ramPct = Math.max(1, Math.min(99, i.ramPct + Math.round((Math.random() - 0.5) * 2)));
    touched = true;
  });
  // task progress
  TASKS.forEach((t) => {
    if (t.status !== 'running') return;
    t.progress = Math.min(100, (t.progress || 0) + 2 + Math.floor(Math.random() * 8));
    if (t.progress >= 100) {
      t.progress = 100;
      t.status = 'success';
      // side-effect hooks
      if (t.op === 'instance.stop') {
        const inst = INSTANCES.find((x) => x.name === t.target);
        if (inst) {
          inst.status = 'stopped';
          inst.cpuPct = 0;
          inst.ramPct = 0;
          inst.uptime = '—';
        }
      }
    }
    touched = true;
  });
  // footer latency drift
  const next = Math.max(8, Math.min(42, __latency + Math.round((Math.random() - 0.5) * 6)));
  if (next !== __latency) {
    __latency = next;
    touched = true;
  }
  if (touched) __notifyStore();
}, 1500);

// ─── ICONS (lucide-react; swap name lookup to PascalCase component) ──
const __iconCache = {};
function __resolveIcon(n) {
  if (__iconCache[n] !== undefined) return __iconCache[n];
  const pascal = n
    .split('-')
    .map((p) => (p ? p[0].toUpperCase() + p.slice(1) : ''))
    .join('');
  const Comp = LucideIcons[pascal] || LucideIcons[n] || null;
  __iconCache[n] = Comp;
  return Comp;
}
const I = ({ n, s = 14, style, color }) => {
  const Comp = __resolveIcon(n);
  if (!Comp) {
    return <span style={{ display: 'inline-block', width: s, height: s, ...style }} />;
  }
  return (
    <Comp
      size={s}
      strokeWidth={1.75}
      style={{ display: 'inline-block', flexShrink: 0, color, ...style }}
    />
  );
};

// ─── MOCK DATA ────────────────────────────────────────────────
const NODES = [
  {
    id: 'node-1',
    name: 'node-1',
    model: 'Dell R730',
    state: 'online',
    cpuPct: 42,
    ramPct: 61,
    watts: 180,
  },
  {
    id: 'node-2',
    name: 'node-2',
    model: 'Minisforum MS-01',
    state: 'online',
    cpuPct: 18,
    ramPct: 34,
    watts: 65,
  },
  {
    id: 'node-3',
    name: 'node-3',
    model: 'HPE ProLiant',
    state: 'sleeping',
    cpuPct: 0,
    ramPct: 0,
    watts: 0,
  },
];

const INSTANCES = [
  {
    name: 'vm-web-1',
    type: 'VM',
    node: 'node-1',
    status: 'running',
    os: 'Debian 13',
    cores: 4,
    ram: 8,
    cpuPct: 45,
    ramPct: 62,
    ip: '10.0.0.50',
    mac: '00:16:3e:1a:2b:3c',
    uptime: '14d 3h',
    tags: ['prod', 'web'],
    health: 92,
    backupAge: '2h',
  },
  {
    name: 'vm-db-1',
    type: 'VM',
    node: 'node-1',
    status: 'running',
    os: 'Debian 13',
    cores: 8,
    ram: 32,
    cpuPct: 78,
    ramPct: 81,
    ip: '10.0.0.51',
    mac: '00:16:3e:1a:2b:3d',
    uptime: '21d 8h',
    tags: ['prod', 'db'],
    health: 88,
    backupAge: '6h',
  },
  {
    name: 'vm-windows-dev',
    type: 'VM',
    node: 'node-2',
    status: 'running',
    os: 'Windows 11',
    cores: 8,
    ram: 16,
    cpuPct: 23,
    ramPct: 55,
    ip: '10.0.0.60',
    mac: '00:16:3e:2a:ff:10',
    uptime: '2d 1h',
    tags: ['dev', 'gpu'],
    health: 74,
    backupAge: '3d',
  },
  {
    name: 'vm-build',
    type: 'VM',
    node: 'node-2',
    status: 'running',
    os: 'Ubuntu 24.04',
    cores: 4,
    ram: 8,
    cpuPct: 65,
    ramPct: 72,
    ip: '10.0.0.61',
    mac: '00:16:3e:2a:ff:11',
    uptime: '5d',
    tags: ['ci'],
    health: 80,
    backupAge: '12h',
  },
  {
    name: 'ct-dns',
    type: 'CT',
    node: 'node-1',
    status: 'running',
    os: 'Alpine 3.20',
    cores: 1,
    ram: 0.5,
    cpuPct: 2,
    ramPct: 18,
    ip: '10.0.0.2',
    mac: '00:16:3e:33:01:02',
    uptime: '45d',
    tags: ['infra'],
    health: 96,
    backupAge: '1d',
  },
  {
    name: 'ct-gitea',
    type: 'CT',
    node: 'node-1',
    status: 'running',
    os: 'Debian 13',
    cores: 2,
    ram: 2,
    cpuPct: 9,
    ramPct: 41,
    ip: '10.0.0.3',
    mac: '00:16:3e:33:01:03',
    uptime: '45d',
    tags: ['dev'],
    health: 91,
    backupAge: '1d',
  },
  {
    name: 'vm-old',
    type: 'VM',
    node: 'node-2',
    status: 'stopped',
    os: 'CentOS 7',
    cores: 2,
    ram: 4,
    cpuPct: 0,
    ramPct: 0,
    ip: '—',
    mac: '00:16:3e:41:99:aa',
    uptime: '—',
    tags: ['legacy'],
    health: 31,
    backupAge: 'never',
  },
  {
    name: 'ct-legacy',
    type: 'CT',
    node: 'node-2',
    status: 'stopped',
    os: 'Ubuntu 20.04',
    cores: 1,
    ram: 1,
    cpuPct: 0,
    ramPct: 0,
    ip: '—',
    mac: '00:16:3e:41:99:ab',
    uptime: '—',
    tags: ['legacy'],
    health: 24,
    backupAge: '12d',
  },
];

const CONTAINERS = [
  {
    id: 'c_a91f',
    name: 'jellyfin',
    image: 'linuxserver/jellyfin:latest',
    ports: '8096:8096',
    cpu: 12,
    ram: 28,
    status: 'running',
    health: 'healthy',
    stack: 'media',
    update: true,
  },
  {
    id: 'c_b12a',
    name: 'gitea',
    image: 'gitea/gitea:1.22',
    ports: '3000:3000',
    cpu: 4,
    ram: 18,
    status: 'running',
    health: 'healthy',
    stack: 'dev',
    update: false,
  },
  {
    id: 'c_c44b',
    name: 'uptime-kuma',
    image: 'louislam/uptime-kuma:1',
    ports: '3001:3001',
    cpu: 1,
    ram: 9,
    status: 'running',
    health: 'healthy',
    stack: 'monitor',
    update: false,
  },
  {
    id: 'c_d71f',
    name: 'pihole',
    image: 'pihole/pihole:latest',
    ports: '53:53/udp',
    cpu: 3,
    ram: 12,
    status: 'running',
    health: 'healthy',
    stack: 'infra',
    update: true,
  },
  {
    id: 'c_e0ac',
    name: 'grafana',
    image: 'grafana/grafana:11.3',
    ports: '3002:3000',
    cpu: 6,
    ram: 15,
    status: 'running',
    health: 'degraded',
    stack: 'monitor',
    update: false,
  },
  {
    id: 'c_f33d',
    name: 'postgres',
    image: 'postgres:16',
    ports: '5432:5432',
    cpu: 9,
    ram: 44,
    status: 'running',
    health: 'healthy',
    stack: 'dev',
    update: false,
  },
  {
    id: 'c_9910',
    name: 'redis',
    image: 'redis:7-alpine',
    ports: '6379:6379',
    cpu: 1,
    ram: 3,
    status: 'running',
    health: 'healthy',
    stack: 'dev',
    update: false,
  },
  {
    id: 'c_2241',
    name: 'n8n',
    image: 'n8nio/n8n:1.70',
    ports: '5678:5678',
    cpu: 2,
    ram: 11,
    status: 'exited',
    health: '—',
    stack: 'automation',
    update: false,
  },
];

const CLUSTERS = [
  {
    name: 'prod-k8s',
    flavor: 'k3s',
    nodes: 3,
    workers: 3,
    version: 'v1.31.2',
    status: 'healthy',
    cpu: 45,
    ram: 62,
  },
  {
    name: 'dev-k8s',
    flavor: 'k3s',
    nodes: 1,
    workers: 1,
    version: 'v1.31.0',
    status: 'healthy',
    cpu: 12,
    ram: 28,
  },
  {
    name: 'edge',
    flavor: 'k3s',
    nodes: 2,
    workers: 1,
    version: 'v1.30.5',
    status: 'degraded',
    cpu: 68,
    ram: 81,
  },
];

let __alertSeq = 200;
const ALERTS = [
  {
    id: 'A-184',
    sev: 'danger',
    t: 'node-3 last heartbeat 14m ago',
    body: 'Sleeping as configured. Wake-on-LAN pending.',
    time: '14m ago',
    read: false,
    target: 'node-3',
    nav: () => window.__nav?.('cluster'),
  },
  {
    id: 'A-183',
    sev: 'warn',
    t: 'Storage pool "default" at 72%',
    body: 'Consider expanding the pool before reaching 85% threshold.',
    time: 'just now',
    read: false,
    target: 'pool/default',
    nav: () => window.__nav?.('storage'),
  },
  {
    id: 'A-182',
    sev: 'warn',
    t: 'Backup failed: db-primary · nightly',
    body: 'exit 2 — fsck on checkpoint returned CRC mismatch',
    time: '1h 12m',
    read: false,
    target: 'db-primary',
    nav: () => window.__nav?.('backups'),
  },
  {
    id: 'A-181',
    sev: 'warn',
    t: 'Replica lag > 30s',
    body: 'db-replica-eu is 58s behind db-primary',
    time: '22m ago',
    read: false,
    target: 'db-replica-eu',
    nav: () => window.__nav?.('alerts'),
  },
  {
    id: 'A-180',
    sev: 'warn',
    t: 'vm-old has no backup',
    body: '95 days since last snapshot',
    time: '2h ago',
    read: true,
    target: 'vm-old',
    nav: () => window.__nav?.('instance:vm-old'),
  },
  {
    id: 'A-179',
    sev: 'info',
    t: 'Update available',
    body: 'Helling 0.1.3 → 0.1.4 — security + perf fixes',
    time: '1h ago',
    read: true,
    target: 'system',
    nav: () => window.__nav?.('settings'),
  },
];
function pushAlert({ sev, t, body, target }) {
  ALERTS.unshift({ id: 'A-' + ++__alertSeq, sev, t, body, time: 'just now', read: false, target });
  if (ALERTS.length > 40) ALERTS.length = 40;
  __notifyStore();
}
window.pushAlert = pushAlert;

const TASKS = [
  {
    id: 'T-8814',
    op: 'snapshot',
    target: 'vm-web-1',
    user: 'admin',
    status: 'running',
    progress: 62,
    started: '14:22:03',
  },
  {
    id: 'T-8813',
    op: 'backup-verify',
    target: 'vm-db-1',
    user: 'system',
    status: 'success',
    progress: 100,
    started: '14:19:45',
  },
  {
    id: 'T-8812',
    op: 'container-pull',
    target: 'jellyfin',
    user: 'admin',
    status: 'success',
    progress: 100,
    started: '14:15:02',
  },
  {
    id: 'T-8811',
    op: 'migrate',
    target: 'vm-build',
    user: 'admin',
    status: 'success',
    progress: 100,
    started: '13:58:11',
  },
  {
    id: 'T-8810',
    op: 'instance-start',
    target: 'ct-gitea',
    user: 'admin',
    status: 'success',
    progress: 100,
    started: '13:40:00',
  },
  {
    id: 'T-8809',
    op: 'backup',
    target: 'vm-old',
    user: 'system',
    status: 'failed',
    progress: 40,
    started: '13:12:00',
    err: 'storage pool full',
  },
];

const AUDIT = [
  {
    ts: '2026-04-22 14:22:03',
    user: 'admin',
    action: 'instance.snapshot',
    target: 'vm-web-1',
    status: 'ok',
    ip: '192.168.1.10',
  },
  {
    ts: '2026-04-22 14:10:51',
    user: 'admin',
    action: 'instance.start',
    target: 'ct-gitea',
    status: 'ok',
    ip: '192.168.1.10',
  },
  {
    ts: '2026-04-22 13:55:22',
    user: 'alice',
    action: 'container.restart',
    target: 'grafana',
    status: 'ok',
    ip: '100.64.0.3',
  },
  {
    ts: '2026-04-22 13:12:00',
    user: 'system',
    action: 'backup.run',
    target: 'vm-old',
    status: 'fail',
    ip: '127.0.0.1',
  },
  {
    ts: '2026-04-22 12:08:15',
    user: 'admin',
    action: 'user.2fa.enable',
    target: 'alice',
    status: 'ok',
    ip: '192.168.1.10',
  },
  {
    ts: '2026-04-22 11:44:01',
    user: 'admin',
    action: 'firewall.rule.add',
    target: 'vm-web-1',
    status: 'ok',
    ip: '192.168.1.10',
  },
  {
    ts: '2026-04-22 10:02:09',
    user: 'deploy',
    action: 'instance.create',
    target: 'vm-build',
    status: 'ok',
    ip: '10.0.0.5',
  },
  {
    ts: '2026-04-22 09:15:00',
    user: 'admin',
    action: 'schedule.create',
    target: 'daily-bk',
    status: 'ok',
    ip: '192.168.1.10',
  },
  {
    ts: '2026-04-22 08:40:42',
    user: 'admin',
    action: 'auth.login',
    target: '—',
    status: 'ok',
    ip: '192.168.1.10',
  },
  {
    ts: '2026-04-22 08:00:00',
    user: 'system',
    action: 'system.startup',
    target: 'hellingd',
    status: 'ok',
    ip: '127.0.0.1',
  },
];

const POOLS = [
  { name: 'default', type: 'zfs', size: 2000, used: 1440, inst: 8 },
  { name: 'fast-pool', type: 'nvme', size: 1000, used: 450, inst: 4 },
  { name: 'archive', type: 'btrfs', size: 8000, used: 4600, inst: 2 },
];

const NETWORKS = [
  { name: 'bridge0', type: 'bridge', cidr: '10.0.0.0/24', dhcp: true, insts: 6 },
  { name: 'dev-net', type: 'bridge', cidr: '10.99.0.0/24', dhcp: true, insts: 3 },
  { name: 'dmz', type: 'ovn', cidr: '172.20.0.0/24', dhcp: false, insts: 1 },
];

const FW_RULES = [
  {
    i: 1,
    dir: 'in',
    act: 'accept',
    proto: 'tcp',
    port: '22',
    src: '192.168.0.0/16',
    dst: 'vm-web-1',
    on: true,
  },
  {
    i: 2,
    dir: 'in',
    act: 'accept',
    proto: 'tcp',
    port: '80,443',
    src: '0.0.0.0/0',
    dst: 'vm-web-1',
    on: true,
  },
  {
    i: 3,
    dir: 'in',
    act: 'drop',
    proto: 'tcp',
    port: '3306',
    src: '0.0.0.0/0',
    dst: 'vm-db-1',
    on: true,
  },
  {
    i: 4,
    dir: 'out',
    act: 'accept',
    proto: 'any',
    port: '*',
    src: '*',
    dst: '0.0.0.0/0',
    on: true,
  },
  {
    i: 5,
    dir: 'in',
    act: 'accept',
    proto: 'udp',
    port: '53',
    src: '10.0.0.0/8',
    dst: 'ct-dns',
    on: true,
  },
];

const SCHEDULES = [
  {
    name: 'daily-backup',
    target: 'tag:prod',
    action: 'backup',
    cron: '0 2 * * *',
    next: 'tonight 02:00',
    last: 'ok',
    on: true,
  },
  {
    name: 'weekly-verify',
    target: 'tag:prod',
    action: 'verify',
    cron: '0 4 * * 0',
    next: 'Sun 04:00',
    last: 'ok',
    on: true,
  },
  {
    name: 'monthly-snapshot',
    target: '*',
    action: 'snapshot',
    cron: '0 3 1 * *',
    next: 'May 1 03:00',
    last: 'ok',
    on: true,
  },
  {
    name: 'cleanup-tmp',
    target: 'tag:tmp',
    action: 'delete',
    cron: '0 1 * * 0',
    next: 'Sun 01:00',
    last: '—',
    on: false,
  },
];

const USERS = [
  { name: 'admin', role: 'admin', twofa: true, lastLogin: '2h ago' },
  { name: 'alice', role: 'operator', twofa: true, lastLogin: '12m ago' },
  { name: 'deploy', role: 'viewer', twofa: false, lastLogin: '4h ago' },
  { name: 'ben', role: 'operator', twofa: true, lastLogin: '3d ago' },
];

const TEMPLATES = [
  { name: 'Jellyfin', cat: 'media', desc: 'Media streaming server', icon: 'film' },
  { name: 'Gitea', cat: 'dev', desc: 'Self-hosted Git service', icon: 'git-branch' },
  { name: 'Uptime Kuma', cat: 'monitor', desc: 'Lightweight monitoring', icon: 'activity' },
  { name: 'Pi-hole', cat: 'infra', desc: 'Network-wide ad blocker', icon: 'shield' },
  { name: 'Nextcloud', cat: 'storage', desc: 'Self-hosted files & collab', icon: 'cloud' },
  { name: 'Grafana', cat: 'monitor', desc: 'Dashboards for metrics', icon: 'bar-chart-3' },
  { name: 'Home Assistant', cat: 'automation', desc: 'Open-source home automation', icon: 'home' },
  { name: 'Vaultwarden', cat: 'security', desc: 'Bitwarden-compatible vault', icon: 'key-round' },
  { name: 'Immich', cat: 'media', desc: 'Self-hosted photo backup', icon: 'image' },
  { name: 'n8n', cat: 'automation', desc: 'Workflow automation', icon: 'workflow' },
  { name: 'Paperless', cat: 'productivity', desc: 'Document management', icon: 'file-text' },
  { name: 'Authentik', cat: 'security', desc: 'Identity provider', icon: 'lock' },
];

const SNAPSHOTS = [
  {
    name: 'pre-cpu-resize',
    date: '2026-04-22 14:10',
    ram: true,
    size: '1.2 GB',
    desc: 'auto before config change',
  },
  { name: 'pre-kernel-upd', date: '2026-04-20 09:02', ram: false, size: '820 MB', desc: 'manual' },
  { name: 'clean-install', date: '2026-04-01 00:00', ram: false, size: '650 MB', desc: 'baseline' },
];

const BACKUPS = [
  {
    ts: '2026-04-22 02:00',
    inst: 'vm-db-1',
    size: '8.2 GB',
    dur: '4m 11s',
    ver: true,
    mode: 'scheduled',
  },
  {
    ts: '2026-04-22 02:00',
    inst: 'vm-web-1',
    size: '1.2 GB',
    dur: '1m 12s',
    ver: true,
    mode: 'scheduled',
  },
  {
    ts: '2026-04-22 02:00',
    inst: 'ct-gitea',
    size: '0.8 GB',
    dur: '0m 42s',
    ver: true,
    mode: 'scheduled',
  },
  {
    ts: '2026-04-22 02:00',
    inst: 'ct-dns',
    size: '0.1 GB',
    dur: '0m 08s',
    ver: true,
    mode: 'scheduled',
  },
  {
    ts: '2026-04-21 13:12',
    inst: 'vm-old',
    size: '—',
    dur: '0m 02s',
    ver: false,
    mode: 'scheduled',
    err: true,
  },
  {
    ts: '2026-04-21 02:00',
    inst: 'vm-db-1',
    size: '8.1 GB',
    dur: '4m 03s',
    ver: true,
    mode: 'scheduled',
  },
  {
    ts: '2026-04-21 02:00',
    inst: 'vm-web-1',
    size: '1.2 GB',
    dur: '1m 09s',
    ver: true,
    mode: 'scheduled',
  },
];

const WARNINGS = [
  { sev: 'warn', msg: 'Storage pool "default" at 72% — backups may slow', target: 'pool:default' },
  { sev: 'warn', msg: 'vm-old has no backup (95 days stopped)', target: 'vm-old' },
  { sev: 'danger', msg: 'Backup of vm-old failed: storage pool full', target: 'vm-old' },
  { sev: 'warn', msg: '3 instances have no firewall rules', target: '—' },
  { sev: 'info', msg: 'Security patch v0.1.1 available — fixes CVE-2026-12', target: 'system' },
];

// ─── SMALL PRIMITIVES ────────────────────────────────────────
const Badge = ({ children, color, dot }) => (
  <span className={'badge' + (dot ? ' badge--dot' : '')} style={{ color, borderColor: color }}>
    {children}
  </span>
);
const STATUS_BADGE = {
  running: (
    <Badge color="var(--h-success)" dot>
      Running
    </Badge>
  ),
  stopped: (
    <Badge color="var(--h-danger)" dot>
      Stopped
    </Badge>
  ),
  warn: (
    <Badge color="var(--h-warn)" dot>
      Warning
    </Badge>
  ),
  healthy: (
    <Badge color="var(--h-success)" dot>
      Healthy
    </Badge>
  ),
  degraded: (
    <Badge color="var(--h-warn)" dot>
      Degraded
    </Badge>
  ),
  sleeping: (
    <Badge color="var(--h-info)" dot>
      Sleeping
    </Badge>
  ),
  online: (
    <Badge color="var(--h-success)" dot>
      Online
    </Badge>
  ),
  failed: (
    <Badge color="var(--h-danger)" dot>
      Failed
    </Badge>
  ),
  success: (
    <Badge color="var(--h-success)" dot>
      Success
    </Badge>
  ),
  exited: (
    <Badge color="var(--h-text-3)" dot>
      Exited
    </Badge>
  ),
};

const Kbd = ({ k }) => {
  // platform format: on mac, cmd-k => ⌘K
  const isMac =
    typeof navigator !== 'undefined' &&
    /Mac/i.test(
      navigator.userAgentData?.platform || navigator.userAgent || navigator.platform || ''
    );
  const parts = String(k).split(/[-\s]/);
  const fmt = (p) => {
    if (isMac)
      return { cmd: '⌘', ctrl: '⌃', alt: '⌥', shift: '⇧', meta: '⌘' }[p] || p.toUpperCase();
    return (
      { cmd: 'Ctrl', meta: 'Ctrl' }[p] ||
      (p.length === 1 ? p.toUpperCase() : p[0].toUpperCase() + p.slice(1))
    );
  };
  return (
    <span style={{ display: 'inline-flex', gap: 2 }}>
      {parts.map((p, i) => (
        <kbd key={i} className="k">
          {fmt(p)}
        </kbd>
      ))}
    </span>
  );
};

const ProgressBar = ({ v, variant }) => (
  <div className={'prog' + (variant ? ' prog--' + variant : '')}>
    <span style={{ width: Math.max(0, Math.min(100, v)) + '%' }} />
  </div>
);

const Copyable = ({ text, mono = true }) => {
  const [ok, setOk] = useState(false);
  const doCopy = (e) => {
    e.stopPropagation();
    try {
      navigator.clipboard.writeText(text);
    } catch (err) {
      // Clipboard access may fail (permissions, insecure context, unsupported API).
      console.warn('Clipboard write failed:', err);
    }
    setOk(true);
    setTimeout(() => setOk(false), 1200);
  };
  return (
    <span
      className={'copyable' + (mono ? ' mono' : '')}
      onClick={doCopy}
      title="Copy"
      style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}
    >
      {text}
      <I n={ok ? 'check' : 'copy'} s={11} style={{ opacity: 0.5 }} />
    </span>
  );
};

// ─── TOP BAR ─────────────────────────────────────────────────
function TopBar({
  onOpenPalette,
  page,
  crumbs,
  onNav,
  density = 'compact',
  onDensity,
  theme = 'dark',
  onTheme,
  onLogout,
}) {
  const [userMenu, setUserMenu] = useState(false);
  const [notifMenu, setNotifMenu] = useState(false);
  useEffect(() => {
    if (!userMenu && !notifMenu) return;
    const close = () => {
      setUserMenu(false);
      setNotifMenu(false);
    };
    window.addEventListener('click', close);
    return () => window.removeEventListener('click', close);
  }, [userMenu, notifMenu]);
  return (
    <header
      style={{
        height: 48,
        background: 'var(--h-surface)',
        borderBottom: '1px solid var(--h-border)',
        display: 'flex',
        alignItems: 'center',
        padding: '0 14px',
        gap: 14,
        flex: '0 0 auto',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 10,
          width: 220,
          flex: '0 0 220px',
          cursor: 'pointer',
        }}
        onClick={() => onNav?.('dashboard')}
      >
        <img src="assets/mark-inverse.png" style={{ width: 22, height: 22 }} alt="" />
        <div style={{ lineHeight: 1 }}>
          <div className="stencil" style={{ fontSize: 14, color: 'var(--h-text)' }}>
            HELLING
          </div>
          <div
            className="mono"
            style={{ fontSize: 9, letterSpacing: '0.18em', color: 'var(--h-text-3)', marginTop: 2 }}
          >
            v0.1 · node-1
          </div>
        </div>
      </div>

      <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: 10 }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 6,
            fontSize: 12,
            color: 'var(--h-text-3)',
          }}
        >
          {crumbs.map((c, i) => (
            <React.Fragment key={i}>
              {i > 0 && <span style={{ color: 'var(--h-text-3)' }}>/</span>}
              <span
                style={{
                  color: i === crumbs.length - 1 ? 'var(--h-text)' : 'var(--h-text-3)',
                  cursor: i < crumbs.length - 1 ? 'pointer' : 'default',
                }}
                onClick={() => {
                  if (i < crumbs.length - 1) {
                    if (c === 'Dashboard') onNav?.('dashboard');
                    else onNav?.(c.toLowerCase());
                  }
                }}
              >
                {c}
              </span>
            </React.Fragment>
          ))}
        </div>
      </div>

      <button
        className="btn btn--sm"
        onClick={onOpenPalette}
        style={{ width: 280, justifyContent: 'space-between', color: 'var(--h-text-3)' }}
      >
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
          <I n="search" s={13} /> Search or run command…
        </span>
        <Kbd k="cmd-k" />
      </button>

      {/* Density toggle */}
      <div className="seg" onClick={(e) => e.stopPropagation()} title="Density">
        <button
          className={density === 'compact' ? 'on' : ''}
          onClick={() => onDensity?.('compact')}
          title="Compact density"
        >
          <I n="rows-3" s={13} />
        </button>
        <button
          className={density === 'comfortable' ? 'on' : ''}
          onClick={() => onDensity?.('comfortable')}
          title="Comfortable density"
        >
          <I n="rows-2" s={13} />
        </button>
      </div>

      {/* Notifications */}
      <NotifBell
        onClick={() => {
          setNotifMenu((v) => !v);
          setUserMenu(false);
        }}
        open={notifMenu}
        onNav={onNav}
      />

      <button
        className="btn btn--sm btn--ghost"
        title={theme === 'dark' ? 'Switch to light' : 'Switch to dark'}
        onClick={() => onTheme?.(theme === 'dark' ? 'light' : 'dark')}
      >
        <I n={theme === 'dark' ? 'sun' : 'moon'} s={15} />
      </button>

      <button className="btn btn--sm btn--ghost" title="Help">
        <I n="circle-help" s={15} />
      </button>

      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          paddingLeft: 10,
          borderLeft: '1px solid var(--h-border)',
          position: 'relative',
          cursor: 'pointer',
        }}
        onClick={(e) => {
          e.stopPropagation();
          setUserMenu((v) => !v);
          setNotifMenu(false);
        }}
      >
        <div
          style={{
            width: 24,
            height: 24,
            borderRadius: '50%',
            background: 'var(--h-accent)',
            color: '#000',
            display: 'grid',
            placeItems: 'center',
            fontWeight: 700,
            fontSize: 11,
          }}
        >
          A
        </div>
        <div style={{ fontSize: 12, lineHeight: 1.15 }}>
          <div style={{ color: 'var(--h-text)' }}>admin</div>
          <div className="mono" style={{ fontSize: 10, color: 'var(--h-text-3)' }}>
            role · admin
          </div>
        </div>
        <I n="chevron-down" s={12} style={{ color: 'var(--h-text-3)' }} />
        {userMenu && (
          <div
            style={{
              position: 'absolute',
              top: 'calc(100% + 6px)',
              right: 0,
              width: 220,
              background: 'var(--h-surface)',
              border: '1px solid var(--h-border)',
              borderRadius: 'var(--h-radius)',
              boxShadow: '0 8px 24px rgba(0,0,0,0.5)',
              zIndex: 200,
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <MenuItem
              icon="user"
              label="Your profile"
              onClick={() => {
                setUserMenu(false);
                onNav?.('rbac:admin');
              }}
            />
            <MenuItem
              icon="key"
              label="API tokens"
              onClick={() => {
                setUserMenu(false);
                onNav?.('rbac:admin');
              }}
            />
            <MenuItem
              icon="settings"
              label="Settings"
              onClick={() => {
                setUserMenu(false);
                onNav?.('settings');
              }}
            />
            <div style={{ height: 1, background: 'var(--h-border)' }} />
            <MenuItem
              icon="sun"
              label={'Density: ' + (density === 'compact' ? 'compact' : 'comfortable')}
              onClick={() => {
                onDensity?.(density === 'compact' ? 'comfortable' : 'compact');
                setUserMenu(false);
              }}
            />
            <div style={{ height: 1, background: 'var(--h-border)' }} />
            <MenuItem
              icon="log-out"
              label="Sign out"
              onClick={() => {
                setUserMenu(false);
                onLogout?.();
              }}
            />
          </div>
        )}
      </div>
    </header>
  );
}

function MenuItem({ icon, label, onClick }) {
  return (
    <button
      onClick={onClick}
      style={{
        width: '100%',
        background: 'transparent',
        border: 'none',
        textAlign: 'left',
        padding: '8px 12px',
        color: 'var(--h-text-2)',
        fontSize: 12,
        fontFamily: 'var(--bzr-font-body)',
        display: 'flex',
        alignItems: 'center',
        gap: 10,
        cursor: 'pointer',
      }}
      onMouseOver={(e) => {
        e.currentTarget.style.background = 'rgba(255,255,255,0.04)';
        e.currentTarget.style.color = 'var(--h-text)';
      }}
      onMouseOut={(e) => {
        e.currentTarget.style.background = 'transparent';
        e.currentTarget.style.color = 'var(--h-text-2)';
      }}
    >
      <I n={icon} s={13} /> {label}
    </button>
  );
}

function NotifBell({ open, onClick, onNav }) {
  useStore();
  const sevOrder = { danger: 0, warn: 1, info: 2 };
  const unread = ALERTS.filter((a) => !a.read);
  const sev = unread.length
    ? unread.slice().sort((a, b) => sevOrder[a.sev] - sevOrder[b.sev])[0].sev
    : null;
  const dot =
    sev === 'danger'
      ? 'var(--h-danger)'
      : sev === 'warn'
        ? 'var(--h-warn)'
        : sev === 'info'
          ? 'var(--h-info)'
          : null;
  return (
    <div style={{ position: 'relative' }} onClick={(e) => e.stopPropagation()}>
      <button
        className="btn btn--sm btn--ghost"
        title={
          unread.length
            ? unread.length + ' unread alert' + (unread.length === 1 ? '' : 's')
            : 'No alerts'
        }
        style={{ position: 'relative' }}
        onClick={onClick}
      >
        <I n="bell" s={15} />
        {dot && (
          <span
            className="mono"
            style={{
              position: 'absolute',
              top: -2,
              right: -2,
              minWidth: 14,
              height: 14,
              borderRadius: 999,
              background: dot,
              color: '#000',
              fontSize: 9,
              fontWeight: 700,
              padding: '0 3px',
              display: 'grid',
              placeItems: 'center',
              lineHeight: 1,
            }}
          >
            {unread.length > 9 ? '9+' : unread.length}
          </span>
        )}
      </button>
      {open && <NotifMenu onNav={onNav} onClose={() => onClick?.()} />}
    </div>
  );
}

function NotifMenu({ onNav, onClose }) {
  useStore();
  const unread = ALERTS.filter((a) => !a.read).length;
  const markAll = (e) => {
    e.stopPropagation();
    ALERTS.forEach((a) => (a.read = true));
    __notifyStore();
  };
  const dismiss = (id, e) => {
    e.stopPropagation();
    const i = ALERTS.findIndex((a) => a.id === id);
    if (i >= 0) {
      ALERTS.splice(i, 1);
      __notifyStore();
    }
  };
  const click = (it) => {
    it.read = true;
    __notifyStore();
    onClose?.();
    it.nav?.();
  };
  return (
    <div
      style={{
        position: 'absolute',
        top: 'calc(100% + 6px)',
        right: -40,
        width: 400,
        background: 'var(--h-surface)',
        border: '1px solid var(--h-border)',
        borderRadius: 'var(--h-radius)',
        boxShadow: '0 8px 24px rgba(0,0,0,0.5)',
        zIndex: 200,
        maxHeight: 460,
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <div
        style={{
          padding: '10px 12px',
          borderBottom: '1px solid var(--h-border)',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <div className="stencil" style={{ fontSize: 12 }}>
          NOTIFICATIONS{' '}
          <span className="mono dim" style={{ marginLeft: 6 }}>
            · {unread} unread
          </span>
        </div>
        <div style={{ display: 'flex', gap: 10 }}>
          <a
            className="link"
            style={{
              fontSize: 11,
              opacity: unread ? 1 : 0.4,
              pointerEvents: unread ? 'auto' : 'none',
            }}
            onClick={markAll}
          >
            Mark all read
          </a>
          <a
            className="link"
            style={{ fontSize: 11 }}
            onClick={() => {
              onClose?.();
              onNav?.('alerts');
            }}
          >
            Manage →
          </a>
        </div>
      </div>
      <div style={{ overflow: 'auto', flex: 1 }}>
        {ALERTS.length === 0 ? (
          <div style={{ padding: 30, textAlign: 'center', color: 'var(--h-text-3)', fontSize: 12 }}>
            <I n="shield-check" s={18} style={{ display: 'block', margin: '0 auto 6px' }} /> All
            clear.
          </div>
        ) : (
          ALERTS.map((it) => (
            <div
              key={it.id}
              className="notif-row"
              style={{
                padding: '10px 12px',
                borderBottom: '1px solid rgba(61,61,61,0.5)',
                display: 'flex',
                gap: 10,
                cursor: 'pointer',
                position: 'relative',
                background: it.read ? 'transparent' : 'rgba(211,135,43,0.04)',
              }}
              onClick={() => click(it)}
              onMouseOver={(e) =>
                (e.currentTarget.style.background = it.read
                  ? 'rgba(255,255,255,0.03)'
                  : 'rgba(211,135,43,0.08)')
              }
              onMouseOut={(e) =>
                (e.currentTarget.style.background = it.read
                  ? 'transparent'
                  : 'rgba(211,135,43,0.04)')
              }
            >
              {!it.read && (
                <span
                  style={{
                    position: 'absolute',
                    left: 4,
                    top: '50%',
                    marginTop: -3,
                    width: 6,
                    height: 6,
                    borderRadius: '50%',
                    background: 'var(--h-accent)',
                  }}
                />
              )}
              <I
                n={
                  it.sev === 'danger' ? 'octagon-x' : it.sev === 'warn' ? 'triangle-alert' : 'info'
                }
                s={14}
                color={
                  it.sev === 'danger'
                    ? 'var(--h-danger)'
                    : it.sev === 'warn'
                      ? 'var(--h-warn)'
                      : 'var(--h-info)'
                }
              />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 12, fontWeight: 600 }}>{it.t}</div>
                <div style={{ fontSize: 11, color: 'var(--h-text-2)' }}>{it.body}</div>
                <div
                  className="mono"
                  style={{ fontSize: 10, color: 'var(--h-text-3)', marginTop: 3 }}
                >
                  {it.target} · {it.time}
                </div>
              </div>
              <button
                className="btn btn--sm btn--ghost"
                title="Dismiss"
                onClick={(e) => dismiss(it.id, e)}
                style={{ padding: '2px 4px', alignSelf: 'flex-start' }}
              >
                <I n="x" s={12} />
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

// ─── RESOURCE TREE (Sidebar) ─────────────────────────────────
function ResourceTree({ page, onNav }) {
  const [open, setOpen] = useState({
    nodes: true,
    'node-1': true,
    'node-2': true,
    'node-3': false,
  });
  const [q, setQ] = useState('');
  const toggle = (k) => setOpen((o) => ({ ...o, [k]: !o[k] }));
  const match = (label) => !q || String(label).toLowerCase().includes(q.toLowerCase());

  // When filtering, auto-expand all nodes so matches in collapsed branches show.
  const forceOpen = q.length > 0;
  const isOpen = (k) => (forceOpen ? true : !!open[k]);

  const Node = ({
    id,
    label,
    icon,
    pad = 12,
    on,
    caret,
    onToggle,
    onClick,
    hideIfNoMatch = true,
  }) => {
    if (hideIfNoMatch && q && !match(label) && caret === undefined) return null;
    const highlight = (txt) => {
      if (!q) return txt;
      const i = String(txt).toLowerCase().indexOf(q.toLowerCase());
      if (i < 0) return txt;
      return (
        <>
          <span>{String(txt).slice(0, i)}</span>
          <mark style={{ background: 'rgba(211,135,43,0.35)', color: 'var(--h-text)', padding: 0 }}>
            {String(txt).slice(i, i + q.length)}
          </mark>
          <span>{String(txt).slice(i + q.length)}</span>
        </>
      );
    };
    return (
      <div
        className={'node' + (on ? ' on' : '')}
        style={{ paddingLeft: pad }}
        onClick={() => {
          onToggle && onToggle();
          onClick && onClick();
        }}
      >
        {caret !== undefined ? (
          <span className="caret">{caret ? '▾' : '▸'}</span>
        ) : (
          <span className="caret" />
        )}
        {icon && (
          <span className="ico">
            <I n={icon} s={13} />
          </span>
        )}
        <span style={{ flex: 1 }}>{highlight(label)}</span>
      </div>
    );
  };

  // Determine if a section has any matches — hide label if not.
  const Section = ({ label, children }) => {
    if (!q)
      return (
        <>
          <div className="section-label">{label}</div>
          {children}
        </>
      );
    // when filtering, still render — children decide their own visibility; header only shown if at least one child is visible.
    // Rough heuristic: search the rendered output by checking the query against child labels via a container test.
    return (
      <>
        <div className="section-label">{label}</div>
        {children}
      </>
    );
  };

  return (
    <aside
      style={{
        width: 232,
        flex: '0 0 232px',
        background: 'var(--h-surface)',
        borderRight: '1px solid var(--h-border)',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}
    >
      <div style={{ padding: '8px 10px', borderBottom: '1px solid var(--h-border)' }}>
        <div style={{ position: 'relative' }}>
          <I
            n="search"
            s={12}
            style={{ position: 'absolute', left: 8, top: 8, color: 'var(--h-text-3)' }}
          />
          <input
            className="input"
            placeholder="Filter resources…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            style={{
              height: 28,
              fontSize: 12,
              paddingLeft: 26,
              paddingRight: q ? 26 : 8,
              width: '100%',
            }}
          />
          {q && (
            <button
              onClick={() => setQ('')}
              title="Clear"
              style={{
                position: 'absolute',
                right: 4,
                top: 4,
                width: 20,
                height: 20,
                border: 'none',
                background: 'transparent',
                color: 'var(--h-text-3)',
                cursor: 'pointer',
                borderRadius: 3,
                display: 'grid',
                placeItems: 'center',
              }}
            >
              <I n="x" s={12} />
            </button>
          )}
        </div>
      </div>

      <div className="tree" style={{ flex: 1, overflow: 'auto', padding: '6px 0' }}>
        {!q && <div className="section-label">PINNED</div>}
        {!q && (
          <Node
            label="vm-web-1"
            icon="monitor"
            pad={16}
            on={page === 'instance:vm-web-1'}
            onClick={() => onNav('instance:vm-web-1')}
          />
        )}
        {!q && (
          <Node
            label="jellyfin"
            icon="box"
            pad={16}
            on={page === 'container:jellyfin'}
            onClick={() => onNav('container:jellyfin')}
          />
        )}
        {!q && (
          <Node
            label="prod-k8s"
            icon="hexagon"
            pad={16}
            on={page === 'cluster:prod-k8s'}
            onClick={() => onNav('cluster:prod-k8s')}
          />
        )}

        {!q && <div className="section-label">RECENT</div>}
        {!q && (
          <Node label="vm-db-1" icon="monitor" pad={16} onClick={() => onNav('instance:vm-db-1')} />
        )}
        {!q && <Node label="audit" icon="list" pad={16} onClick={() => onNav('audit')} />}

        <div className="section-label">DATACENTER</div>
        <Node
          label="Dashboard"
          icon="layout-dashboard"
          on={page === 'dashboard'}
          onClick={() => onNav('dashboard')}
        />

        {match('Nodes') || q ? (
          <Node
            label="Nodes"
            icon="server"
            caret={isOpen('nodes')}
            onToggle={() => toggle('nodes')}
            hideIfNoMatch={false}
          />
        ) : null}
        {isOpen('nodes') &&
          NODES.map((n) => {
            const nodeVms = INSTANCES.filter((x) => x.node === n.id && x.type === 'VM');
            const nodeCts = INSTANCES.filter((x) => x.node === n.id && x.type === 'CT');
            const anyMatch =
              !q ||
              match(n.name) ||
              nodeVms.some((v) => match(v.name)) ||
              nodeCts.some((c) => match(c.name)) ||
              match('Podman') ||
              match('Kubernetes');
            if (!anyMatch) return null;
            return (
              <React.Fragment key={n.id}>
                <Node
                  label={n.name}
                  icon="cpu"
                  pad={24}
                  caret={isOpen(n.id)}
                  onToggle={() => toggle(n.id)}
                  hideIfNoMatch={false}
                />
                {isOpen(n.id) && (
                  <>
                    <Node
                      label="VMs"
                      icon="monitor"
                      pad={38}
                      on={page === 'instances'}
                      onClick={() => onNav('instances')}
                    />
                    {nodeVms.map((x) => (
                      <Node
                        key={x.name}
                        label={x.name}
                        icon={x.status === 'running' ? 'circle' : 'circle-off'}
                        pad={52}
                        on={page === 'instance:' + x.name}
                        onClick={() => onNav('instance:' + x.name)}
                      />
                    ))}
                    <Node
                      label="CTs"
                      icon="package-2"
                      pad={38}
                      onClick={() => onNav('instances')}
                    />
                    {nodeCts.map((x) => (
                      <Node
                        key={x.name}
                        label={x.name}
                        icon={x.status === 'running' ? 'circle' : 'circle-off'}
                        pad={52}
                        on={page === 'instance:' + x.name}
                        onClick={() => onNav('instance:' + x.name)}
                      />
                    ))}
                    <Node
                      label="Podman"
                      icon="container"
                      pad={38}
                      on={page === 'containers'}
                      onClick={() => onNav('containers')}
                    />
                    <Node
                      label="Kubernetes"
                      icon="hexagon"
                      pad={38}
                      on={page === 'kubernetes'}
                      onClick={() => onNav('kubernetes')}
                    />
                  </>
                )}
              </React.Fragment>
            );
          })}

        <div className="section-label">RESOURCES</div>
        <Node
          label="Templates"
          icon="layout-grid"
          on={page === 'templates'}
          onClick={() => onNav('templates')}
        />
        <Node
          label="Marketplace"
          icon="shopping-bag"
          on={page === 'marketplace'}
          onClick={() => onNav('marketplace')}
        />
        <Node
          label="Storage"
          icon="database"
          on={page === 'storage'}
          onClick={() => onNav('storage')}
        />
        <Node
          label="Networking"
          icon="network"
          on={page === 'networking'}
          onClick={() => onNav('networking')}
        />
        <Node
          label="Firewall"
          icon="shield"
          on={page === 'firewall'}
          onClick={() => onNav('firewall')}
        />
        <Node label="Images" icon="disc" on={page === 'images'} onClick={() => onNav('images')} />
        <Node
          label="Backups"
          icon="archive"
          on={page === 'backups'}
          onClick={() => onNav('backups')}
        />
        <Node
          label="Schedules"
          icon="calendar-clock"
          on={page === 'schedules'}
          onClick={() => onNav('schedules')}
        />
        <Node label="BMC" icon="power" on={page === 'bmc'} onClick={() => onNav('bmc')} />
        <Node
          label="Cluster"
          icon="boxes"
          on={page === 'cluster'}
          onClick={() => onNav('cluster')}
        />

        <div className="section-label">OBSERVABILITY</div>
        <Node
          label="Metrics"
          icon="chart-line"
          on={page === 'metrics'}
          onClick={() => onNav('metrics')}
        />
        <Node label="Alerts" icon="bell" on={page === 'alerts'} onClick={() => onNav('alerts')} />

        <div className="section-label">ADMIN</div>
        <Node label="Users" icon="users" on={page === 'users'} onClick={() => onNav('users')} />
        <Node label="Audit" icon="list" on={page === 'audit'} onClick={() => onNav('audit')} />
        <Node label="Logs" icon="scroll-text" on={page === 'logs'} onClick={() => onNav('logs')} />
        <Node label="Operations" icon="activity" on={page === 'ops'} onClick={() => onNav('ops')} />
        <Node
          label="Settings"
          icon="settings"
          on={page === 'settings'}
          onClick={() => onNav('settings')}
        />
      </div>

      <div
        style={{
          padding: '8px 12px',
          borderTop: '1px solid var(--h-border)',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <FooterStatus />
      </div>
    </aside>
  );
}

function FooterStatus() {
  useStore();
  const latency = window.__getLatency?.() ?? 18;
  const healthy = latency < 28;
  return (
    <>
      <div
        className="mono"
        style={{
          fontSize: 10,
          letterSpacing: '0.14em',
          color: 'var(--h-text-3)',
          textTransform: 'uppercase',
        }}
      >
        <span style={{ color: healthy ? 'var(--h-success)' : 'var(--h-warn)' }}>●</span>{' '}
        {healthy ? 'healthy' : 'degraded'}
      </div>
      <div className="mono" style={{ fontSize: 10, color: 'var(--h-text-3)' }}>
        {latency}ms
      </div>
    </>
  );
}

// ─── TASK LOG DRAWER ─────────────────────────────────────────
function TaskDrawer({ open, onToggle }) {
  useStore();
  const [filter, setFilter] = useState('all');
  const [groupBy, setGroupBy] = useState('none');
  const running = TASKS.filter((t) => t.status === 'running').length;
  const completed = TASKS.filter((t) => t.status === 'success' || t.status === 'failed').length;
  const filtered = TASKS.filter((t) => filter === 'all' || t.status === filter);
  const clearCompleted = (e) => {
    e.stopPropagation();
    for (let i = TASKS.length - 1; i >= 0; i--) {
      if (TASKS[i].status === 'success') TASKS.splice(i, 1);
    }
    __notifyStore();
    window.toast?.info(
      'Cleared completed tasks',
      completed + ' task' + (completed === 1 ? '' : 's') + ' removed from log',
    );
  };
  const groups =
    groupBy === 'target'
      ? Object.entries(
          filtered.reduce((m, t) => ((m[t.target] = m[t.target] || []).push(t), m), {}),
        )
      : [['', filtered]];
  return (
    <div className={'drawer' + (open ? '' : ' closed')}>
      <div className="drawer__handle" onClick={onToggle}>
        <span>
          <I n={open ? 'chevron-down' : 'chevron-up'} s={12} />
          &nbsp;&nbsp;Task Log · {running} running · {TASKS.length} recent
        </span>
        <span style={{ display: 'inline-flex', gap: 14, alignItems: 'center' }}>
          <span style={{ color: running ? 'var(--h-success)' : 'var(--h-text-3)' }}>
            ● {running ? 'live' : 'idle'}
          </span>
          <Kbd k="ctrl-`" />
        </span>
      </div>
      {open && (
        <div className="drawer__body">
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              padding: '6px 10px',
              borderBottom: '1px solid var(--h-border)',
              gap: 10,
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
              <div className="seg">
                {[
                  ['all', 'All', TASKS.length],
                  ['running', 'Running', running],
                  ['success', 'Success', TASKS.filter((t) => t.status === 'success').length],
                  ['failed', 'Failed', TASKS.filter((t) => t.status === 'failed').length],
                ].map(([k, l, n]) => (
                  <button key={k} className={filter === k ? 'on' : ''} onClick={() => setFilter(k)}>
                    {l}{' '}
                    <span className="mono dim" style={{ marginLeft: 4 }}>
                      {n}
                    </span>
                  </button>
                ))}
              </div>
              <div className="seg">
                <button
                  className={groupBy === 'none' ? 'on' : ''}
                  onClick={() => setGroupBy('none')}
                >
                  Flat
                </button>
                <button
                  className={groupBy === 'target' ? 'on' : ''}
                  onClick={() => setGroupBy('target')}
                >
                  Group by target
                </button>
              </div>
            </div>
            <button className="btn btn--sm" disabled={completed === 0} onClick={clearCompleted}>
              <I n="x" s={11} /> Clear completed {completed ? '(' + completed + ')' : ''}
            </button>
          </div>
          <table className="tbl">
            <thead>
              <tr>
                <th style={{ width: 90 }}>ID</th>
                <th style={{ width: 140 }}>Operation</th>
                <th>Target</th>
                <th style={{ width: 100 }}>User</th>
                <th style={{ width: 110 }}>Status</th>
                <th style={{ width: 220 }}>Progress</th>
                <th style={{ width: 100 }}>Started</th>
              </tr>
            </thead>
            <tbody>
              {filtered.length === 0 ? (
                <tr>
                  <td
                    colSpan={7}
                    style={{
                      padding: 30,
                      textAlign: 'center',
                      color: 'var(--h-text-3)',
                      fontSize: 12,
                    }}
                  >
                    No {filter === 'all' ? 'tasks' : filter + ' tasks'}.
                  </td>
                </tr>
              ) : (
                groups.map(([g, ts]) => (
                  <React.Fragment key={g || 'flat'}>
                    {g && (
                      <tr>
                        <td
                          colSpan={7}
                          style={{
                            padding: '6px 10px',
                            background: 'var(--h-surface-2)',
                            fontSize: 10,
                            color: 'var(--h-text-3)',
                            textTransform: 'uppercase',
                            letterSpacing: 0.5,
                          }}
                        >
                          {g}{' '}
                          <span className="mono" style={{ marginLeft: 6 }}>
                            · {ts.length}
                          </span>
                        </td>
                      </tr>
                    )}
                    {ts.map((t) => (
                      <tr key={t.id}>
                        <td className="mono" style={{ color: 'var(--h-text-3)' }}>
                          {t.id}
                        </td>
                        <td className="mono">{t.op}</td>
                        <td className="mono">{t.target}</td>
                        <td>{t.user}</td>
                        <td>{STATUS_BADGE[t.status]}</td>
                        <td>
                          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                            <div style={{ flex: 1 }}>
                              <ProgressBar
                                v={t.progress}
                                variant={
                                  t.status === 'failed'
                                    ? 'danger'
                                    : t.status === 'running'
                                      ? undefined
                                      : undefined
                                }
                              />
                            </div>
                            <span
                              className="mono"
                              style={{
                                fontSize: 11,
                                color: 'var(--h-text-3)',
                                width: 34,
                                textAlign: 'right',
                              }}
                            >
                              {t.progress}%
                            </span>
                          </div>
                          {t.err && (
                            <div style={{ color: 'var(--h-danger)', fontSize: 11, marginTop: 2 }}>
                              {t.err}
                            </div>
                          )}
                        </td>
                        <td className="mono" style={{ color: 'var(--h-text-3)' }}>
                          {t.started}
                        </td>
                      </tr>
                    ))}
                  </React.Fragment>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ─── COMMAND PALETTE ─────────────────────────────────────────
function CommandPalette({ open, onClose, onNav }) {
  const [q, setQ] = useState('');
  const [sel, setSel] = useState(0);
  const actions = useMemo(
    () => [
      {
        id: 'nav.dashboard',
        label: 'Go to Dashboard',
        cat: 'Navigation',
        k: 'g d',
        run: () => onNav('dashboard'),
      },
      {
        id: 'nav.instances',
        label: 'Go to Instances',
        cat: 'Navigation',
        k: 'g i',
        run: () => onNav('instances'),
      },
      {
        id: 'nav.containers',
        label: 'Go to Containers',
        cat: 'Navigation',
        k: 'g c',
        run: () => onNav('containers'),
      },
      {
        id: 'nav.kubernetes',
        label: 'Go to Kubernetes',
        cat: 'Navigation',
        k: 'g k',
        run: () => onNav('kubernetes'),
      },
      {
        id: 'nav.storage',
        label: 'Go to Storage',
        cat: 'Navigation',
        k: 'g s',
        run: () => onNav('storage'),
      },
      {
        id: 'nav.networking',
        label: 'Go to Networking',
        cat: 'Navigation',
        k: 'g n',
        run: () => onNav('networking'),
      },
      {
        id: 'nav.firewall',
        label: 'Go to Firewall',
        cat: 'Navigation',
        k: 'g f',
        run: () => onNav('firewall'),
      },
      {
        id: 'nav.backups',
        label: 'Go to Backups',
        cat: 'Navigation',
        k: 'g b',
        run: () => onNav('backups'),
      },
      {
        id: 'nav.metrics',
        label: 'Go to Metrics',
        cat: 'Navigation',
        k: 'g m',
        run: () => onNav('metrics'),
      },
      { id: 'nav.alerts', label: 'Go to Alerts', cat: 'Navigation', run: () => onNav('alerts') },
      {
        id: 'nav.marketplace',
        label: 'Go to Marketplace',
        cat: 'Navigation',
        run: () => onNav('marketplace'),
      },
      {
        id: 'nav.audit',
        label: 'Go to Audit',
        cat: 'Navigation',
        k: 'g a',
        run: () => onNav('audit'),
      },
      {
        id: 'nav.logs',
        label: 'Go to Logs',
        cat: 'Navigation',
        k: 'g l',
        run: () => onNav('logs'),
      },
      {
        id: 'nav.settings',
        label: 'Go to Settings',
        cat: 'Navigation',
        k: 'g ,',
        run: () => onNav('settings'),
      },
      {
        id: 'instance.create',
        label: 'Create Instance…',
        cat: 'Instances',
        run: () => onNav('new-instance'),
      },
      {
        id: 'container.create',
        label: 'Create Container…',
        cat: 'Containers',
        run: () => onNav('new-instance'),
      },
      {
        id: 'template.deploy',
        label: 'Deploy from Template…',
        cat: 'Templates',
        run: () => onNav('marketplace'),
      },
      {
        id: 'cluster.create',
        label: 'Create Kubernetes Cluster…',
        cat: 'Kubernetes',
        run: () => onNav('kubernetes'),
      },
      {
        id: 'backup.verify',
        label: 'Verify all backups',
        cat: 'Backups',
        run: () => {
          onNav('backups');
          window.toast?.info(
            'Backup verification queued',
            'Running fsck on all snapshots — 14 items',
          );
        },
      },
      {
        id: 'search.vms4gb',
        label: 'Show VMs using > 4 GB RAM',
        cat: 'Query',
        run: () => onNav('instances'),
      },
      {
        id: 'search.whychanged',
        label: 'What changed today?',
        cat: 'Query',
        run: () => onNav('audit'),
      },
      {
        id: 'system.doctor',
        label: 'Run `helling doctor`',
        cat: 'System',
        run: () => {
          onNav('ops');
          window.toast?.info('helling doctor started', 'Checking 42 system invariants…');
        },
      },
      {
        id: 'system.reboot',
        label: 'Reboot node',
        cat: 'System',
        run: () =>
          window.openModal?.('confirm', {
            title: 'Reboot node',
            body: 'This will cause brief downtime for all VMs on this node.',
            danger: true,
            confirmText: 'Reboot now',
            confirmMatch: 'reboot',
            onConfirm: () =>
              window.toast?.warn('Reboot queued', 'Node-1 will restart in 60 seconds'),
          }),
      },
      {
        id: 'user.theme',
        label: 'Toggle theme (light / dark)',
        cat: 'Preferences',
        run: () => window.toast?.info('Theme toggle', 'Light theme coming in 0.2'),
      },
    ],
    [onNav],
  );

  const filtered = actions.filter(
    (a) =>
      !q ||
      a.label.toLowerCase().includes(q.toLowerCase()) ||
      a.cat.toLowerCase().includes(q.toLowerCase()),
  );
  // If search has 2+ chars, offer a "Search everything" action
  const withSearch =
    q.length >= 2
      ? [
          ...filtered,
          {
            id: 'search.global',
            label: `Search "${q}" across everything…`,
            cat: 'Search',
            run: () => {
              window.__searchQuery = q;
              onNav('search');
            },
          },
        ]
      : filtered;

  useEffect(() => {
    if (open) setSel(0);
  }, [open, q]);

  const onKey = (e) => {
    if (e.key === 'Escape') {
      onClose();
      return;
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setSel((s) => Math.min(withSearch.length - 1, s + 1));
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      setSel((s) => Math.max(0, s - 1));
    }
    if (e.key === 'Enter') {
      const a = withSearch[sel];
      if (a) {
        a.run();
        onClose();
      }
    }
  };

  if (!open) return null;
  return (
    <div className="palette-bg" onClick={onClose}>
      <div className="palette" onClick={(e) => e.stopPropagation()}>
        <input
          autoFocus
          placeholder="Run a command or jump to anything…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          onKeyDown={onKey}
        />
        <div className="results">
          {withSearch.map((a, i) => (
            <div
              key={a.id}
              className={'row' + (i === sel ? ' on' : '')}
              onMouseEnter={() => setSel(i)}
              onClick={() => {
                a.run();
                onClose();
              }}
            >
              <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
                <span className="cat" style={{ width: 80 }}>
                  {a.cat}
                </span>
                <span className="label">{a.label}</span>
              </div>
              {a.k && <Kbd k={a.k} />}
            </div>
          ))}
          {withSearch.length === 0 && (
            <div style={{ padding: '24px 16px', textAlign: 'center', color: 'var(--h-text-3)' }}>
              No results. Press <Kbd k="escape" /> to dismiss.
            </div>
          )}
        </div>
        <div
          style={{
            padding: '8px 14px',
            borderTop: '1px solid var(--h-border)',
            display: 'flex',
            justifyContent: 'space-between',
            color: 'var(--h-text-3)',
            fontSize: 11,
          }}
        >
          <span>
            <Kbd k="up" />
            <Kbd k="down" /> navigate · <Kbd k="enter" /> run · <Kbd k="escape" /> close
          </span>
          <span className="mono">
            {withSearch.length} / {actions.length}
          </span>
        </div>
      </div>
    </div>
  );
}

// expose to other files
Object.assign(window, {
  React,
  useState,
  useEffect,
  useRef,
  useMemo,
  useCallback,
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
});
