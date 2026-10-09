import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { infiniteQueryOptions, useInfiniteQuery } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { flattenHistory, hasOlderPages, oldestSequence, threadLength } from "./thread-history";

/** How much of a thread is read at a time. */
export const HISTORY_PAGE_SIZE = 50;

/**
 * How a thread's history is read and kept. Every reader of the cache spends
 * these exact options, so a second observer — the hand-off menu reading the
 * newest suggestion — neither refetches nor reads pages of another size.
 */
export function threadHistoryQueryOptions(threadId: string) {
  return infiniteQueryOptions({
    queryKey: queries.assistant.messages(threadId).queryKey,
    queryFn: ({ pageParam, signal }) =>
      apiService.assistantService.listMessages(threadId, {
        limit: HISTORY_PAGE_SIZE,
        before: pageParam,
        signal,
      }),
    initialPageParam: undefined as number | undefined,
    getNextPageParam: (lastPage, pages) => (lastPage.hasMore ? oldestSequence(pages) : undefined),
    staleTime: Number.POSITIVE_INFINITY,
  });
}

/**
 * A thread's messages, read a page at a time from the newest end.
 *
 * The first page is what opens instantly; older pages are fetched as the
 * reader scrolls up, and a page fetched once is kept because history never
 * changes beneath it. The pages are never refetched wholesale after a turn:
 * the turn hook appends the saved rows to the newest page instead.
 */
export function useThreadHistory(threadId: string) {
  const query = useInfiniteQuery(threadHistoryQueryOptions(threadId));

  const pages = query.data?.pages;
  const messages = useMemo(() => flattenHistory(pages ?? []), [pages]);
  const hasOlder = hasOlderPages(pages ?? []);
  const newest = pages?.[0];
  const length = threadLength(newest?.total ?? messages.length, newest?.limit ?? 0);

  const { fetchNextPage, isFetchingNextPage } = query;
  const loadOlder = useCallback(() => {
    if (hasOlder && !isFetchingNextPage) {
      void fetchNextPage();
    }
  }, [fetchNextPage, hasOlder, isFetchingNextPage]);

  return {
    messages,
    isLoading: query.isLoading,
    hasOlder,
    isLoadingOlder: isFetchingNextPage,
    loadOlder,
    total: newest?.total ?? messages.length,
    limit: newest?.limit ?? 0,
    length,
  };
}
