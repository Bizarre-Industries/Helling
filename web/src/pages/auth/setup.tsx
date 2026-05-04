// Helling WebUI - first-admin setup.

import { useEffect, useId, useState } from 'react';
import { authSetup, authSetupStatus } from '../../api/generated/sdk.gen';
import type { Error as ApiError } from '../../api/generated/types.gen';
import { I } from '../../primitives/icon';

interface PageSetupProps {
  onDone?: () => void;
  onCancel?: () => void;
}

interface SetupForm {
  username: string;
  password: string;
  confirmPassword: string;
  setupToken: string;
}

type ToastBusGlobal = {
  toast?: { success?: (title: string, msg?: string) => void };
};

const getToast = () =>
  typeof window !== 'undefined' ? (window as unknown as ToastBusGlobal).toast : undefined;

const initialForm: SetupForm = {
  username: 'admin',
  password: '',
  confirmPassword: '',
  setupToken: '',
};

export default function PageSetup({ onDone, onCancel }: PageSetupProps) {
  const formId = useId();
  const [form, setForm] = useState<SetupForm>(initialForm);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const passwordMismatch =
    form.password !== '' && form.confirmPassword !== '' && form.password !== form.confirmPassword;

  const set = (field: keyof SetupForm, value: string) => {
    setForm((current) => ({ ...current, [field]: value }));
  };

  useEffect(() => {
    let cancelled = false;
    const checkSetupState = async () => {
      try {
        const res = await authSetupStatus();
        if (!cancelled && res?.data && !res.data.setup_required) {
          onDone?.();
        }
      } catch {
        // Keep the setup form visible when status cannot be reached.
      }
    };
    void checkSetupState();
    return () => {
      cancelled = true;
    };
  }, [onDone]);

  const finish = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const username = form.username.trim();
    const setupToken = form.setupToken.trim();
    if (!username || !form.password || !setupToken) {
      setError('Admin username, password, and setup token are required.');
      return;
    }
    if (form.password.length < 8) {
      setError('Password must be at least 8 characters.');
      return;
    }
    if (passwordMismatch) {
      setError('Passwords do not match.');
      return;
    }

    setBusy(true);
    setError(null);
    try {
      const res = await authSetup({
        body: {
          username,
          password: form.password,
          setup_token: setupToken,
        },
      });
      if (res?.error) {
        if ((res.error as Partial<ApiError>).code === 'already_setup') {
          onDone?.();
          return;
        }
        setError(setupErrorMessage(res.error));
        return;
      }
      getToast()?.success?.('Helling is ready', 'Admin account created.');
      onDone?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Cannot connect to Helling.');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div
      style={{
        minHeight: '100vh',
        background: 'var(--h-bg)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 24,
      }}
    >
      <section
        aria-labelledby={`${formId}-title`}
        style={{
          width: 'min(520px, 100%)',
          background: 'var(--h-surface)',
          border: '1px solid var(--h-border)',
          borderRadius: 'var(--h-radius)',
        }}
      >
        <div
          style={{
            padding: '22px 28px 18px',
            borderBottom: '1px solid var(--h-border)',
            display: 'flex',
            alignItems: 'center',
            gap: 16,
          }}
        >
          <img
            src="assets/mark-inverse.png"
            alt=""
            aria-hidden="true"
            style={{ width: 44, height: 44 }}
          />
          <div>
            <div className="mono dim" style={{ fontSize: 11 }}>
              FIRST-BOOT SETUP
            </div>
            <h1
              id={`${formId}-title`}
              className="stencil"
              style={{ fontSize: 22, margin: '2px 0 0' }}
            >
              Create the first admin.
            </h1>
          </div>
        </div>

        <form onSubmit={finish} style={{ padding: 28 }}>
          <p className="muted" style={{ fontSize: 13, marginTop: 0 }}>
            This one-time step creates the initial Helling administrator. It does not change disks,
            networking, or reboot the host.
          </p>

          {error && (
            <div
              className="alert alert--danger"
              role="alert"
              aria-live="assertive"
              style={{ margin: '18px 0' }}
            >
              <I n="triangle-alert" s={14} />
              <span>{error}</span>
            </div>
          )}

          <div className="field">
            <label htmlFor={`${formId}-username`}>Username</label>
            <input
              id={`${formId}-username`}
              className="input input--mono"
              autoComplete="username"
              required
              maxLength={64}
              value={form.username}
              onChange={(e) => set('username', e.target.value)}
            />
          </div>

          <div className="field">
            <label htmlFor={`${formId}-password`}>Password</label>
            <input
              id={`${formId}-password`}
              type="password"
              className="input"
              autoComplete="new-password"
              required
              minLength={8}
              maxLength={256}
              value={form.password}
              onChange={(e) => set('password', e.target.value)}
            />
            <div className="hint">Use at least 8 characters.</div>
          </div>

          <div className="field">
            <label htmlFor={`${formId}-confirm`}>Confirm password</label>
            <input
              id={`${formId}-confirm`}
              type="password"
              className="input"
              autoComplete="new-password"
              required
              minLength={8}
              maxLength={256}
              aria-invalid={passwordMismatch}
              aria-describedby={passwordMismatch ? `${formId}-confirm-error` : undefined}
              value={form.confirmPassword}
              onChange={(e) => set('confirmPassword', e.target.value)}
            />
            {passwordMismatch && (
              <div id={`${formId}-confirm-error`} className="err">
                Passwords do not match.
              </div>
            )}
          </div>

          <div className="field">
            <label htmlFor={`${formId}-token`}>Setup token</label>
            <input
              id={`${formId}-token`}
              type="password"
              className="input input--mono"
              autoComplete="one-time-code"
              required
              minLength={32}
              maxLength={128}
              aria-describedby={`${formId}-token-help`}
              value={form.setupToken}
              onChange={(e) => set('setupToken', e.target.value)}
            />
            <div id={`${formId}-token-help`} className="hint">
              On the installed host, run{' '}
              <span className="mono">sudo cat /etc/helling/setup-token</span>. If setup is locked,
              verify that file exists and is readable by hellingd.
            </div>
          </div>

          <div
            style={{
              display: 'flex',
              gap: 10,
              justifyContent: 'flex-end',
              marginTop: 22,
            }}
          >
            {onCancel && (
              <button type="button" className="btn" onClick={onCancel}>
                <I n="arrow-left" s={13} /> Back to sign in
              </button>
            )}
            <button type="submit" className="btn btn--primary" disabled={busy || passwordMismatch}>
              <I n="check" s={13} /> {busy ? 'Creating admin...' : 'Create account'}
            </button>
          </div>
        </form>
      </section>
    </div>
  );
}

function setupErrorMessage(error: ApiError | unknown): string {
  const apiError = error as Partial<ApiError> | undefined;
  switch (apiError?.code) {
    case 'already_setup':
      return 'Setup is already complete. Return to sign in.';
    case 'invalid_setup_token':
      return 'Setup token is invalid. Re-read /etc/helling/setup-token on the installed host.';
    case 'setup_locked':
      return 'Setup is locked because /etc/helling/setup-token is unavailable to hellingd.';
    case 'bad_request':
      return apiError.message || 'Check the username, password, and setup token.';
    default:
      return apiError?.message || 'Setup failed.';
  }
}
