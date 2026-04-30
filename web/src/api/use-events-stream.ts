// web/src/api/use-events-stream.ts
//
// Audit F-03 + F-42 stage-1.
//
// Stage 1 (this file): poll GET /api/v1/events?limit=50 every 5s, dedupe by
// event id, dispatch by type into the TanStack Query cache (invalidateQueries
// for the affected resource). Replaces the 1.5s `setInterval` mock loop in
// shell.jsx.
//
// Stage 2 (deferred to v0.1-beta backend): swap fetch+poll for an EventSource
// against the same path; the dispatch shape stays identical. See
// api/openapi.yaml — full SSE streaming lands beta-side.

import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useRef } from 'react';

import { getAccessToken } from './auth-store';

export type HellingEvent = {
  id: string;
  type: string;
  /** ISO 8601 UTC. */
  timestamp: string;
  payload?: unknown;
};

type EventsResponse = {
  events?: HellingEvent[];
};

const POLL_MS = 5000;
const SEEN_LIMIT = 1000;

/** Map an event type to one or more query keys to invalidate. */
function dispatchKeys(eventType: string): string[][] {
  // Keep keys in sync with web/src/api/queries.ts.
  if (eventType.startsWith('instance.') || eventType.startsWith('incus.instance.')) {
    return [['incus', 'instances']];
  }
  if (eventType.startsWith('container.') || eventType.startsWith('podman.')) {
    return [['podman', 'containers']];
  }
  if (eventType.startsWith('storage.')) return [['incus', 'storage-pools']];
  if (eventType.startsWith('network.')) return [['incus', 'networks']];
  if (eventType.startsWith('image.')) return [['incus', 'images']];
  if (eventType.startsWith('operation.') || eventType.startsWith('task.')) {
    return [['incus', 'operations']];
  }
  return [];
}

async function fetchEventsBatch(): Promise<HellingEvent[]> {
  const token = getAccessToken();
  if (!token) return [];
  const resp = await fetch('/api/v1/events?limit=50', {
    headers: { Authorization: `Bearer ${token}`, Accept: 'application/json' },
    credentials: 'same-origin',
  });
  if (!resp.ok) {
    throw new Error(`events: HTTP ${resp.status}`);
  }
  const body = (await resp.json()) as EventsResponse;
  return body.events ?? [];
}

/** Poll the events endpoint and invalidate query keys per event type. */
export function useEventsStream() {
  const qc = useQueryClient();
  const seenRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | null = null;

    async function tick() {
      if (cancelled) return;
      try {
        const events = await fetchEventsBatch();
        const seen = seenRef.current;
        for (const ev of events) {
          if (seen.has(ev.id)) continue;
          seen.add(ev.id);
          if (seen.size > SEEN_LIMIT) {
            // Trim oldest entries; Set keeps insertion order.
            const toDrop = seen.size - SEEN_LIMIT;
            const it = seen.values();
            for (let i = 0; i < toDrop; i++) {
              const next = it.next();
              if (next.done) break;
              seen.delete(next.value);
            }
          }
          for (const key of dispatchKeys(ev.type)) {
            qc.invalidateQueries({ queryKey: key });
          }
        }
      } catch {
        // Swallow; next tick retries. Unauthenticated callers no-op.
      }
      if (!cancelled) {
        timer = setTimeout(tick, POLL_MS);
      }
    }

    tick();
    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
    };
  }, [qc]);
}
