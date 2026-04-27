/* Helling WebUI — app root */
/* eslint-disable */
import { useCallback, useEffect, useState } from 'react';
import { clearAccessToken, isAuthenticated, subscribeAuthChange } from './api/auth-store';
import { ErrorBoundary } from './error-boundary';
import PageLogin from './pages/auth/login';
import './shell.jsx';
import './infra.jsx';
import './pages.jsx';
import './pages2.jsx';

const {
  TopBar,
  ResourceTree,
  TaskDrawer,
  CommandPalette,
  ToastStack,
  Modal,
  ConfirmModal,
  PageDashboard,
  PageInstances,
  PageInstanceDetail,
  PageContainers,
  PageKubernetes,
  PageStorage,
  PageNetworking,
  PageFirewall,
  PageImages,
  PageBackups,
  PageSchedules,
  PageTemplates,
  PageBMC,
  PageCluster,
  PageUsers,
  PageAudit,
  PageLogs,
  PageOps,
  PageSettings,
  PageSetup,
  PageNewInstance,
  PageConsole,
  PageMetrics,
  PageAlerts,
  PageRBAC,
  PageFirewallEditor,
  PageMarketplace,
  PageFileBrowser,
  PageSearchResults,
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

function App() {
  const [authed, setAuthed] = useState(() => isAuthenticated());
  const [setupDone, setSetupDone] = useState(true); // once booted, skip setup
  const [page, setPage] = useState('dashboard');
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [density, setDensity] = useState(() => {
    try {
      const v = localStorage.getItem('helling-density');
      return v === 'comfortable' || v === 'compact' ? v : 'compact';
    } catch {
      return 'compact';
    }
  });
  const [theme, setTheme] = useState(() => {
    try {
      const stored = localStorage.getItem('helling-theme');
      if (stored === 'light' || stored === 'dark') return stored;
    } catch {}
    // Audit F-44 (b): first paint honors OS preference when no stored value.
    try {
      if (window.matchMedia?.('(prefers-color-scheme: light)').matches) return 'light';
    } catch {}
    return 'dark';
  });
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
      setPage('dashboard');
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

  const nav = useCallback((p) => {
    setPage(p);
    setPaletteOpen(false);
  }, []);
  useEffect(() => {
    window.__nav = nav;
  }, [nav]);

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
        <PageSetup onDone={() => setSetupDone(true)} />
        <ToastStack />
        {modalState && <ModalHost state={modalState} onClose={() => setModalState(null)} />}
      </>
    );
  }

  if (!authed) {
    return (
      <>
        <PageLogin onLogin={() => setAuthed(true)} onEnterSetup={() => setSetupDone(false)} />
        <ToastStack />
      </>
    );
  }

  // figure out crumbs + page render
  let crumbs, body;
  if (page.startsWith('instance:')) {
    const name = page.split(':')[1];
    crumbs = ['Datacenter', 'Instances', name];
    body = <PageInstanceDetail name={name} onNav={nav} />;
  } else if (page.startsWith('console:')) {
    const name = page.split(':')[1];
    crumbs = ['Datacenter', 'Instances', name, 'Console'];
    body = <PageConsole name={name} onNav={nav} />;
  } else if (page.startsWith('container:')) {
    const name = page.split(':')[1];
    crumbs = ['Datacenter', 'Containers', name];
    body = <PageContainerDetail name={name} onNav={nav} />;
  } else if (page.startsWith('cluster:')) {
    crumbs = ['Datacenter', 'Kubernetes', page.split(':')[1]];
    body = <PageKubernetes />;
  } else if (page.startsWith('files:')) {
    const [, scope, id] = page.split(':');
    crumbs = [
      scope === 'backup' ? 'Resources' : 'Datacenter',
      scope === 'backup' ? 'Backups' : 'Containers',
      id,
      'Files',
    ];
    body = <PageFileBrowser scope={scope} id={id} onNav={nav} />;
  } else if (page.startsWith('rbac:')) {
    const u = page.split(':')[1];
    crumbs = ['Admin', 'Users', u];
    body = <PageUserDetail user={u} onNav={nav} />;
  } else if (page === 'new-instance') {
    crumbs = ['Datacenter', 'Instances', 'New'];
    body = <PageNewInstance onNav={nav} />;
  } else {
    crumbs = CRUMBS[page] || ['Datacenter', page];
    const M = {
      dashboard: PageDashboard,
      instances: PageInstances,
      containers: PageContainers,
      kubernetes: PageKubernetes,
      storage: PageStorage,
      networking: PageNetworking,
      firewall: PageFirewall,
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
    body = <P onNav={nav} />;
  }

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
        onLogout={() => clearAccessToken()}
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
              {body}
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
