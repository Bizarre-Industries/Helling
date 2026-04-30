// web/src/stores/ui-store.ts
//
// Audit F-07: replace `window.openModal`, `window.closeModal`, `window.toast`,
// `window.__nav`, `window.useStore`, etc. with a typed external store consumed
// via `useSyncExternalStore`. Avoids zustand/redux deps until ConfigProvider
// (Phase 4) lands and we have antd's own state primitives.
//
// Persistence:
//   density  -> localStorage 'helling.density'  (compact|comfortable)
//   theme    -> localStorage 'helling.theme'    (light|dark|system)
//
// Toasts and modal state are ephemeral (no persistence).

import { useSyncExternalStore } from 'react';

export type Density = 'compact' | 'comfortable';
export type Theme = 'light' | 'dark' | 'system';

export type ToastTone = 'info' | 'success' | 'warning' | 'danger';

export type Toast = {
  id: string;
  tone: ToastTone;
  message: string;
  ttl?: number; // ms; default 4000
};

export type Modal = {
  kind: string;
  payload?: unknown;
} | null;

type UIState = {
  density: Density;
  theme: Theme;
  modal: Modal;
  paletteOpen: boolean;
  drawerOpen: boolean;
  toasts: Toast[];
};

const DENSITY_KEY = 'helling.density';
const THEME_KEY = 'helling.theme';

function readPersistedDensity(): Density {
  if (typeof localStorage === 'undefined') return 'comfortable';
  const v = localStorage.getItem(DENSITY_KEY);
  return v === 'compact' || v === 'comfortable' ? v : 'comfortable';
}

function readPersistedTheme(): Theme {
  if (typeof localStorage === 'undefined') return 'system';
  const v = localStorage.getItem(THEME_KEY);
  return v === 'light' || v === 'dark' || v === 'system' ? v : 'system';
}

let state: UIState = {
  density: readPersistedDensity(),
  theme: readPersistedTheme(),
  modal: null,
  paletteOpen: false,
  drawerOpen: false,
  toasts: [],
};

const listeners = new Set<() => void>();

function emit() {
  for (const l of listeners) l();
}

function setState(patch: Partial<UIState>) {
  state = { ...state, ...patch };
  emit();
}

function subscribe(cb: () => void) {
  listeners.add(cb);
  return () => listeners.delete(cb);
}

function getSnapshot() {
  return state;
}

// ---- public API ----

export function useUIStore() {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}

export function setDensity(d: Density) {
  if (typeof localStorage !== 'undefined') localStorage.setItem(DENSITY_KEY, d);
  setState({ density: d });
}

export function setTheme(t: Theme) {
  if (typeof localStorage !== 'undefined') localStorage.setItem(THEME_KEY, t);
  setState({ theme: t });
}

export function openModal(kind: string, payload?: unknown) {
  setState({ modal: { kind, payload } });
}

export function closeModal() {
  setState({ modal: null });
}

export function setPaletteOpen(open: boolean) {
  setState({ paletteOpen: open });
}

export function setDrawerOpen(open: boolean) {
  setState({ drawerOpen: open });
}

let toastSeq = 0;

export function pushToast(tone: ToastTone, message: string, ttl = 4000): string {
  const id = `t_${Date.now()}_${++toastSeq}`;
  setState({ toasts: [...state.toasts, { id, tone, message, ttl }] });
  if (ttl > 0) {
    setTimeout(() => dismissToast(id), ttl);
  }
  return id;
}

export function dismissToast(id: string) {
  setState({ toasts: state.toasts.filter((t) => t.id !== id) });
}
