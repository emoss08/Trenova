import {
  findThread,
  recentThreadsOf,
  threadListQuery,
  type ThreadPages,
} from "@/lib/thread-list";
import type { AssistantThread } from "@/types/assistant";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useCallback } from "react";

const NO_THREADS: AssistantThread[] = [];

export type DeskThreads = {
  /** The list's first page: the person's pinned and most recent conversations. */
  threads: AssistantThread[];
  /** The open conversation as the list holds it; null when no page read so far does. */
  listedThread: AssistantThread | null;
  isSuccess: boolean;
  isLoading: boolean;
};

/**
 * What the Desk's frame reads from the conversation list: the open
 * conversation and the first page, never the list itself, which only the rail
 * subscribes to (DeskRailThreads).
 *
 * Both are selected from the cache and structurally shared, and only the
 * fields read here are tracked, so the rail reading an older page, or asking
 * for the next one, leaves the frame and every page under it alone. They
 * change only when the open conversation or the first page does.
 */
export function useDeskThreads(activeThreadId: string | null): DeskThreads {
  const select = useCallback(
    (data: ThreadPages) => ({
      listed: findThread(data, activeThreadId),
      recent: recentThreadsOf(data),
    }),
    [activeThreadId],
  );
  const query = useInfiniteQuery({ ...threadListQuery(), select });

  return {
    threads: query.data?.recent ?? NO_THREADS,
    listedThread: query.data?.listed ?? null,
    isSuccess: query.isSuccess,
    isLoading: query.isLoading,
  };
}
