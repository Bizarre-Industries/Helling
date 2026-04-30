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
  error: Error | null;
  data: T | undefined;
};

export type QueryStateViewProps<T> = {
  query: QueryLike<T>;
  /** Returns true when data should be considered empty. Default: array.length === 0. */
  isEmpty?: (data: T) => boolean;
  /** Rendered when loading. Default: skeleton placeholder. */
  loadingFallback?: ReactNode;
  /** Rendered when error is non-null. Default: error card. */
  errorFallback?: (err: Error) => ReactNode;
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

export function QueryStateView<T>(props: QueryStateViewProps<T>) {
  const { query, isEmpty = defaultIsEmpty, loadingFallback, errorFallback, emptyFallback, children } = props;

  if (query.isLoading) {
    return <div style={skeletonStyle}>{loadingFallback ?? 'Loading…'}</div>;
  }
  if (query.error) {
    if (errorFallback) return <>{errorFallback(query.error)}</>;
    return (
      <div style={cardStyle} role="alert">
        <strong style={{ color: 'var(--h-danger, #e57373)' }}>Failed to load.</strong>
        <div style={{ marginTop: 8, fontSize: 13, opacity: 0.8 }}>{query.error.message}</div>
      </div>
    );
  }
  if (query.data === undefined) {
    return <div style={skeletonStyle}>{loadingFallback ?? 'Loading…'}</div>;
  }
  if (isEmpty(query.data)) {
    return <div style={cardStyle}>{emptyFallback ?? 'Nothing here yet.'}</div>;
  }
  return <>{children(query.data)}</>;
}
