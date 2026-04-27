// In-memory store for the JWT access token issued by /api/v1/auth/login.
//
// Per docs/spec/auth.md §2.2 (ADR-031, docs/standards/security.md):
//   - Access token storage: memory only.
//   - Refresh token storage: httpOnly, Secure, SameSite=Strict cookie set by hellingd.
//
// localStorage is XSS-exfiltratable, so the access token never touches it. On a
// page reload the in-memory token is gone; the app calls /api/v1/auth/refresh
// (which sends the refresh cookie automatically) at boot to mint a new access
// token. Refresh failure routes the user to /login.
//
// The 'auth:session-changed' event lets non-React consumers (and React hooks
// that subscribe via useSyncExternalStore) react to login/logout. The
// 'auth:session-expired' event is fired separately by client.ts on a 401.

let accessToken: string | null = null;

const SESSION_CHANGED_EVENT = 'auth:session-changed';

function emitChange(): void {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new CustomEvent(SESSION_CHANGED_EVENT));
}

export function getAccessToken(): string | null {
  return accessToken;
}

export function setAccessToken(token: string | null): void {
  if (accessToken === token) return;
  accessToken = token;
  emitChange();
}

export function clearAccessToken(): void {
  setAccessToken(null);
}

export function isAuthenticated(): boolean {
  return accessToken !== null;
}

export function subscribeAuthChange(listener: () => void): () => void {
  if (typeof window === 'undefined') return () => {};
  window.addEventListener(SESSION_CHANGED_EVENT, listener);
  return () => window.removeEventListener(SESSION_CHANGED_EVENT, listener);
}
