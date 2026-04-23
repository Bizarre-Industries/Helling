// Thin localStorage wrapper for the JWT access token issued by /api/v1/auth/login.
// Refresh token lives in an httpOnly cookie per docs/spec/auth.md §2.2; this module
// only touches the access token, which is memory + localStorage per the same spec.

const ACCESS_TOKEN_KEY = 'helling.access_token';

export function getAccessToken(): string | null {
  try {
    return localStorage.getItem(ACCESS_TOKEN_KEY);
  } catch {
    return null;
  }
}

export function setAccessToken(token: string | null): void {
  try {
    if (token) {
      localStorage.setItem(ACCESS_TOKEN_KEY, token);
    } else {
      localStorage.removeItem(ACCESS_TOKEN_KEY);
    }
  } catch {
    // Silently ignore — the token simply won't persist across reloads.
  }
}

export function clearAccessToken(): void {
  setAccessToken(null);
}

export function isAuthenticated(): boolean {
  return getAccessToken() !== null;
}
