/// <reference types="vite/client" />

declare module '*.css' {
  const content: string;
  export default content;
}

declare module '*.jsx' {
  import type { ComponentType } from 'react';
  const value: ComponentType<Record<string, unknown>>;
  export default value;
}

// The ported prototype files attach many components and mock-data arrays to
// window for cross-file resolution. Keep Window loosely typed here so the
// .jsx files can destructure without TS complaint. Narrow per-key in a
// follow-up once individual components are converted to proper TSX modules.
interface Window {
  [key: string]: unknown;
}
