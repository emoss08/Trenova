"use no memo";
import { noteThreadsInView, threadListQuery, threadsOf } from "@/lib/thread-list";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { DeskRail, type DeskRailProps } from "./desk-rail";
import type { DeskRailPaging } from "./desk-rail-list";

export type DeskRailThreadsProps = Omit<DeskRailProps, "threads" | "paging" | "onThreadsInView">;

/**
 * The rail with the person's conversations, read a page at a time.
 *
 * The rail is the only part of the Desk that reads the whole list, so it is
 * the only part that subscribes to it: a page arriving, or the next one being
 * asked for, re-renders the rail and nothing around it. The frame reads only
 * what it needs from the same cache (the open conversation and the first
 * page), each of which stays the same object when an older page arrives.
 */
export function DeskRailThreads(props: DeskRailThreadsProps) {
  const queryClient = useQueryClient();
  const query = useInfiniteQuery(threadListQuery());
  const threads = threadsOf(query.data);
  const { hasNextPage, isFetchingNextPage, isFetchNextPageError, fetchNextPage } = query;

  const loadMore = useCallback(() => {
    void fetchNextPage({ cancelRefetch: false });
  }, [fetchNextPage]);
  const paging = useMemo<DeskRailPaging>(
    () => ({
      hasMore: hasNextPage,
      loadingMore: isFetchingNextPage,
      failed: isFetchNextPageError,
      loadMore,
    }),
    [hasNextPage, isFetchNextPageError, isFetchingNextPage, loadMore],
  );
  const onThreadsInView = useCallback(
    (threadIds: readonly string[]) => noteThreadsInView(queryClient, threadIds),
    [queryClient],
  );

  return (
    <DeskRail {...props} threads={threads} paging={paging} onThreadsInView={onThreadsInView} />
  );
}
