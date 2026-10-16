import { assistant } from "@/lib/queries/assistant";
import { apiService } from "@/services/api";
import type { ConversationScheduleList } from "@/types/assistant";
import { infiniteQueryOptions } from "@tanstack/react-query";

/** How many schedules the Scheduled page reads at a time. */
export const SCHEDULE_PAGE_SIZE = 50;

/**
 * Where the next page of schedules starts: after everything read so far, or
 * nothing once that is all of them. An empty page ends the list too, so a
 * total that ran ahead of the rows (one deleted between reads) cannot ask
 * for the same empty page forever.
 */
export function nextScheduleOffset(
  lastPage: ConversationScheduleList,
  pages: readonly ConversationScheduleList[],
): number | undefined {
  const loaded = pages.reduce((count, page) => count + page.items.length, 0);

  return lastPage.items.length > 0 && loaded < lastPage.total ? loaded : undefined;
}

/**
 * Every schedule the person keeps, across their conversations, a page at a
 * time: the ones that run soonest first, paused ones after. The page and its
 * route's prefetch read it from here, so both use the same key and paging.
 */
export function allSchedulesQuery() {
  return infiniteQueryOptions({
    queryKey: assistant.allSchedules().queryKey,
    initialPageParam: 0,
    queryFn: ({ pageParam, signal }) =>
      apiService.assistantService.listSchedules({
        signal,
        limit: SCHEDULE_PAGE_SIZE,
        offset: pageParam,
      }),
    getNextPageParam: nextScheduleOffset,
  });
}
