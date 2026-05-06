import { afterEach, describe, expect, it, vi } from 'vitest';
import { getAccessToken, setAccessToken } from './auth-store';
import { performLogout } from './logout';

afterEach(() => {
  vi.restoreAllMocks();
  setAccessToken(null);
});

describe('performLogout', () => {
  it('revokes the server session before clearing the access token', async () => {
    setAccessToken('jwt-session');
    const requestLogout = vi.fn().mockResolvedValue({});

    await performLogout(requestLogout);

    expect(requestLogout).toHaveBeenCalledWith({ throwOnError: true });
    expect(getAccessToken()).toBeNull();
  });

  it('still clears the local access token when server logout fails', async () => {
    setAccessToken('jwt-session');
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    const requestLogout = vi.fn().mockRejectedValue(new Error('offline'));

    await performLogout(requestLogout);

    expect(getAccessToken()).toBeNull();
    expect(console.warn).toHaveBeenCalledOnce();
  });
});
