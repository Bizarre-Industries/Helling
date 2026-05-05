import { clearAccessToken } from './auth-store';
import { logout as logoutRequest } from './generated';

type LogoutRequest = typeof logoutRequest;

/**
 * Revoke the httpOnly session cookie on the daemon, then always clear the
 * in-memory access token so the UI leaves the authenticated route boundary.
 */
export async function performLogout(requestLogout: LogoutRequest = logoutRequest): Promise<void> {
  try {
    await requestLogout({ throwOnError: true });
  } catch (err) {
    // Local logout must still succeed if the server is offline or the cookie was
    // already invalidated. Keep this as a warning for dev visibility only.
    console.warn('server logout failed; clearing local session', err);
  } finally {
    clearAccessToken();
  }
}
