import type { WatchtowerCounts, WatchtowerItem } from "@/lib/graphql/watchtower";
import { watchtower, type WatchtowerQueuePages } from "@/lib/queries/watchtower";
import type { QueryClient } from "@tanstack/react-query";

/** Where an item with no due time sorts: after every dated one, as the server reads it. */
const UNDATED = 9_000_000_000_000;

/** Whether an item belongs in the queue: open and awake. */
export function belongsInQueue(item: Pick<WatchtowerItem, "resolvedAt" | "status">): boolean {
  return item.resolvedAt == null && item.status !== "Snoozed" && item.status !== "Resolved";
}

/** The queue's order: soonest due first, undated last, then by id, as the cursor pages it. */
export function compareQueueItems(
  a: Pick<WatchtowerItem, "dueAt" | "id">,
  b: Pick<WatchtowerItem, "dueAt" | "id">,
): number {
  const due = (a.dueAt ?? UNDATED) - (b.dueAt ?? UNDATED);
  if (due !== 0) {
    return due;
  }
  return a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
}

function readPages(queryClient: QueryClient): WatchtowerQueuePages | undefined {
  return queryClient.getQueryData<WatchtowerQueuePages>(watchtower.queue.queryKey);
}

/** The queue's rows as they are cached, in order. */
export function cachedQueueItems(queryClient: QueryClient): WatchtowerItem[] {
  return readPages(queryClient)?.pages.flatMap((page) => page.items) ?? [];
}

/** A row the queue holds, if it holds it. */
export function cachedQueueItem(queryClient: QueryClient, id: string): WatchtowerItem | undefined {
  return cachedQueueItems(queryClient).find((item) => item.id === id);
}

/** The rows an agent run is working, so a run that moves can redraw them. */
export function cachedQueueItemsForRun(queryClient: QueryClient, runId: string): WatchtowerItem[] {
  return cachedQueueItems(queryClient).filter((item) => item.agentWork?.runId === runId);
}

/**
 * Puts an item where it now belongs in the cached queue: it leaves wherever it
 * was, and comes back in its sorted place if it is still open and awake. An
 * item that sorts after everything loaded while more pages wait is left for
 * those pages to bring, so the queue never shows a row out of order.
 */
export function placeInQueue(queryClient: QueryClient, item: WatchtowerItem): void {
  queryClient.setQueryData<WatchtowerQueuePages>(watchtower.queue.queryKey, (data) => {
    if (data === undefined) {
      return data;
    }
    const pages = data.pages.map((page) => ({
      ...page,
      items: page.items.filter((row) => row.id !== item.id),
    }));
    if (!belongsInQueue(item) || pages.length === 0) {
      return { ...data, pages };
    }

    for (let index = 0; index < pages.length; index += 1) {
      const page = pages[index];
      const position = page.items.findIndex((row) => compareQueueItems(item, row) < 0);
      const isLast = index === pages.length - 1;
      if (position !== -1) {
        page.items = [...page.items.slice(0, position), item, ...page.items.slice(position)];
        return { ...data, pages };
      }
      if (isLast && !page.hasNextPage) {
        page.items = [...page.items, item];
        return { ...data, pages };
      }
    }

    return { ...data, pages };
  });
}

/** Changes one cached row in place, wherever it is. */
export function patchQueueItem(
  queryClient: QueryClient,
  id: string,
  patch: (item: WatchtowerItem) => WatchtowerItem,
): void {
  queryClient.setQueryData<WatchtowerQueuePages>(watchtower.queue.queryKey, (data) =>
    data === undefined
      ? data
      : {
          ...data,
          pages: data.pages.map((page) => ({
            ...page,
            items: page.items.map((item) => (item.id === id ? patch(item) : item)),
          })),
        },
  );
}

/** Takes a row off the cached queue. */
export function removeFromQueue(queryClient: QueryClient, id: string): void {
  queryClient.setQueryData<WatchtowerQueuePages>(watchtower.queue.queryKey, (data) =>
    data === undefined
      ? data
      : {
          ...data,
          pages: data.pages.map((page) => ({
            ...page,
            items: page.items.filter((item) => item.id !== id),
          })),
        },
  );
}

/**
 * Moves the cached counts by one item leaving the queue (`delta` -1) or coming
 * back to it (+1), so the rail's number follows the queue before the server
 * answers.
 */
export function shiftQueueCounts(
  queryClient: QueryClient,
  item: Pick<WatchtowerItem, "severity">,
  delta: 1 | -1,
): void {
  queryClient.setQueryData<WatchtowerCounts>(watchtower.counts().queryKey, (counts) =>
    counts === undefined
      ? counts
      : {
          ...counts,
          active: Math.max(0, counts.active + delta),
          activeCritical:
            item.severity === "Critical"
              ? Math.max(0, counts.activeCritical + delta)
              : counts.activeCritical,
        },
  );
}

/** Moves today's cleared count by one. */
export function shiftCleared(queryClient: QueryClient, since: number, delta: 1 | -1): void {
  queryClient.setQueryData<number>(watchtower.cleared(since).queryKey, (cleared) =>
    cleared === undefined ? cleared : Math.max(0, cleared + delta),
  );
}

/** The workflow fields a watchtower notice's entity carries in the same shape the queue holds them. */
type NoticeFields = Partial<
  Pick<
    WatchtowerItem,
    | "version"
    | "status"
    | "resolvedAt"
    | "snoozedUntil"
    | "dueAt"
    | "severity"
    | "title"
    | "summary"
  >
>;

function noticeFields(entity: Record<string, unknown>): NoticeFields | null {
  if (typeof entity.version !== "number") {
    return null;
  }
  const fields: NoticeFields = { version: entity.version };
  const numberOrNull = (value: unknown) =>
    typeof value === "number" ? value : value === null || value === undefined ? null : undefined;
  if (typeof entity.status === "string") fields.status = entity.status as WatchtowerItem["status"];
  if (typeof entity.severity === "string")
    fields.severity = entity.severity as WatchtowerItem["severity"];
  if (typeof entity.title === "string") fields.title = entity.title;
  if (typeof entity.summary === "string") fields.summary = entity.summary;
  const resolvedAt = numberOrNull(entity.resolvedAt);
  if (resolvedAt !== undefined) fields.resolvedAt = resolvedAt;
  const snoozedUntil = numberOrNull(entity.snoozedUntil);
  if (snoozedUntil !== undefined) fields.snoozedUntil = snoozedUntil;
  const dueAt = numberOrNull(entity.dueAt);
  if (dueAt !== undefined) fields.dueAt = dueAt;
  return fields;
}

/**
 * Applies what a notice says about an item to its cached row at once, the
 * way every other list patches a row from its notice: the version and the
 * workflow state move before anything is read again, so an action taken in
 * between is made against the item as it now stands. A notice older than
 * the row (they can arrive out of order) changes nothing, and an item the
 * notice says is resolved or snoozed leaves the queue.
 */
export function applyNoticeToQueue(
  queryClient: QueryClient,
  id: string,
  entity: Record<string, unknown> | undefined,
): void {
  const known = cachedQueueItem(queryClient, id);
  const fields = entity === undefined ? null : noticeFields(entity);
  if (known === undefined || fields === null || (fields.version ?? 0) < known.version) {
    return;
  }
  const next = { ...known, ...fields };
  if (!belongsInQueue(next)) {
    removeFromQueue(queryClient, id);
    return;
  }
  if (next.dueAt !== known.dueAt) {
    placeInQueue(queryClient, next);
    return;
  }
  patchQueueItem(queryClient, id, () => next);
}
