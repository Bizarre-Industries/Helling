// Configures the hey-api-generated fetch client for Helling runtime.
//
// - Base URL defaults to same-origin; dev uses Vite's /api proxy from vite.config.ts,
//   production serves through Caddy (ADR-037).
// - Interceptor injects the stored JWT access token (auth-store.ts).
// - 401 responses clear the stored token; the app's route guard picks up the change
//   on the next render and redirects to /login. (Redirect logic lives in app.jsx.)

import { clearAccessToken, getAccessToken } from './auth-store';
import { client } from './generated/client.gen';

client.setConfig({
  baseUrl: '/api/v1',
});

client.interceptors.request.use((request) => {
  const token = getAccessToken();
  if (token) {
    request.headers.set('Authorization', `Bearer ${token}`);
  }
  return request;
});

client.interceptors.response.use((response) => {
  if (response.status === 401) {
    clearAccessToken();
    console.warn('Session expired or unauthorized. Redirecting to login.');
    window.dispatchEvent(
      new CustomEvent('auth:session-expired', {
        detail: { status: response.status, reason: 'unauthorized' },
      }),
    );
  }
  return response;
});

export { client };
