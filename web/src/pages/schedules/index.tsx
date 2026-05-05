import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useMemo, useState } from 'react';

import {
  deleteScheduleMutation,
  listSchedulesOptions,
  listSchedulesQueryKey,
  runScheduleMutation,
} from '../../api/generated/@tanstack/react-query.gen';
import type { Schedule } from '../../api/generated/types.gen';
import { QueryStateView, describeQueryError } from '../../components/QueryStateView';
import { openConfirm, toast } from '../../legacy/window-globals';
import { I } from '../../primitives/icon';

type ScheduleFilter = 'all' | 'active' | 'disabled';

export default function PageSchedules() {
  const queryClient = useQueryClient();
  const [filter, setFilter] = useState<ScheduleFilter>('all');
  const [runStatus, setRunStatus] = useState<Record<string, string>>({});
  const query = useQuery(listSchedulesOptions());
  const invalidate = () => queryClient.invalidateQueries({ queryKey: listSchedulesQueryKey() });
  const runSchedule = useMutation({
    ...runScheduleMutation(),
    onMutate: (variables) => {
      setRunStatus((current) => ({ ...current, [variables.path.id]: 'Run queued...' }));
    },
    onSuccess: (_data, variables) => {
      setRunStatus((current) => ({
        ...current,
        [variables.path.id]: 'Run submitted. Refreshing latest status.',
      }));
      toast('success', 'Schedule run submitted');
      void invalidate();
    },
    onError: (error, variables) => {
      const info = describeQueryError(error);
      setRunStatus((current) => ({
        ...current,
        [variables.path.id]: `Run failed: ${info.details ?? info.message}`,
      }));
      toast('danger', 'Schedule run failed', info.details ?? info.message);
    },
  });
  const deleteSchedule = useMutation({
    ...deleteScheduleMutation(),
    onSuccess: () => {
      toast('success', 'Schedule deleted');
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
  const schedules = useMemo(() => rowsQuery.data ?? [], [rowsQuery.data]);
  const filteredSchedules = useMemo(
    () =>
      schedules.filter((schedule) => {
        if (filter === 'active') return schedule.enabled;
        if (filter === 'disabled') return !schedule.enabled;
        return true;
      }),
    [filter, schedules],
  );
  const filterOptions: Array<[ScheduleFilter, string]> = [
    ['all', `All schedules (${schedules.length})`],
    ['active', `Active (${schedules.filter((schedule) => schedule.enabled).length})`],
    ['disabled', `Disabled (${schedules.filter((schedule) => !schedule.enabled).length})`],
  ];

  return (
    <div>
      <div className="toolbar">
        <div className="lft">
          <div className="seg">
            {filterOptions.map(([value, label]) => (
              <button
                key={value}
                type="button"
                className={filter === value ? 'on' : undefined}
                aria-pressed={filter === value}
                onClick={() => setFilter(value)}
              >
                {label}
              </button>
            ))}
          </div>
        </div>
        <div className="rgt">
          <button type="button" className="btn btn--sm btn--primary" disabled>
            <I n="plus" s={13} /> New schedule
          </button>
        </div>
      </div>
      <div className="card" style={{ marginBottom: 12, padding: 12 }}>
        <span className="mono">helling schedule create</span> is the v0.2 creation path while the
        dashboard editor is being completed.
      </div>

      <QueryStateView<Schedule[]>
        query={rowsQuery}
        emptyFallback={
          <div>
            <strong>No backup schedules configured.</strong>
            <div className="dim" style={{ marginTop: 6 }}>
              Add schedules with <span className="mono">helling schedule create</span>; they will
              appear here after the API refreshes.
            </div>
          </div>
        }
      >
        {() =>
          filteredSchedules.length === 0 ? (
            <div className="card" style={{ padding: 24, textAlign: 'center' }}>
              No schedules match this filter.
            </div>
          ) : (
            <table className="tbl">
              <thead>
                <tr>
                  <th style={{ width: 60 }}>Enabled</th>
                  <th>Name</th>
                  <th>Target</th>
                  <th>Kind</th>
                  <th>OnCalendar</th>
                  <th>Next run</th>
                  <th>Last status</th>
                  <th style={{ textAlign: 'right' }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {filteredSchedules.map((s) => (
                  <tr key={s.id}>
                    <td>
                      <span className="mono">{s.enabled ? 'enabled' : 'disabled'}</span>
                    </td>
                    <td className="mono" style={{ fontWeight: 600 }}>
                      {s.name}
                    </td>
                    <td className="mono dim">{s.target}</td>
                    <td>
                      <span className="badge mono">{s.kind}</span>
                    </td>
                    <td className="mono">{s.on_calendar}</td>
                    <td className="mono dim">{s.next_run_at ?? '-'}</td>
                    <td className="mono dim">{runStatus[s.id] ?? s.last_status ?? 'never'}</td>
                    <td style={{ textAlign: 'right' }}>
                      <button
                        type="button"
                        className="btn btn--sm btn--ghost"
                        disabled={runSchedule.isPending}
                        onClick={() => runSchedule.mutate({ path: { id: s.id } })}
                      >
                        <I n="play" s={13} /> Run now
                      </button>
                      <button type="button" className="btn btn--sm btn--ghost" disabled>
                        <I n="pencil" s={13} /> Edit
                      </button>
                      <button
                        type="button"
                        className="btn btn--sm btn--ghost"
                        disabled={deleteSchedule.isPending}
                        aria-label={`Delete schedule ${s.name}`}
                        onClick={() =>
                          openConfirm({
                            title: `Delete schedule "${s.name}"?`,
                            body: 'This removes the schedule and cannot be undone.',
                            danger: true,
                            confirmLabel: 'Delete schedule',
                            onConfirm: () => deleteSchedule.mutate({ path: { id: s.id } }),
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
          )
        }
      </QueryStateView>
    </div>
  );
}
