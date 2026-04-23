/* Helling — shared UI infra: modals, toasts, empties, charts, switch */
/* eslint-disable */
import React, { useState, useEffect } from 'react';

// ─── Toast system ───────────────────────────────────────────────
const ToastBus = {
  _listeners: new Set(),
  _id: 0,
  push(t) {
    const id = ++this._id;
    const toast = {id, kind:'info', ttl:4200, ...t};
    this._listeners.forEach(fn => fn({type:'push', toast}));
    if (toast.ttl > 0) setTimeout(() => this.dismiss(id), toast.ttl);
    return id;
  },
  dismiss(id) { this._listeners.forEach(fn => fn({type:'dismiss', id})); },
  subscribe(fn) { this._listeners.add(fn); return () => this._listeners.delete(fn); },
};
window.toast = {
  info:    (title, body, opts) => ToastBus.push({kind:'info',    title, body, ...(opts||{})}),
  success: (title, body, opts) => ToastBus.push({kind:'success', title, body, ...(opts||{})}),
  warn:    (title, body, opts) => ToastBus.push({kind:'warn',    title, body, ...(opts||{})}),
  danger:  (title, body, opts) => ToastBus.push({kind:'danger',  title, body, ...(opts||{})}),
};

function ToastStack() {
  const [items, setItems] = useState([]);
  useEffect(() => ToastBus.subscribe(ev => {
    if (ev.type === 'push') setItems(xs => [...xs, ev.toast]);
    else if (ev.type === 'dismiss') setItems(xs => xs.filter(x => x.id !== ev.id));
  }), []);
  const iconFor = k => k==='success'?'circle-check':k==='warn'?'triangle-alert':k==='danger'?'octagon-x':'info';
  return (
    <div className="toast-stack">
      {items.map(t => (
        <div key={t.id} className={"toast toast--"+t.kind}>
          <I n={iconFor(t.kind)} s={16}/>
          <div style={{flex:1, minWidth:0}}>
            <div className="t-title">{t.title}</div>
            {t.body && <div className="t-body">{t.body}</div>}
          </div>
          {t.action && (
            <button className="t-action" onClick={()=>{ t.action.run?.(); ToastBus.dismiss(t.id); }}>
              {t.action.label}
            </button>
          )}
          <button className="t-close" onClick={()=>ToastBus.dismiss(t.id)}>
            <I n="x" s={12}/>
          </button>
        </div>
      ))}
    </div>
  );
}

// ─── Modal ──────────────────────────────────────────────────────
function Modal({open, onClose, title, size, danger, children, footer}) {
  useEffect(() => {
    if (!open) return;
    const onKey = e => { if (e.key === 'Escape') onClose?.(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);
  if (!open) return null;
  const sizeCls = size === 'lg' ? ' modal--lg' : size === 'xl' ? ' modal--xl' : size === 'wizard' ? ' modal--wizard' : '';
  return (
    <div className="modal-bg" onClick={onClose}>
      <div className={"modal"+sizeCls+(danger?' modal--danger':'')} onClick={e=>e.stopPropagation()}>
        <header>
          <div className="stencil" style={{fontSize:14}}>{title}</div>
          <button className="btn btn--sm btn--ghost" onClick={onClose}><I n="x" s={14}/></button>
        </header>
        <div className="body">{children}</div>
        {footer && <footer>{footer}</footer>}
      </div>
    </div>
  );
}

// ─── ConfirmModal ───────────────────────────────────────────────
function ConfirmModal({open, onClose, onConfirm, title, body, danger, confirmText='Confirm', confirmMatch}) {
  const [typed, setTyped] = useState('');
  useEffect(() => { if (open) setTyped(''); }, [open]);
  const canConfirm = !confirmMatch || typed === confirmMatch;
  return (
    <Modal open={open} onClose={onClose} title={title} danger={danger}
      footer={<>
        <button className="btn btn--sm" onClick={onClose}>Cancel</button>
        <button
          className={"btn btn--sm "+(danger?'btn--danger':'btn--primary')}
          disabled={!canConfirm}
          onClick={() => { onConfirm?.(); onClose?.(); }}>
          {confirmText}
        </button>
      </>}>
      <div style={{fontSize:13, color:'var(--h-text)', lineHeight:1.55}}>{body}</div>
      {confirmMatch && (
        <div className="confirm-hint">
          Type <span className="mono" style={{fontWeight:700}}>{confirmMatch}</span> to confirm.
          <input className="input input--mono" style={{marginTop:8, width:'100%'}}
            value={typed} onChange={e=>setTyped(e.target.value)} autoFocus/>
        </div>
      )}
    </Modal>
  );
}

// ─── EmptyState ─────────────────────────────────────────────────
function EmptyState({icon='inbox', title, body, action}) {
  return (
    <div className="empty">
      <div className="glyph"><I n={icon} s={32}/></div>
      <h3>{title}</h3>
      {body && <p>{body}</p>}
      {action}
    </div>
  );
}

// ─── Switch ─────────────────────────────────────────────────────
function Switch({on, onChange, label}) {
  const sw = (
    <label className={"switch"+(on?' on':'')}>
      <input type="checkbox" checked={!!on} onChange={e=>onChange?.(e.target.checked)}/>
      <span className="s-track"/><span className="s-thumb"/>
    </label>
  );
  if (!label) return sw;
  return (
    <label style={{display:'inline-flex', alignItems:'center', gap:8, cursor:'pointer', fontSize:13}}>
      {sw}<span>{label}</span>
    </label>
  );
}

// ─── Sparkline ──────────────────────────────────────────────────
function Sparkline({data, color='var(--h-accent)', w=120, h=24, fill=true}) {
  if (!data || !data.length) return <svg className="sparkline" viewBox={`0 0 ${w} ${h}`}/>;
  const max = Math.max(...data, 1), min = Math.min(...data, 0);
  const range = max - min || 1;
  const step = w / (data.length - 1 || 1);
  const pts = data.map((v, i) => [i*step, h - ((v-min)/range)*(h-2) - 1]);
  const path = pts.map((p,i)=>(i?'L':'M')+p[0].toFixed(1)+','+p[1].toFixed(1)).join(' ');
  const area = path + ` L${w},${h} L0,${h} Z`;
  return (
    <svg className="sparkline" viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none">
      {fill && <path d={area} fill={color} opacity="0.15"/>}
      <path d={path} fill="none" stroke={color} strokeWidth="1.25"/>
    </svg>
  );
}

// ─── MultiChart — time-series with grid + axes ────────────────
function MultiChart({series, w=800, h=260, yMax, yLabel='', xLabels=[]}) {
  // series: [{name, color, data:[]}]
  const pad = {t:16, r:16, b:26, l:44};
  const cw = w - pad.l - pad.r, ch = h - pad.t - pad.b;
  const n = series[0]?.data.length || 1;
  const max = yMax || Math.max(1, ...series.flatMap(s => s.data));
  const step = cw / Math.max(1, n-1);
  const scaleY = v => ch - (v/max)*ch;
  const gridY = [0, 0.25, 0.5, 0.75, 1].map(f => f*max);
  const gridX = Math.min(8, n);
  return (
    <svg viewBox={`0 0 ${w} ${h}`} className="chart">
      {/* y grid */}
      {gridY.map((v, i) => (
        <g key={i}>
          <line x1={pad.l} x2={pad.l+cw} y1={pad.t+scaleY(v)} y2={pad.t+scaleY(v)}
            stroke="rgba(255,255,255,0.05)" strokeDasharray="2 4"/>
          <text x={pad.l-6} y={pad.t+scaleY(v)+3} textAnchor="end"
            fill="var(--h-text-3)" fontSize="10" fontFamily="var(--bzr-font-mono)">
            {Math.round(v)}
          </text>
        </g>
      ))}
      {/* x labels */}
      {xLabels.map((lab, i) => (
        <text key={i} x={pad.l + (i/(xLabels.length-1||1))*cw} y={pad.t+ch+16}
          textAnchor="middle" fill="var(--h-text-3)" fontSize="10" fontFamily="var(--bzr-font-mono)">
          {lab}
        </text>
      ))}
      {/* series */}
      {series.map((s, si) => {
        const pts = s.data.map((v,i)=>[pad.l+i*step, pad.t+scaleY(v)]);
        const path = pts.map((p,i)=>(i?'L':'M')+p[0].toFixed(1)+','+p[1].toFixed(1)).join(' ');
        const area = path + ` L${pad.l+cw},${pad.t+ch} L${pad.l},${pad.t+ch} Z`;
        return (
          <g key={si}>
            <path d={area} fill={s.color} opacity="0.08"/>
            <path d={path} fill="none" stroke={s.color} strokeWidth="1.5"/>
          </g>
        );
      })}
      {/* y label */}
      {yLabel && <text x={10} y={pad.t-6} fill="var(--h-text-3)" fontSize="10"
        fontFamily="var(--bzr-font-mono)" letterSpacing="0.12em">{yLabel.toUpperCase()}</text>}
      {/* legend */}
      <g>
        {series.map((s,i) => (
          <g key={i} transform={`translate(${pad.l + i*120}, ${h-6})`}>
            <rect width="10" height="2" y="-3" fill={s.color}/>
            <text x={14} y={0} fontSize="10" fontFamily="var(--bzr-font-mono)"
              fill="var(--h-text-2)" letterSpacing="0.08em">{s.name.toUpperCase()}</text>
          </g>
        ))}
      </g>
    </svg>
  );
}

// ─── File path crumbs ───────────────────────────────────────────
function FilePath({path, onNav}) {
  const parts = path.split('/').filter(Boolean);
  return (
    <div className="fb-path">
      <span className="seg" onClick={()=>onNav('/')}>/</span>
      {parts.map((p, i) => (
        <React.Fragment key={i}>
          <span className="seg" onClick={()=>onNav('/' + parts.slice(0,i+1).join('/'))}>{p}</span>
          {i < parts.length - 1 && <span className="sep">/</span>}
        </React.Fragment>
      ))}
    </div>
  );
}

// expose
Object.assign(window, {
  ToastBus, ToastStack, Modal, ConfirmModal, EmptyState, Switch,
  Sparkline, MultiChart, FilePath,
});
