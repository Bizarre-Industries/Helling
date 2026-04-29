// Helling WebUI — `<Switch>` primitive.
//
// Extracted from infra.jsx during Phase 2A so per-route modules can
// ES-import the toggle without going through window.* coupling.
// infra.jsx still attaches the same component to window.Switch for
// legacy page modules that have not been split yet.

import type { ReactNode } from 'react';

interface SwitchProps {
  on?: boolean;
  onChange?: (value: boolean) => void;
  label?: ReactNode;
}

export function Switch({ on, onChange, label }: SwitchProps) {
  const sw = (
    <label className={`switch${on ? ' on' : ''}`}>
      <input type="checkbox" checked={!!on} onChange={(e) => onChange?.(e.target.checked)} />
      <span className="s-track" />
      <span className="s-thumb" />
    </label>
  );
  if (!label) return sw;
  return (
    <label
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 8,
        cursor: 'pointer',
        fontSize: 13,
      }}
    >
      {sw}
      <span>{label}</span>
    </label>
  );
}
