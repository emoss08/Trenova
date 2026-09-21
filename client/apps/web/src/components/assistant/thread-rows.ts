import { calendarDayOf, calendarDaysAgo } from "@/lib/calendar-days";
import type { AssistantMessage } from "@/types/assistant";
import type { ThreadEntry } from "./thread-view";

export type DayMarker = {
  kind: "day";
  key: string;
  /** The first instant of the day's messages, for formatting the date. */
  at: number;
  daysAgo: number;
};

export type ThreadRowItem = DayMarker | { kind: "entry"; entry: ThreadEntry };

/**
 * Interleaves a day marker before the first message of each calendar day, as
 * the reader's clock counts days. A thread that spans a week reads with its
 * dates in it rather than as one unbroken column of times.
 */
export function withDayMarkers(
  entries: readonly ThreadEntry[],
  now: number,
  timezone: string,
): ThreadRowItem[] {
  const rows: ThreadRowItem[] = [];
  let currentDay: number | null = null;
  // A day can recur when a row's clock is out of step with its neighbours.
  // Each marker still needs a key of its own, or the window measures one
  // against the other.
  const seen = new Map<number, number>();

  for (const entry of entries) {
    const at = entry.message.createdAt;
    const day = at > 0 ? calendarDayOf(at, timezone) : null;
    if (day !== null && day !== currentDay) {
      currentDay = day;
      const repeat = seen.get(day) ?? 0;
      seen.set(day, repeat + 1);
      rows.push({
        kind: "day",
        key: repeat === 0 ? `day-${day}` : `day-${day}-${repeat}`,
        at,
        daysAgo: calendarDaysAgo(at, now, timezone),
      });
    }
    rows.push({ kind: "entry", entry });
  }

  return rows;
}

/**
 * The messages that arrived after a point: the ones worth animating in.
 *
 * A windowed list mounts rows every time they scroll into view, and a page of
 * older history mounts a screenful at once. Neither is an arrival. Only a
 * message numbered past what the thread held when it was opened is new to
 * the reader, and only that one rises.
 */
export function arrivedSince(
  sinceSequence: number,
  messages: readonly AssistantMessage[],
): Set<string> {
  const fresh = new Set<string>();
  for (const message of messages) {
    if (message.sequence > sinceSequence) {
      fresh.add(message.id);
    }
  }

  return fresh;
}

/** The highest sequence in a list, or -1 for none. */
export function highestSequence(messages: readonly AssistantMessage[]): number {
  let highest = -1;
  for (const message of messages) {
    if (message.sequence > highest) {
      highest = message.sequence;
    }
  }

  return highest;
}
