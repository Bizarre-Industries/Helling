// Helling WebUI — Login page (extracted from pages.jsx during Phase 2A).
//
// Wires the local-account credentials form to authLogin + authMfaComplete from the
// generated SDK. On success calls setAccessToken (memory-only per
// docs/spec/auth.md §2.2), which fires the 'auth:session-changed' event;
// app.jsx's subscription flips authed=true.
//
// Props:
//   onLogin       — optional callback fired after successful token issue.
//                   The auth-store subscription drives the actual route
//                   transition; this prop is kept for explicit callers.
//   onEnterSetup  — optional callback shown as a "First-time setup" link
//                   when set. app.jsx passes setSetupDone(false).

import { type KeyboardEvent, useId, useState } from 'react';
import { setAccessToken } from '../../api/auth-store';
import { authLogin, authMfaComplete } from '../../api/generated/sdk.gen';
import { I } from '../../primitives/icon';

interface PageLoginProps {
  onLogin?: () => void;
  onEnterSetup?: () => void;
}

type Stage = 'creds' | 'totp';

export default function PageLogin({ onLogin, onEnterSetup }: PageLoginProps) {
  const formId = useId();
  const [stage, setStage] = useState<Stage>('creds');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [totp, setTotp] = useState('');
  const [mfaToken, setMfaToken] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const completeLogin = (accessToken: string) => {
    setAccessToken(accessToken);
    if (typeof onLogin === 'function') onLogin();
  };

  const submitCreds = async () => {
    if (!username || !password) {
      setError('Username and password required.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const res = await authLogin({ body: { username, password } });
      const payload = res?.data?.data;
      if (res?.error || !payload) {
        const detail =
          (res?.error as { detail?: string; title?: string } | undefined)?.detail ||
          (res?.error as { detail?: string; title?: string } | undefined)?.title ||
          'Login failed.';
        setError(detail);
        return;
      }
      if (payload.mfa_required && payload.mfa_token) {
        setMfaToken(payload.mfa_token);
        setTotp('');
        setStage('totp');
        return;
      }
      if (payload.access_token) {
        completeLogin(payload.access_token);
        return;
      }
      setError('Login response missing access token.');
    } catch (e) {
      setError((e as Error)?.message || 'Network error.');
    } finally {
      setBusy(false);
    }
  };

  const submitTotp = async () => {
    if (!mfaToken) {
      setError('MFA session expired. Sign in again.');
      setStage('creds');
      return;
    }
    if (totp.length !== 6 && totp.length !== 16) {
      setError('Enter the 6-digit TOTP code or 16-character recovery code.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const res = await authMfaComplete({
        body: { mfa_token: mfaToken, totp_code: totp },
      });
      const payload = res?.data?.data;
      if (res?.error || !payload?.access_token) {
        const detail =
          (res?.error as { detail?: string; title?: string } | undefined)?.detail ||
          (res?.error as { detail?: string; title?: string } | undefined)?.title ||
          'MFA verification failed.';
        setError(detail);
        return;
      }
      completeLogin(payload.access_token);
    } catch (e) {
      setError((e as Error)?.message || 'Network error.');
    } finally {
      setBusy(false);
    }
  };

  const onCredsKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !busy) submitCreds();
  };
  const onTotpKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !busy) submitTotp();
  };

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'var(--h-bg)',
        display: 'grid',
        placeItems: 'center',
      }}
    >
      <div
        style={{
          position: 'absolute',
          top: 20,
          left: 20,
          display: 'flex',
          alignItems: 'center',
          gap: 10,
        }}
      >
        <img src="assets/mark-inverse.png" alt="Helling" style={{ width: 28, height: 28 }} />
        <div>
          <div className="stencil" style={{ fontSize: 16 }}>
            HELLING
          </div>
          <div className="mono dim" style={{ fontSize: 10, letterSpacing: '0.18em' }}>
            v0.1 · node-1
          </div>
        </div>
      </div>
      <div
        style={{
          position: 'absolute',
          bottom: 20,
          left: 20,
          right: 20,
          fontSize: 11,
          color: 'var(--h-text-3)',
          display: 'flex',
          justifyContent: 'space-between',
        }}
        className="mono"
      >
        <span>CATCH THE STARS.</span>
        <span>Debian · Incus · Podman · k3s</span>
      </div>

      <div className="card" style={{ width: 380, padding: 28, background: 'var(--h-surface)' }}>
        <div className="eyebrow" style={{ marginBottom: 14 }}>
          SIGN IN
        </div>
        <h1 className="stencil" style={{ fontSize: 24, margin: '0 0 22px' }}>
          Access your cluster
        </h1>

        {error && (
          <div
            role="alert"
            className="mono"
            style={{
              fontSize: 12,
              padding: '8px 10px',
              marginBottom: 14,
              borderRadius: 4,
              border: '1px solid var(--bzr-danger, #d4554b)',
              color: 'var(--bzr-danger, #d4554b)',
              background: 'rgba(212, 85, 75, 0.08)',
            }}
          >
            {error}
          </div>
        )}
        {stage === 'creds' ? (
          <>
            <label
              htmlFor={`${formId}-username`}
              className="mono dim"
              style={{ fontSize: 10, letterSpacing: '0.14em', textTransform: 'uppercase' }}
            >
              Username
            </label>
            <input
              id={`${formId}-username`}
              className="input"
              style={{ marginTop: 4, marginBottom: 14, width: '100%' }}
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              onKeyDown={onCredsKey}
              autoComplete="username"
              autoFocus
              disabled={busy}
            />
            <label
              htmlFor={`${formId}-password`}
              className="mono dim"
              style={{ fontSize: 10, letterSpacing: '0.14em', textTransform: 'uppercase' }}
            >
              Password
            </label>
            <input
              id={`${formId}-password`}
              type="password"
              className="input"
              style={{ marginTop: 4, marginBottom: 14, width: '100%' }}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              onKeyDown={onCredsKey}
              autoComplete="current-password"
              disabled={busy}
            />
            <button
              type="button"
              className="btn btn--primary"
              style={{ width: '100%', justifyContent: 'center' }}
              onClick={submitCreds}
              disabled={busy}
            >
              {busy ? 'Signing in…' : 'Continue'} <I n="arrow-right" s={13} />
            </button>
            <div className="mono dim" style={{ fontSize: 11, marginTop: 16, textAlign: 'center' }}>
              Helling local account
              {typeof onEnterSetup === 'function' && (
                <>
                  {' · '}
                  <button
                    type="button"
                    className="link"
                    style={{ border: 0, background: 'transparent', padding: 0, cursor: 'pointer' }}
                    onClick={onEnterSetup}
                  >
                    First-time setup
                  </button>
                </>
              )}
            </div>
          </>
        ) : (
          <>
            <div style={{ textAlign: 'center', marginBottom: 14 }}>
              <I n="shield" s={28} color="var(--h-accent)" />
            </div>
            <div
              style={{
                textAlign: 'center',
                color: 'var(--h-text-2)',
                marginBottom: 18,
                fontSize: 13,
              }}
            >
              Enter the 6-digit code from your
              <br />
              authenticator app or a 16-char recovery code.
            </div>
            <input
              className="input mono"
              style={{
                width: '100%',
                marginBottom: 18,
                textAlign: 'center',
                fontSize: 20,
                letterSpacing: '0.3em',
                padding: '12px 8px',
              }}
              value={totp}
              onChange={(e) => setTotp(e.target.value.replace(/\s/g, '').toLowerCase())}
              onKeyDown={onTotpKey}
              maxLength={16}
              autoComplete="one-time-code"
              autoFocus
              inputMode="text"
              disabled={busy}
              aria-label="TOTP or recovery code"
            />
            <button
              type="button"
              className="btn btn--primary"
              style={{ width: '100%', justifyContent: 'center' }}
              onClick={submitTotp}
              disabled={busy}
            >
              {busy ? 'Verifying…' : 'Sign in'} <I n="arrow-right" s={13} />
            </button>
            <div className="mono dim" style={{ fontSize: 11, marginTop: 14, textAlign: 'center' }}>
              <button
                type="button"
                className="link"
                style={{ border: 0, background: 'transparent', padding: 0, cursor: 'pointer' }}
                onClick={() => {
                  setStage('creds');
                  setError(null);
                  setTotp('');
                }}
              >
                Back
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
