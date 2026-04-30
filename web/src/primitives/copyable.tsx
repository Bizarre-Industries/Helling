// Helling WebUI — Copyable text primitive (extracted from shell.jsx Phase 2A).
//
// Renders text inline with a copy-icon affordance; on click writes text to the
// clipboard and shows a 1.2s checkmark confirmation. Pages use this for IPs,
// IDs, hashes, CIDR blocks, etc.

import { useState } from 'react';
import type { CSSProperties, MouseEvent } from 'react';

import { I } from './icon';

type Props = {
  text: string;
  mono?: boolean;
  style?: CSSProperties;
};

export function Copyable({ text, mono = true, style }: Props) {
  const [ok, setOk] = useState(false);
  const doCopy = (e: MouseEvent<HTMLSpanElement>) => {
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
      style={{ display: 'inline-flex', alignItems: 'center', gap: 4, ...style }}
    >
      {text}
      <I n={ok ? 'check' : 'copy'} s={11} style={{ opacity: 0.5 }} />
    </span>
  );
}
