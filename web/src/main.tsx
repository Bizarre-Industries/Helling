import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

// Load design-system CSS first (tokens, type, fonts), then the Helling shell.
import './styles/ds/tokens.css';
import './styles/ds/colors_and_type.css';
import './styles/app.css';

// Side-effect imports populate window.* globals referenced by App.
// Order matters: shell defines primitives, infra adds shared UI, pages add
// route bodies, app composes them.
import './shell.jsx';
import './infra.jsx';
import './pages.jsx';
import './pages2.jsx';
import App from './app.jsx';

const container = document.getElementById('root');
if (!container) {
  throw new Error('Helling WebUI: #root not found in document');
}

createRoot(container).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
