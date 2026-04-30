// Helling WebUI — typed access to legacy mock arrays.
//
// shell.jsx attaches mock data (INSTANCES, CONTAINERS, etc.) to window.*
// during initialization. Phase 2A page extractions need typed access to
// these arrays; this module is the single bridge.
//
// Phase 3A replaces these arrays with `useQuery` hooks fed by an MSW mock
// adapter — at that point this entire module can be deleted and importers
// migrated to the new hooks. Until then, every new .tsx page module
// imports mock data from here, never from `window` directly.
//
// Types are intentionally permissive (Record<string, unknown> shapes).
// Phase 3B introduces canonical IncusInstanceDetail / PodmanContainerDetail
// from the actual API surface; importers can narrow then.

export interface MockNode extends Record<string, unknown> {
  id: string;
  name: string;
  state: string;
}
export interface MockInstance extends Record<string, unknown> {
  name: string;
  type: string;
  node: string;
  status: string;
  cpuPct: number;
  ramPct: number;
  ip: string;
  uptime: string;
  tags: string[];
  health: number;
  backupAge: string;
}
export interface MockContainer extends Record<string, unknown> {
  name: string;
  image: string;
  status: string;
}
export interface MockTask extends Record<string, unknown> {
  id: string;
  type: string;
  target: string;
  status: string;
  progress: number;
}
export interface MockAlert extends Record<string, unknown> {
  id: string | number;
  severity: string;
  message: string;
  ts: string;
}
export interface MockAuditEntry extends Record<string, unknown> {
  ts: string;
  user: string;
  action: string;
  target: string;
  status: string;
  ip: string;
}
export interface MockWarning extends Record<string, unknown> {
  sev: 'danger' | 'warn' | 'info';
  msg: string;
  target: string;
}
export interface MockUser extends Record<string, unknown> {
  name: string;
  role: string;
  twofa: boolean;
  lastLogin: string;
}

export interface MockNetwork {
  name: string;
  type: string;
  cidr: string;
  dhcp: boolean;
  insts: number;
}

export interface MockSchedule {
  name: string;
  target: string;
  action: string;
  cron: string;
  next: string;
  last: string;
  on: boolean;
}

interface MocksGlobal {
  NODES?: MockNode[];
  INSTANCES?: MockInstance[];
  CONTAINERS?: MockContainer[];
  CLUSTERS?: Record<string, unknown>[];
  ALERTS?: MockAlert[];
  TASKS?: MockTask[];
  AUDIT?: MockAuditEntry[];
  POOLS?: Record<string, unknown>[];
  NETWORKS?: MockNetwork[];
  FW_RULES?: Record<string, unknown>[];
  SCHEDULES?: MockSchedule[];
  USERS?: MockUser[];
  TEMPLATES?: Record<string, unknown>[];
  SNAPSHOTS?: Record<string, unknown>[];
  BACKUPS?: Record<string, unknown>[];
  WARNINGS?: MockWarning[];
}

const w = (): MocksGlobal =>
  typeof window === 'undefined' ? {} : (window as unknown as MocksGlobal);

export const getNodes = (): MockNode[] => w().NODES ?? [];
export const getInstances = (): MockInstance[] => w().INSTANCES ?? [];
export const getContainers = (): MockContainer[] => w().CONTAINERS ?? [];
export const getClusters = (): Record<string, unknown>[] => w().CLUSTERS ?? [];
export const getAlerts = (): MockAlert[] => w().ALERTS ?? [];
export const getTasks = (): MockTask[] => w().TASKS ?? [];
export const getAudit = (): MockAuditEntry[] => w().AUDIT ?? [];
export const getPools = (): Record<string, unknown>[] => w().POOLS ?? [];
export const getNetworks = (): MockNetwork[] => w().NETWORKS ?? [];
export const getFirewallRules = (): Record<string, unknown>[] => w().FW_RULES ?? [];
export const getSchedules = (): MockSchedule[] => w().SCHEDULES ?? [];
export const getUsers = (): MockUser[] => w().USERS ?? [];
export const getTemplates = (): Record<string, unknown>[] => w().TEMPLATES ?? [];
export const getSnapshots = (): Record<string, unknown>[] => w().SNAPSHOTS ?? [];
export const getBackups = (): Record<string, unknown>[] => w().BACKUPS ?? [];
export const getWarnings = (): MockWarning[] => w().WARNINGS ?? [];
