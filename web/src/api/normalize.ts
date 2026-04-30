// web/src/api/normalize.ts
//
// Audit F-02: canonical Instance + Container types + normalizers at the API
// boundary. Reconciles Incus's mixed-case "Running" vs mock lowercase "running"
// before the data hits a component. Phase D consumers always render against
// these canonical types.

import type { IncusInstance, PodmanContainer } from './queries';

export type CanonicalStatus = 'running' | 'stopped' | 'frozen' | 'error' | 'unknown';

export type Instance = {
  name: string;
  type: string;
  status: CanonicalStatus;
  /** Original raw status string for debugging. */
  rawStatus: string;
};

export type Container = {
  id: string;
  /** First name from PodmanContainer.Names, with leading slash stripped. */
  name: string;
  image: string;
  status: CanonicalStatus;
  rawStatus: string;
};

export function canonicalizeStatus(raw: string | undefined): CanonicalStatus {
  if (!raw) return 'unknown';
  const s = raw.toLowerCase().trim();
  if (s === 'running') return 'running';
  if (s === 'stopped' || s === 'exited' || s === 'created' || s === 'configured') return 'stopped';
  if (s === 'frozen' || s === 'paused') return 'frozen';
  if (s === 'error' || s === 'errored' || s === 'restarting') return 'error';
  return 'unknown';
}

export function normalizeIncusInstance(raw: IncusInstance): Instance {
  return {
    name: raw.name,
    type: raw.type,
    status: canonicalizeStatus(raw.status),
    rawStatus: raw.status,
  };
}

export function normalizeIncusInstances(raws: IncusInstance[] | undefined): Instance[] {
  return (raws ?? []).map(normalizeIncusInstance);
}

export function normalizePodmanContainer(raw: PodmanContainer): Container {
  const firstName = (raw.Names && raw.Names[0]) ?? raw.Id.slice(0, 12);
  return {
    id: raw.Id,
    name: firstName.startsWith('/') ? firstName.slice(1) : firstName,
    image: raw.Image,
    status: canonicalizeStatus(raw.State),
    rawStatus: raw.State,
  };
}

export function normalizePodmanContainers(raws: PodmanContainer[] | undefined): Container[] {
  return (raws ?? []).map(normalizePodmanContainer);
}
