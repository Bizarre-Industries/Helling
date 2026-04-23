import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

// Load design-system CSS first (tokens, type, fonts), then the Helling shell.
import './styles/ds/tokens.css';
import './styles/ds/colors_and_type.css';
import './styles/app.css';

// Configure the hey-api fetch client (baseUrl + auth interceptors) before any
// component renders; importing for its side-effect registers the interceptors
// on the generated client singleton.
import './api/client';

// Side-effect imports populate window.* globals referenced by App.
// Order matters: shell defines primitives, infra adds shared UI, pages add
// route bodies, app composes them.
import './shell.jsx';
import './infra.jsx';
import './pages.jsx';
// Additional page registrations kept in a separate module by design.
// Must load after `pages.jsx` so base page globals are available first.
import './pages2.jsx';
import App from './app.jsx';

const container = document.getElementById('root');
if (!container) {
  throw new Error('Helling WebUI: #root not found in document');
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: 0,
    },
  },
});

createRoot(container).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>,
);
