import {
  type WatchtowerFilter,
  fetchWatchtowerCleared,
  fetchWatchtowerCounts,
  fetchWatchtowerFeed,
  fetchWatchtowerHandOffAgents,
  fetchWatchtowerItem,
  type WatchtowerPage,
} from "@/lib/graphql/watchtower";
import type { WatchtowerSourceKind } from "@trenova/graphql/generated/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import type { InfiniteData } from "@tanstack/react-query";

type QueryContext = { signal?: AbortSignal };

/**
 * The exceptions a dispatcher owns once a move is covered: freight slipping, a stop
 * missed, a truck gone quiet, a tender run dry, a driver out of hours. The dispatch
 * console reads them from the Watchtower rather than keeping a list of its own.
 */
export const DISPATCH_EXCEPTION_KINDS: WatchtowerSourceKind[] = [
  "EtaAtRisk",
  "MissedPickup",
  "MissedDelivery",
  "NoCheckCall",
  "TenderExhausted",
  "HOSViolation",
];

const DISPATCH_EXCEPTIONS_FILTER: WatchtowerFilter = {
  kinds: DISPATCH_EXCEPTION_KINDS,
  statuses: ["Open", "Acknowledged", "InProgress"],
  unresolvedOnly: true,
  orderByDue: true,
  first: 50,
};

export const watchtower = createQueryKeys("watchtower", {
  counts: () => ({
    queryKey: ["counts"],
    queryFn: ({ signal }: QueryContext) => fetchWatchtowerCounts({ signal }),
  }),
  // The queue the Watchtower page works through, paged in the order it is
  // read; see watchtowerQueueQuery.
  queue: null,
  cleared: (since: number) => ({
    queryKey: [since],
    queryFn: ({ signal }: QueryContext) => fetchWatchtowerCleared(since, { signal }),
  }),
  handOffAgents: () => ({
    queryKey: ["hand-off-agents"],
    queryFn: ({ signal }: QueryContext) => fetchWatchtowerHandOffAgents({ signal }),
  }),
  item: (id: string) => ({
    queryKey: [id],
    queryFn: ({ signal }: QueryContext) => fetchWatchtowerItem(id, { signal }),
  }),
  dispatchExceptions: () => ({
    queryKey: ["dispatch-exceptions"],
    queryFn: ({ signal }: QueryContext) =>
      fetchWatchtowerFeed(DISPATCH_EXCEPTIONS_FILTER, { signal }),
  }),
});

/**
 * What the queue holds: open, awake items soonest deadline first and undated
 * ones last, the order the server's index reads them in.
 */
export const WATCHTOWER_QUEUE_FILTER: WatchtowerFilter = {
  statuses: ["Open", "Acknowledged", "InProgress"],
  unresolvedOnly: true,
  orderByDue: true,
  first: 50,
};

/** The queue's cached pages, as `useInfiniteQuery` holds them. */
export type WatchtowerQueuePages = InfiniteData<WatchtowerPage, string | null>;

export function watchtowerQueueQuery() {
  return {
    queryKey: watchtower.queue.queryKey,
    queryFn: ({ pageParam, signal }: { pageParam?: unknown; signal?: AbortSignal }) =>
      fetchWatchtowerFeed(
        { ...WATCHTOWER_QUEUE_FILTER, after: typeof pageParam === "string" ? pageParam : null },
        { signal },
      ),
    initialPageParam: null as string | null,
    getNextPageParam: (last: WatchtowerPage): string | null =>
      last.hasNextPage ? last.endCursor : null,
  };
}
