// Helling WebUI — `<I>` icon primitive.
//
// Extracted from shell.jsx during Phase 2A so per-route page modules can
// ES-import the icon component without going through window.* coupling.
// shell.jsx still attaches the same component to window.I for legacy
// page modules that haven't been split yet.

import type { CSSProperties, ElementType } from 'react';
import { ICONS, type IconName } from '../icons';

interface IconProps {
  n: IconName | string;
  s?: number;
  style?: CSSProperties;
  color?: string;
}

export function I({ n, s = 14, style, color }: IconProps) {
  const Comp = (ICONS as Record<string, ElementType | undefined>)[n];
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
}
