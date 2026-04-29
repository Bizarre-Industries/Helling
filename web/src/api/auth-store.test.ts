// Regression guard for Phase 1 F-37 + F-38 work.
//
// Spec: docs/spec/auth.md §2.2 — access token lives in memory only.
// localStorage must never be touched by auth-store. Subscribers must be
// notified on set/clear so the App route boundary flips authed state.
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  clearAccessToken,
  getAccessToken,
  isAuthenticated,
  setAccessToken,
  subscribeAuthChange,
} from './auth-store';

afterEach(() => {
  clearAccessToken();
  localStorage.clear();
});

describe('auth-store', () => {
  it('starts unauthenticated', () => {
    expect(getAccessToken()).toBeNull();
    expect(isAuthenticated()).toBe(false);
  });

  it('setAccessToken stores in memory and reports authenticated', () => {
    setAccessToken('jwt-abc');
    expect(getAccessToken()).toBe('jwt-abc');
    expect(isAuthenticated()).toBe(true);
  });

  it('never writes the access token to localStorage (audit F-37)', () => {
    setAccessToken('jwt-leak-canary');
    // Spec: docs/spec/auth.md §2.2 — access token is memory only.
    expect(localStorage.getItem('helling.access_token')).toBeNull();
    // No localStorage key should reference the token.
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      const value = key ? localStorage.getItem(key) : null;
      expect(value).not.toBe('jwt-leak-canary');
    }
  });

  it('clearAccessToken resets state', () => {
    setAccessToken('jwt-abc');
    clearAccessToken();
    expect(getAccessToken()).toBeNull();
    expect(isAuthenticated()).toBe(false);
  });

  it('subscribeAuthChange fires on set and clear', () => {
    const listener = vi.fn();
    const unsubscribe = subscribeAuthChange(listener);
    setAccessToken('jwt-1');
    setAccessToken('jwt-2');
    clearAccessToken();
    unsubscribe();
    setAccessToken('jwt-3'); // should not notify after unsubscribe
    expect(listener).toHaveBeenCalledTimes(3);
  });

  it('subscribeAuthChange does not fire when token is unchanged', () => {
    setAccessToken('jwt-same');
    const listener = vi.fn();
    const unsubscribe = subscribeAuthChange(listener);
    setAccessToken('jwt-same');
    unsubscribe();
    expect(listener).not.toHaveBeenCalled();
  });
});
