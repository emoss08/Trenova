import { calendarDaysAgo } from "@/lib/calendar-days";
import { caseStateOf } from "@/lib/case-state";
import type { AssistantThread } from "@/types/assistant";

/**
 * A shelf of the rail: what the person pinned, how long ago the rest was
 * touched, and below them the cases put away until later and the cases
 * whose record has closed.
 */
export type DeskShelfKey =
  | "pinned"
  | "today"
  | "yesterday"
  | "week"
  | "month"
  | "older"
  | "snoozed"
  | "settled";

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

const SHELF_ORDER: readonly DeskShelfKey[] = [
  "pinned",
  "today",
  "yesterday",
  "week",
  "month",
  "older",
  "snoozed",
  "settled",
];

function touchedAt(thread: AssistantThread): number {
  return thread.lastMessageAt > 0 ? thread.lastMessageAt : thread.createdAt;
}

function newestFirst(a: AssistantThread, b: AssistantThread): number {
  return touchedAt(b) - touchedAt(a);
}

function shelfFor(daysAgo: number): DeskShelfKey {
  if (daysAgo <= 0) return "today";
  if (daysAgo === 1) return "yesterday";
  if (daysAgo <= 7) return "week";
  if (daysAgo <= 30) return "month";
  return "older";
}

function shelfOf(
  thread: AssistantThread,
  now: number,
  pinnedFirst: boolean,
  timezone: string,
): DeskShelfKey {
  if (pinnedFirst && thread.pinned) {
    return "pinned";
  }
  const state = thread.case ? caseStateOf(thread.case, now) : null;
  if (state === "Snoozed") {
    return "snoozed";
  }
  if (state === "Settled") {
    return "settled";
  }

  return shelfFor(calendarDaysAgo(touchedAt(thread), now, timezone));
}

/**
 * The rail's order: what the person pinned on top, then the rest shelved by
 * how recently it was touched on the reader's own calendar, newest first on
 * each shelf, then the snoozed cases and the settled ones. Shelves with
 * nothing on them are left out.
 */
export function groupDeskThreadsByRecency(
  threads: readonly AssistantThread[],
  now: number,
  { pinnedFirst, timezone }: GroupDeskThreadsOptions,
): DeskThreadShelf[] {
  if (threads.length === 0) {
    return [];
  }

  const byShelf = new Map<DeskShelfKey, AssistantThread[]>();
  for (const thread of [...threads].sort(newestFirst)) {
    const key = shelfOf(thread, now, pinnedFirst, timezone ?? "");
    const shelf = byShelf.get(key);
    if (shelf) {
      shelf.push(thread);
    } else {
      byShelf.set(key, [thread]);
    }
  }

  const shelves: DeskThreadShelf[] = [];
  for (const key of SHELF_ORDER) {
    const shelf = byShelf.get(key);
    if (shelf) {
      shelves.push({ key, threads: shelf });
    }
  }

  return shelves;
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
