import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import {
  deleteHostFirewallRuleMutation,
  listHostFirewallRulesOptions,
  listHostFirewallRulesQueryKey,
} from '../../api/generated/@tanstack/react-query.gen';
import type { FirewallRule } from '../../api/generated/types.gen';
import { QueryStateView, describeQueryError } from '../../components/QueryStateView';
import { openConfirm, toast } from '../../legacy/window-globals';
import { I } from '../../primitives/icon';

export default function PageFirewall() {
  const queryClient = useQueryClient();
  const query = useQuery(listHostFirewallRulesOptions());
  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: listHostFirewallRulesQueryKey() });
  const deleteRule = useMutation({
    ...deleteHostFirewallRuleMutation(),
    onSuccess: () => {
      toast('success', 'Firewall rule deleted');
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
              Host rules
            </button>
          </div>
        </div>
        <div className="rgt">
          <button type="button" className="btn btn--sm btn--primary" disabled>
            <I n="plus" s={13} /> Add rule
          </button>
        </div>
      </div>
      <div className="card" style={{ marginBottom: 12, padding: 12 }}>
        <span className="mono">helling firewall add</span> is the v0.2 creation path while the
        dashboard editor is being completed.
      </div>

      <QueryStateView<FirewallRule[]>
        query={rowsQuery}
        emptyFallback={
          <div>
            <strong>No Helling-managed host firewall rules.</strong>
            <div className="dim" style={{ marginTop: 6 }}>
              Helling is not applying host traffic restrictions yet. Add allow or drop rules with{' '}
              <span className="mono">helling firewall add</span>.
            </div>
          </div>
        }
      >
        {(rules) => (
          <table className="tbl">
            <thead>
              <tr>
                <th style={{ width: 60 }}>On</th>
                <th>Direction</th>
                <th>Action</th>
                <th>Proto</th>
                <th>Port</th>
                <th>Source</th>
                <th>Destination</th>
                <th>nft comment</th>
                <th style={{ textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {rules.map((r) => (
                <tr key={r.id}>
                  <td>{r.enabled ? 'on' : 'off'}</td>
                  <td>
                    <span className="badge mono">{r.direction}</span>
                  </td>
                  <td className="mono">{r.action}</td>
                  <td className="mono">{r.protocol}</td>
                  <td className="mono">{r.destination_port ?? '-'}</td>
                  <td className="mono dim">{r.source_cidr ?? '-'}</td>
                  <td className="mono dim">{r.destination_cidr ?? '-'}</td>
                  <td className="mono dim">{r.nft_comment}</td>
                  <td style={{ textAlign: 'right' }}>
                    <button
                      type="button"
                      className="btn btn--sm btn--ghost"
                      disabled={deleteRule.isPending}
                      aria-label={`Delete firewall rule ${r.nft_comment}`}
                      onClick={() =>
                        openConfirm({
                          title: `Delete firewall rule ${r.nft_comment}?`,
                          body: 'This removes the host firewall rule and cannot be undone.',
                          danger: true,
                          confirmLabel: 'Delete rule',
                          onConfirm: () => deleteRule.mutate({ path: { id: r.id } }),
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
