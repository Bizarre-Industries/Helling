/* Helling — new pages: wizards, console, metrics, alerts, rbac, firewall editor,
   marketplace, file browser, search, container detail, setup, modals */
/* eslint-disable */
import React, { useState, useMemo } from 'react';
import './shell.jsx';
import './infra.jsx';
import './pages.jsx';

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
  Sparkline,
  MultiChart,
  FilePath,
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
  PageUsers,
  PageAudit,
  PageLogs,
  PageOps,
  PageSettings,
  PageLogin,
} = window;

// ─── SETUP WIZARD ───────────────────────────────────────────────
// ─── CREATE INSTANCE (full page, 5-step) ────────────────────────
function PageNewInstance({ onNav }) {
  const [step, setStep] = useState(0);
  const STEPS = ['General', 'OS / Image', 'Storage', 'Network', 'Cloud-init', 'Review'];
  const [cfg, setCfg] = useState({
    type: 'VM',
    name: 'vm-new-1',
    node: 'node-1',
    cores: 2,
    ram: 4,
    os: 'debian-13-cloud',
    disk: 32,
    pool: 'default',
    bus: 'virtio-scsi',
    net: 'bridge0',
    mac: 'auto',
    fw: true,
    vlan: '',
    username: 'admin',
    ssh: '',
    userData: '',
    startAfter: true,
  });
  const set = (k, v) => setCfg((c) => ({ ...c, [k]: v }));
  const next = () => setStep((s) => Math.min(STEPS.length - 1, s + 1));
  const prev = () => setStep((s) => Math.max(0, s - 1));

  return (
    <div style={{ maxWidth: 1100, padding: '18px 20px' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-end',
          marginBottom: 14,
        }}
      >
        <div>
          <div className="eyebrow">DATACENTER / INSTANCES / NEW</div>
          <h1
            className="stencil"
            style={{ fontSize: 26, margin: '4px 0 0', letterSpacing: '0.01em' }}
          >
            Create instance
          </h1>
          <div className="muted" style={{ fontSize: 13, marginTop: 4 }}>
            Full VM or unprivileged container. Takes about 30 seconds on local storage.
          </div>
        </div>
        <button className="btn btn--sm" onClick={() => onNav('instances')}>
          <I n="x" s={13} /> Cancel
        </button>
      </div>

      <div
        style={{
          background: 'var(--h-surface)',
          border: '1px solid var(--h-border)',
          borderRadius: 'var(--h-radius)',
        }}
      >
        <div className="stepper">
          {STEPS.map((s, i) => (
            <React.Fragment key={s}>
              <div className={'s ' + (i === step ? 'on' : i < step ? 'done' : '')}>
                <span className="n">{i < step ? '✓' : i + 1}</span>
                {s}
              </div>
              {i < STEPS.length - 1 && <div className="sep" />}
            </React.Fragment>
          ))}
        </div>
        <div style={{ padding: '22px 28px', minHeight: 440 }}>
          {step === 0 && <WizGeneral cfg={cfg} set={set} />}
          {step === 1 && <WizImage cfg={cfg} set={set} />}
          {step === 2 && <WizStorage cfg={cfg} set={set} />}
          {step === 3 && <WizNetwork cfg={cfg} set={set} />}
          {step === 4 && <WizCloudInit cfg={cfg} set={set} />}
          {step === 5 && <WizReview cfg={cfg} />}
        </div>
        <div
          style={{
            padding: '14px 20px',
            borderTop: '1px solid var(--h-border)',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <div className="mono dim" style={{ fontSize: 11 }}>
            Step {step + 1} / {STEPS.length}
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            <button className="btn btn--sm" onClick={() => onNav('instances')}>
              Cancel
            </button>
            {step > 0 && (
              <button className="btn btn--sm" onClick={prev}>
                Back
              </button>
            )}
            {step < STEPS.length - 1 && (
              <button className="btn btn--sm btn--primary" onClick={next}>
                Next
              </button>
            )}
            {step === STEPS.length - 1 && (
              <button
                className="btn btn--sm btn--primary"
                onClick={() => {
                  window.toast?.success(
                    'Creating ' + cfg.name,
                    'Provisioning ' +
                      (cfg.type === 'VM' ? 'VM' : 'container') +
                      ' — track progress in the task drawer',
                  );
                  onNav('instance:' + cfg.name);
                }}
              >
                <I n="play" s={13} /> Create & start
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
function WizGeneral({ cfg, set }) {
  return (
    <>
      <div className="field">
        <label>Instance type</label>
        <div className="choices">
          {[
            {
              id: 'VM',
              t: 'Full VM (KVM)',
              d: 'Own kernel. Isolation and compat with any OS. ~150 MB RAM overhead.',
            },
            {
              id: 'CT',
              t: 'Linux container (LXC)',
              d: 'Shared kernel. Near-zero overhead. Linux only.',
            },
          ].map((o) => (
            <div
              key={o.id}
              className={'opt' + (cfg.type === o.id ? ' on' : '')}
              onClick={() => set('type', o.id)}
            >
              <div className="t">{o.t}</div>
              <div className="d">{o.d}</div>
            </div>
          ))}
        </div>
      </div>
      <div className="field-row-3">
        <div className="field">
          <label>Name</label>
          <input
            className="input input--mono"
            value={cfg.name}
            onChange={(e) => set('name', e.target.value)}
          />
          <div className="hint">Lowercase, hyphens, max 32 chars</div>
        </div>
        <div className="field">
          <label>Node</label>
          <select className="input" value={cfg.node} onChange={(e) => set('node', e.target.value)}>
            {NODES.map((n) => (
              <option key={n.id} value={n.id}>
                {n.name}
              </option>
            ))}
          </select>
        </div>
        <div className="field">
          <label>Start after creation</label>
          <Switch
            on={cfg.startAfter}
            onChange={(v) => set('startAfter', v)}
            label="Boot immediately"
          />
        </div>
      </div>
      <div className="field-row">
        <div className="field">
          <label>CPU cores</label>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <input
              type="range"
              min="1"
              max="16"
              value={cfg.cores}
              onChange={(e) => set('cores', +e.target.value)}
              style={{ flex: 1 }}
            />
            <span className="mono" style={{ width: 40, textAlign: 'right' }}>
              {cfg.cores}
            </span>
          </div>
          <div className="hint">Available on {cfg.node}: 12 cores</div>
        </div>
        <div className="field">
          <label>Memory (GB)</label>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <input
              type="range"
              min="1"
              max="64"
              value={cfg.ram}
              onChange={(e) => set('ram', +e.target.value)}
              style={{ flex: 1 }}
            />
            <span className="mono" style={{ width: 40, textAlign: 'right' }}>
              {cfg.ram}
            </span>
          </div>
          <div className="hint">Available on {cfg.node}: 48 GB free</div>
        </div>
      </div>
    </>
  );
}
function WizImage({ cfg, set }) {
  const IMGS = [
    { id: 'debian-13-cloud', os: 'Debian 13', size: '560 MB', date: '2026-03-20', type: 'cloud' },
    {
      id: 'ubuntu-2404-cloud',
      os: 'Ubuntu 24.04 LTS',
      size: '612 MB',
      date: '2026-02-14',
      type: 'cloud',
    },
    { id: 'fedora-40-cloud', os: 'Fedora 40', size: '488 MB', date: '2026-03-02', type: 'cloud' },
    {
      id: 'alpine-3.20-cloud',
      os: 'Alpine 3.20',
      size: '38 MB',
      date: '2026-03-18',
      type: 'cloud',
    },
    { id: 'rocky-9-cloud', os: 'Rocky 9', size: '580 MB', date: '2026-02-28', type: 'cloud' },
    { id: 'windows-11-ent', os: 'Windows 11', size: '5.1 GB', date: '2026-01-10', type: 'iso' },
    { id: 'freebsd-14', os: 'FreeBSD 14', size: '420 MB', date: '2025-12-04', type: 'iso' },
  ];
  return (
    <>
      <h3 className="stencil" style={{ fontSize: 14, margin: '0 0 10px', color: 'var(--h-text)' }}>
        PICK AN OS IMAGE
      </h3>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))',
          gap: 10,
        }}
      >
        {IMGS.map((im) => (
          <div
            key={im.id}
            className={'opt choices opt' + (cfg.os === im.id ? ' on' : '')}
            style={{
              padding: 14,
              cursor: 'pointer',
              border: '1px solid ' + (cfg.os === im.id ? 'var(--h-accent)' : 'var(--h-border)'),
              borderRadius: 'var(--h-radius)',
              display: 'flex',
              flexDirection: 'column',
              gap: 4,
              background: cfg.os === im.id ? 'rgba(198,255,36,0.06)' : 'transparent',
            }}
            onClick={() => set('os', im.id)}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div
                style={{
                  fontWeight: 600,
                  fontSize: 13,
                  color: cfg.os === im.id ? 'var(--h-accent)' : 'var(--h-text)',
                }}
              >
                {im.os}
              </div>
              <span
                className="badge"
                style={{ fontSize: 9, borderColor: 'var(--h-border)', color: 'var(--h-text-3)' }}
              >
                {im.type}
              </span>
            </div>
            <div className="mono dim" style={{ fontSize: 10 }}>
              {im.id}
            </div>
            <div className="mono dim" style={{ fontSize: 10, marginTop: 4 }}>
              {im.size} · {im.date}
            </div>
          </div>
        ))}
      </div>
      <a
        className="link"
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 6,
          marginTop: 14,
          fontSize: 12,
        }}
      >
        <I n="download" s={12} /> Download more images
      </a>
    </>
  );
}
function WizStorage({ cfg, set }) {
  return (
    <>
      <div className="field-row">
        <div className="field">
          <label>Disk size (GB)</label>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <input
              type="range"
              min="8"
              max="512"
              value={cfg.disk}
              onChange={(e) => set('disk', +e.target.value)}
              style={{ flex: 1 }}
            />
            <span className="mono" style={{ width: 50, textAlign: 'right' }}>
              {cfg.disk} GB
            </span>
          </div>
          <div className="hint">Thin-provisioned on ZFS; actual usage grows on write.</div>
        </div>
        <div className="field">
          <label>Pool</label>
          <select className="input" value={cfg.pool} onChange={(e) => set('pool', e.target.value)}>
            <option value="default">default — ZFS mirror — 1.4 TB free</option>
            <option value="nvme-fast">nvme-fast — ZFS stripe — 612 GB free</option>
            <option value="slow-rust">slow-rust — ZFS raidz1 — 8.1 TB free</option>
          </select>
        </div>
      </div>
      <div className="field">
        <label>Bus / controller</label>
        <div className="choices">
          {[
            {
              id: 'virtio-scsi',
              t: 'VirtIO SCSI (recommended)',
              d: 'Paravirtualized, TRIM, fastest on Linux guests.',
            },
            {
              id: 'virtio-blk',
              t: 'VirtIO Block',
              d: 'Older; slightly faster for single-disk setups.',
            },
            {
              id: 'sata',
              t: 'SATA emulated',
              d: 'Compat mode for Windows without VirtIO drivers.',
            },
          ].map((o) => (
            <div
              key={o.id}
              className={'opt' + (cfg.bus === o.id ? ' on' : '')}
              onClick={() => set('bus', o.id)}
            >
              <div className="t">{o.t}</div>
              <div className="d">{o.d}</div>
            </div>
          ))}
        </div>
      </div>
      <div className="field">
        <label>Options</label>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <Switch on={true} label="Discard / TRIM" />
          <Switch on={true} label="SSD emulation" />
          <Switch on={false} label="IO-thread pinning" />
          <Switch on={false} label="Encrypt at rest (LUKS)" />
        </div>
      </div>
    </>
  );
}
function WizNetwork({ cfg, set }) {
  return (
    <>
      <div className="field-row">
        <div className="field">
          <label>Bridge</label>
          <select className="input" value={cfg.net} onChange={(e) => set('net', e.target.value)}>
            <option>bridge0 — 10.0.0.0/24 — DHCP</option>
            <option>dev-net — 10.99.0.0/24 — DHCP</option>
            <option>dmz — 10.10.0.0/24 — no DHCP</option>
          </select>
        </div>
        <div className="field">
          <label>VLAN tag (optional)</label>
          <input
            className="input input--mono"
            placeholder="e.g. 100"
            value={cfg.vlan}
            onChange={(e) => set('vlan', e.target.value)}
          />
        </div>
      </div>
      <div className="field-row">
        <div className="field">
          <label>MAC address</label>
          <input
            className="input input--mono"
            value={cfg.mac}
            onChange={(e) => set('mac', e.target.value)}
          />
          <div className="hint">Set "auto" for a stable random MAC.</div>
        </div>
        <div className="field">
          <label>Firewall</label>
          <Switch
            on={cfg.fw}
            onChange={(v) => set('fw', v)}
            label="Apply datacenter firewall to this NIC"
          />
          <div className="hint">
            Rules inherit from group. Override per-instance on detail page.
          </div>
        </div>
      </div>
      <div className="alert alert--info" style={{ marginTop: 14 }}>
        <I n="info" s={14} />
        <span>
          Multi-NIC setups and PCI passthrough are edited on the instance detail page after
          creation.
        </span>
      </div>
    </>
  );
}
function WizCloudInit({ cfg, set }) {
  return (
    <>
      <p className="muted" style={{ fontSize: 13, marginBottom: 14 }}>
        Cloud-init runs once on first boot. Leave blank to log in via console and set things up by
        hand.
      </p>
      <div className="field-row">
        <div className="field">
          <label>First username</label>
          <input
            className="input input--mono"
            value={cfg.username}
            onChange={(e) => set('username', e.target.value)}
          />
        </div>
        <div className="field">
          <label>SSH public key</label>
          <input
            className="input input--mono"
            placeholder="ssh-ed25519 AAAA… you@mac"
            value={cfg.ssh}
            onChange={(e) => set('ssh', e.target.value)}
          />
          <div className="hint">
            Or paste later from <span className="mono">rbac:admin</span>
          </div>
        </div>
      </div>
      <div className="field">
        <label>User-data (YAML)</label>
        <textarea
          className="input"
          rows="10"
          value={cfg.userData}
          onChange={(e) => set('userData', e.target.value)}
          placeholder={`#cloud-config
package_update: true
packages:
  - htop
  - vim
runcmd:
  - echo "hello from helling" > /etc/motd`}
        />
      </div>
    </>
  );
}
function WizReview({ cfg }) {
  return (
    <>
      <h3 className="stencil" style={{ fontSize: 14, margin: '0 0 10px' }}>
        CONFIRM & CREATE
      </h3>
      <dl className="desc">
        <dt>Type</dt>
        <dd>{cfg.type === 'VM' ? 'KVM Virtual Machine' : 'LXC Container'}</dd>
        <dt>Name</dt>
        <dd className="mono">{cfg.name}</dd>
        <dt>Node</dt>
        <dd className="mono">{cfg.node}</dd>
        <dt>Resources</dt>
        <dd className="mono">
          {cfg.cores} cores · {cfg.ram} GB RAM
        </dd>
        <dt>OS image</dt>
        <dd className="mono">{cfg.os}</dd>
        <dt>Disk</dt>
        <dd className="mono">
          {cfg.disk} GB on {cfg.pool} ({cfg.bus})
        </dd>
        <dt>Network</dt>
        <dd className="mono">
          {cfg.net}
          {cfg.vlan ? ` vlan ${cfg.vlan}` : ''} · MAC {cfg.mac} · fw {cfg.fw ? 'on' : 'off'}
        </dd>
        <dt>Cloud-init</dt>
        <dd>
          {cfg.username || '(none)'} · {cfg.ssh ? 'SSH key' : 'no SSH key'}
          {cfg.userData ? ' · user-data' : ''}
        </dd>
        <dt>Start</dt>
        <dd>{cfg.startAfter ? 'Boot after creation' : 'Leave stopped'}</dd>
      </dl>
      <div className="alert alert--info" style={{ marginTop: 18 }}>
        <I n="info" s={14} />
        <span>
          Equivalent CLI:{' '}
          <span className="mono">
            helling instance create {cfg.name} --type {cfg.type.toLowerCase()} --cores {cfg.cores}{' '}
            --ram {cfg.ram} --image {cfg.os} --disk {cfg.disk} --net {cfg.net}
          </span>
        </span>
      </div>
    </>
  );
}

// ─── CONSOLE VIEWER (noVNC + serial) ────────────────────────────
function PageConsole({ instance = 'web-prod-1', onNav }) {
  const [mode, setMode] = useState('graphical'); // graphical | serial
  const [fullscreen, setFullscreen] = useState(false);
  const [keymap, setKeymap] = useState('en-us');
  return (
    <div style={{ maxWidth: 1280, padding: '18px 20px' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-end',
          marginBottom: 12,
        }}
      >
        <div>
          <div className="eyebrow">DATACENTER / INSTANCES / {instance.toUpperCase()} / CONSOLE</div>
          <h1 className="stencil" style={{ fontSize: 24, margin: '4px 0 0' }}>
            Console ·{' '}
            <span className="mono" style={{ color: 'var(--h-text-2)' }}>
              {instance}
            </span>
          </h1>
        </div>
        <div style={{ display: 'flex', gap: 6 }}>
          <div className="seg">
            <button
              className={'seg__b ' + (mode === 'graphical' ? 'on' : '')}
              onClick={() => setMode('graphical')}
            >
              <I n="monitor" s={12} /> Graphical
            </button>
            <button
              className={'seg__b ' + (mode === 'serial' ? 'on' : '')}
              onClick={() => setMode('serial')}
            >
              <I n="terminal" s={12} /> Serial
            </button>
          </div>
          <button className="btn btn--sm">
            <I n="power" s={12} /> Send Ctrl-Alt-Del
          </button>
          <button className="btn btn--sm" onClick={() => setFullscreen((f) => !f)}>
            <I n={fullscreen ? 'minimize' : 'maximize'} s={12} />{' '}
            {fullscreen ? 'Exit' : 'Fullscreen'}
          </button>
        </div>
      </div>

      <div
        style={{ display: 'grid', gridTemplateColumns: fullscreen ? '1fr' : '1fr 280px', gap: 14 }}
      >
        {/* Canvas */}
        <div
          style={{
            background: '#000',
            border: '1px solid var(--h-border)',
            borderRadius: 'var(--h-radius)',
            overflow: 'hidden',
            aspectRatio: '16/9',
            position: 'relative',
          }}
        >
          {mode === 'graphical' ? <VNCCanvas /> : <SerialConsole />}
          {/* Toolbar overlay */}
          <div
            style={{
              position: 'absolute',
              top: 10,
              left: 10,
              display: 'flex',
              gap: 4,
              background: 'rgba(0,0,0,0.6)',
              padding: '4px 8px',
              borderRadius: 4,
              backdropFilter: 'blur(6px)',
              border: '1px solid rgba(255,255,255,0.1)',
            }}
          >
            <span className="mono" style={{ fontSize: 10, color: 'var(--h-accent)' }}>
              ● LIVE
            </span>
            <span className="mono dim" style={{ fontSize: 10 }}>
              · 1280×800 · {mode === 'graphical' ? 'VNC' : 'ttyS0 115200'}
            </span>
          </div>
          <div style={{ position: 'absolute', bottom: 10, right: 10, display: 'flex', gap: 4 }}>
            <button
              className="btn btn--sm"
              style={{ background: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(6px)' }}
            >
              <I n="camera" s={12} />
            </button>
            <button
              className="btn btn--sm"
              style={{ background: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(6px)' }}
            >
              <I n="clipboard" s={12} />
            </button>
          </div>
        </div>
        {/* Sidebar */}
        {!fullscreen && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <div className="card" style={{ padding: 12 }}>
              <div className="eyebrow" style={{ marginBottom: 8 }}>
                POWER
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 6 }}>
                <button className="btn btn--sm">
                  <I n="play" s={12} /> Start
                </button>
                <button className="btn btn--sm">
                  <I n="pause" s={12} /> Suspend
                </button>
                <button className="btn btn--sm">
                  <I n="refresh-cw" s={12} /> Reset
                </button>
                <button className="btn btn--sm" style={{ color: 'var(--h-danger)' }}>
                  <I n="power" s={12} /> Stop
                </button>
              </div>
            </div>
            <div className="card" style={{ padding: 12 }}>
              <div className="eyebrow" style={{ marginBottom: 8 }}>
                KEYBOARD
              </div>
              <div className="field" style={{ marginBottom: 8 }}>
                <label>Keymap</label>
                <select
                  className="input"
                  value={keymap}
                  onChange={(e) => setKeymap(e.target.value)}
                >
                  <option value="en-us">English (US)</option>
                  <option value="en-gb">English (UK)</option>
                  <option value="de">German</option>
                  <option value="fr">French</option>
                  <option value="nl">Dutch</option>
                </select>
              </div>
              <div className="eyebrow" style={{ marginBottom: 6 }}>
                SEND KEYS
              </div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                {[
                  'Ctrl-Alt-Del',
                  'Ctrl-Alt-F1',
                  'Ctrl-C',
                  'Ctrl-D',
                  'Esc',
                  'Tab',
                  'PrtSc',
                  'F2',
                ].map((k) => (
                  <button
                    key={k}
                    className="mono"
                    style={{
                      fontSize: 10,
                      padding: '4px 8px',
                      background: 'var(--h-bg-2)',
                      border: '1px solid var(--h-border)',
                      borderRadius: 3,
                      color: 'var(--h-text-2)',
                      cursor: 'pointer',
                    }}
                  >
                    {k}
                  </button>
                ))}
              </div>
            </div>
            <div className="card" style={{ padding: 12 }}>
              <div className="eyebrow" style={{ marginBottom: 8 }}>
                MEDIA
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <button className="btn btn--sm">
                  <I n="disc" s={12} /> Mount ISO…
                </button>
                <button className="btn btn--sm">
                  <I n="usb" s={12} /> Attach USB…
                </button>
                <button className="btn btn--sm">
                  <I n="upload" s={12} /> Upload file…
                </button>
              </div>
            </div>
            <div
              className="card"
              style={{
                padding: 12,
                background: 'rgba(198,255,36,0.03)',
                border: '1px solid rgba(198,255,36,0.2)',
              }}
            >
              <div className="eyebrow" style={{ color: 'var(--h-accent)', marginBottom: 6 }}>
                CLI EQUIVALENT
              </div>
              <code
                style={{
                  fontSize: 10,
                  color: 'var(--h-text-2)',
                  display: 'block',
                  wordBreak: 'break-all',
                }}
              >
                helling console {instance} --mode {mode}
              </code>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
function VNCCanvas() {
  // Stylized fake desktop
  return (
    <div
      style={{
        width: '100%',
        height: '100%',
        background: 'linear-gradient(135deg, #1a2b4a 0%, #0d1628 100%)',
        position: 'relative',
        display: 'flex',
        flexDirection: 'column',
        fontFamily: 'system-ui',
      }}
    >
      {/* top bar */}
      <div
        style={{
          height: 24,
          background: 'rgba(0,0,0,0.4)',
          display: 'flex',
          alignItems: 'center',
          padding: '0 10px',
          gap: 12,
          fontSize: 11,
          color: '#bbb',
        }}
      >
        <span>Activities</span>
        <span>·</span>
        <span>Terminal</span>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: 10 }}>
          <span>⏵</span>
          <span>🔊</span>
          <span>14:32</span>
        </div>
      </div>
      {/* windows */}
      <div style={{ flex: 1, position: 'relative', padding: 30 }}>
        <div
          style={{
            position: 'absolute',
            top: 40,
            left: 60,
            width: 420,
            height: 240,
            background: '#1a1a1a',
            border: '1px solid #333',
            borderRadius: 6,
            boxShadow: '0 20px 60px rgba(0,0,0,0.5)',
            overflow: 'hidden',
          }}
        >
          <div
            style={{
              height: 22,
              background: '#2a2a2a',
              padding: '4px 10px',
              fontSize: 10,
              color: '#aaa',
            }}
          >
            admin@web-prod-1: ~
          </div>
          <div
            style={{
              padding: 10,
              fontFamily: 'JetBrains Mono, monospace',
              fontSize: 10,
              color: '#c8e6c9',
              lineHeight: 1.6,
            }}
          >
            <div>admin@web-prod-1:~$ systemctl status nginx</div>
            <div style={{ color: '#81c784' }}>● nginx.service - A high performance web server</div>
            <div style={{ color: '#bbb' }}>
              {' '}
              Loaded: loaded (/lib/systemd/system/nginx.service; enabled)
            </div>
            <div style={{ color: '#81c784' }}>
              {' '}
              Active: active (running) since Tue 2026-03-18 08:14:22 UTC; 3 days ago
            </div>
            <div>&nbsp;</div>
            <div>
              admin@web-prod-1:~${' '}
              <span style={{ background: '#c8e6c9', color: '#000', padding: '0 3px' }}>_</span>
            </div>
          </div>
        </div>
        <div
          style={{
            position: 'absolute',
            top: 80,
            left: 380,
            width: 380,
            height: 220,
            background: 'rgba(255,255,255,0.95)',
            border: '1px solid #ccc',
            borderRadius: 6,
            boxShadow: '0 20px 60px rgba(0,0,0,0.5)',
            overflow: 'hidden',
          }}
        >
          <div
            style={{
              height: 22,
              background: '#eee',
              padding: '4px 10px',
              fontSize: 10,
              color: '#333',
              borderBottom: '1px solid #ddd',
            }}
          >
            about:helling
          </div>
          <div style={{ padding: 16, fontSize: 11, color: '#333' }}>
            <div style={{ fontWeight: 600, fontSize: 14, marginBottom: 8 }}>Debian 13 "Trixie"</div>
            <div style={{ color: '#666', lineHeight: 1.6 }}>
              Kernel 6.10.0-amd64
              <br />
              uptime 3 days, 4 hours
              <br />2 vCPU · 4 GB RAM
            </div>
          </div>
        </div>
      </div>
      {/* taskbar */}
      <div
        style={{
          height: 32,
          background: 'rgba(0,0,0,0.5)',
          display: 'flex',
          alignItems: 'center',
          padding: '0 10px',
          gap: 6,
        }}
      >
        {['📁', '🌐', '💻', '⚙'].map((x, i) => (
          <div
            key={i}
            style={{
              width: 22,
              height: 22,
              background: 'rgba(255,255,255,0.08)',
              borderRadius: 4,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 12,
            }}
          >
            {x}
          </div>
        ))}
      </div>
    </div>
  );
}
function SerialConsole() {
  const lines = [
    [
      '[    0.000000]',
      'Linux version 6.10.0-amd64 (debian@helling) (gcc-13) #1 SMP Debian 6.10.7-1',
    ],
    ['[    0.002140]', 'Command line: BOOT_IMAGE=/boot/vmlinuz-6.10.0-amd64 root=UUID=… ro quiet'],
    ['[    0.038120]', 'ACPI: Early table checksum verification disabled'],
    ['[    0.041002]', 'smpboot: CPU0: Intel(R) Xeon(R) Gold 6338 (family: 0x6, model: 0x6a)'],
    ['[    1.204118]', 'systemd[1]: Starting systemd 256.4 running in system mode'],
    ['[    1.410002]', 'systemd[1]: Reached target local-fs.target - Local File Systems'],
    ['[    1.680012]', 'systemd[1]: Started systemd-networkd.service'],
    [
      '[    1.912004]',
      'cloud-init[482]: Cloud-init v. 24.1 running "init-local" at Tue, 21 Mar 2026 14:31:54',
    ],
    [
      '[    2.104210]',
      'nginx[518]: nginx: the configuration file /etc/nginx/nginx.conf syntax is ok',
    ],
    [
      '[    2.330120]',
      'systemd[1]: Startup finished in 1.840s (kernel) + 320ms (initrd) + 1.012s (userspace)',
    ],
    ['', ''],
    ['', 'Debian GNU/Linux 13 web-prod-1 ttyS0'],
    ['', ''],
    ['', 'web-prod-1 login: '],
  ];
  return (
    <div
      style={{
        width: '100%',
        height: '100%',
        background: '#000',
        fontFamily: 'JetBrains Mono, monospace',
        fontSize: 11,
        color: '#d4d4d4',
        padding: '12px 16px',
        overflow: 'auto',
        lineHeight: 1.55,
      }}
    >
      {lines.map(([t, msg], i) => (
        <div key={i}>
          <span style={{ color: '#6a6a6a' }}>{t}</span> <span>{msg}</span>
        </div>
      ))}
      <div>
        <span style={{ background: '#c8e6c9', color: '#000' }}>&nbsp;</span>
      </div>
    </div>
  );
}

// ─── METRICS DASHBOARD ──────────────────────────────────────────
function PageMetrics({ onNav }) {
  const [range, setRange] = useState('6h');
  const [focus, setFocus] = useState(null);
  const RANGES = ['15m', '1h', '6h', '24h', '7d', '30d'];
  return (
    <div style={{ maxWidth: 1400, padding: '18px 20px' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-end',
          marginBottom: 14,
        }}
      >
        <div>
          <div className="eyebrow">OBSERVABILITY / METRICS</div>
          <h1
            className="stencil"
            style={{ fontSize: 26, margin: '4px 0 0', letterSpacing: '0.01em' }}
          >
            Metrics
          </h1>
          <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>
            Live system telemetry · scraped every 10 s · retained for 90 days
          </div>
        </div>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <div className="seg">
            {RANGES.map((r) => (
              <button
                key={r}
                className={'seg__b ' + (range === r ? 'on' : '')}
                onClick={() => setRange(r)}
              >
                {r}
              </button>
            ))}
          </div>
          <button className="btn btn--sm">
            <I n="calendar" s={12} /> Custom
          </button>
          <button className="btn btn--sm">
            <I n="refresh-cw" s={12} /> Refresh
          </button>
          <button className="btn btn--sm">
            <I n="download" s={12} /> Export
          </button>
        </div>
      </div>

      {/* Scope chips */}
      <div
        style={{
          display: 'flex',
          gap: 6,
          alignItems: 'center',
          marginBottom: 18,
          flexWrap: 'wrap',
        }}
      >
        <span className="eyebrow" style={{ marginRight: 4 }}>
          SCOPE
        </span>
        {[
          { l: 'All nodes', on: true },
          { l: 'node-1', on: false },
          { l: 'node-2', on: false },
          { l: 'node-3', on: false },
          { l: 'VMs only', on: false },
          { l: 'Containers only', on: false },
        ].map((c) => (
          <span key={c.l} className={'chip ' + (c.on ? 'chip--on' : '')}>
            {c.l} {c.on && <I n="x" s={10} />}
          </span>
        ))}
        <button className="btn btn--sm" style={{ marginLeft: 6 }}>
          <I n="plus" s={11} /> Add filter
        </button>
      </div>

      {/* KPI cards */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(4, 1fr)',
          gap: 10,
          marginBottom: 14,
        }}
      >
        {[
          { l: 'CPU', v: '34%', d: '+2.1%', k: 'cpu' },
          { l: 'Memory', v: '58%', d: '−0.8%', k: 'mem' },
          { l: 'Disk IO', v: '412 MB/s', d: '+18%', k: 'io' },
          { l: 'Network', v: '1.2 Gb/s', d: '+3%', k: 'net' },
        ].map((k) => (
          <div
            key={k.k}
            className="card"
            style={{
              padding: 14,
              cursor: 'pointer',
              borderColor: focus === k.k ? 'var(--h-accent)' : 'var(--h-border)',
            }}
            onClick={() => setFocus(k.k === focus ? null : k.k)}
          >
            <div className="eyebrow">{k.l}</div>
            <div className="stencil" style={{ fontSize: 26, margin: '4px 0 2px' }}>
              {k.v}
            </div>
            <div className="mono dim" style={{ fontSize: 10 }}>
              {k.d} vs last period
            </div>
            <MiniChart points={24} accent={focus === k.k} />
          </div>
        ))}
      </div>

      {/* Big charts */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
        <BigChart
          title="CPU utilization · by node"
          series={[
            { name: 'node-1', color: '#c6ff24' },
            { name: 'node-2', color: '#8a9bff' },
            { name: 'node-3', color: '#ff8aa9' },
          ]}
        />
        <BigChart
          title="Memory used · by node"
          series={[
            { name: 'node-1', color: '#c6ff24' },
            { name: 'node-2', color: '#8a9bff' },
            { name: 'node-3', color: '#ff8aa9' },
          ]}
        />
        <BigChart
          title="Disk IO · read vs write"
          series={[
            { name: 'read', color: '#8bffd4' },
            { name: 'write', color: '#ffd36a' },
          ]}
        />
        <BigChart
          title="Network throughput · in vs out"
          series={[
            { name: 'in', color: '#c6ff24' },
            { name: 'out', color: '#8a9bff' },
          ]}
        />
      </div>

      {/* Top consumers */}
      <div className="card" style={{ marginTop: 10, padding: 16 }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: 10,
          }}
        >
          <h3 className="stencil" style={{ fontSize: 14, margin: 0 }}>
            TOP CONSUMERS · CPU
          </h3>
          <div className="seg">
            <button className="seg__b on">CPU</button>
            <button className="seg__b">Memory</button>
            <button className="seg__b">Disk</button>
            <button className="seg__b">Net</button>
          </div>
        </div>
        <table className="tbl">
          <thead>
            <tr>
              <th>INSTANCE</th>
              <th>NODE</th>
              <th>USAGE</th>
              <th>AVG {range}</th>
              <th>PEAK</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {[
              ['db-primary', 'node-1', 78, 64, 92],
              ['web-prod-1', 'node-1', 42, 38, 71],
              ['ci-runner-3', 'node-2', 38, 24, 98],
              ['redis-cache', 'node-3', 18, 14, 22],
              ['monitoring', 'node-2', 12, 9, 18],
            ].map(([name, node, u, a, p]) => (
              <tr key={name}>
                <td>
                  <a className="link" onClick={() => onNav('instance:' + name)}>
                    {name}
                  </a>
                </td>
                <td className="mono">{node}</td>
                <td style={{ width: 240 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                    <div
                      style={{
                        flex: 1,
                        height: 6,
                        background: 'var(--h-bg-2)',
                        borderRadius: 3,
                        overflow: 'hidden',
                      }}
                    >
                      <div
                        style={{
                          width: u + '%',
                          height: '100%',
                          background:
                            u > 80
                              ? 'var(--h-danger)'
                              : u > 60
                                ? 'var(--h-warn)'
                                : 'var(--h-accent)',
                        }}
                      />
                    </div>
                    <span className="mono" style={{ fontSize: 11, width: 40 }}>
                      {u}%
                    </span>
                  </div>
                </td>
                <td className="mono">{a}%</td>
                <td className="mono">{p}%</td>
                <td>
                  <button className="btn btn--sm" onClick={() => onNav('instance:' + name)}>
                    Open →
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
function MiniChart({ points = 24, accent = false }) {
  const vals = useMemo(() => {
    let v = 40 + Math.random() * 20;
    return Array.from({ length: points }, (_, i) => {
      v += (Math.random() - 0.5) * 12;
      v = Math.max(10, Math.min(90, v));
      return v;
    });
  }, [points]);
  const d = vals
    .map((v, i) => (i === 0 ? 'M' : 'L') + (i / (points - 1)) * 100 + ' ' + (40 - (v / 100) * 36))
    .join(' ');
  const dArea = d + ` L 100 40 L 0 40 Z`;
  return (
    <svg
      viewBox="0 0 100 40"
      preserveAspectRatio="none"
      style={{ width: '100%', height: 40, marginTop: 8 }}
    >
      <path d={dArea} fill={accent ? 'rgba(198,255,36,0.2)' : 'rgba(255,255,255,0.04)'} />
      <path
        d={d}
        fill="none"
        stroke={accent ? 'var(--h-accent)' : 'var(--h-text-3)'}
        strokeWidth="1"
      />
    </svg>
  );
}
function BigChart({ title, series }) {
  const W = 800,
    H = 200;
  const data = useMemo(
    () =>
      series.map((s) => {
        let v = 30 + Math.random() * 30;
        return {
          ...s,
          pts: Array.from({ length: 60 }, () => {
            v += (Math.random() - 0.5) * 8;
            v = Math.max(5, Math.min(95, v));
            return v;
          }),
        };
      }),
    [series],
  );
  const path = (pts) =>
    pts
      .map(
        (v, i) =>
          (i === 0 ? 'M' : 'L') + (i / (pts.length - 1)) * W + ' ' + (H - (v / 100) * (H - 20)),
      )
      .join(' ');
  return (
    <div className="card" style={{ padding: 14 }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 8,
        }}
      >
        <h3 className="stencil" style={{ fontSize: 13, margin: 0 }}>
          {title}
        </h3>
        <div style={{ display: 'flex', gap: 12 }}>
          {data.map((s) => (
            <div
              key={s.name}
              style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 11 }}
            >
              <span
                style={{ width: 10, height: 2, background: s.color, display: 'inline-block' }}
              />
              <span className="mono dim">{s.name}</span>
            </div>
          ))}
        </div>
      </div>
      <svg
        viewBox={`0 0 ${W} ${H}`}
        preserveAspectRatio="none"
        style={{ width: '100%', height: H }}
      >
        {/* grid */}
        {[0, 0.25, 0.5, 0.75, 1].map((g) => (
          <line
            key={g}
            x1="0"
            x2={W}
            y1={g * (H - 20) + 10}
            y2={g * (H - 20) + 10}
            stroke="var(--h-border)"
            strokeWidth="0.5"
            strokeDasharray="2 3"
          />
        ))}
        {data.map((s) => (
          <g key={s.name}>
            <path d={path(s.pts) + ` L ${W} ${H} L 0 ${H} Z`} fill={s.color} opacity="0.08" />
            <path d={path(s.pts)} fill="none" stroke={s.color} strokeWidth="1.5" />
          </g>
        ))}
      </svg>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 4 }}>
        {['−6h', '−4h', '−2h', 'now'].map((t) => (
          <span key={t} className="mono dim" style={{ fontSize: 10 }}>
            {t}
          </span>
        ))}
      </div>
    </div>
  );
}

// ─── ALERTS ─────────────────────────────────────────────────────
function PageAlerts({ onNav }) {
  const [tab, setTab] = useState('rules');
  return (
    <div style={{ maxWidth: 1280, padding: '18px 20px' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-end',
          marginBottom: 14,
        }}
      >
        <div>
          <div className="eyebrow">OBSERVABILITY / ALERTS</div>
          <h1 className="stencil" style={{ fontSize: 26, margin: '4px 0 0' }}>
            Alerts
          </h1>
          <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>
            3 firing · 12 active rules · 4 channels configured
          </div>
        </div>
        <div style={{ display: 'flex', gap: 6 }}>
          <button className="btn btn--sm">
            <I n="bell-off" s={12} /> Silence all (1h)
          </button>
          <button className="btn btn--primary btn--sm">
            <I n="plus" s={12} /> New rule
          </button>
        </div>
      </div>

      <div className="tabs" style={{ marginBottom: 14 }}>
        {[
          ['rules', 'Rules (12)'],
          ['firing', 'Firing (3)'],
          ['channels', 'Channels (4)'],
          ['history', 'History'],
        ].map(([id, l]) => (
          <button
            key={id}
            className={'tab ' + (tab === id ? 'tab--on' : '')}
            onClick={() => setTab(id)}
          >
            {l}
          </button>
        ))}
      </div>

      {tab === 'firing' && <AlertsFiring />}
      {tab === 'rules' && <AlertsRules />}
      {tab === 'channels' && <AlertsChannels />}
      {tab === 'history' && <AlertsHistory />}
    </div>
  );
}
function AlertsFiring() {
  const [silenced, setSilenced] = useState(false);
  const rows = silenced
    ? []
    : [
        {
          sev: 'crit',
          rule: 'Disk > 90%',
          tgt: 'node-2 / rpool',
          fired: '4m ago',
          val: '92.1%',
          acker: null,
        },
        {
          sev: 'warn',
          rule: 'Backup failed',
          tgt: 'db-primary · nightly',
          fired: '1h 12m',
          val: 'exit 2',
          acker: 'alice@helling',
        },
        {
          sev: 'warn',
          rule: 'Replica lag > 30s',
          tgt: 'db-replica-eu',
          fired: '22m ago',
          val: '58s',
          acker: null,
        },
      ];
  if (rows.length === 0)
    return (
      <div className="card" style={{ padding: 40 }}>
        <EmptyState
          icon="shield-check"
          title="All clear"
          body={
            silenced
              ? 'All 3 firing alerts silenced for 1h. They will resume firing at 16:42 UTC unless resolved.'
              : 'No firing alerts. Rules green across 12 nodes, 48 instances, and 7 pools.'
          }
          action={
            silenced ? (
              <button className="btn btn--sm" onClick={() => setSilenced(false)}>
                Unsilence
              </button>
            ) : (
              <button className="btn btn--sm" onClick={() => setSilenced(true)}>
                Silence everything (1h)
              </button>
            )
          }
        />
      </div>
    );
  return (
    <div className="card">
      <div
        style={{
          padding: '8px 12px',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          borderBottom: '1px solid var(--h-border)',
        }}
      >
        <div className="mono dim" style={{ fontSize: 11 }}>
          {rows.length} firing · live tail
        </div>
        <button className="btn btn--sm" onClick={() => setSilenced(true)}>
          <I n="bell-off" s={11} /> Silence all (1h)
        </button>
      </div>
      <table className="tbl">
        <thead>
          <tr>
            <th>SEV</th>
            <th>RULE</th>
            <th>TARGET</th>
            <th>VALUE</th>
            <th>FIRED</th>
            <th>ACK</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={i}>
              <td>
                <span className={'dot dot--' + (r.sev === 'crit' ? 'danger' : 'warn')} />{' '}
                <span
                  className="mono"
                  style={{
                    fontSize: 10,
                    color: r.sev === 'crit' ? 'var(--h-danger)' : 'var(--h-warn)',
                  }}
                >
                  {r.sev.toUpperCase()}
                </span>
              </td>
              <td>{r.rule}</td>
              <td className="mono">{r.tgt}</td>
              <td className="mono">{r.val}</td>
              <td className="mono dim">{r.fired}</td>
              <td>
                {r.acker ? (
                  <span className="mono dim" style={{ fontSize: 10 }}>
                    ack · {r.acker}
                  </span>
                ) : (
                  <button className="btn btn--sm">Ack</button>
                )}
              </td>
              <td>
                <button className="btn btn--sm">
                  <I n="bell-off" s={11} /> Silence
                </button>{' '}
                <button className="btn btn--sm">Details</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
function AlertsRules() {
  const rules = [
    {
      name: 'CPU > 85% for 5m',
      for: 'all-nodes',
      sev: 'warn',
      ch: ['email', 'slack'],
      on: true,
      last: 'never',
    },
    {
      name: 'Memory > 90% for 2m',
      for: 'all-nodes',
      sev: 'crit',
      ch: ['email', 'slack', 'pagerduty'],
      on: true,
      last: '3d ago',
    },
    {
      name: 'Disk > 90%',
      for: 'pools',
      sev: 'crit',
      ch: ['email', 'pagerduty'],
      on: true,
      last: '4m ago (firing)',
    },
    {
      name: 'Backup failed',
      for: 'any-backup-job',
      sev: 'warn',
      ch: ['email', 'slack'],
      on: true,
      last: '1h ago (firing)',
    },
    {
      name: 'VM stopped unexpectedly',
      for: 'instances[auto-restart]',
      sev: 'warn',
      ch: ['slack'],
      on: true,
      last: '8d ago',
    },
    {
      name: 'Cluster quorum lost',
      for: 'cluster',
      sev: 'crit',
      ch: ['email', 'slack', 'pagerduty', 'sms'],
      on: true,
      last: 'never',
    },
    {
      name: 'Node unreachable > 30s',
      for: 'all-nodes',
      sev: 'crit',
      ch: ['email', 'slack', 'pagerduty'],
      on: true,
      last: 'never',
    },
    {
      name: 'ZFS scrub errors > 0',
      for: 'pools',
      sev: 'warn',
      ch: ['email'],
      on: true,
      last: '22d ago',
    },
    {
      name: 'Certificate expires < 14d',
      for: 'all-certs',
      sev: 'warn',
      ch: ['email'],
      on: true,
      last: 'never',
    },
    {
      name: 'Login failures > 10 / 5m',
      for: 'panel',
      sev: 'warn',
      ch: ['email', 'slack'],
      on: false,
      last: 'never',
    },
    {
      name: 'Replica lag > 30s',
      for: 'db-replica-*',
      sev: 'warn',
      ch: ['slack'],
      on: true,
      last: '22m ago (firing)',
    },
    {
      name: 'Container OOM killed',
      for: 'all-containers',
      sev: 'warn',
      ch: ['slack'],
      on: true,
      last: '2d ago',
    },
  ];
  return (
    <div className="card">
      <table className="tbl">
        <thead>
          <tr>
            <th>ENABLED</th>
            <th>NAME</th>
            <th>APPLIES TO</th>
            <th>SEV</th>
            <th>CHANNELS</th>
            <th>LAST TRIGGER</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {rules.map((r, i) => (
            <tr key={i}>
              <td>
                <Switch on={r.on} />
              </td>
              <td className="mono">{r.name}</td>
              <td className="mono dim">{r.for}</td>
              <td>
                <span
                  className="badge"
                  style={{
                    color: r.sev === 'crit' ? 'var(--h-danger)' : 'var(--h-warn)',
                    borderColor: r.sev === 'crit' ? 'var(--h-danger)' : 'var(--h-warn)',
                  }}
                >
                  {r.sev}
                </span>
              </td>
              <td>
                <div style={{ display: 'flex', gap: 4 }}>
                  {r.ch.map((c) => (
                    <span key={c} className="chip mono" style={{ fontSize: 9 }}>
                      {c}
                    </span>
                  ))}
                </div>
              </td>
              <td
                className="mono dim"
                style={{ color: r.last.includes('firing') ? 'var(--h-danger)' : 'var(--h-text-3)' }}
              >
                {r.last}
              </td>
              <td>
                <button className="btn btn--sm">Edit</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
function AlertsChannels() {
  const ch = [
    { type: 'email', name: 'Ops team', target: 'ops@company.tld', tested: '1 day ago' },
    {
      type: 'slack',
      name: '#helling-alerts',
      target: 'webhook → slack.com/T03…',
      tested: '8h ago',
    },
    {
      type: 'pagerduty',
      name: 'On-call rotation',
      target: 'service P12AB34',
      tested: '3 days ago',
    },
    {
      type: 'webhook',
      name: 'Internal audit',
      target: 'https://audit.company.tld/hooks/helling',
      tested: 'never',
    },
  ];
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
      {ch.map((c, i) => (
        <div key={i} className="card" style={{ padding: 14 }}>
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: 6,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <div
                style={{
                  width: 28,
                  height: 28,
                  background: 'var(--h-bg-2)',
                  border: '1px solid var(--h-border)',
                  borderRadius: 4,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <I
                  n={
                    c.type === 'email'
                      ? 'mail'
                      : c.type === 'slack'
                        ? 'message-square'
                        : c.type === 'pagerduty'
                          ? 'siren'
                          : 'webhook'
                  }
                  s={14}
                />
              </div>
              <div>
                <div style={{ fontSize: 13, fontWeight: 600 }}>{c.name}</div>
                <div className="mono dim" style={{ fontSize: 10 }}>
                  {c.type.toUpperCase()}
                </div>
              </div>
            </div>
            <div style={{ display: 'flex', gap: 4 }}>
              <button className="btn btn--sm">Test</button>
              <button className="btn btn--sm">Edit</button>
            </div>
          </div>
          <div
            className="mono dim"
            style={{
              fontSize: 11,
              marginBottom: 4,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {c.target}
          </div>
          <div className="mono dim" style={{ fontSize: 10 }}>
            Last tested: {c.tested}
          </div>
        </div>
      ))}
      <div
        className="card"
        style={{
          padding: 20,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          border: '1px dashed var(--h-border)',
          cursor: 'pointer',
          color: 'var(--h-text-2)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <I n="plus" s={14} /> <span style={{ fontSize: 13 }}>Add channel</span>
        </div>
      </div>
    </div>
  );
}
function AlertsHistory() {
  const h = [
    ['15:42:11', 'crit', 'Disk > 90%', 'node-2 / rpool', 'fired', ''],
    ['14:30:02', 'warn', 'Replica lag > 30s', 'db-replica-eu', 'fired', ''],
    ['14:28:40', 'warn', 'Backup failed', 'db-primary nightly', 'fired', ''],
    ['11:15:00', 'warn', 'Container OOM killed', 'ci-runner-3', 'resolved', 'auto'],
    ['09:02:18', 'crit', 'Node unreachable > 30s', 'node-3', 'resolved', 'auto'],
    ['08:45:00', 'info', 'Certificate renewed', 'panel.helling.io', 'info', ''],
  ];
  return (
    <div className="card">
      <table className="tbl">
        <thead>
          <tr>
            <th>TIME</th>
            <th>SEV</th>
            <th>RULE</th>
            <th>TARGET</th>
            <th>STATE</th>
            <th>NOTE</th>
          </tr>
        </thead>
        <tbody>
          {h.map((r, i) => (
            <tr key={i}>
              <td className="mono dim">{r[0]}</td>
              <td>
                <span
                  className="mono"
                  style={{
                    fontSize: 10,
                    color:
                      r[1] === 'crit'
                        ? 'var(--h-danger)'
                        : r[1] === 'warn'
                          ? 'var(--h-warn)'
                          : 'var(--h-text-3)',
                  }}
                >
                  {r[1].toUpperCase()}
                </span>
              </td>
              <td>{r[2]}</td>
              <td className="mono">{r[3]}</td>
              <td>
                <span
                  className="badge"
                  style={{
                    color:
                      r[4] === 'fired'
                        ? 'var(--h-danger)'
                        : r[4] === 'resolved'
                          ? 'var(--h-success)'
                          : 'var(--h-text-3)',
                  }}
                >
                  {r[4]}
                </span>
              </td>
              <td className="mono dim">{r[5]}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ─── RBAC DETAIL ────────────────────────────────────────────────
function PageRBAC({ onNav }) {
  const [tab, setTab] = useState('users');
  return (
    <div style={{ maxWidth: 1280, padding: '18px 20px' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-end',
          marginBottom: 14,
        }}
      >
        <div>
          <div className="eyebrow">ADMIN / ACCESS</div>
          <h1 className="stencil" style={{ fontSize: 26, margin: '4px 0 0' }}>
            Access control
          </h1>
          <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>
            12 users · 5 roles · 4 tokens · 8 SSH keys
          </div>
        </div>
        <div style={{ display: 'flex', gap: 6 }}>
          <button className="btn btn--sm">
            <I n="download" s={12} /> Export audit
          </button>
          <button className="btn btn--primary btn--sm">
            <I n="plus" s={12} /> Invite user
          </button>
        </div>
      </div>
      <div className="tabs" style={{ marginBottom: 14 }}>
        {[
          ['users', 'Users'],
          ['roles', 'Roles & matrix'],
          ['tokens', 'API tokens'],
          ['keys', 'SSH keys'],
          ['sso', 'SSO & policies'],
        ].map(([id, l]) => (
          <button
            key={id}
            className={'tab ' + (tab === id ? 'tab--on' : '')}
            onClick={() => setTab(id)}
          >
            {l}
          </button>
        ))}
      </div>
      {tab === 'users' && <RBACUsers onNav={onNav} />}
      {tab === 'roles' && <RBACMatrix />}
      {tab === 'tokens' && <RBACTokens />}
      {tab === 'keys' && <RBACKeys />}
      {tab === 'sso' && <RBACSSO />}
    </div>
  );
}
function RBACUsers({ onNav }) {
  const users = [
    { name: 'alice@helling', role: 'admin', mfa: true, last: '12m ago', status: 'active' },
    { name: 'bob@helling', role: 'operator', mfa: true, last: '2h ago', status: 'active' },
    { name: 'carol@helling', role: 'viewer', mfa: false, last: '3d ago', status: 'active' },
    { name: 'ci-bot', role: 'automation', mfa: false, last: '42s ago', status: 'active' },
    { name: 'dan@contractor', role: 'viewer', mfa: true, last: '14d ago', status: 'pending' },
  ];
  return (
    <div className="card">
      <table className="tbl">
        <thead>
          <tr>
            <th>
              <input type="checkbox" />
            </th>
            <th>USER</th>
            <th>ROLE</th>
            <th>MFA</th>
            <th>LAST SEEN</th>
            <th>STATUS</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {users.map((u, i) => (
            <tr
              key={i}
              onClick={() => onNav?.('rbac:' + u.name.split('@')[0])}
              style={{ cursor: 'pointer' }}
            >
              <td onClick={(e) => e.stopPropagation()}>
                <input type="checkbox" />
              </td>
              <td>
                <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                  <div
                    style={{
                      width: 26,
                      height: 26,
                      borderRadius: '50%',
                      background: `linear-gradient(135deg, hsl(${i * 67}, 50%, 50%), hsl(${i * 67 + 40}, 50%, 40%))`,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontWeight: 700,
                      fontSize: 11,
                    }}
                  >
                    {u.name[0].toUpperCase()}
                  </div>
                  <span className="mono" style={{ color: 'var(--h-accent)' }}>
                    {u.name}
                  </span>
                </div>
              </td>
              <td>
                <span className="chip chip--on">{u.role}</span>
              </td>
              <td>
                {u.mfa ? (
                  <span style={{ color: 'var(--h-success)' }}>
                    <I n="shield-check" s={14} />
                  </span>
                ) : (
                  <span style={{ color: 'var(--h-warn)' }}>
                    <I n="shield-off" s={14} />
                  </span>
                )}
              </td>
              <td className="mono dim">{u.last}</td>
              <td>
                <span
                  className="badge"
                  style={{
                    color: u.status === 'active' ? 'var(--h-success)' : 'var(--h-warn)',
                    borderColor: u.status === 'active' ? 'var(--h-success)' : 'var(--h-warn)',
                  }}
                >
                  {u.status}
                </span>
              </td>
              <td>
                <button className="btn btn--sm">Edit</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
function RBACMatrix() {
  const roles = ['admin', 'operator', 'dev', 'viewer', 'automation'];
  const perms = [
    ['Instances', ['create', 'start/stop', 'delete', 'console', 'migrate']],
    ['Storage', ['create pool', 'resize disk', 'destroy', 'snapshot', 'restore']],
    ['Network', ['create bridge', 'edit firewall', 'load balancer', 'vpn']],
    ['Cluster', ['add node', 'remove node', 'edit quorum', 'join token']],
    ['Admin', ['manage users', 'view audit', 'change settings', 'rotate secrets']],
  ];
  const grants = {
    admin: { all: true },
    operator: {
      Instances: ['create', 'start/stop', 'console', 'migrate'],
      Storage: ['resize disk', 'snapshot', 'restore'],
      Network: ['edit firewall'],
      Cluster: [],
    },
    dev: { Instances: ['create', 'start/stop', 'console'], Storage: ['snapshot'] },
    viewer: { Instances: ['console'] },
    automation: { Instances: ['create', 'start/stop', 'delete'], Storage: ['snapshot', 'restore'] },
  };
  const has = (role, group, p) => {
    const g = grants[role];
    if (g?.all) return true;
    return g?.[group]?.includes(p);
  };
  return (
    <div className="card" style={{ padding: 14, overflow: 'auto' }}>
      <table className="tbl" style={{ width: '100%' }}>
        <thead>
          <tr>
            <th style={{ width: 200 }}>PERMISSION</th>
            {roles.map((r) => (
              <th key={r} style={{ textAlign: 'center', textTransform: 'uppercase' }}>
                {r}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {perms.map(([group, list]) => (
            <React.Fragment key={group}>
              <tr style={{ background: 'var(--h-bg-2)' }}>
                <td
                  colSpan={roles.length + 1}
                  className="mono"
                  style={{
                    fontSize: 10,
                    letterSpacing: '0.1em',
                    padding: '6px 12px',
                    color: 'var(--h-text-3)',
                  }}
                >
                  {group.toUpperCase()}
                </td>
              </tr>
              {list.map((p) => (
                <tr key={p}>
                  <td style={{ paddingLeft: 20 }}>{p}</td>
                  {roles.map((r) => (
                    <td key={r} style={{ textAlign: 'center' }}>
                      {has(r, group, p) ? (
                        <I n="check" s={14} style={{ color: 'var(--h-accent)' }} />
                      ) : (
                        <span style={{ color: 'var(--h-text-3)' }}>—</span>
                      )}
                    </td>
                  ))}
                </tr>
              ))}
            </React.Fragment>
          ))}
        </tbody>
      </table>
    </div>
  );
}
function RBACTokens() {
  const tokens = [
    {
      name: 'terraform-ci',
      scope: 'instances:write',
      last: '42s ago',
      created: '2025-11-02',
      expires: '2026-11-02',
    },
    {
      name: 'backup-script',
      scope: 'backup:read',
      last: '3m ago',
      created: '2025-08-14',
      expires: 'never',
    },
    {
      name: 'grafana-reader',
      scope: 'metrics:read',
      last: '12s ago',
      created: '2026-01-08',
      expires: '2026-07-08',
    },
    {
      name: 'old-ci (disabled)',
      scope: 'instances:write',
      last: '48d ago',
      created: '2024-12-01',
      expires: 'expired',
    },
  ];
  return (
    <div className="card">
      <table className="tbl">
        <thead>
          <tr>
            <th>NAME</th>
            <th>SCOPE</th>
            <th>LAST USED</th>
            <th>CREATED</th>
            <th>EXPIRES</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {tokens.map((t, i) => (
            <tr key={i}>
              <td className="mono">{t.name}</td>
              <td>
                <span className="chip">{t.scope}</span>
              </td>
              <td className="mono dim">{t.last}</td>
              <td className="mono dim">{t.created}</td>
              <td
                className="mono"
                style={{ color: t.expires === 'expired' ? 'var(--h-danger)' : 'var(--h-text-2)' }}
              >
                {t.expires}
              </td>
              <td>
                <button className="btn btn--sm">Rotate</button>{' '}
                <button className="btn btn--sm" style={{ color: 'var(--h-danger)' }}>
                  Revoke
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
function RBACKeys() {
  const keys = [
    {
      fp: 'SHA256:3n7g…aB9+',
      type: 'ed25519',
      user: 'alice@helling',
      comment: 'alice@macbook',
      added: '2025-11-02',
    },
    {
      fp: 'SHA256:wL4k…Cv1x',
      type: 'ed25519',
      user: 'bob@helling',
      comment: 'bob@thinkpad',
      added: '2025-09-14',
    },
    {
      fp: 'SHA256:Ym3p…Dq7e',
      type: 'rsa-4096',
      user: 'ci-bot',
      comment: 'ci-runner',
      added: '2024-02-10',
    },
  ];
  return (
    <div className="card">
      <table className="tbl">
        <thead>
          <tr>
            <th>FINGERPRINT</th>
            <th>TYPE</th>
            <th>USER</th>
            <th>COMMENT</th>
            <th>ADDED</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {keys.map((k, i) => (
            <tr key={i}>
              <td className="mono" style={{ fontSize: 10 }}>
                {k.fp}
              </td>
              <td>
                <span className="chip mono">{k.type}</span>
              </td>
              <td className="mono">{k.user}</td>
              <td className="mono dim">{k.comment}</td>
              <td className="mono dim">{k.added}</td>
              <td>
                <button className="btn btn--sm" style={{ color: 'var(--h-danger)' }}>
                  Remove
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
function RBACSSO() {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
      <div className="card" style={{ padding: 14 }}>
        <h3 className="stencil" style={{ fontSize: 13, margin: '0 0 10px' }}>
          SSO PROVIDER
        </h3>
        <div className="field">
          <label>Provider</label>
          <select className="input">
            <option>OIDC — generic</option>
            <option>Google Workspace</option>
            <option>Microsoft Entra ID</option>
            <option>Okta</option>
            <option>Authentik</option>
          </select>
        </div>
        <div className="field">
          <label>Issuer URL</label>
          <input
            className="input input--mono"
            defaultValue="https://auth.company.tld/application/o/helling/"
          />
        </div>
        <div className="field">
          <label>Client ID</label>
          <input className="input input--mono" defaultValue="helling-panel" />
        </div>
        <div className="field">
          <label>Client secret</label>
          <input className="input input--mono" type="password" defaultValue="••••••••••••••••" />
        </div>
        <Switch on={true} label="Auto-create users on first login" />
      </div>
      <div className="card" style={{ padding: 14 }}>
        <h3 className="stencil" style={{ fontSize: 13, margin: '0 0 10px' }}>
          SECURITY POLICY
        </h3>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <Switch on={true} label="Require MFA for all users" />
          <Switch on={true} label="Require MFA for admin role" />
          <Switch on={false} label="Require hardware keys (WebAuthn)" />
          <Switch on={true} label="Log out idle sessions after 8 hours" />
          <Switch on={false} label="Restrict panel to VPN/LAN" />
          <Switch on={true} label="Block logins from new countries (email confirm)" />
        </div>
        <div className="field" style={{ marginTop: 14 }}>
          <label>Allowed email domains</label>
          <input className="input input--mono" defaultValue="company.tld, helling.io" />
        </div>
      </div>
    </div>
  );
}

// ─── FIREWALL EDITOR ────────────────────────────────────────────
function PageFirewallEditor({ onNav }) {
  const [rules, setRules] = useState([
    {
      id: 1,
      on: true,
      action: 'allow',
      dir: 'in',
      proto: 'tcp',
      src: '0.0.0.0/0',
      dport: '80',
      note: 'HTTP',
      hits: '12.4M',
    },
    {
      id: 2,
      on: true,
      action: 'allow',
      dir: 'in',
      proto: 'tcp',
      src: '0.0.0.0/0',
      dport: '443',
      note: 'HTTPS',
      hits: '48.1M',
    },
    {
      id: 3,
      on: true,
      action: 'allow',
      dir: 'in',
      proto: 'tcp',
      src: '10.0.0.0/8',
      dport: '22',
      note: 'SSH (LAN)',
      hits: '4.8K',
    },
    {
      id: 4,
      on: true,
      action: 'allow',
      dir: 'in',
      proto: 'icmp',
      src: '0.0.0.0/0',
      dport: '*',
      note: 'Ping',
      hits: '91K',
    },
    {
      id: 5,
      on: true,
      action: 'deny',
      dir: 'in',
      proto: 'tcp',
      src: '0.0.0.0/0',
      dport: '22',
      note: 'Block SSH from WAN',
      hits: '2.1M',
    },
    {
      id: 6,
      on: false,
      action: 'allow',
      dir: 'in',
      proto: 'tcp',
      src: '10.0.0.0/24',
      dport: '5432',
      note: 'Postgres (disabled)',
      hits: '0',
    },
    {
      id: 7,
      on: true,
      action: 'deny',
      dir: 'in',
      proto: '*',
      src: '0.0.0.0/0',
      dport: '*',
      note: 'Default deny',
      hits: '38M',
    },
  ]);
  const [selected, setSelected] = useState(null);
  const move = (id, dir) => {
    setRules((rs) => {
      const i = rs.findIndex((r) => r.id === id);
      const j = i + dir;
      if (j < 0 || j >= rs.length) return rs;
      const copy = [...rs];
      [copy[i], copy[j]] = [copy[j], copy[i]];
      return copy;
    });
  };

  const rule = selected ? rules.find((r) => r.id === selected) : null;

  return (
    <div style={{ maxWidth: 1400, padding: '18px 20px' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-end',
          marginBottom: 14,
        }}
      >
        <div>
          <div className="eyebrow" style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <span
              style={{ cursor: 'pointer' }}
              onClick={() => onNav?.('firewall')}
              title="Back to basic view"
            >
              NETWORKING / FIREWALL
            </span>
            <span>/ DATACENTER / ADVANCED</span>
          </div>
          <h1 className="stencil" style={{ fontSize: 26, margin: '4px 0 0' }}>
            Firewall editor
          </h1>
          <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>
            Applied to all instances unless overridden. Rules evaluated top-to-bottom.{' '}
            <a
              onClick={() => onNav?.('firewall')}
              style={{ color: 'var(--h-accent)', cursor: 'pointer', textDecoration: 'underline' }}
            >
              Switch to basic view →
            </a>
          </div>
        </div>
        <div style={{ display: 'flex', gap: 6 }}>
          <button className="btn btn--sm">
            <I n="upload" s={12} /> Import
          </button>
          <button className="btn btn--sm">
            <I n="download" s={12} /> Export YAML
          </button>
          <button className="btn btn--primary btn--sm">
            <I n="plus" s={12} /> Add rule
          </button>
        </div>
      </div>

      <div className="alert alert--warn" style={{ marginBottom: 14 }}>
        <I n="triangle-alert" s={14} />
        <span>
          You have 2 unsaved changes. Click "Apply" to activate, or changes will be discarded in 15
          minutes.
        </span>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: 6 }}>
          <button className="btn btn--sm">Discard</button>
          <button className="btn btn--sm btn--primary">Apply</button>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 340px', gap: 12 }}>
        <div className="card" style={{ overflow: 'hidden' }}>
          <table className="tbl">
            <thead>
              <tr>
                <th style={{ width: 32 }}></th>
                <th style={{ width: 50 }}>#</th>
                <th style={{ width: 44 }}>ON</th>
                <th style={{ width: 80 }}>ACTION</th>
                <th style={{ width: 60 }}>DIR</th>
                <th style={{ width: 80 }}>PROTO</th>
                <th>SOURCE</th>
                <th>PORT</th>
                <th>NOTE</th>
                <th>HITS</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {rules.map((r, i) => (
                <tr
                  key={r.id}
                  className={(selected === r.id ? 'sel ' : '') + (!r.on ? 'off ' : '')}
                  onClick={() => setSelected(r.id)}
                  style={{ cursor: 'pointer', opacity: r.on ? 1 : 0.5 }}
                >
                  <td style={{ color: 'var(--h-text-3)', cursor: 'grab' }}>
                    <I n="grip-vertical" s={14} />
                  </td>
                  <td className="mono dim">{i + 1}</td>
                  <td>
                    <Switch on={r.on} />
                  </td>
                  <td>
                    <span
                      style={{
                        display: 'inline-block',
                        padding: '2px 8px',
                        borderRadius: 3,
                        fontSize: 10,
                        fontWeight: 600,
                        letterSpacing: '0.05em',
                        color: r.action === 'allow' ? '#000' : '#fff',
                        background: r.action === 'allow' ? 'var(--h-accent)' : 'var(--h-danger)',
                      }}
                    >
                      {r.action.toUpperCase()}
                    </span>
                  </td>
                  <td className="mono" style={{ fontSize: 10 }}>
                    {r.dir.toUpperCase()}
                  </td>
                  <td className="mono">{r.proto}</td>
                  <td className="mono dim">{r.src}</td>
                  <td className="mono">{r.dport}</td>
                  <td style={{ fontSize: 12 }}>{r.note}</td>
                  <td className="mono dim" style={{ fontSize: 10 }}>
                    {r.hits}
                  </td>
                  <td style={{ whiteSpace: 'nowrap' }}>
                    <button
                      className="btn btn--sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        move(r.id, -1);
                      }}
                    >
                      <I n="arrow-up" s={11} />
                    </button>
                    <button
                      className="btn btn--sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        move(r.id, 1);
                      }}
                    >
                      <I n="arrow-down" s={11} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Right panel — rule editor */}
        <div>
          {rule ? (
            <FWRuleEditor rule={rule} />
          ) : (
            <div
              className="card"
              style={{ padding: 20, textAlign: 'center', color: 'var(--h-text-3)' }}
            >
              <I n="shield" s={32} style={{ color: 'var(--h-text-3)', marginBottom: 8 }} />
              <div style={{ fontSize: 13, marginBottom: 4 }}>No rule selected</div>
              <div className="muted" style={{ fontSize: 12 }}>
                Click a row to edit its details here.
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
function FWRuleEditor({ rule }) {
  return (
    <div className="card" style={{ padding: 14, position: 'sticky', top: 80 }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 10,
        }}
      >
        <h3 className="stencil" style={{ fontSize: 13, margin: 0 }}>
          RULE #{rule.id}
        </h3>
        <span className="badge">draft</span>
      </div>
      <div className="field">
        <label>Note</label>
        <input className="input" defaultValue={rule.note} />
      </div>
      <div className="field-row">
        <div className="field">
          <label>Action</label>
          <div className="seg" style={{ width: '100%' }}>
            <button
              className={'seg__b ' + (rule.action === 'allow' ? 'on' : '')}
              style={{ flex: 1 }}
            >
              Allow
            </button>
            <button
              className={'seg__b ' + (rule.action === 'deny' ? 'on' : '')}
              style={{ flex: 1 }}
            >
              Deny
            </button>
            <button className="seg__b" style={{ flex: 1 }}>
              Log
            </button>
          </div>
        </div>
        <div className="field">
          <label>Direction</label>
          <div className="seg" style={{ width: '100%' }}>
            <button className={'seg__b ' + (rule.dir === 'in' ? 'on' : '')} style={{ flex: 1 }}>
              In
            </button>
            <button className={'seg__b ' + (rule.dir === 'out' ? 'on' : '')} style={{ flex: 1 }}>
              Out
            </button>
          </div>
        </div>
      </div>
      <div className="field">
        <label>Protocol</label>
        <select className="input" defaultValue={rule.proto}>
          <option>*</option>
          <option>tcp</option>
          <option>udp</option>
          <option>icmp</option>
          <option>icmpv6</option>
        </select>
      </div>
      <div className="field-row">
        <div className="field">
          <label>Source</label>
          <input className="input input--mono" defaultValue={rule.src} />
        </div>
        <div className="field">
          <label>Source port</label>
          <input className="input input--mono" defaultValue="*" />
        </div>
      </div>
      <div className="field-row">
        <div className="field">
          <label>Destination</label>
          <input className="input input--mono" defaultValue="any" />
        </div>
        <div className="field">
          <label>Dest. port</label>
          <input className="input input--mono" defaultValue={rule.dport} />
        </div>
      </div>
      <div className="field">
        <label>Options</label>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          <Switch on={true} label="Log matches to audit" />
          <Switch on={false} label="Rate limit (10 req/s)" />
          <Switch on={false} label="Stateful connection tracking" />
        </div>
      </div>
      <div style={{ display: 'flex', gap: 6, marginTop: 14 }}>
        <button className="btn btn--sm" style={{ flex: 1, color: 'var(--h-danger)' }}>
          <I n="trash-2" s={12} /> Delete
        </button>
        <button className="btn btn--primary btn--sm" style={{ flex: 1 }}>
          Save rule
        </button>
      </div>
    </div>
  );
}

// ─── MARKETPLACE ────────────────────────────────────────────────
function PageMarketplace({ onNav }) {
  const [cat, setCat] = useState('featured');
  const apps = [
    {
      id: 'nextcloud',
      name: 'Nextcloud',
      tag: 'Files, calendar, contacts',
      stars: '★ 4.9',
      users: '45k',
      cat: 'featured productivity',
      logo: '☁️',
    },
    {
      id: 'gitea',
      name: 'Gitea',
      tag: 'Self-hosted Git',
      stars: '★ 4.8',
      users: '32k',
      cat: 'featured dev',
      logo: '🫱',
    },
    {
      id: 'plex',
      name: 'Plex',
      tag: 'Media server',
      stars: '★ 4.7',
      users: '28k',
      cat: 'featured media',
      logo: '▶️',
    },
    {
      id: 'homeassist',
      name: 'Home Assistant',
      tag: 'Smart home',
      stars: '★ 4.9',
      users: '39k',
      cat: 'featured home',
      logo: '🏠',
    },
    {
      id: 'grafana',
      name: 'Grafana',
      tag: 'Observability dashboards',
      stars: '★ 4.8',
      users: '41k',
      cat: 'featured dev observability',
      logo: '📊',
    },
    {
      id: 'postgres',
      name: 'PostgreSQL 16',
      tag: 'Relational database',
      stars: '★ 4.9',
      users: '67k',
      cat: 'featured db',
      logo: '🐘',
    },
    {
      id: 'redis',
      name: 'Redis 7',
      tag: 'In-memory store',
      stars: '★ 4.8',
      users: '52k',
      cat: 'db',
      logo: '🟥',
    },
    {
      id: 'traefik',
      name: 'Traefik',
      tag: 'Reverse proxy',
      stars: '★ 4.7',
      users: '22k',
      cat: 'net',
      logo: '🔀',
    },
    {
      id: 'mail',
      name: 'Mailcow',
      tag: 'Full mail server',
      stars: '★ 4.6',
      users: '11k',
      cat: 'mail',
      logo: '✉️',
    },
    {
      id: 'wg',
      name: 'WireGuard VPN',
      tag: 'Zero-conf VPN',
      stars: '★ 4.9',
      users: '34k',
      cat: 'net',
      logo: '🌐',
    },
    {
      id: 'jellyfin',
      name: 'Jellyfin',
      tag: 'Media server',
      stars: '★ 4.7',
      users: '18k',
      cat: 'media',
      logo: '🎬',
    },
    {
      id: 'paperless',
      name: 'Paperless-ngx',
      tag: 'Document archive',
      stars: '★ 4.8',
      users: '14k',
      cat: 'productivity',
      logo: '📄',
    },
    {
      id: 'minio',
      name: 'MinIO',
      tag: 'S3-compatible object store',
      stars: '★ 4.9',
      users: '24k',
      cat: 'db storage',
      logo: '🪣',
    },
    {
      id: 'vault',
      name: 'Vaultwarden',
      tag: 'Password manager',
      stars: '★ 4.9',
      users: '38k',
      cat: 'productivity',
      logo: '🔐',
    },
  ];
  const CATS = [
    ['featured', 'Featured'],
    ['dev', 'Developer'],
    ['productivity', 'Productivity'],
    ['media', 'Media'],
    ['home', 'Smart home'],
    ['db', 'Databases'],
    ['net', 'Networking'],
    ['mail', 'Mail'],
  ];
  const filtered =
    cat === 'featured'
      ? apps.filter((a) => a.cat.includes('featured'))
      : apps.filter((a) => a.cat.includes(cat));
  return (
    <div style={{ maxWidth: 1280, padding: '18px 20px' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-end',
          marginBottom: 14,
        }}
      >
        <div>
          <div className="eyebrow">LIBRARY / MARKETPLACE</div>
          <h1 className="stencil" style={{ fontSize: 26, margin: '4px 0 0' }}>
            Marketplace
          </h1>
          <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>
            One-click apps · fully containerized · signed by maintainers
          </div>
        </div>
        <div style={{ display: 'flex', gap: 6 }}>
          <div style={{ position: 'relative' }}>
            <I
              n="search"
              s={14}
              style={{
                position: 'absolute',
                left: 10,
                top: '50%',
                transform: 'translateY(-50%)',
                color: 'var(--h-text-3)',
              }}
            />
            <input
              className="input"
              placeholder="Search 240+ apps…"
              style={{ paddingLeft: 32, width: 260 }}
            />
          </div>
        </div>
      </div>

      <div style={{ display: 'flex', gap: 6, marginBottom: 14, flexWrap: 'wrap' }}>
        {CATS.map(([id, l]) => (
          <span
            key={id}
            className={'chip ' + (cat === id ? 'chip--on' : '')}
            onClick={() => setCat(id)}
            style={{ cursor: 'pointer' }}
          >
            {l}
          </span>
        ))}
      </div>

      {/* Hero */}
      {cat === 'featured' && (
        <div
          className="card"
          style={{
            padding: 24,
            marginBottom: 14,
            background: 'linear-gradient(135deg, rgba(198,255,36,0.08), rgba(198,255,36,0.01))',
            border: '1px solid rgba(198,255,36,0.3)',
            position: 'relative',
            overflow: 'hidden',
          }}
        >
          <div
            style={{
              position: 'absolute',
              top: -40,
              right: -40,
              width: 180,
              height: 180,
              borderRadius: '50%',
              background: 'radial-gradient(circle, rgba(198,255,36,0.15), transparent 70%)',
            }}
          />
          <div className="eyebrow" style={{ color: 'var(--h-accent)', marginBottom: 8 }}>
            ▲ STAFF PICK · NEW THIS WEEK
          </div>
          <h2 className="stencil" style={{ fontSize: 28, margin: '0 0 6px' }}>
            Nextcloud + Onlyoffice stack
          </h2>
          <p className="muted" style={{ maxWidth: 520, marginBottom: 12, fontSize: 13 }}>
            Pre-wired files, calendar, contacts, and collaborative docs. Deploys as two containers
            behind Traefik with automatic Let's Encrypt.
          </p>
          <div style={{ display: 'flex', gap: 6 }}>
            <button className="btn btn--primary btn--sm">
              <I n="download" s={12} /> One-click install
            </button>
            <button className="btn btn--sm">View source</button>
          </div>
        </div>
      )}

      {/* Grid */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))',
          gap: 10,
        }}
      >
        {filtered.map((a) => (
          <div key={a.id} className="card" style={{ padding: 14, cursor: 'pointer' }}>
            <div style={{ display: 'flex', gap: 10, marginBottom: 8 }}>
              <div
                style={{
                  width: 40,
                  height: 40,
                  background: 'var(--h-bg-2)',
                  border: '1px solid var(--h-border)',
                  borderRadius: 6,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 22,
                }}
              >
                {a.logo}
              </div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontWeight: 600, fontSize: 13 }}>{a.name}</div>
                <div
                  className="muted"
                  style={{
                    fontSize: 11,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {a.tag}
                </div>
              </div>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div className="mono dim" style={{ fontSize: 10 }}>
                {a.stars} · {a.users} installs
              </div>
              <button className="btn btn--sm btn--primary">Install</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// ─── FILE BROWSER ───────────────────────────────────────────────
function PageFileBrowser({ target = 'web-prod-1', onNav }) {
  const [path, setPath] = useState(['home', 'admin']);
  const [selected, setSelected] = useState(null);
  const tree = {
    '': [
      { name: 'bin', type: 'dir', size: '', mtime: '2026-03-01' },
      { name: 'etc', type: 'dir', size: '', mtime: '2026-03-20' },
      { name: 'home', type: 'dir', size: '', mtime: '2026-03-18' },
      { name: 'var', type: 'dir', size: '', mtime: '2026-03-21' },
      { name: 'usr', type: 'dir', size: '', mtime: '2026-01-15' },
    ],
    home: [
      { name: 'admin', type: 'dir', size: '', mtime: '2026-03-21' },
      { name: 'ci-bot', type: 'dir', size: '', mtime: '2026-03-19' },
    ],
    'home/admin': [
      { name: '.bashrc', type: 'file', size: '3.8 KB', mtime: '2026-03-18' },
      { name: '.ssh', type: 'dir', size: '', mtime: '2026-03-18' },
      { name: 'scripts', type: 'dir', size: '', mtime: '2026-03-20' },
      { name: 'backups', type: 'dir', size: '', mtime: '2026-03-21' },
      { name: 'README.md', type: 'file', size: '1.2 KB', mtime: '2026-03-14' },
      { name: 'deploy.yaml', type: 'file', size: '4.1 KB', mtime: '2026-03-21' },
      { name: 'db-dump-2026-03-20.sql.gz', type: 'file', size: '612 MB', mtime: '2026-03-20' },
      { name: 'snapshot-pre-migrate.tar', type: 'file', size: '4.2 GB', mtime: '2026-03-18' },
    ],
  };
  const current = tree[path.join('/')] || [];
  const goUp = () => setPath((p) => p.slice(0, -1));
  const enter = (name, type) => {
    if (type === 'dir') setPath((p) => [...p, name]);
    else setSelected(name);
  };
  return (
    <div style={{ maxWidth: 1280, padding: '18px 20px' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-end',
          marginBottom: 14,
        }}
      >
        <div>
          <div className="eyebrow">DATACENTER / INSTANCES / {target.toUpperCase()} / FILES</div>
          <h1 className="stencil" style={{ fontSize: 24, margin: '4px 0 0' }}>
            Files ·{' '}
            <span className="mono" style={{ color: 'var(--h-text-2)' }}>
              {target}
            </span>
          </h1>
        </div>
        <div style={{ display: 'flex', gap: 6 }}>
          <button className="btn btn--sm">
            <I n="upload" s={12} /> Upload
          </button>
          <button className="btn btn--sm">
            <I n="folder-plus" s={12} /> New folder
          </button>
          <button className="btn btn--sm">
            <I n="terminal" s={12} /> Open shell
          </button>
        </div>
      </div>

      {/* Path bar */}
      <div
        className="card"
        style={{
          padding: '10px 14px',
          marginBottom: 8,
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          fontFamily: 'JetBrains Mono, monospace',
          fontSize: 12,
        }}
      >
        <button className="btn btn--sm" onClick={goUp} disabled={path.length === 0}>
          <I n="arrow-up" s={12} />
        </button>
        <span style={{ color: 'var(--h-text-3)' }}>/</span>
        {path.map((p, i) => (
          <React.Fragment key={i}>
            <a className="link" onClick={() => setPath(path.slice(0, i + 1))}>
              {p}
            </a>
            <span style={{ color: 'var(--h-text-3)' }}>/</span>
          </React.Fragment>
        ))}
        <div style={{ marginLeft: 'auto', display: 'flex', gap: 6 }}>
          <input
            className="input input--mono"
            style={{ padding: '4px 8px', fontSize: 11, width: 200 }}
            placeholder="Search files…"
          />
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 320px', gap: 10 }}>
        <div className="card">
          <table className="tbl">
            <thead>
              <tr>
                <th style={{ width: 30 }}>
                  <input type="checkbox" />
                </th>
                <th>NAME</th>
                <th style={{ width: 100 }}>SIZE</th>
                <th style={{ width: 120 }}>MODIFIED</th>
                <th style={{ width: 80 }}>PERMS</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {current.map((f) => (
                <tr
                  key={f.name}
                  className={selected === f.name ? 'sel' : ''}
                  onClick={() => enter(f.name, f.type)}
                  onDoubleClick={() => enter(f.name, f.type)}
                  style={{ cursor: 'pointer' }}
                >
                  <td>
                    <input type="checkbox" onClick={(e) => e.stopPropagation()} />
                  </td>
                  <td>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <I
                        n={
                          f.type === 'dir'
                            ? 'folder'
                            : f.name.endsWith('.sql.gz') || f.name.endsWith('.tar')
                              ? 'archive'
                              : f.name.endsWith('.yaml') || f.name.endsWith('.md')
                                ? 'file-text'
                                : 'file'
                        }
                        s={14}
                        style={{ color: f.type === 'dir' ? 'var(--h-accent)' : 'var(--h-text-3)' }}
                      />
                      <span className="mono" style={{ fontSize: 12 }}>
                        {f.name}
                      </span>
                    </div>
                  </td>
                  <td className="mono dim">{f.size}</td>
                  <td className="mono dim">{f.mtime}</td>
                  <td className="mono dim" style={{ fontSize: 10 }}>
                    {f.type === 'dir' ? 'drwxr-xr-x' : '-rw-r--r--'}
                  </td>
                  <td>
                    <button className="btn btn--sm" onClick={(e) => e.stopPropagation()}>
                      <I n="more-horizontal" s={12} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {/* Preview pane */}
        <div>
          {selected ? (
            <FilePreview name={selected} />
          ) : (
            <div
              className="card"
              style={{ padding: 20, textAlign: 'center', color: 'var(--h-text-3)' }}
            >
              <I n="file" s={32} style={{ marginBottom: 8 }} />
              <div style={{ fontSize: 13, marginBottom: 4 }}>No file selected</div>
              <div className="muted" style={{ fontSize: 12 }}>
                Click a file to preview.
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
function FilePreview({ name }) {
  const isText = /\.(md|yaml|yml|conf|json|txt|sh|bashrc)$/.test(name) || name === '.bashrc';
  return (
    <div className="card" style={{ padding: 14 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
        <I n="file-text" s={18} style={{ color: 'var(--h-accent)' }} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            className="mono"
            style={{
              fontSize: 13,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {name}
          </div>
          <div className="muted" style={{ fontSize: 11 }}>
            Modified 2026-03-21 · 4.1 KB · admin:admin
          </div>
        </div>
      </div>
      <div style={{ display: 'flex', gap: 6, marginBottom: 12 }}>
        <button className="btn btn--sm" style={{ flex: 1 }}>
          <I n="download" s={12} /> Download
        </button>
        <button className="btn btn--sm" style={{ flex: 1 }}>
          <I n="edit" s={12} /> Edit
        </button>
        <button className="btn btn--sm" style={{ color: 'var(--h-danger)' }}>
          <I n="trash-2" s={12} />
        </button>
      </div>
      {isText && (
        <div
          style={{
            background: 'var(--h-bg)',
            border: '1px solid var(--h-border)',
            borderRadius: 4,
            padding: 10,
            fontFamily: 'JetBrains Mono, monospace',
            fontSize: 10,
            color: 'var(--h-text-2)',
            maxHeight: 320,
            overflow: 'auto',
            lineHeight: 1.6,
          }}
        >
          <div style={{ color: 'var(--h-text-3)' }}># {name}</div>
          <div>
            version: <span style={{ color: 'var(--h-accent)' }}>1.2</span>
          </div>
          <div>services:</div>
          <div>&nbsp;&nbsp;web:</div>
          <div>
            &nbsp;&nbsp;&nbsp;&nbsp;image:{' '}
            <span style={{ color: '#8bffd4' }}>nginx:1.27-alpine</span>
          </div>
          <div>
            &nbsp;&nbsp;&nbsp;&nbsp;ports: [<span style={{ color: '#ffd36a' }}>"80:80"</span>,{' '}
            <span style={{ color: '#ffd36a' }}>"443:443"</span>]
          </div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;volumes:</div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;- ./html:/usr/share/nginx/html:ro</div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;- ./certs:/etc/ssl/certs:ro</div>
          <div>
            &nbsp;&nbsp;&nbsp;&nbsp;restart:{' '}
            <span style={{ color: '#8bffd4' }}>unless-stopped</span>
          </div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;healthcheck:</div>
          <div>
            &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;test: [
            <span style={{ color: '#ffd36a' }}>"CMD"</span>,{' '}
            <span style={{ color: '#ffd36a' }}>"curl"</span>,{' '}
            <span style={{ color: '#ffd36a' }}>"-f"</span>,{' '}
            <span style={{ color: '#ffd36a' }}>"http://localhost/"</span>]
          </div>
        </div>
      )}
      {!isText && (
        <div
          style={{
            padding: 30,
            textAlign: 'center',
            border: '1px dashed var(--h-border)',
            borderRadius: 4,
            color: 'var(--h-text-3)',
          }}
        >
          <I n="file" s={48} style={{ marginBottom: 10 }} />
          <div className="mono" style={{ fontSize: 11 }}>
            Binary file · no preview
          </div>
        </div>
      )}
    </div>
  );
}

function PageContainerDetail({ name = 'jellyfin', onNav }) {
  const c = (typeof CONTAINERS !== 'undefined' ? CONTAINERS : []).find((x) => x.name === name) || {
    name,
    image: 'unknown',
    status: 'running',
    health: 'healthy',
    stack: 'misc',
    ports: '—',
    cpu: 20,
    ram: 30,
  };
  const [tab, setTab] = useState('overview');
  return (
    <div style={{ maxWidth: 1280, padding: '18px 20px' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-end',
          marginBottom: 14,
        }}
      >
        <div>
          <div className="eyebrow">DATACENTER / CONTAINERS / {name.toUpperCase()}</div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginTop: 4 }}>
            <h1 className="stencil" style={{ fontSize: 26, margin: 0 }}>
              {name}
            </h1>
            <span className="dot dot--running" />
            <span className="mono" style={{ fontSize: 11, color: 'var(--h-success)' }}>
              RUNNING
            </span>
            <span className="badge mono" style={{ color: 'var(--h-text-2)' }}>
              {c.stack}
            </span>
            <span className="badge mono" style={{ color: 'var(--h-text-2)' }}>
              {c.image}
            </span>
          </div>
        </div>
        <div style={{ display: 'flex', gap: 6 }}>
          <button className="btn btn--sm">
            <I n="terminal" s={12} /> Shell
          </button>
          <button className="btn btn--sm">
            <I n="rotate-cw" s={12} /> Restart
          </button>
          <button className="btn btn--sm">
            <I n="square" s={12} /> Stop
          </button>
          <button className="btn btn--sm" onClick={() => onNav?.('files:container:' + name)}>
            <I n="folder" s={12} /> Files
          </button>
          <button className="btn btn--sm" style={{ color: 'var(--h-danger)' }}>
            <I n="trash-2" s={12} />
          </button>
        </div>
      </div>

      <div className="tabs" style={{ marginBottom: 14 }}>
        {[
          ['overview', 'Overview'],
          ['logs', 'Logs'],
          ['env', 'Environment'],
          ['volumes', 'Volumes'],
          ['network', 'Network'],
          ['compose', 'Compose'],
          ['stats', 'Stats'],
        ].map(([id, l]) => (
          <button key={id} className={'tab ' + (tab === id ? 'on' : '')} onClick={() => setTab(id)}>
            {l}
          </button>
        ))}
      </div>

      {tab === 'overview' && <CtOverview c={c} />}
      {tab === 'logs' && <CtLogs />}
      {tab === 'env' && <CtEnv />}
      {tab === 'volumes' && <CtVolumes />}
      {tab === 'network' && <CtNetwork />}
      {tab === 'compose' && <CtCompose name={name} />}
      {tab === 'stats' && <CtStats />}
    </div>
  );
}
function CtOverview({ c }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 14 }}>
      <div className="card" style={{ padding: 16 }}>
        <h3 className="stencil" style={{ fontSize: 13, margin: '0 0 10px' }}>
          RUNTIME
        </h3>
        <dl className="desc">
          <dt>Container ID</dt>
          <dd className="mono" style={{ wordBreak: 'break-all' }}>
            {(c.id || 'c_a91f82c3') + 'd4e0b9e8f7c6a5b4c3d2e1f0'}
          </dd>
          <dt>Image</dt>
          <dd className="mono">{c.image}</dd>
          <dt>Digest</dt>
          <dd className="mono dim" style={{ fontSize: 10 }}>
            sha256:9f8a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0e9f8a
          </dd>
          <dt>Created</dt>
          <dd className="mono">2026-02-14 08:14:22</dd>
          <dt>Started</dt>
          <dd className="mono">2026-04-18 02:00:01 · uptime 4d 12h</dd>
          <dt>Entrypoint</dt>
          <dd className="mono">/init</dd>
          <dt>CMD</dt>
          <dd className="mono dim">(inherited from image)</dd>
          <dt>Restart policy</dt>
          <dd className="mono">unless-stopped</dd>
          <dt>Ports</dt>
          <dd className="mono">{c.ports}</dd>
          <dt>Health</dt>
          <dd>
            <span style={{ color: 'var(--h-success)' }}>● healthy</span> · last check 12s ago
          </dd>
        </dl>
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div className="card" style={{ padding: 12 }}>
          <div className="eyebrow" style={{ marginBottom: 8 }}>
            CPU
          </div>
          <div className="stencil" style={{ fontSize: 24 }}>
            {c.cpu}%
          </div>
          <MiniChart points={32} />
        </div>
        <div className="card" style={{ padding: 12 }}>
          <div className="eyebrow" style={{ marginBottom: 8 }}>
            MEMORY
          </div>
          <div className="stencil" style={{ fontSize: 24 }}>
            {c.ram}%
          </div>
          <MiniChart points={32} />
        </div>
        <div className="card" style={{ padding: 12 }}>
          <div className="eyebrow" style={{ marginBottom: 8 }}>
            NETWORK
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, fontSize: 12 }}>
            <div>
              <span className="mono dim" style={{ fontSize: 10 }}>
                RX
              </span>
              <div className="mono">412 MB</div>
            </div>
            <div>
              <span className="mono dim" style={{ fontSize: 10 }}>
                TX
              </span>
              <div className="mono">1.2 GB</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
function CtLogs() {
  const lines = [
    ['14:32:22.104', 'info', 'HTTP 200 GET /api/users'],
    ['14:32:24.982', 'info', 'HTTP 200 GET /web/index.html'],
    ['14:32:25.014', 'info', 'Session refresh: user=alice ip=10.0.0.14'],
    ['14:32:30.441', 'warn', 'Slow query: SELECT * FROM items WHERE …  took 218ms'],
    ['14:32:32.102', 'info', 'HTTP 200 POST /api/playlist'],
    ['14:32:34.581', 'info', 'HTTP 304 GET /api/library (cache hit)'],
    ['14:32:38.002', 'error', 'Upstream timeout: radarr:7878 after 5000ms'],
    ['14:32:38.020', 'info', 'Retry 1/3 upstream radarr:7878'],
    ['14:32:39.918', 'info', 'HTTP 200 POST /api/sync'],
    ['14:32:42.801', 'info', 'Background job: library-scan completed (4128 items in 12.4s)'],
    ['14:32:45.010', 'info', 'HTTP 200 GET /api/sessions'],
  ];
  return (
    <div
      className="card"
      style={{
        padding: 0,
        background: '#0a0a0a',
        height: '60vh',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <div
        style={{
          padding: '8px 14px',
          borderBottom: '1px solid var(--h-border)',
          display: 'flex',
          gap: 8,
          alignItems: 'center',
        }}
      >
        <div className="seg">
          <button className="seg__b on">All</button>
          <button className="seg__b">info</button>
          <button className="seg__b">warn</button>
          <button className="seg__b">error</button>
        </div>
        <input
          className="input input--mono"
          style={{ flex: 1, maxWidth: 300 }}
          placeholder="Filter: grep regex…"
        />
        <div style={{ marginLeft: 'auto', display: 'flex', gap: 6 }}>
          <Switch on={true} label="Tail" />
          <button className="btn btn--sm">
            <I n="download" s={12} /> Export
          </button>
        </div>
      </div>
      <div
        style={{
          flex: 1,
          overflow: 'auto',
          padding: '8px 14px',
          fontFamily: 'JetBrains Mono, monospace',
          fontSize: 11,
          lineHeight: 1.65,
        }}
      >
        {lines.map((l, i) => (
          <div key={i} style={{ display: 'flex', gap: 10 }}>
            <span style={{ color: 'var(--h-text-3)', minWidth: 100 }}>{l[0]}</span>
            <span
              style={{
                minWidth: 44,
                fontSize: 9,
                letterSpacing: '0.08em',
                color:
                  l[1] === 'error'
                    ? 'var(--h-danger)'
                    : l[1] === 'warn'
                      ? 'var(--h-warn)'
                      : 'var(--h-text-3)',
              }}
            >
              {l[1].toUpperCase()}
            </span>
            <span style={{ color: 'var(--h-text)', flex: 1 }}>{l[2]}</span>
          </div>
        ))}
        <div style={{ color: 'var(--h-accent)', marginTop: 4 }}>● live · scroll to pause</div>
      </div>
    </div>
  );
}
function CtEnv() {
  const env = [
    ['TZ', 'Europe/Amsterdam', false],
    ['PUID', '1000', false],
    ['PGID', '1000', false],
    ['NEXTCLOUD_ADMIN_USER', 'admin', false],
    ['NEXTCLOUD_ADMIN_PASSWORD', '••••••••••••', true],
    ['POSTGRES_HOST', 'db.internal', false],
    ['POSTGRES_PASSWORD', '••••••••••••••••', true],
    ['SMTP_HOST', 'mail.company.tld', false],
    ['SMTP_USER', 'noreply@company.tld', false],
    ['OVERLEAF_LICENSE', '••••••••••••', true],
  ];
  return (
    <div className="card">
      <table className="tbl">
        <thead>
          <tr>
            <th>KEY</th>
            <th>VALUE</th>
            <th style={{ width: 100 }}>TYPE</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {env.map(([k, v, secret], i) => (
            <tr key={i}>
              <td className="mono">{k}</td>
              <td className="mono" style={{ color: secret ? 'var(--h-text-3)' : 'var(--h-text)' }}>
                {v}
              </td>
              <td>
                {secret ? (
                  <span
                    className="chip"
                    style={{ color: 'var(--h-warn)', borderColor: 'var(--h-warn)' }}
                  >
                    secret
                  </span>
                ) : (
                  <span className="chip">plain</span>
                )}
              </td>
              <td style={{ textAlign: 'right' }}>
                <button className="btn btn--sm">
                  <I n="eye" s={11} />
                </button>
                <button className="btn btn--sm">
                  <I n="copy" s={11} />
                </button>
                <button className="btn btn--sm">
                  <I n="pencil" s={11} />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div style={{ padding: 12, borderTop: '1px solid var(--h-border)' }}>
        <button className="btn btn--sm">
          <I n="plus" s={12} /> Add variable
        </button>
      </div>
    </div>
  );
}
function CtVolumes() {
  const vols = [
    { src: '/srv/appdata/jellyfin/config', dst: '/config', mode: 'rw', size: '412 MB' },
    { src: '/srv/media', dst: '/media', mode: 'ro', size: '2.4 TB' },
    { src: 'jellyfin-cache (named)', dst: '/cache', mode: 'rw', size: '18 GB' },
    { src: '/etc/localtime', dst: '/etc/localtime', mode: 'ro', size: '—' },
  ];
  return (
    <div className="card">
      <table className="tbl">
        <thead>
          <tr>
            <th>HOST / VOLUME</th>
            <th>CONTAINER PATH</th>
            <th>MODE</th>
            <th>SIZE</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {vols.map((v, i) => (
            <tr key={i}>
              <td className="mono">{v.src}</td>
              <td className="mono">{v.dst}</td>
              <td>
                <span className="chip">{v.mode}</span>
              </td>
              <td className="mono">{v.size}</td>
              <td style={{ textAlign: 'right' }}>
                <button className="btn btn--sm">Browse</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div style={{ padding: 12, borderTop: '1px solid var(--h-border)' }}>
        <button className="btn btn--sm">
          <I n="plus" s={12} /> Attach volume
        </button>
      </div>
    </div>
  );
}
function CtNetwork() {
  return (
    <div className="card" style={{ padding: 16 }}>
      <dl className="desc">
        <dt>Network mode</dt>
        <dd className="mono">bridge</dd>
        <dt>Network</dt>
        <dd className="mono">media-net (172.20.0.0/16)</dd>
        <dt>IP address</dt>
        <dd className="mono">172.20.0.14</dd>
        <dt>MAC</dt>
        <dd className="mono">02:42:ac:14:00:0e</dd>
        <dt>Port bindings</dt>
        <dd>
          <div className="mono" style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <span>0.0.0.0:8096 → 8096/tcp (web)</span>
            <span>0.0.0.0:8920 → 8920/tcp (https)</span>
            <span>0.0.0.0:1900 → 1900/udp (DLNA)</span>
            <span>0.0.0.0:7359 → 7359/udp (autodiscovery)</span>
          </div>
        </dd>
        <dt>Connected networks</dt>
        <dd>
          <div style={{ display: 'flex', gap: 6 }}>
            <span className="chip chip--on">media-net</span>
            <span className="chip">metrics</span>
          </div>
        </dd>
      </dl>
    </div>
  );
}
function CtCompose({ name }) {
  const yaml = `# ${name}/compose.yaml
services:
  ${name}:
    image: linuxserver/${name}:latest
    container_name: ${name}
    restart: unless-stopped
    environment:
      - TZ=Europe/Amsterdam
      - PUID=1000
      - PGID=1000
    volumes:
      - /srv/appdata/${name}/config:/config
      - /srv/media:/media:ro
    ports:
      - "8096:8096"
      - "8920:8920"
    networks:
      - media-net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.${name}.rule=Host(\`${name}.local\`)"

networks:
  media-net:
    external: true
`;
  return (
    <div
      className="card"
      style={{ padding: 0, background: '#0a0a0a', fontFamily: 'JetBrains Mono, monospace' }}
    >
      <div
        style={{
          padding: '8px 14px',
          borderBottom: '1px solid var(--h-border)',
          display: 'flex',
          gap: 6,
          alignItems: 'center',
        }}
      >
        <span className="mono dim" style={{ fontSize: 11 }}>
          compose.yaml · read-only · managed by Helling
        </span>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: 6 }}>
          <button className="btn btn--sm">
            <I n="pencil" s={12} /> Edit
          </button>
          <button className="btn btn--sm">
            <I n="refresh-cw" s={12} /> Redeploy
          </button>
          <button className="btn btn--sm">
            <I n="download" s={12} /> Export
          </button>
        </div>
      </div>
      <pre
        style={{
          margin: 0,
          padding: '14px 16px',
          fontSize: 11,
          color: '#d4d4d4',
          lineHeight: 1.6,
          overflow: 'auto',
          maxHeight: '60vh',
        }}
      >
        <code>{yaml}</code>
      </pre>
    </div>
  );
}
function CtStats() {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
      <BigChart
        title="CPU"
        series={[
          { name: 'user', color: '#c6ff24' },
          { name: 'system', color: '#8a9bff' },
        ]}
      />
      <BigChart
        title="Memory"
        series={[
          { name: 'rss', color: '#c6ff24' },
          { name: 'cache', color: '#8bffd4' },
        ]}
      />
      <BigChart
        title="Network"
        series={[
          { name: 'rx', color: '#c6ff24' },
          { name: 'tx', color: '#ff8aa9' },
        ]}
      />
      <BigChart
        title="Disk IO"
        series={[
          { name: 'read', color: '#8bffd4' },
          { name: 'write', color: '#ffd36a' },
        ]}
      />
    </div>
  );
}

// ─── USER DETAIL (profile + audit) ─────────────────────────────
function PageUserDetail({ user = 'alice', onNav }) {
  const [tab, setTab] = useState('profile');
  return (
    <div style={{ maxWidth: 1280, padding: '18px 20px' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-end',
          marginBottom: 14,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 14 }}>
          <div
            style={{
              width: 56,
              height: 56,
              borderRadius: '50%',
              background: `linear-gradient(135deg, hsl(200, 60%, 50%), hsl(280, 60%, 40%))`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontWeight: 700,
              fontSize: 22,
              color: '#000',
            }}
          >
            {user[0].toUpperCase()}
          </div>
          <div>
            <div className="eyebrow">ADMIN / USERS</div>
            <h1 className="stencil" style={{ fontSize: 26, margin: '4px 0 2px' }}>
              {user}
            </h1>
            <div className="mono dim" style={{ fontSize: 12 }}>
              {user}@helling.io · active 12m ago
            </div>
          </div>
        </div>
        <div style={{ display: 'flex', gap: 6 }}>
          <button className="btn btn--sm">
            <I n="key" s={12} /> Reset password
          </button>
          <button className="btn btn--sm">
            <I n="log-out" s={12} /> Revoke all sessions
          </button>
          <button className="btn btn--sm" style={{ color: 'var(--h-danger)' }}>
            <I n="user-x" s={12} /> Suspend
          </button>
        </div>
      </div>

      <div className="tabs" style={{ marginBottom: 14 }}>
        {[
          ['profile', 'Profile'],
          ['sessions', 'Sessions'],
          ['tokens', 'API tokens'],
          ['ssh', 'SSH keys'],
          ['activity', 'Activity'],
        ].map(([id, l]) => (
          <button key={id} className={'tab ' + (tab === id ? 'on' : '')} onClick={() => setTab(id)}>
            {l}
          </button>
        ))}
      </div>

      {tab === 'profile' && (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
          <div className="card" style={{ padding: 16 }}>
            <h3 className="stencil" style={{ fontSize: 13, margin: '0 0 10px' }}>
              ACCOUNT
            </h3>
            <div className="field">
              <label>Display name</label>
              <input
                className="input"
                defaultValue={user.charAt(0).toUpperCase() + user.slice(1) + ' Martin'}
              />
            </div>
            <div className="field">
              <label>Email</label>
              <input className="input" defaultValue={user + '@helling.io'} />
            </div>
            <div className="field">
              <label>Role</label>
              <select className="input">
                <option>admin</option>
                <option>operator</option>
                <option>dev</option>
                <option>viewer</option>
              </select>
            </div>
            <div className="field">
              <label>Department</label>
              <input className="input" defaultValue="Infrastructure" />
            </div>
          </div>
          <div className="card" style={{ padding: 16 }}>
            <h3 className="stencil" style={{ fontSize: 13, margin: '0 0 10px' }}>
              SECURITY
            </h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              <Switch on={true} label="Two-factor authentication (TOTP)" />
              <Switch on={false} label="Require WebAuthn hardware key" />
              <Switch on={true} label="Email on new device login" />
              <Switch on={false} label="Session visible to admins" />
            </div>
            <div className="field" style={{ marginTop: 14 }}>
              <label>Password last changed</label>
              <div className="mono dim">32 days ago</div>
            </div>
            <div className="field">
              <label>Failed login attempts</label>
              <div className="mono">0 in last 24h</div>
            </div>
          </div>
        </div>
      )}
      {tab === 'sessions' && (
        <div className="card">
          <table className="tbl">
            <thead>
              <tr>
                <th>DEVICE</th>
                <th>LOCATION</th>
                <th>IP</th>
                <th>STARTED</th>
                <th>LAST ACTIVE</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>
                  Chrome on macOS{' '}
                  <span className="chip chip--on" style={{ marginLeft: 6 }}>
                    this session
                  </span>
                </td>
                <td>Amsterdam, NL</td>
                <td className="mono">10.0.0.14</td>
                <td className="mono dim">2h ago</td>
                <td className="mono dim">just now</td>
                <td>
                  <button className="btn btn--sm" disabled>
                    Active
                  </button>
                </td>
              </tr>
              <tr>
                <td>Safari on iOS</td>
                <td>Amsterdam, NL</td>
                <td className="mono">10.0.0.22</td>
                <td className="mono dim">8h ago</td>
                <td className="mono dim">12m ago</td>
                <td>
                  <button className="btn btn--sm">Revoke</button>
                </td>
              </tr>
              <tr>
                <td>helling-cli</td>
                <td>Rotterdam, NL (VPN)</td>
                <td className="mono">10.9.0.4</td>
                <td className="mono dim">3 days ago</td>
                <td className="mono dim">42s ago</td>
                <td>
                  <button className="btn btn--sm">Revoke</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      )}
      {tab === 'tokens' && <RBACTokens />}
      {tab === 'ssh' && <RBACKeys />}
      {tab === 'activity' && (
        <div className="card">
          <table className="tbl">
            <thead>
              <tr>
                <th>TIME</th>
                <th>ACTION</th>
                <th>TARGET</th>
                <th>IP</th>
                <th>RESULT</th>
              </tr>
            </thead>
            <tbody>
              {[
                ['14:32:02', 'instance.start', 'web-prod-1', '10.0.0.14', 'ok'],
                ['14:22:03', 'instance.snapshot', 'vm-web-1', '10.0.0.14', 'ok'],
                ['14:10:51', 'instance.start', 'ct-gitea', '10.0.0.14', 'ok'],
                ['13:48:02', 'firewall.rule.add', 'dc/rule-8', '10.0.0.14', 'ok'],
                ['12:08:15', 'user.2fa.enable', '(self)', '10.0.0.14', 'ok'],
                ['09:34:17', 'login', '-', '10.0.0.14', 'ok'],
                ['09:33:48', 'login', '-', '10.0.0.14', 'denied · wrong password'],
              ].map((r, i) => (
                <tr key={i}>
                  <td className="mono dim">{r[0]}</td>
                  <td className="mono">{r[1]}</td>
                  <td className="mono">{r[2]}</td>
                  <td className="mono dim">{r[3]}</td>
                  <td>
                    <span
                      className="badge"
                      style={{
                        color: r[4].startsWith('denied') ? 'var(--h-danger)' : 'var(--h-success)',
                        borderColor: r[4].startsWith('denied')
                          ? 'var(--h-danger)'
                          : 'var(--h-success)',
                      }}
                    >
                      {r[4]}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// Modal stubs — dispatched from app.jsx ModalHost
function WizardCreateInstance({ onClose }) {
  return (
    <Modal
      open={true}
      onClose={onClose}
      title="Create instance"
      size="lg"
      footer={
        <>
          <button className="btn btn--sm" onClick={onClose}>
            Cancel
          </button>
          <button className="btn btn--primary btn--sm" onClick={onClose}>
            Create
          </button>
        </>
      }
    >
      <PageNewInstance onNav={onClose} />
    </Modal>
  );
}
function ModalInstallApp({ onClose, app }) {
  return (
    <Modal
      open={true}
      onClose={onClose}
      title={`Install ${app?.name || 'app'}`}
      footer={
        <>
          <button className="btn btn--sm" onClick={onClose}>
            Cancel
          </button>
          <button className="btn btn--primary btn--sm" onClick={onClose}>
            Install
          </button>
        </>
      }
    >
      <div style={{ padding: 4 }}>
        <p className="muted" style={{ fontSize: 13, marginBottom: 14 }}>
          One-click install — configures container, volumes, and reverse-proxy entry.
        </p>
        <div className="field">
          <label>Hostname</label>
          <input className="input input--mono" defaultValue={app?.id || 'my-app'} />
        </div>
        <div className="field">
          <label>Subdomain</label>
          <input className="input input--mono" defaultValue={(app?.id || 'app') + '.local'} />
        </div>
        <div className="field">
          <label>Storage pool</label>
          <select className="input">
            <option>default</option>
            <option>nvme-fast</option>
          </select>
        </div>
      </div>
    </Modal>
  );
}
function ModalFirewallRule({ onClose }) {
  return (
    <Modal
      open={true}
      onClose={onClose}
      title="New firewall rule"
      footer={
        <>
          <button className="btn btn--sm" onClick={onClose}>
            Cancel
          </button>
          <button className="btn btn--primary btn--sm" onClick={onClose}>
            Save rule
          </button>
        </>
      }
    >
      <FWRuleEditor
        rule={{
          id: 'new',
          action: 'allow',
          dir: 'in',
          proto: 'tcp',
          src: '0.0.0.0/0',
          dport: '',
          note: '',
        }}
      />
    </Modal>
  );
}
function ModalCloudInit({ onClose }) {
  const [yaml, setYaml] = useState(`#cloud-config
package_update: true
packages:
  - htop
  - vim
runcmd:
  - echo "hello from helling" > /etc/motd`);
  return (
    <Modal
      open={true}
      onClose={onClose}
      title="Edit cloud-init"
      size="lg"
      footer={
        <>
          <button className="btn btn--sm" onClick={onClose}>
            Cancel
          </button>
          <button className="btn btn--primary btn--sm" onClick={onClose}>
            Save & apply on next boot
          </button>
        </>
      }
    >
      <div className="field">
        <label>user-data (YAML)</label>
        <textarea
          className="input input--mono"
          rows="18"
          value={yaml}
          onChange={(e) => setYaml(e.target.value)}
        />
      </div>
    </Modal>
  );
}

Object.assign(window, {
  PageNewInstance,
  PageConsole,
  PageMetrics,
  PageAlerts,
  PageRBAC,
  PageFirewallEditor,
  PageMarketplace,
  PageFileBrowser,
  PageContainerDetail,
  PageUserDetail,
  WizardCreateInstance,
  ModalInstallApp,
  ModalFirewallRule,
  ModalCloudInit,
});
