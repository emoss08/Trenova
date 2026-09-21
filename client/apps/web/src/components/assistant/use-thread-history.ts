import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AssistantMessagePage } from "@/types/assistant";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import {
  flattenHistory,
  hasOlderPages,
  oldestSequence,
  threadLength,
  type ThreadHistory,
} from "./thread-history";

/** How much of a thread is read at a time. */
export const HISTORY_PAGE_SIZE = 50;

/**
 * A thread's messages, read a page at a time from the newest end.
 *
 * The first page is what opens instantly; older pages are fetched as the
 * reader scrolls up, and a page fetched once is kept because history never
 * changes beneath it. The pages are never refetched wholesale after a turn:
 * the turn hook appends the saved rows to the newest page instead.
 */
export function useThreadHistory(threadId: string) {
  const query = useInfiniteQuery<
    AssistantMessagePage,
    Error,
    ThreadHistory,
    ReturnType<typeof queries.assistant.messages>["queryKey"],
    number | undefined
  >({
    queryKey: queries.assistant.messages(threadId).queryKey,
    queryFn: ({ pageParam, signal }) =>
      apiService.assistantService.listMessages(threadId, {
        limit: HISTORY_PAGE_SIZE,
        before: pageParam,
        signal,
      }),
    initialPageParam: undefined,
    getNextPageParam: (lastPage, pages) => (lastPage.hasMore ? oldestSequence(pages) : undefined),
    staleTime: Number.POSITIVE_INFINITY,
  });

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
