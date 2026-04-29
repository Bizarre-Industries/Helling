// Helling WebUI — First-boot Setup wizard.
//
// Extracted from pages2.jsx during Phase 2A. The 5 step components
// (SetupWelcome / SetupDisks / SetupNetwork / SetupAdmin / SetupReview)
// are co-located non-exports — only PageSetup itself is consumed
// outside this file.
//
// onDone: optional — fired after the user clicks Finish & reboot.

import React, { useState } from 'react';
import { I } from '../../primitives/icon';
import { Switch } from '../../primitives/switch';

interface SetupConfig {
  hostname: string;
  timezone: string;
  disks: string[];
  pool: string;
  ipMode: 'dhcp' | 'static';
  ip: string;
  gateway: string;
  dns: string;
  adminName: string;
  adminEmail: string;
  adminPw: string;
  adminPw2: string;
  telemetry: boolean;
}

type SetField = <K extends keyof SetupConfig>(k: K, v: SetupConfig[K]) => void;

interface PageSetupProps {
  onDone?: () => void;
}

// window.toast is provided by infra.jsx (legacy ToastBus). Phase 2B will
// replace this with a typed UI store import; for now declare the shape.
type ToastBusGlobal = {
  toast?: { success?: (title: string, msg?: string) => void };
};
const getToast = () =>
  typeof window !== 'undefined' ? (window as unknown as ToastBusGlobal).toast : undefined;

const STEPS = ['Welcome', 'Disks', 'Network', 'Admin', 'Review'] as const;

export default function PageSetup({ onDone }: PageSetupProps) {
  const [step, setStep] = useState(0);
  const [cfg, setCfg] = useState<SetupConfig>({
    hostname: 'helling-01',
    timezone: 'Europe/Amsterdam',
    disks: ['/dev/nvme0n1', '/dev/nvme1n1'],
    pool: 'zfs-mirror',
    ipMode: 'dhcp',
    ip: '',
    gateway: '',
    dns: '1.1.1.1',
    adminName: 'admin',
    adminEmail: '',
    adminPw: '',
    adminPw2: '',
    telemetry: true,
  });
  const set: SetField = (k, v) => setCfg((c) => ({ ...c, [k]: v }));

  const next = () => setStep((s) => Math.min(STEPS.length - 1, s + 1));
  const prev = () => setStep((s) => Math.max(0, s - 1));

  return (
    <div
      style={{
        minHeight: '100vh',
        background: 'var(--h-bg)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 40,
      }}
    >
      <div
        style={{
          width: 820,
          background: 'var(--h-surface)',
          border: '1px solid var(--h-border)',
          borderRadius: 'var(--h-radius)',
        }}
      >
        {/* Header */}
        <div
          style={{
            padding: '22px 28px 18px',
            borderBottom: '1px solid var(--h-border)',
            display: 'flex',
            alignItems: 'center',
            gap: 16,
          }}
        >
          <img src="assets/mark-inverse.png" alt="Helling" style={{ width: 44, height: 44 }} />
          <div>
            <div
              className="mono"
              style={{ fontSize: 10, letterSpacing: '0.18em', color: 'var(--h-text-3)' }}
            >
              FIRST-BOOT SETUP · v0.1
            </div>
            <div
              className="stencil"
              style={{ fontSize: 22, letterSpacing: '0.01em', marginTop: 2 }}
            >
              Welcome to Helling.
            </div>
          </div>
        </div>

        {/* Stepper */}
        <div className="stepper">
          {STEPS.map((s, i) => (
            <React.Fragment key={s}>
              <div className={`s ${i === step ? 'on' : i < step ? 'done' : ''}`}>
                <span className="n">{i < step ? '✓' : i + 1}</span>
                {s}
              </div>
              {i < STEPS.length - 1 && <div className="sep" />}
            </React.Fragment>
          ))}
        </div>

        {/* Body */}
        <div style={{ padding: '22px 28px', minHeight: 380 }}>
          {step === 0 && <SetupWelcome cfg={cfg} set={set} />}
          {step === 1 && <SetupDisks cfg={cfg} set={set} />}
          {step === 2 && <SetupNetwork cfg={cfg} set={set} />}
          {step === 3 && <SetupAdmin cfg={cfg} set={set} />}
          {step === 4 && <SetupReview cfg={cfg} />}
        </div>

        {/* Footer */}
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
            {step > 0 && (
              <button type="button" className="btn" onClick={prev}>
                <I n="arrow-left" s={13} /> Back
              </button>
            )}
            {step < STEPS.length - 1 && (
              <button type="button" className="btn btn--primary" onClick={next}>
                Continue <I n="arrow-right" s={13} />
              </button>
            )}
            {step === STEPS.length - 1 && (
              <button
                type="button"
                className="btn btn--primary"
                onClick={() => {
                  getToast()?.success?.(
                    'Helling is ready',
                    'Booting services — this takes about 30 seconds.',
                  );
                  setTimeout(() => onDone?.(), 400);
                }}
              >
                <I n="check" s={13} /> Finish & reboot
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

interface StepProps {
  cfg: SetupConfig;
  set: SetField;
}

function SetupWelcome({ cfg, set }: StepProps) {
  return (
    <>
      <h2 className="stencil" style={{ fontSize: 20, margin: '0 0 6px' }}>
        Let's name this machine.
      </h2>
      <p className="muted" style={{ maxWidth: 560, fontSize: 13, marginBottom: 22 }}>
        Helling runs on bare metal. This host will become <span className="mono">node-1</span> — the
        seed of your datacenter. Add more nodes from <span className="mono">Cluster → Join</span>{' '}
        later.
      </p>
      <div className="field-row">
        <div className="field">
          <label>Hostname</label>
          <input
            className="input input--mono"
            value={cfg.hostname}
            onChange={(e) => set('hostname', e.target.value)}
          />
        </div>
        <div className="field">
          <label>Timezone</label>
          <select
            className="input"
            value={cfg.timezone}
            onChange={(e) => set('timezone', e.target.value)}
          >
            {[
              'UTC',
              'Europe/Amsterdam',
              'Europe/London',
              'America/New_York',
              'America/Los_Angeles',
              'Asia/Tokyo',
            ].map((t) => (
              <option key={t}>{t}</option>
            ))}
          </select>
        </div>
      </div>
      <div className="field">
        <label>Telemetry</label>
        <Switch
          on={cfg.telemetry}
          onChange={(v) => set('telemetry', v)}
          label="Share anonymous system metrics to help improve Helling"
        />
      </div>
    </>
  );
}

interface DiskRow {
  path: string;
  model: string;
  size: string;
  health: string;
}

function SetupDisks({ cfg, set }: StepProps) {
  const DISKS: DiskRow[] = [
    { path: '/dev/nvme0n1', model: 'Samsung 980 PRO 1TB', size: '931 GB', health: '100%' },
    { path: '/dev/nvme1n1', model: 'Samsung 980 PRO 1TB', size: '931 GB', health: '100%' },
    { path: '/dev/sda', model: 'WD Red 4TB', size: '3.6 TB', health: '92%' },
    { path: '/dev/sdb', model: 'WD Red 4TB', size: '3.6 TB', health: '94%' },
  ];
  const toggle = (p: string) =>
    set('disks', cfg.disks.includes(p) ? cfg.disks.filter((x) => x !== p) : [...cfg.disks, p]);
  return (
    <>
      <h2 className="stencil" style={{ fontSize: 20, margin: '0 0 6px' }}>
        Pick the disks Helling manages.
      </h2>
      <p className="muted" style={{ fontSize: 13, marginBottom: 18 }}>
        Selected disks will be wiped and joined into a ZFS pool. Boot disk is never touched.
      </p>
      <table className="tbl" style={{ borderCollapse: 'separate', borderSpacing: 0 }}>
        <thead>
          <tr>
            <th style={{ width: 40 }} />
            <th>DEVICE</th>
            <th>MODEL</th>
            <th>SIZE</th>
            <th>HEALTH</th>
          </tr>
        </thead>
        <tbody>
          {DISKS.map((d) => (
            <tr
              key={d.path}
              className={cfg.disks.includes(d.path) ? 'sel' : ''}
              onClick={() => toggle(d.path)}
              style={{ cursor: 'pointer' }}
            >
              <td>
                <input
                  type="checkbox"
                  checked={cfg.disks.includes(d.path)}
                  onChange={() => toggle(d.path)}
                />
              </td>
              <td className="mono">{d.path}</td>
              <td>{d.model}</td>
              <td className="mono">{d.size}</td>
              <td className="mono" style={{ color: 'var(--h-success)' }}>
                {d.health}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="field" style={{ marginTop: 18 }}>
        <label>Pool layout</label>
        <div className="choices">
          {[
            {
              id: 'zfs-mirror',
              t: 'Mirror (recommended)',
              d: 'Max redundancy. Needs 2+ disks. Half the total capacity.',
            },
            { id: 'zfs-raidz1', t: 'RAIDZ1', d: 'One-disk failure tolerance. Needs 3+ disks.' },
            {
              id: 'zfs-stripe',
              t: 'Stripe',
              d: 'Maximum capacity. No redundancy. Lose any disk, lose everything.',
            },
          ].map((o) => (
            <div
              key={o.id}
              className={`opt${cfg.pool === o.id ? ' on' : ''}`}
              onClick={() => set('pool', o.id)}
            >
              <div className="t">{o.t}</div>
              <div className="d">{o.d}</div>
            </div>
          ))}
        </div>
      </div>
    </>
  );
}

function SetupNetwork({ cfg, set }: StepProps) {
  return (
    <>
      <h2 className="stencil" style={{ fontSize: 20, margin: '0 0 6px' }}>
        Set up networking.
      </h2>
      <p className="muted" style={{ fontSize: 13, marginBottom: 18 }}>
        Helling manages a bridge, a firewall, and an overlay network for VMs. Static IPs are
        recommended for servers.
      </p>
      <div className="field">
        <label>IP assignment</label>
        <div className="choices">
          <div
            className={`opt${cfg.ipMode === 'dhcp' ? ' on' : ''}`}
            onClick={() => set('ipMode', 'dhcp')}
          >
            <div className="t">DHCP</div>
            <div className="d">Let your router assign an address.</div>
          </div>
          <div
            className={`opt${cfg.ipMode === 'static' ? ' on' : ''}`}
            onClick={() => set('ipMode', 'static')}
          >
            <div className="t">Static</div>
            <div className="d">Lock an IP. Required for multi-node clusters.</div>
          </div>
        </div>
      </div>
      {cfg.ipMode === 'static' && (
        <div className="field-row-3">
          <div className="field">
            <label>IP / CIDR</label>
            <input
              className="input input--mono"
              placeholder="192.168.1.10/24"
              value={cfg.ip}
              onChange={(e) => set('ip', e.target.value)}
            />
          </div>
          <div className="field">
            <label>Gateway</label>
            <input
              className="input input--mono"
              placeholder="192.168.1.1"
              value={cfg.gateway}
              onChange={(e) => set('gateway', e.target.value)}
            />
          </div>
          <div className="field">
            <label>DNS</label>
            <input
              className="input input--mono"
              value={cfg.dns}
              onChange={(e) => set('dns', e.target.value)}
            />
          </div>
        </div>
      )}
      <div className="alert alert--info" style={{ marginTop: 18 }}>
        <I n="info" s={14} />
        <span>
          Current link: <span className="mono">eno1</span> · detected{' '}
          <span className="mono" style={{ color: 'var(--h-text)' }}>
            192.168.1.42
          </span>{' '}
          via DHCP · 1 Gbit/s
        </span>
      </div>
    </>
  );
}

function SetupAdmin({ cfg, set }: StepProps) {
  const len = cfg.adminPw?.length ?? 0;
  const strength = len >= 12 ? 'strong' : len >= 8 ? 'ok' : 'weak';
  return (
    <>
      <h2 className="stencil" style={{ fontSize: 20, margin: '0 0 6px' }}>
        Create the admin account.
      </h2>
      <p className="muted" style={{ fontSize: 13, marginBottom: 18 }}>
        This account has full control. You can revoke root SSH after setup from{' '}
        <span className="mono">Admin → Security</span>.
      </p>
      <div className="field-row">
        <div className="field">
          <label>Username</label>
          <input
            className="input input--mono"
            value={cfg.adminName}
            onChange={(e) => set('adminName', e.target.value)}
          />
        </div>
        <div className="field">
          <label>Email (for alerts)</label>
          <input
            className="input"
            value={cfg.adminEmail}
            onChange={(e) => set('adminEmail', e.target.value)}
            placeholder="you@company.tld"
          />
        </div>
      </div>
      <div className="field-row">
        <div className="field">
          <label>Password</label>
          <input
            type="password"
            className="input"
            value={cfg.adminPw}
            onChange={(e) => set('adminPw', e.target.value)}
          />
          <div
            className="hint"
            style={{
              color:
                strength === 'strong'
                  ? 'var(--h-success)'
                  : strength === 'ok'
                    ? 'var(--h-warn)'
                    : 'var(--h-danger)',
            }}
          >
            Strength: {strength}
          </div>
        </div>
        <div className="field">
          <label>Confirm</label>
          <input
            type="password"
            className="input"
            value={cfg.adminPw2}
            onChange={(e) => set('adminPw2', e.target.value)}
          />
          {cfg.adminPw && cfg.adminPw2 && cfg.adminPw !== cfg.adminPw2 && (
            <div className="err">Passwords don't match</div>
          )}
        </div>
      </div>
    </>
  );
}

interface ReviewProps {
  cfg: SetupConfig;
}

function SetupReview({ cfg }: ReviewProps) {
  const K = ({ label, val }: { label: string; val: string }) => (
    <>
      <dt>{label}</dt>
      <dd className="mono">{val}</dd>
    </>
  );
  return (
    <>
      <h2 className="stencil" style={{ fontSize: 20, margin: '0 0 6px' }}>
        Review and commit.
      </h2>
      <p className="muted" style={{ fontSize: 13, marginBottom: 18 }}>
        This is irreversible — selected disks will be wiped. Back out now if anything looks off.
      </p>
      <dl className="desc">
        <K label="Hostname" val={cfg.hostname} />
        <K label="Timezone" val={cfg.timezone} />
        <K label="Disks" val={cfg.disks.join(', ') || '(none)'} />
        <K label="Pool" val={cfg.pool} />
        <K label="Network" val={cfg.ipMode === 'dhcp' ? 'DHCP' : `${cfg.ip} via ${cfg.gateway}`} />
        <K label="DNS" val={cfg.dns} />
        <K label="Admin" val={`${cfg.adminName} (${cfg.adminEmail})`} />
        <K label="Telemetry" val={cfg.telemetry ? 'enabled' : 'off'} />
      </dl>
      <div className="alert alert--warn" style={{ marginTop: 18 }}>
        <I n="triangle-alert" s={14} />
        <span>
          After "Finish & reboot", data on <span className="mono">{cfg.disks.join(', ')}</span> will
          be erased.
        </span>
      </div>
    </>
  );
}
