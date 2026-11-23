import type { Connection } from "@/lib/graphql/connection";
import { keepPreviousData, useInfiniteQuery, type InfiniteData } from "@tanstack/react-query";

/** One read of a page: how many rows, after which cursor, and the read's abort signal. */
export type ConnectionPage = { first: number; after: string | null; signal: AbortSignal };

/** A per-worker list read a page at a time, as the components draw it. */
export type WorkerList<Row> = {
  /** Every loaded row in order; the same array until a page arrives or a refetch changes one. */
  rows: Row[];
  /** The first page's `totalCount` when the operation selects it, else null. */
  total: number | null;
  hasMore: boolean;
  /** Reads the next page; does nothing while one is in flight or when there is none. */
  loadMore: () => void;
  loadingMore: boolean;
  /** The next page failed; `loadMore` asks again. */
  loadMoreFailed: boolean;
  /** Nothing has been read yet. */
  isPending: boolean;
  /** The first page failed. */
  isError: boolean;
  error: unknown;
  /** Showing the previous variables' rows while the new ones load (`keepPreviousRows`). */
  isStale: boolean;
  refetch: () => void;
};

export const WORKER_LIST_PAGE_SIZE = 50;

/** The key of a worker's list read (the hook appends its page size): under `[root, workerId]`, which `useWorkerInvalidation` refreshes. */
export function workerListKey(root: string, workerId: string, variables: readonly unknown[] = []) {
  return [root, workerId, "list", ...variables] as const;
}

type Flat<Row> = { rows: Row[]; total: number | null };

// Module level, so TanStack keeps the flattened result (and `rows`) the same object until the
// pages change.
function flatten<Row>(data: InfiniteData<Connection<Row>, string | null>): Flat<Row> {
  return {
    rows: data.pages.flatMap((page) => (page.edges ?? []).map((edge) => edge.node)),
    total: data.pages[0]?.totalCount ?? null,
  };
}

/**
 * Reads one of a worker's growing lists as a cursor connection, a page at a time.
 *
 *   const ledger = useWorkerList({
 *     root: WORKER_PTO_LEDGER_KEY,
 *     workerId,
 *     fetchPage: ({ first, after, signal }) => fetchWorkerPtoLedger({ workerId, first, after }, { signal }),
 *   });
 *
 * The key sits under `[root, workerId]`, so a write's `useWorkerInvalidation` refetches every
 * page already loaded, in place: the rows keep their keys and the region its scroll. Pages are
 * asked for only by `loadMore`, which the region's end sentinel calls; a hidden tab's sentinel
 * does not run, so it reads nothing.
 */
export function useWorkerList<Row>({
  root,
  workerId,
  variables = [],
  fetchPage,
  pageSize = WORKER_LIST_PAGE_SIZE,
  enabled = true,
  keepPreviousRows = false,
  refetchInterval,
  queryKey,
}: {
  /** The area's key root, the one its `WORKER_WRITE_REACH` row names. */
  root: string;
  workerId: string;
  /** What else the read depends on (filters, a search); part of the key. */
  variables?: readonly unknown[];
  fetchPage: (page: ConnectionPage) => Promise<Connection<Row>>;
  pageSize?: number;
  enabled?: boolean;
  /** Keep the rows of the previous variables on screen while new ones load (a search). */
  keepPreviousRows?: boolean;
  /**
   * Reads every loaded page again on an interval while it returns a number (a row's document
   * preview still being drawn: `previewPendingRefetchMs(query, …)`), given the loaded rows and
   * the query, whose identity lasts as long as the read.
   */
  refetchInterval?: (rows: Row[], query: object) => number | false;
  /**
   * The key prefix, for a read another surface keys its own way (a record's agent memories,
   * under `[DESK_MEMORIES_KEY, "record", "Worker", id]`): it replaces `[root, workerId, "list",
   * ...variables]`, and must still sit under what the area's `WORKER_WRITE_REACH` row refreshes.
   */
  queryKey?: readonly unknown[];
}): WorkerList<Row> {
  const query = useInfiniteQuery({
    queryKey: [...(queryKey ?? workerListKey(root, workerId, variables)), pageSize],
    queryFn: ({ pageParam, signal }) => fetchPage({ first: pageSize, after: pageParam, signal }),
    initialPageParam: null as string | null,
    getNextPageParam: (page: Connection<Row>) =>
      page.pageInfo.hasNextPage && page.pageInfo.endCursor ? page.pageInfo.endCursor : null,
    select: flatten<Row>,
    enabled,
    placeholderData: keepPreviousRows ? keepPreviousData : undefined,
    refetchInterval: refetchInterval
      ? (read) => refetchInterval(read.state.data ? flatten(read.state.data).rows : [], read)
      : undefined,
  });

  const { hasNextPage, isFetching, isFetchingNextPage, fetchNextPage } = query;
  return {
    rows: query.data?.rows ?? [],
    total: query.data?.total ?? null,
    hasMore: hasNextPage,
    loadMore: () => {
      if (!hasNextPage || isFetching) return;
      void fetchNextPage({ cancelRefetch: false });
    },
    loadingMore: isFetchingNextPage,
    loadMoreFailed: query.isFetchNextPageError,
    isPending: query.isPending,
    isError: query.isError && !query.isFetchNextPageError,
    error: query.error,
    isStale: query.isPlaceholderData,
    refetch: () => void query.refetch(),
  };
}
