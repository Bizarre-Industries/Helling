// web/src/stores/system-store.ts
//
// Audit F-07: short-lived store for tick / latency / task / alert state that
// currently lives on `window.useTick`, `window.__getLatency`, `window.pushTask`,
// `window.pushAlert`. After Phase 3 (audit F-01/F-42) this content moves into
// TanStack Query keys and this file gets deleted.

import { useSyncExternalStore } from 'react';

export type SystemTask = {
  id: string;
  label: string;
  progress: number; // 0..1
  status?: 'queued' | 'running' | 'done' | 'failed';
};

export type SystemAlert = {
  id: string;
  severity: 'info' | 'warning' | 'danger';
  message: string;
  createdAt: number; // unix ms
};

type SystemState = {
  tick: number; // unix ms; bumped by 1.5s interval until Phase 3 SSE swap
  latencyMs: number;
  tasks: SystemTask[];
  alerts: SystemAlert[];
};

let state: SystemState = {
  tick: Date.now(),
  latencyMs: 0,
  tasks: [],
  alerts: [],
};

const listeners = new Set<() => void>();

function emit() {
  for (const l of listeners) l();
}

function setState(patch: Partial<SystemState>) {
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

export function useSystemStore() {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}

export function bumpTick(latencyMs?: number) {
  setState({
    tick: Date.now(),
    latencyMs: latencyMs ?? state.latencyMs,
  });
}

export function pushTask(task: SystemTask) {
  setState({ tasks: [...state.tasks, task] });
}

export function updateTask(id: string, patch: Partial<SystemTask>) {
  setState({
    tasks: state.tasks.map((t) => (t.id === id ? { ...t, ...patch } : t)),
  });
}

export function pushAlert(alert: SystemAlert) {
  setState({ alerts: [...state.alerts, alert] });
}

export function clearAlerts() {
  setState({ alerts: [] });
}
