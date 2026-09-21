import { toUserWallClock } from "@trenova/shared/lib/date";

const DAY_MS = 24 * 60 * 60 * 1000;

/** The reader's calendar day of an instant, as a UTC midnight for arithmetic. */
export function calendarDayOf(unix: number, timezone: string): number {
  const local = toUserWallClock(unix, timezone) ?? new Date(unix * 1000);
  return Date.UTC(local.getFullYear(), local.getMonth(), local.getDate());
}

/**
 * Days between two instants as the reader's calendar counts them, so 23:50
 * last night is "yesterday" and not "today" because it was fewer than 24 hours
 * ago.
 */
export function calendarDaysAgo(at: number, now: number, timezone: string): number {
  return Math.round((calendarDayOf(now, timezone) - calendarDayOf(at, timezone)) / DAY_MS);
}
