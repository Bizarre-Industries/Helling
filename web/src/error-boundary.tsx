/* Helling WebUI — error boundary
 *
 * Audit F-39: any throw in any page whitescreens the app today. React 19
 * still requires a class component for getDerivedStateFromError /
 * componentDidCatch.
 *
 * Two layers expected (audit Section 03):
 *   - root in main.tsx around <App />: catches throws above the route boundary
 *   - per-route in app.jsx around {body}: keeps topbar + sidebar + drawers
 *     alive so the user can navigate away from a broken page
 *
 * The `scope` prop is shown in the fallback so we can tell which boundary
 * tripped from screenshots / bug reports. The optional `resetKey` prop lets
 * the parent reset error state when the route changes (pass `key={page}`).
 */
import { Component, type ErrorInfo, type ReactNode } from 'react';

interface ErrorBoundaryProps {
  children?: ReactNode;
  scope?: string;
  variant?: 'root' | 'page';
  resetKey?: unknown;
}

interface ErrorBoundaryState {
  error: Error | null;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    // Surface to console for now. A future PR wires this to a real reporter
    // (audit F-46/F-49 — supply chain + observability follow-up).
    // eslint-disable-next-line no-console
    console.error(`[ErrorBoundary:${this.props.scope || 'unknown'}]`, error, info);
  }

  componentDidUpdate(prevProps: ErrorBoundaryProps): void {
    if (this.state.error && prevProps.resetKey !== this.props.resetKey) {
      this.setState({ error: null });
    }
  }

  render(): ReactNode {
    if (!this.state.error) return this.props.children;

    const { scope = 'app', variant = 'page' } = this.props;
    const message = this.state.error?.message || String(this.state.error) || 'Unknown error';

    if (variant === 'root') {
      return (
        <div
          role="alert"
          style={{
            minHeight: '100vh',
            display: 'grid',
            placeItems: 'center',
            background: 'var(--h-bg, #0a0a0a)',
            color: 'var(--h-text, #e8e8e6)',
            padding: 24,
          }}
        >
          <div style={{ maxWidth: 520, width: '100%' }}>
            <div
              className="mono"
              style={{
                fontSize: 11,
                letterSpacing: '0.2em',
                color: 'var(--bzr-danger, #d4554b)',
                marginBottom: 8,
              }}
            >
              ✦ HELLING / FATAL
            </div>
            <h1 className="stencil" style={{ fontSize: 28, margin: '0 0 14px' }}>
              The console crashed
            </h1>
            <p style={{ color: 'var(--h-text-2, #9a9a96)', marginBottom: 18 }}>
              An unrecoverable error escaped the WebUI. The Helling daemon and your workloads are
              unaffected. Reload to try again.
            </p>
            <pre
              className="mono"
              style={{
                fontSize: 12,
                padding: 12,
                borderRadius: 4,
                border: '1px solid var(--h-border, #2a2a28)',
                background: 'rgba(0,0,0,0.3)',
                color: 'var(--bzr-danger, #d4554b)',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                marginBottom: 18,
              }}
            >
              {scope}: {message}
            </pre>
            <button
              type="button"
              className="btn btn--primary"
              onClick={() => window.location.reload()}
            >
              Reload
            </button>
          </div>
        </div>
      );
    }

    return (
      <div
        role="alert"
        style={{
          padding: 24,
          margin: 24,
          borderRadius: 4,
          border: '1px solid var(--bzr-danger, #d4554b)',
          background: 'rgba(212, 85, 75, 0.06)',
        }}
      >
        <div
          className="mono"
          style={{
            fontSize: 11,
            letterSpacing: '0.2em',
            color: 'var(--bzr-danger, #d4554b)',
            marginBottom: 6,
          }}
        >
          ✦ PAGE / ERROR
        </div>
        <h2 className="stencil" style={{ fontSize: 20, margin: '0 0 10px' }}>
          This page crashed
        </h2>
        <p style={{ color: 'var(--h-text-2, #9a9a96)', marginBottom: 12 }}>
          The rest of the console is fine. Pick another page from the sidebar or reload to retry.
        </p>
        <pre
          className="mono"
          style={{
            fontSize: 12,
            padding: 10,
            borderRadius: 4,
            border: '1px solid var(--h-border, #2a2a28)',
            background: 'rgba(0,0,0,0.25)',
            color: 'var(--bzr-danger, #d4554b)',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
            margin: 0,
          }}
        >
          {scope}: {message}
        </pre>
      </div>
    );
  }
}
