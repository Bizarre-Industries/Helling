// web/src/components/QueryStateView.tsx
//
// Audit F-04: a single wrapper that renders skeleton on loading, an error card
// on error, and an empty-state card when the query returns an empty list.
// Phase D list pages wrap their query result with this so they don't each
// hand-roll the three-state UI.
//
// Stays stack-agnostic until ADR-051's antd port (Phase 4) lands; the antd
// equivalent is <Skeleton> / <Result> / <Empty>, which a Phase 4 PR can swap
// in by editing only this file.

import type { CSSProperties, ReactNode } from 'react';

export type QueryLike<T> = {
  isLoading: boolean;
  error: unknown | null;
  data: T | undefined;
  refetch?: () => void;
};

export type QueryStateViewProps<T> = {
  query: QueryLike<T>;
  /** Returns true when data should be considered empty. Default: array.length === 0. */
  isEmpty?: (data: T) => boolean;
  /** Rendered when loading. Default: skeleton placeholder. */
  loadingFallback?: ReactNode;
  /** Rendered when error is non-null. Default: error card. */
  errorFallback?: (err: unknown) => ReactNode;
  /** Rendered when data is empty. Default: empty card. */
  emptyFallback?: ReactNode;
  /** Children render with the resolved non-empty data. */
  children: (data: T) => ReactNode;
};

const skeletonStyle: CSSProperties = {
  padding: 24,
  borderRadius: 8,
  background: 'var(--h-tint-hover, rgba(255,255,255,0.04))',
  color: 'var(--h-text-muted, #888)',
  textAlign: 'center',
  fontSize: 14,
};

const cardStyle: CSSProperties = {
  padding: 24,
  borderRadius: 8,
  background: 'var(--h-tint-pressed, rgba(255,255,255,0.06))',
  border: '1px solid var(--h-divider-soft, rgba(255,255,255,0.08))',
  textAlign: 'center',
};

function defaultIsEmpty<T>(data: T): boolean {
  if (Array.isArray(data)) return data.length === 0;
  return false;
}

type ErrorEnvelope = {
  code?: unknown;
  message?: unknown;
  details?: unknown;
  status?: unknown;
};

const nonRetryableStatus = new Set([400, 401, 403, 404, 409, 422]);
const nonRetryableCodes = [
  'bad_request',
  'conflict',
  'forbidden',
  'invalid',
  'mfa',
  'no_session',
  'not_found',
  'unauthorized',
];

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function stringifyDetails(details: unknown): string | null {
  if (details === undefined || details === null || details === '') return null;
  if (typeof details === 'string') return details;
  try {
    return JSON.stringify(details);
  } catch {
    return String(details);
  }
}

export function describeQueryError(error: unknown): {
  code: string | null;
  message: string;
  details: string | null;
  retryable: boolean;
} {
  const envelope: ErrorEnvelope = isRecord(error) ? error : {};
  const code = typeof envelope.code === 'string' ? envelope.code : null;
  const status = typeof envelope.status === 'number' ? envelope.status : null;
  const message =
    typeof envelope.message === 'string'
      ? envelope.message
      : error instanceof Error
        ? error.message
        : typeof error === 'string'
          ? error
          : 'The request failed.';
  const details = stringifyDetails(envelope.details);
  const normalizedCode = code?.toLowerCase() ?? '';
  const retryable =
    status === null || !nonRetryableStatus.has(status)
      ? !nonRetryableCodes.some((token) => normalizedCode.includes(token))
      : false;

  return { code, message, details, retryable };
}

function StaleDataNotice({
  errorInfo,
  onRetry,
}: {
  errorInfo: ReturnType<typeof describeQueryError>;
  onRetry?: () => void;
}) {
  return (
    <output style={{ ...cardStyle, marginBottom: 12, padding: 12, display: 'block' }}>
      <strong>Showing stale data.</strong>
      <div style={{ marginTop: 6, fontSize: 13, opacity: 0.82 }}>{errorInfo.message}</div>
      {onRetry && errorInfo.retryable ? (
        <button type="button" className="btn btn--sm" style={{ marginTop: 12 }} onClick={onRetry}>
          Retry
        </button>
      ) : null}
    </output>
  );
}

function ErrorDetails({ details }: { details: string | null }) {
  if (!details) return null;
  return (
    <pre
      className="mono"
      style={{
        margin: '10px auto 0',
        maxWidth: 640,
        whiteSpace: 'pre-wrap',
        textAlign: 'left',
        fontSize: 12,
        opacity: 0.72,
      }}
    >
      {details}
    </pre>
  );
}

function ErrorCard({
  errorInfo,
  onRetry,
}: {
  errorInfo: ReturnType<typeof describeQueryError>;
  onRetry?: () => void;
}) {
  return (
    <div style={cardStyle} role="alert">
      <strong style={{ color: 'var(--h-danger, #e57373)' }}>Failed to load.</strong>
      <div style={{ marginTop: 8, fontSize: 13, opacity: 0.8 }}>{errorInfo.message}</div>
      {errorInfo.code ? (
        <div className="mono" style={{ marginTop: 8, fontSize: 12, opacity: 0.72 }}>
          {errorInfo.code}
        </div>
      ) : null}
      <ErrorDetails details={errorInfo.details} />
      {onRetry && errorInfo.retryable ? (
        <button type="button" className="btn btn--sm" style={{ marginTop: 12 }} onClick={onRetry}>
          Retry
        </button>
      ) : null}
    </div>
  );
}

export function QueryStateView<T>(props: QueryStateViewProps<T>) {
  const {
    query,
    isEmpty = defaultIsEmpty,
    loadingFallback,
    errorFallback,
    emptyFallback,
    children,
  } = props;

  if (query.isLoading) {
    return <div style={skeletonStyle}>{loadingFallback ?? 'Loading…'}</div>;
  }
  if (query.error) {
    const errorInfo = describeQueryError(query.error);
    if (query.data !== undefined && !isEmpty(query.data)) {
      return (
        <>
          <StaleDataNotice errorInfo={errorInfo} onRetry={query.refetch} />
          {children(query.data)}
        </>
      );
    }
    if (errorFallback) return <>{errorFallback(query.error)}</>;
    return <ErrorCard errorInfo={errorInfo} onRetry={query.refetch} />;
  }
  if (query.data === undefined) {
    return <div style={skeletonStyle}>{loadingFallback ?? 'Loading…'}</div>;
  }
  if (isEmpty(query.data)) {
    return <div style={cardStyle}>{emptyFallback ?? 'Nothing here yet.'}</div>;
  }
  return <>{children(query.data)}</>;
}
