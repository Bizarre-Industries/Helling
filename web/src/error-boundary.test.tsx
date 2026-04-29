// Regression guard for Phase 1 F-39 work.
//
// The root + per-route ErrorBoundary is the cheapest blast-radius
// reduction in the codebase. This test confirms it catches throws,
// shows the right variant fallback, and resets when resetKey changes.
import { fireEvent, render, screen } from '@testing-library/react';
import { useState } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ErrorBoundary } from './error-boundary';

function Boom({ message = 'Boom!' }: { message?: string }): never {
  throw new Error(message);
}

let consoleErrorSpy: ReturnType<typeof vi.spyOn>;

beforeEach(() => {
  // React logs the caught error itself + ErrorBoundary's own console.error.
  // Silence both to keep test output clean; tests still assert on the DOM.
  consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
});

afterEach(() => {
  consoleErrorSpy.mockRestore();
});

describe('ErrorBoundary', () => {
  it('renders children when no error', () => {
    render(
      <ErrorBoundary scope="test">
        <div>healthy</div>
      </ErrorBoundary>,
    );
    expect(screen.getByText('healthy')).toBeInTheDocument();
  });

  it('renders page-variant fallback on throw with the scope and message', () => {
    render(
      <ErrorBoundary scope="instances">
        <Boom message="page exploded" />
      </ErrorBoundary>,
    );
    expect(screen.getByText('This page crashed')).toBeInTheDocument();
    expect(screen.getByText(/instances: page exploded/)).toBeInTheDocument();
  });

  it('renders root-variant fallback when variant="root"', () => {
    render(
      <ErrorBoundary scope="root" variant="root">
        <Boom message="catastrophe" />
      </ErrorBoundary>,
    );
    expect(screen.getByText('The console crashed')).toBeInTheDocument();
    expect(screen.getByText(/root: catastrophe/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Reload' })).toBeInTheDocument();
  });

  it('resets fallback when resetKey changes', () => {
    function Harness() {
      const [page, setPage] = useState<'broken' | 'fixed'>('broken');
      return (
        <>
          <button type="button" onClick={() => setPage('fixed')}>
            navigate
          </button>
          <ErrorBoundary scope={page} resetKey={page}>
            {page === 'broken' ? <Boom /> : <div>recovered</div>}
          </ErrorBoundary>
        </>
      );
    }
    render(<Harness />);
    expect(screen.getByText('This page crashed')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'navigate' }));
    expect(screen.getByText('recovered')).toBeInTheDocument();
  });
});
