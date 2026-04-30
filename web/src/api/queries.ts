// Real-data hooks for the Helling WebUI dashboard (PR G).
//
// Thin TanStack Query wrappers that hit the proxy routes mounted by hellingd
// (ADR-014) using the same auth store as the hey-api generated client. Per
// docs/spec/webui-spec.md the dashboard derives counts from these responses.
// When the caller is unauthenticated the hooks stay disabled so the dashboard
// keeps rendering mock data during dev.

import { useQuery } from '@tanstack/react-query';

import { getAccessToken } from './auth-store';

/** Incus instance row as returned by /api/incus/1.0/instances?recursion=1. */
export type IncusInstance = {
  name: string;
  status: string;
  type: string;
};

/** Podman container row as returned by /libpod/containers/json. */
export type PodmanContainer = {
  Id: string;
  Names: string[];
  State: string;
  Image: string;
};

const STALE_TIME_MS = 15_000;

async function authedFetch(path: string): Promise<Response> {
  const token = getAccessToken();
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return fetch(path, { headers, credentials: 'same-origin' });
}

async function fetchIncusInstances(): Promise<IncusInstance[]> {
  const resp = await authedFetch('/api/incus/1.0/instances?recursion=1');
  if (!resp.ok) {
    throw new Error(`incus instances: HTTP ${resp.status}`);
  }
  const body = (await resp.json()) as { metadata?: IncusInstance[] } | IncusInstance[];
  if (Array.isArray(body)) {
    return body;
  }
  return body.metadata ?? [];
}

async function fetchPodmanContainers(): Promise<PodmanContainer[]> {
  const resp = await authedFetch('/api/podman/libpod/containers/json?all=true');
  if (!resp.ok) {
    throw new Error(`podman containers: HTTP ${resp.status}`);
  }
  const body = (await resp.json()) as PodmanContainer[] | null;
  return body ?? [];
}

export function useInstancesQuery() {
  return useQuery<IncusInstance[], Error>({
    queryKey: ['incus', 'instances'],
    queryFn: fetchIncusInstances,
    staleTime: STALE_TIME_MS,
    enabled: Boolean(getAccessToken()),
    retry: false,
  });
}

export function useContainersQuery() {
  return useQuery<PodmanContainer[], Error>({
    queryKey: ['podman', 'containers'],
    queryFn: fetchPodmanContainers,
    staleTime: STALE_TIME_MS,
    enabled: Boolean(getAccessToken()),
    retry: false,
  });
}

/** Incus storage pool row as returned by /api/incus/1.0/storage-pools?recursion=1. */
export type IncusStoragePool = {
  name: string;
  driver: string;
  status?: string;
  used_by?: string[];
};

/** Incus network row as returned by /api/incus/1.0/networks?recursion=1. */
export type IncusNetwork = {
  name: string;
  type: string;
  managed?: boolean;
  used_by?: string[];
};

/** Incus image row as returned by /api/incus/1.0/images?recursion=1. */
export type IncusImage = {
  fingerprint: string;
  filename?: string;
  size?: number;
  public?: boolean;
};

/** Incus operation row as returned by /api/incus/1.0/operations?recursion=1. */
export type IncusOperation = {
  id: string;
  description?: string;
  status?: string;
  status_code?: number;
  created_at?: string;
};

async function fetchIncusList<T>(path: string, label: string): Promise<T[]> {
  const resp = await authedFetch(path);
  if (!resp.ok) {
    throw new Error(`${label}: HTTP ${resp.status}`);
  }
  const body = (await resp.json()) as { metadata?: T[] } | T[];
  if (Array.isArray(body)) {
    return body;
  }
  return body.metadata ?? [];
}

export function useStoragePoolsQuery() {
  return useQuery<IncusStoragePool[], Error>({
    queryKey: ['incus', 'storage-pools'],
    queryFn: () => fetchIncusList<IncusStoragePool>('/api/incus/1.0/storage-pools?recursion=1', 'incus storage-pools'),
    staleTime: STALE_TIME_MS,
    enabled: Boolean(getAccessToken()),
    retry: false,
  });
}

export function useNetworksQuery() {
  return useQuery<IncusNetwork[], Error>({
    queryKey: ['incus', 'networks'],
    queryFn: () => fetchIncusList<IncusNetwork>('/api/incus/1.0/networks?recursion=1', 'incus networks'),
    staleTime: STALE_TIME_MS,
    enabled: Boolean(getAccessToken()),
    retry: false,
  });
}

export function useImagesQuery() {
  return useQuery<IncusImage[], Error>({
    queryKey: ['incus', 'images'],
    queryFn: () => fetchIncusList<IncusImage>('/api/incus/1.0/images?recursion=1', 'incus images'),
    staleTime: STALE_TIME_MS,
    enabled: Boolean(getAccessToken()),
    retry: false,
  });
}

export function useTasksQuery() {
  return useQuery<IncusOperation[], Error>({
    queryKey: ['incus', 'operations'],
    queryFn: () => fetchIncusList<IncusOperation>('/api/incus/1.0/operations?recursion=1', 'incus operations'),
    staleTime: STALE_TIME_MS,
    enabled: Boolean(getAccessToken()),
    retry: false,
  });
}

/** Summary counts used by PageDashboard. Returns mock fallbacks when queries
 * are disabled (no access token) or still loading. */
export function useDashboardCounts(
  mockInstances: number,
  mockRunning: number,
  mockContainers: number,
  mockContainersRunning: number,
) {
  const instances = useInstancesQuery();
  const containers = useContainersQuery();

  const live = Boolean(getAccessToken());
  if (!live) {
    return {
      live: false,
      totalInstances: mockInstances,
      runningInstances: mockRunning,
      totalContainers: mockContainers,
      runningContainers: mockContainersRunning,
      loading: false,
    };
  }
  if (instances.isLoading || containers.isLoading) {
    return {
      live: true,
      totalInstances: mockInstances,
      runningInstances: mockRunning,
      totalContainers: mockContainers,
      runningContainers: mockContainersRunning,
      loading: true,
    };
  }
  const inst = instances.data ?? [];
  const cts = containers.data ?? [];
  return {
    live: true,
    totalInstances: inst.length,
    runningInstances: inst.filter((i) => i.status === 'Running').length,
    totalContainers: cts.length,
    runningContainers: cts.filter((c) => c.State === 'running').length,
    loading: false,
  };
}
