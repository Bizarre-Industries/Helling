import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';

import {
  deleteWebhookMutation,
  listWebhooksOptions,
  listWebhooksQueryKey,
  testWebhookMutation,
} from '../../api/generated/@tanstack/react-query.gen';
import type { Webhook } from '../../api/generated/types.gen';
import { QueryStateView, describeQueryError } from '../../components/QueryStateView';
import { openConfirm, toast } from '../../legacy/window-globals';
import { I } from '../../primitives/icon';

export default function PageWebhooks() {
  const queryClient = useQueryClient();
  const [lastTest, setLastTest] = useState<Record<string, string>>({});
  const query = useQuery(listWebhooksOptions());
  const invalidate = () => queryClient.invalidateQueries({ queryKey: listWebhooksQueryKey() });
  const testWebhook = useMutation({
    ...testWebhookMutation(),
    onMutate: (variables) => {
      setLastTest((current) => ({ ...current, [variables.path.id]: 'Sending...' }));
    },
    onSuccess: (response, variables) => {
      const delivery = response.data;
      const statusBits = [
        delivery?.status ?? 'sent',
        delivery?.attempt ? `attempt ${delivery.attempt}` : '',
        delivery?.http_status ? `HTTP ${delivery.http_status}` : '',
        delivery?.error ?? '',
      ].filter(Boolean);
      const status = statusBits.join(' - ');
      setLastTest((current) => ({ ...current, [variables.path.id]: status }));
      if (delivery?.status === 'failed') {
        toast('danger', 'Webhook test failed', delivery.error ?? status);
      } else {
        toast('success', 'Webhook test completed', status);
      }
      void invalidate();
    },
    onError: (error, variables) => {
      const info = describeQueryError(error);
      setLastTest((current) => ({
        ...current,
        [variables.path.id]: `failed: ${info.details ?? info.message}`,
      }));
      toast('danger', 'Webhook test failed', info.details ?? info.message);
    },
  });
  const deleteWebhook = useMutation({
    ...deleteWebhookMutation(),
    onSuccess: () => {
      toast('success', 'Webhook deleted');
      void invalidate();
    },
    onError: (error) => {
      const info = describeQueryError(error);
      toast('danger', 'Delete failed', info.details ?? info.message);
    },
  });
  const rowsQuery = {
    isLoading: query.isLoading,
    error: query.error,
    data: query.data?.data,
    refetch: query.refetch,
  };
  return (
    <div>
      <div className="toolbar">
        <div className="lft">
          <div className="seg">
            <button type="button" className="on">
              Outbound
            </button>
          </div>
        </div>
        <div className="rgt">
          <button type="button" className="btn btn--sm btn--primary" disabled>
            <I n="plus" s={13} /> New webhook
          </button>
        </div>
      </div>
      <div className="card" style={{ marginBottom: 12, padding: 12 }}>
        <span className="mono">helling webhook create</span> is the v0.2 creation path while the
        dashboard editor is being completed.
      </div>

      <QueryStateView<Webhook[]>
        query={rowsQuery}
        emptyFallback={
          <div>
            <strong>No outbound webhooks configured.</strong>
            <div className="dim" style={{ marginTop: 6 }}>
              Add HTTPS webhooks with <span className="mono">helling webhook create</span>; secrets
              are never returned to this page.
            </div>
          </div>
        }
      >
        {(webhooks) => (
          <table className="tbl">
            <thead>
              <tr>
                <th>Enabled</th>
                <th>Name</th>
                <th>URL</th>
                <th>Events</th>
                <th>Last test</th>
                <th>Updated</th>
                <th style={{ textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {webhooks.map((w) => (
                <tr key={w.id}>
                  <td className="mono">{w.enabled ? 'on' : 'off'}</td>
                  <td className="mono">{w.name}</td>
                  <td className="mono dim">{w.url}</td>
                  <td className="mono">{w.events.join(', ')}</td>
                  <td className="mono dim">{lastTest[w.id] ?? '-'}</td>
                  <td className="mono dim">{w.updated_at}</td>
                  <td style={{ textAlign: 'right' }}>
                    <button
                      type="button"
                      className="btn btn--sm btn--ghost"
                      disabled={testWebhook.isPending}
                      onClick={() => testWebhook.mutate({ path: { id: w.id } })}
                    >
                      <I n="send" s={13} /> Test
                    </button>
                    <button type="button" className="btn btn--sm btn--ghost" disabled>
                      <I n="pencil" s={13} /> Edit
                    </button>
                    <button
                      type="button"
                      className="btn btn--sm btn--ghost"
                      disabled={deleteWebhook.isPending}
                      aria-label={`Delete webhook ${w.name}`}
                      onClick={() =>
                        openConfirm({
                          title: `Delete webhook "${w.name}"?`,
                          body: 'This removes the webhook and cannot be undone.',
                          danger: true,
                          confirmLabel: 'Delete webhook',
                          onConfirm: () => deleteWebhook.mutate({ path: { id: w.id } }),
                        })
                      }
                    >
                      <I n="trash-2" s={13} /> Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </QueryStateView>
    </div>
  );
}
