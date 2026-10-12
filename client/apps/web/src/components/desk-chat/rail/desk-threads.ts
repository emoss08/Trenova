import { calendarDayStart } from "@/lib/calendar-days";
import type { ThreadListFilter } from "@/services/assistant";
import type { AssistantThread } from "@/types/assistant";

/**
 * A shelf of the rail: what the person pinned, then how long ago the rest
 * was touched on the reader's own calendar.
 */
export type DeskShelfKey = "pinned" | "today" | "yesterday" | "week" | "month" | "older";

export type DeskThreadShelf = {
  key: DeskShelfKey;
  threads: AssistantThread[];
};

export type GroupDeskThreadsOptions = {
  /** Hold what the person pinned above the calendar rather than shelving it by date. */
  pinnedFirst: boolean;
  /** The calendar the days are counted on; the reader's own by default. */
  timezone?: string;
};

/** Every shelf, in the order the rail draws them. */
export const DESK_SHELF_KEYS: readonly DeskShelfKey[] = [
  "pinned",
  "today",
  "yesterday",
  "week",
  "month",
  "older",
];

/**
 * Where each dated shelf begins, in Unix seconds: the reader's midnight
 * today, yesterday, seven days ago and thirty days ago. Older is everything
 * before the last.
 */
export type DeskShelfEdges = {
  today: number;
  yesterday: number;
  week: number;
  month: number;
};

/** The shelves' edges on the reader's calendar, for the day `now` falls on. */
export function deskShelfEdges(now: number, timezone: string): DeskShelfEdges {
  return {
    today: calendarDayStart(now, 0, timezone),
    yesterday: calendarDayStart(now, 1, timezone),
    week: calendarDayStart(now, 7, timezone),
    month: calendarDayStart(now, 30, timezone),
  };
}

/** The edges newest first, as the server bands the shelf counts by them. */
export function deskShelfEdgeList(edges: DeskShelfEdges): number[] {
  return [edges.today, edges.yesterday, edges.week, edges.month];
}

/**
 * When a conversation was last touched: its last message, or when it was
 * made if it has none. The server orders and ranges each shelf on the same
 * instant, so a conversation is read on the shelf it is drawn on.
 */
export function threadTouchedAt(thread: Pick<AssistantThread, "lastMessageAt" | "createdAt">) {
  return Math.max(thread.lastMessageAt, thread.createdAt);
}

/** What the server is asked for to read one shelf. */
export function deskShelfFilter(key: DeskShelfKey, edges: DeskShelfEdges): ThreadListFilter {
  switch (key) {
    case "pinned":
      return { pinned: true };
    case "today":
      return { pinned: false, touchedFrom: edges.today };
    case "yesterday":
      return { pinned: false, touchedFrom: edges.yesterday, touchedBefore: edges.today };
    case "week":
      return { pinned: false, touchedFrom: edges.week, touchedBefore: edges.yesterday };
    case "month":
      return { pinned: false, touchedFrom: edges.month, touchedBefore: edges.week };
    default:
      return { pinned: false, touchedBefore: edges.month };
  }
}

/** The shelf a conversation sits on. */
export function deskShelfOf(
  thread: AssistantThread,
  edges: DeskShelfEdges,
  pinnedFirst: boolean,
): DeskShelfKey {
  if (pinnedFirst && thread.pinned) {
    return "pinned";
  }
  const at = threadTouchedAt(thread);
  if (at >= edges.today) return "today";
  if (at >= edges.yesterday) return "yesterday";
  if (at >= edges.week) return "week";
  if (at >= edges.month) return "month";
  return "older";
}

function newestFirst(a: AssistantThread, b: AssistantThread): number {
  return threadTouchedAt(b) - threadTouchedAt(a);
}

/**
 * Conversations on the shelves given edges put them on, in the rail's order,
 * newest first on each. Shelves with nothing on them are left out.
 */
export function groupDeskThreadsByShelf(
  threads: readonly AssistantThread[],
  edges: DeskShelfEdges,
  pinnedFirst: boolean,
): DeskThreadShelf[] {
  const byShelf = new Map<DeskShelfKey, AssistantThread[]>();
  for (const thread of [...threads].sort(newestFirst)) {
    const key = deskShelfOf(thread, edges, pinnedFirst);
    const shelf = byShelf.get(key);
    if (shelf) {
      shelf.push(thread);
    } else {
      byShelf.set(key, [thread]);
    }
  }

  const shelves: DeskThreadShelf[] = [];
  for (const key of DESK_SHELF_KEYS) {
    const shelf = byShelf.get(key);
    if (shelf) {
      shelves.push({ key, threads: shelf });
    }
  }

  return shelves;
}

/** The later of two copies of one conversation, as two lists read at different times hold it. */
function fresher(a: AssistantThread, b: AssistantThread): AssistantThread {
  if (a.updatedAt !== b.updatedAt) {
    return a.updatedAt > b.updatedAt ? a : b;
  }
  return a.version >= b.version ? a : b;
}

/**
 * Conversations read by several lists, each kept once in its latest copy and
 * put on the shelf that copy's last touch puts it on, keyed by shelf.
 *
 * A conversation touched since an older shelf was read is held by that shelf
 * and a newer one until the older is read again; it is drawn once, where it
 * is now. `also` adds conversations known some other way, such as the open
 * one read on its own.
 */
export function shelveLatest(
  lists: readonly (readonly AssistantThread[])[],
  edges: DeskShelfEdges,
  also: readonly AssistantThread[] = [],
): Map<DeskShelfKey, AssistantThread[]> {
  const latest = new Map<string, AssistantThread>();
  const take = (thread: AssistantThread) => {
    const known = latest.get(thread.id);
    latest.set(thread.id, known ? fresher(known, thread) : thread);
  };
  for (const list of lists) {
    for (const thread of list) {
      take(thread);
    }
  }
  for (const thread of also) {
    take(thread);
  }

  return new Map(
    groupDeskThreadsByShelf([...latest.values()], edges, true).map((shelf) => [
      shelf.key,
      shelf.threads,
    ]),
  );
}

/**
 * The rail's order: what the person pinned on top, then the rest shelved by
 * how recently it was touched on the reader's own calendar, newest first on
 * each shelf. Shelves with nothing on them are left out.
 */
export function groupDeskThreadsByRecency(
  threads: readonly AssistantThread[],
  now: number,
  { pinnedFirst, timezone }: GroupDeskThreadsOptions,
): DeskThreadShelf[] {
  if (threads.length === 0) {
    return [];
  }

  return groupDeskThreadsByShelf(threads, deskShelfEdges(now, timezone ?? ""), pinnedFirst);
}

/** Whether a conversation's title or agent contains the search, case ignored. */
export function matchesThreadSearch(
  thread: AssistantThread,
  agentName: string,
  query: string,
): boolean {
  const needle = query.trim().toLowerCase();
  if (needle === "") {
    return true;
  }

  return `${thread.title} ${agentName}`.toLowerCase().includes(needle);
}
