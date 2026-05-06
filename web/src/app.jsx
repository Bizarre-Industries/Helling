/* Helling WebUI — app root */
/* eslint-disable */
import { Suspense, lazy, useCallback, useEffect, useState } from 'react';
import { clearAccessToken, isAuthenticated, subscribeAuthChange } from './api/auth-store';
import { performLogout } from './api/logout';
import { ErrorBoundary } from './error-boundary';
// Phase 2C (audit F-29): each extracted page is lazy-loaded so the initial
// chunk only ships shell + dashboard, and per-route navigation triggers a
// dedicated chunk fetch.
const PageAudit = lazy(() => import('./pages/admin/audit'));
const PageLogs = lazy(() => import('./pages/admin/logs'));
const PageOps = lazy(() => import('./pages/admin/ops'));
const PageUsers = lazy(() => import('./pages/admin/users'));
const PageLogin = lazy(() => import('./pages/auth/login'));
const PageSetup = lazy(() => import('./pages/auth/setup'));
const PageNetworking = lazy(() => import('./pages/networking'));
const PageBMC = lazy(() => import('./pages/bmc'));
const PageSchedules = lazy(() => import('./pages/schedules'));
const PageFirewall = lazy(() => import('./pages/firewall'));
const PageWebhooks = lazy(() => import('./pages/webhooks'));
const PageAPIDocs = lazy(() => import('./pages/api-docs'));
const PageSearchResults = lazy(() => import('./pages/search/results'));
import './shell.jsx';
import './infra.jsx';
import './pages.jsx';
import './pages2.jsx';

function PageSkeleton() {
  return (
    <div
      style={{
        padding: 24,
        color: 'var(--h-text-muted, #888)',
        fontSize: 13,
        textAlign: 'center',
      }}
    >
      Loading…
    </div>
  );
}

const {
  TopBar,
  ResourceTree,
  TaskDrawer,
  CommandPalette,
  ToastStack,
  ConfirmModal,
  PageDashboard,
  PageInstances,
  PageInstanceDetail,
  PageContainers,
  PageKubernetes,
  PageStorage,
  PageImages,
  PageBackups,
  PageTemplates,
  PageCluster,
  PageSettings,
  PageNewInstance,
  PageConsole,
  PageMetrics,
  PageAlerts,
  PageRBAC,
  PageFirewallEditor,
  PageMarketplace,
  PageFileBrowser,
  PageContainerDetail,
  PageUserDetail,
  WizardCreateInstance,
  ModalInstallApp,
  ModalFirewallRule,
  ModalCloudInit,
} = window;

const CRUMBS = {
  dashboard: ['Datacenter', 'Dashboard'],
  instances: ['Datacenter', 'Instances'],
  containers: ['Datacenter', 'Containers'],
  kubernetes: ['Datacenter', 'Kubernetes'],
  storage: ['Resources', 'Storage'],
  networking: ['Resources', 'Networking'],
  firewall: ['Resources', 'Firewall'],
  webhooks: ['Admin', 'Webhooks'],
  'api-docs': ['Admin', 'API Docs'],
  images: ['Resources', 'Images'],
  backups: ['Resources', 'Backups'],
  schedules: ['Resources', 'Schedules'],
  templates: ['Resources', 'Templates'],
  bmc: ['Resources', 'BMC'],
  cluster: ['Datacenter', 'Cluster'],
  metrics: ['Observability', 'Metrics'],
  alerts: ['Observability', 'Alerts'],
  marketplace: ['Resources', 'Marketplace'],
  users: ['Admin', 'Users'],
  audit: ['Admin', 'Audit'],
  logs: ['Admin', 'Logs'],
  ops: ['Admin', 'Operations'],
  settings: ['Admin', 'Settings'],
  search: ['Search', 'Results'],
};

const PAGE_PATHS = {
  dashboard: '/',
  instances: '/instances',
  containers: '/containers',
  kubernetes: '/kubernetes',
  storage: '/storage',
  networking: '/networking',
  firewall: '/firewall',
  webhooks: '/webhooks',
  'api-docs': '/api-docs',
  images: '/images',
  backups: '/backups',
  schedules: '/schedules',
  templates: '/templates',
  bmc: '/bmc',
  cluster: '/cluster',
  metrics: '/metrics',
  alerts: '/alerts',
  marketplace: '/marketplace',
  users: '/users',
  audit: '/audit',
  logs: '/logs',
  ops: '/ops',
  settings: '/settings',
  search: '/search',
};

const PATH_PAGES = Object.fromEntries(
  Object.entries(PAGE_PATHS).map(([page, path]) => [path, page]),
);

function pageFromLocation() {
  const path = window.location.pathname.replace(/\/+$/, '') || '/';
  return PATH_PAGES[path] || 'dashboard';
}

function initialSetupDone() {
  try {
    return localStorage.getItem('helling-setup-dismissed') === '1';
  } catch {
    return false;
  }
}

function initialDensity() {
  try {
    const v = localStorage.getItem('helling-density');
    return v === 'comfortable' || v === 'compact' ? v : 'compact';
  } catch {
    return 'compact';
  }
}

function initialTheme() {
  try {
    const stored = localStorage.getItem('helling-theme');
    if (stored === 'light' || stored === 'dark') return stored;
  } catch {}
  // Audit F-44 (b): first paint honors OS preference when no stored value.
  try {
    if (window.matchMedia?.('(prefers-color-scheme: light)').matches) return 'light';
  } catch {}
  return 'dark';
}

function getPageContent(page, nav) {
  if (page.startsWith('instance:')) {
    const name = page.split(':')[1];
    return {
      crumbs: ['Datacenter', 'Instances', name],
      body: <PageInstanceDetail name={name} onNav={nav} />,
    };
  }
  if (page.startsWith('console:')) {
    const name = page.split(':')[1];
    return {
      crumbs: ['Datacenter', 'Instances', name, 'Console'],
      body: <PageConsole name={name} onNav={nav} />,
    };
  }
  if (page.startsWith('container:')) {
    const name = page.split(':')[1];
    return {
      crumbs: ['Datacenter', 'Containers', name],
      body: <PageContainerDetail name={name} onNav={nav} />,
    };
  }
  if (page.startsWith('cluster:')) {
    return { crumbs: ['Datacenter', 'Kubernetes', page.split(':')[1]], body: <PageKubernetes /> };
  }
  if (page.startsWith('files:')) {
    const [, scope, id] = page.split(':');
    return {
      crumbs: [
        scope === 'backup' ? 'Resources' : 'Datacenter',
        scope === 'backup' ? 'Backups' : 'Containers',
        id,
        'Files',
      ],
      body: <PageFileBrowser scope={scope} id={id} onNav={nav} />,
    };
  }
  if (page.startsWith('rbac:')) {
    const u = page.split(':')[1];
    return { crumbs: ['Admin', 'Users', u], body: <PageUserDetail user={u} onNav={nav} /> };
  }
  if (page === 'new-instance') {
    return { crumbs: ['Datacenter', 'Instances', 'New'], body: <PageNewInstance onNav={nav} /> };
  }

  const M = {
    dashboard: PageDashboard,
    instances: PageInstances,
    containers: PageContainers,
    kubernetes: PageKubernetes,
    storage: PageStorage,
    networking: PageNetworking,
    firewall: PageFirewall,
    webhooks: PageWebhooks,
    'api-docs': PageAPIDocs,
    images: PageImages,
    backups: PageBackups,
    schedules: PageSchedules,
    templates: PageTemplates,
    bmc: PageBMC,
    cluster: PageCluster,
    users: PageUsers,
    audit: PageAudit,
    logs: PageLogs,
    ops: PageOps,
    settings: PageSettings,
    metrics: PageMetrics,
    alerts: PageAlerts,
    marketplace: PageMarketplace,
    search: PageSearchResults,
    access: PageRBAC,
    rbac: PageRBAC,
    'firewall-editor': PageFirewallEditor,
  };
  const P = M[page] || PageDashboard;
  return { crumbs: CRUMBS[page] || ['Datacenter', page], body: <P onNav={nav} /> };
}

function App() {
  const [authed, setAuthed] = useState(() => isAuthenticated());
  const [setupDone, setSetupDone] = useState(initialSetupDone);
  const [page, setPage] = useState(() => pageFromLocation());
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [density, setDensity] = useState(initialDensity);
  const [theme, setTheme] = useState(initialTheme);
  const [modalState, setModalState] = useState(null); // {kind, props}

  useEffect(() => {
    document.body.classList.toggle('light-mode', theme === 'light');
    try {
      localStorage.setItem('helling-theme', theme);
    } catch {}
  }, [theme]);

  // Track auth-store transitions so login/logout/expired-session events flip
  // the App route boundary without prop drilling. Per docs/spec/auth.md §2.2
  // the access token lives in memory only, so a refresh starts unauthed.
  useEffect(() => {
    const sync = () => setAuthed(isAuthenticated());
    const onExpired = () => {
      clearAccessToken();
      setAuthed(false);
      nav('dashboard', { replace: true });
    };
    const unsubscribe = subscribeAuthChange(sync);
    window.addEventListener('auth:session-expired', onExpired);
    return () => {
      unsubscribe();
      window.removeEventListener('auth:session-expired', onExpired);
    };
  }, []);

  // attach density + expose modal opener globally
  useEffect(() => {
    document.body.classList.toggle('density-comfortable', density === 'comfortable');
    try {
      localStorage.setItem('helling-density', density);
    } catch {}
  }, [density]);

  useEffect(() => {
    window.openModal = (kind, props) => setModalState({ kind, props: props || {} });
    window.closeModal = () => setModalState(null);
  }, []);

  const nav = useCallback((p, opts = {}) => {
    setPage(p);
    setPaletteOpen(false);
    const path = PAGE_PATHS[p];
    if (path && window.location.pathname !== path) {
      const method = opts.replace ? 'replaceState' : 'pushState';
      window.history[method]({ page: p }, '', path);
    }
  }, []);
  useEffect(() => {
    window.__nav = nav;
  }, [nav]);

  useEffect(() => {
    const onPopState = () => setPage(pageFromLocation());
    window.addEventListener('popstate', onPopState);
    return () => window.removeEventListener('popstate', onPopState);
  }, []);

  const handleLogout = useCallback(() => {
    void performLogout();
  }, []);

  useEffect(() => {
    const onKey = (e) => {
      const meta = e.metaKey || e.ctrlKey;
      if (meta && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setPaletteOpen(true);
      }
      if (e.ctrlKey && e.key === '`') {
        e.preventDefault();
        setDrawerOpen((d) => !d);
      }
      if (e.key === 'Escape' && paletteOpen) setPaletteOpen(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [paletteOpen]);

  if (!setupDone) {
    return (
      <>
        <Suspense fallback={<PageSkeleton />}>
          <PageSetup
            onDone={() => {
              try {
                localStorage.setItem('helling-setup-dismissed', '1');
              } catch {}
              setSetupDone(true);
            }}
            onCancel={() => {
              try {
                localStorage.setItem('helling-setup-dismissed', '1');
              } catch {}
              setSetupDone(true);
            }}
          />
        </Suspense>
        <ToastStack />
        {modalState && <ModalHost state={modalState} onClose={() => setModalState(null)} />}
      </>
    );
  }

  if (!authed) {
    return (
      <>
        <Suspense fallback={<PageSkeleton />}>
          <PageLogin
            onLogin={() => setAuthed(true)}
            onEnterSetup={() => {
              try {
                localStorage.removeItem('helling-setup-dismissed');
              } catch {}
              setSetupDone(false);
            }}
          />
        </Suspense>
        <ToastStack />
      </>
    );
  }

  const { crumbs, body } = getPageContent(page, nav);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh', overflow: 'hidden' }}>
      <TopBar
        onOpenPalette={() => setPaletteOpen(true)}
        page={page}
        crumbs={crumbs}
        onNav={nav}
        density={density}
        onDensity={setDensity}
        theme={theme}
        onTheme={setTheme}
        onLogout={handleLogout}
      />
      <div style={{ display: 'flex', flex: 1, overflow: 'hidden', minHeight: 0 }}>
        <ResourceTree page={page} onNav={nav} />
        <main
          style={{
            flex: 1,
            overflow: 'auto',
            background: 'var(--h-bg)',
            paddingBottom: drawerOpen ? 340 : 36,
          }}
        >
          <div key={page} className="page-fade">
            <ErrorBoundary scope={page} resetKey={page}>
              <Suspense fallback={<PageSkeleton />}>{body}</Suspense>
            </ErrorBoundary>
          </div>
        </main>
      </div>
      <TaskDrawer open={drawerOpen} onToggle={() => setDrawerOpen((d) => !d)} />
      <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} onNav={nav} />
      <ToastStack />
      {modalState && <ModalHost state={modalState} onClose={() => setModalState(null)} />}
    </div>
  );
}

// modal host — dispatches on kind
function ModalHost({ state, onClose }) {
  const { kind, props } = state;
  if (kind === 'confirm') {
    return <ConfirmModal open={true} onClose={onClose} {...props} />;
  }
  if (kind === 'create-vm') {
    return <WizardCreateInstance onClose={onClose} {...props} />;
  }
  if (kind === 'install-app') {
    return <ModalInstallApp onClose={onClose} {...props} />;
  }
  if (kind === 'new-rule') {
    return <ModalFirewallRule onClose={onClose} {...props} />;
  }
  if (kind === 'edit-cloud-init') {
    return <ModalCloudInit onClose={onClose} {...props} />;
  }
  return null;
}

export default App;
window.App = App;
