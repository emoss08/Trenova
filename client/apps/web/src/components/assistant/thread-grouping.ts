import { calendarDaysAgo } from "@/lib/calendar-days";
import type { AssistantThread } from "@/types/assistant";

export type ThreadGroupLabel = "Today" | "Yesterday" | "Previous 7 days" | "Older";

export type ThreadGroup = {
  label: ThreadGroupLabel;
  threads: AssistantThread[];
};

const SHELVES: ThreadGroupLabel[] = ["Today", "Yesterday", "Previous 7 days", "Older"];

/** When a thread was last touched: its last message, or its creation. */
function touchedAt(thread: AssistantThread): number {
  return thread.lastMessageAt > 0 ? thread.lastMessageAt : thread.createdAt;
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
