import { useQuery } from '@tanstack/react-query';
import { useMemo, useRef, useState } from 'react';

import { getAccessToken } from '../../api/auth-store';
import { listAuditEventsOptions } from '../../api/generated/@tanstack/react-query.gen';
import type { AuditEvent } from '../../api/generated/types.gen';
import { QueryStateView, describeQueryError } from '../../components/QueryStateView';
import { toast } from '../../legacy/window-globals';
import { I } from '../../primitives/icon';

type AuditOutcomeFilter = '' | 'success' | 'failure' | 'denied';

type AuditQueryParams = {
  action?: string;
  actor?: string;
  outcome?: Exclude<AuditOutcomeFilter, ''>;
  since?: string;
  until?: string;
  limit: number;
};

function auditExportParams(queryParams: AuditQueryParams) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries({ ...queryParams, limit: 500 })) {
    params.set(key, String(value));
  }
  return params;
}

function auditExportHeaders() {
  const headers = new Headers({ Accept: 'application/x-ndjson' });
  const token = getAccessToken();
  if (token) headers.set('Authorization', `Bearer ${token}`);
  return headers;
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}

export default function PageAudit() {
  const [filters, setFilters] = useState<{
    action: string;
    actor: string;
    outcome: AuditOutcomeFilter;
    since: string;
    until: string;
  }>({
    action: '',
    actor: '',
    outcome: '',
    since: '',
    until: '',
  });
  const [exporting, setExporting] = useState(false);
  const exportAbort = useRef<AbortController | null>(null);
  const queryParams = useMemo<AuditQueryParams>(() => {
    const params: AuditQueryParams = { limit: 100 };
    if (filters.actor) params.actor = filters.actor;
    if (filters.action) params.action = filters.action;
    if (filters.outcome) params.outcome = filters.outcome;
    if (filters.since) params.since = filters.since;
    if (filters.until) params.until = filters.until;
    return params;
  }, [filters]);
  const query = useQuery(listAuditEventsOptions({ query: queryParams }));
  const rowsQuery = {
    isLoading: query.isLoading,
    error: query.error,
    data: query.data?.data,
    refetch: query.refetch,
  };
  const exportAudit = async () => {
    if (exporting) {
      exportAbort.current?.abort();
      return;
    }
    const params = auditExportParams(queryParams);
    const controller = new AbortController();
    exportAbort.current = controller;
    setExporting(true);
    try {
      const response = await fetch(`/api/v1/audit/export?${params.toString()}`, {
        credentials: 'include',
        headers: auditExportHeaders(),
        signal: controller.signal,
      });
      if (!response.ok) throw new Error(`Export failed with HTTP ${response.status}`);
      const blob = await response.blob();
      downloadBlob(blob, 'audit-export.jsonl');
      toast('success', 'Audit export ready', `${blob.size} bytes`);
    } catch (error) {
      if ((error as Error).name === 'AbortError') {
        toast('warn', 'Audit export canceled');
      } else {
        const info = describeQueryError(error);
        toast('danger', 'Audit export failed', info.details ?? info.message);
      }
    } finally {
      setExporting(false);
      exportAbort.current = null;
    }
  };

  return (
    <div>
      <div className="toolbar">
        <div className="lft">
          <span className="mono dim">Latest 100 audit events</span>
        </div>
        <div className="rgt">
          <button type="button" className="btn btn--sm" onClick={exportAudit}>
            <I n={exporting ? 'x' : 'download'} s={13} />{' '}
            {exporting ? 'Cancel export' : 'Export JSONL'}
          </button>
        </div>
      </div>
      <div className="card" style={{ marginBottom: 12, padding: 12 }}>
        <div className="form-grid form-grid--compact">
          <label>
            Actor
            <input
              className="input"
              value={filters.actor}
              onChange={(event) =>
                setFilters((current) => ({ ...current, actor: event.target.value }))
              }
            />
          </label>
          <label>
            Action
            <input
              className="input"
              value={filters.action}
              onChange={(event) =>
                setFilters((current) => ({ ...current, action: event.target.value }))
              }
            />
          </label>
          <label>
            Outcome
            <select
              className="input"
              value={filters.outcome}
              onChange={(event) =>
                setFilters((current) => ({
                  ...current,
                  outcome: event.target.value as AuditOutcomeFilter,
                }))
              }
            >
              <option value="">Any</option>
              <option value="success">Success</option>
              <option value="failure">Failure</option>
              <option value="denied">Denied</option>
            </select>
          </label>
          <label>
            Since
            <input
              className="input"
              placeholder="2026-05-05 00:00:00"
              value={filters.since}
              onChange={(event) =>
                setFilters((current) => ({ ...current, since: event.target.value }))
              }
            />
          </label>
          <label>
            Until
            <input
              className="input"
              placeholder="2026-05-05 23:59:59"
              value={filters.until}
              onChange={(event) =>
                setFilters((current) => ({ ...current, until: event.target.value }))
              }
            />
          </label>
        </div>
      </div>

      <QueryStateView<AuditEvent[]> query={rowsQuery} emptyFallback="No audit events found.">
        {(audit) => (
          <table className="tbl">
            <thead>
              <tr>
                <th>Timestamp</th>
                <th>Actor</th>
                <th>Action</th>
                <th>Target</th>
                <th>Outcome</th>
                <th>Message</th>
              </tr>
            </thead>
            <tbody>
              {audit.map((a, i) => (
                <tr key={`${a.time}-${a.action}-${i}`}>
                  <td className="mono dim">{a.time}</td>
                  <td className="mono">{a.actor ?? '-'}</td>
                  <td className="mono">{a.action}</td>
                  <td className="mono" style={{ color: 'var(--h-accent)' }}>
                    {[a.target_type, a.target_id].filter(Boolean).join('/') || '-'}
                  </td>
                  <td className="mono">{a.outcome}</td>
                  <td className="dim">{a.message}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </QueryStateView>
    </div>
  );
}
