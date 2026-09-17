import type { AssistantThread } from "@/types/assistant";
import { toUserWallClock } from "@trenova/shared/lib/date";

export type ThreadGroupLabel = "Today" | "Yesterday" | "Previous 7 days" | "Older";

export type ThreadGroup = {
  label: ThreadGroupLabel;
  threads: AssistantThread[];
};

const SHELVES: ThreadGroupLabel[] = ["Today", "Yesterday", "Previous 7 days", "Older"];
const DAY_SECONDS = 24 * 60 * 60;

/** When a thread was last touched: its last message, or its creation. */
function touchedAt(thread: AssistantThread): number {
  return thread.lastMessageAt > 0 ? thread.lastMessageAt : thread.createdAt;
}

/**
 * Days between two instants as the reader's calendar counts them, so a
 * conversation from 23:50 last night is "Yesterday" and not "Today" because it
 * was fewer than 24 hours ago.
 */
function calendarDaysAgo(at: number, now: number, timezone: string): number {
  const startOfDay = (unix: number) => {
    const local = toUserWallClock(unix, timezone) ?? new Date(unix * 1000);
    return Date.UTC(local.getFullYear(), local.getMonth(), local.getDate());
  };

  return Math.round((startOfDay(now) - startOfDay(at)) / (DAY_SECONDS * 1000));
}

function shelfFor(daysAgo: number): ThreadGroupLabel {
  if (daysAgo <= 0) return "Today";
  if (daysAgo === 1) return "Yesterday";
  if (daysAgo <= 7) return "Previous 7 days";
  return "Older";
}

/**
 * Shelves conversations by how recently they were touched, newest first on
 * each shelf, leaving out shelves with nothing on them.
 */
export function groupThreadsByRecency(
  threads: readonly AssistantThread[],
  now: number,
  timezone?: string,
): ThreadGroup[] {
  const byShelf = new Map<ThreadGroupLabel, AssistantThread[]>();

  for (const thread of [...threads].sort((a, b) => touchedAt(b) - touchedAt(a))) {
    const label = shelfFor(calendarDaysAgo(touchedAt(thread), now, timezone ?? "UTC"));
    const shelf = byShelf.get(label);
    if (shelf) {
      shelf.push(thread);
    } else {
      byShelf.set(label, [thread]);
    }
  }

  return SHELVES.flatMap((label) => {
    const shelf = byShelf.get(label);
    return shelf ? [{ label, threads: shelf }] : [];
  });
}
