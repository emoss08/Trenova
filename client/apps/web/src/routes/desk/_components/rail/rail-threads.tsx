"use no memo";
import type {
  DeskRailShelf,
  DeskShelfCount,
  DeskShelfPaging,
} from "@/components/desk-chat/rail/desk-rail-rows";
import {
  DESK_SHELF_KEYS,
  deskShelfEdgeList,
  deskShelfEdges,
  deskShelfFilter,
  shelveLatest,
  type DeskShelfEdges,
  type DeskShelfKey,
} from "@/components/desk-chat/rail/desk-threads";
import { useCalendarDay } from "@/hooks/use-calendar-day";
import {
  noteThreadsInView,
  threadShelfCountsQuery,
  threadShelfQuery,
  threadsOf,
} from "@/lib/thread-list";
import type { AssistantThread, AssistantThreadShelfCounts } from "@/types/assistant";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useInfiniteQuery, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { caseOfThread } from "../record-case/case-subject";
import { DeskRail, railViewOf, type DeskRailProps } from "./desk-rail";

export type DeskRailThreadsProps = Omit<
  DeskRailProps,
  "shelves" | "today" | "noConversations" | "onLoadMore" | "onThreadsInView" | "view" | "edges"
> & {
  /** The open conversation, however it was read, so its row shows even on a shelf never read. */
  activeThread: AssistantThread | null;
};

/** Which band of the shelf counts each dated shelf is; the pinned shelf is counted on its own. */
const COUNT_BAND: Record<Exclude<DeskShelfKey, "pinned">, number> = {
  today: 0,
  yesterday: 1,
  week: 2,
  month: 3,
  older: 4,
};

function shelfCount(
  key: DeskShelfKey,
  counts: AssistantThreadShelfCounts | undefined,
  failed: boolean,
): DeskShelfCount {
  if (!counts) {
    return failed ? "unknown" : "counting";
  }

  return key === "pinned" ? counts.pinned : (counts.bands[COUNT_BAND[key]] ?? "unknown");
}

const NO_THREADS: AssistantThread[] = [];

/**
 * One shelf's query. It hands back the shelf's conversations flattened from
 * its pages, one array per cache entry, so a shelf that did not change keeps
 * its array.
 */
function useShelfQuery(key: DeskShelfKey, edges: DeskShelfEdges, cases: boolean) {
  return useInfiniteQuery({
    ...threadShelfQuery(deskShelfFilter(key, edges, cases)),
    select: threadsOf,
  });
}

/**
 * The rail with the person's conversations, each shelf read on its own a
 * page at a time.
 *
 * A conversation touched since its shelf was read is held by two shelves
 * until the older one is read again; the later copy wins, and every
 * conversation is drawn once, on the shelf its last touch puts it on.
 */
export function DeskRailThreads({ activeThread, ...props }: DeskRailThreadsProps) {
  const queryClient = useQueryClient();
  const timezone = resolveUserTimezone(useAuthStore((state) => state.user?.timezone));
  const today = useCalendarDay(timezone);
  const edges = useMemo(() => deskShelfEdges(today, timezone), [today, timezone]);

  // The rail is the cases' or the conversations', by the place in front, and
  // lists that half of the person's work: conversations about no record, or
  // the ones asked on a record's case, which it draws as cases. Each half is
  // shelved, counted and paged on its own.
  const view = railViewOf(
    props.place,
    activeThread !== null && caseOfThread(activeThread) !== null,
  );
  const cases = view === "cases";
  const counts = useQuery(threadShelfCountsQuery(deskShelfEdgeList(edges), cases));
  const queries = {
    pinned: useShelfQuery("pinned", edges, cases),
    today: useShelfQuery("today", edges, cases),
    yesterday: useShelfQuery("yesterday", edges, cases),
    week: useShelfQuery("week", edges, cases),
    month: useShelfQuery("month", edges, cases),
    older: useShelfQuery("older", edges, cases),
  };

  const pinned = queries.pinned.data ?? NO_THREADS;
  const todays = queries.today.data ?? NO_THREADS;
  const yesterdays = queries.yesterday.data ?? NO_THREADS;
  const week = queries.week.data ?? NO_THREADS;
  const month = queries.month.data ?? NO_THREADS;
  const older = queries.older.data ?? NO_THREADS;
  const grouped = shelveLatest(
    [pinned, todays, yesterdays, week, month, older],
    edges,
    activeThread && Boolean(activeThread.subjectId) === cases ? [activeThread] : [],
  );

  const shelves: DeskRailShelf[] = DESK_SHELF_KEYS.map((key) => {
    const query = queries[key];
    const paging: DeskShelfPaging = {
      loading: (query.isPending && query.fetchStatus === "fetching") || query.isFetchingNextPage,
      hasMore: query.hasNextPage,
      failed: (query.isError && !query.isFetching) || query.isFetchNextPageError,
    };

    return {
      key,
      count: shelfCount(key, counts.data, counts.isError),
      threads: grouped.get(key) ?? [],
      paging,
    };
  });
  const noConversations =
    counts.data !== undefined &&
    counts.data.pinned === 0 &&
    counts.data.bands.every((band) => band === 0) &&
    shelves.every((shelf) => shelf.threads.length === 0);

  const onLoadMore = useLatestCallback((key: DeskShelfKey) => {
    const query = queries[key];
    if (query.data === undefined) {
      void query.refetch();
      return;
    }
    void query.fetchNextPage({ cancelRefetch: false });
  });
  const onThreadsInView = useCallback(
    (threadIds: readonly string[]) => noteThreadsInView(queryClient, threadIds),
    [queryClient],
  );

  return (
    <DeskRail
      {...props}
      view={view}
      edges={edges}
      shelves={shelves}
      today={today}
      noConversations={noConversations}
      onLoadMore={onLoadMore}
      onThreadsInView={onThreadsInView}
    />
  );
}
