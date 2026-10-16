import type { WatchtowerItem } from "@/lib/graphql/watchtower";
import { compareQueueItems } from "@/lib/watchtower-cache";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  formatMinutesSpan,
  formatUnixDateTimeShort,
  formatUnixTime,
  toUserWallClock,
} from "@trenova/shared/lib/date";

/**
 * The deadline lanes the queue is read in, by how long is left: inside four
 * hours (or already past), before the end of the day, days out, and nothing
 * due at all.
 */
export type Lane = "now" | "today" | "later" | "fyi";

export const LANES: readonly Lane[] = ["now", "today", "later", "fyi"];

/** How close a deadline is before it is in the act-now lane. */
export const ACT_NOW_MINUTES = 240;
const TODAY_MINUTES = 1440;
/** A countdown under an hour is drawn hot. */
const HOT_MINUTES = 60;
/** The window the deadline clock drains over. */
export const CLOCK_WINDOW_MINUTES = 240;

/** Whole minutes until a due time, negative once it has passed; null when there is none. */
export function minutesLeft(dueAt: number | null | undefined, now: number): number | null {
  if (dueAt == null) {
    return null;
  }
  return Math.trunc((dueAt - now) / 60);
}

export function laneOf(minutes: number | null): Lane {
  if (minutes === null) {
    return "fyi";
  }
  if (minutes <= ACT_NOW_MINUTES) {
    return "now";
  }
  return minutes <= TODAY_MINUTES ? "today" : "later";
}

export function laneLabel(t: TranslateFn, lane: Lane): string {
  switch (lane) {
    case "now":
      return t("Act now");
    case "today":
      return t("Today");
    case "later":
      return t("Coming up");
    default:
      return t("For your information");
  }
}

/** The queue in lanes, soonest first within each, undated last; empty lanes left out. */
export function groupByLane(
  items: readonly WatchtowerItem[],
  now: number,
): { lane: Lane; items: WatchtowerItem[] }[] {
  const byLane = new Map<Lane, WatchtowerItem[]>(LANES.map((lane) => [lane, []]));
  for (const item of [...items].sort(compareQueueItems)) {
    byLane.get(laneOf(minutesLeft(item.dueAt, now)))?.push(item);
  }

  return LANES.flatMap((lane) => {
    const laneItems = byLane.get(lane) ?? [];
    return laneItems.length === 0 ? [] : [{ lane, items: laneItems }];
  });
}

/**
 * A countdown the way the card and the queue say it: "Now", "51m running"
 * once it has passed, "45m", "3h 10m", and whole days past one.
 */
export function formatCountdown(t: TranslateFn, minutes: number | null): string | null {
  if (minutes === null) {
    return null;
  }
  if (minutes < 0) {
    return t("{0} running", formatMinutesSpan(-minutes));
  }
  if (minutes === 0) {
    return t("Now");
  }
  if (minutes < TODAY_MINUTES) {
    return formatMinutesSpan(minutes);
  }

  return t("{0, plural, one {# day} other {# days}}", Math.round(minutes / TODAY_MINUTES));
}

/** Whether a row's countdown is drawn hot: under an hour left, or past. */
export function isHot(minutes: number | null): boolean {
  return minutes !== null && minutes <= HOT_MINUTES;
}

/** Whether the queue row shows its countdown rather than the unopened dot. */
export function showsCountdown(minutes: number | null): boolean {
  return minutes !== null && minutes < TODAY_MINUTES;
}

/** What the clock says under the countdown: what happens then, and whether it is still to come. */
export function clockCaption(
  t: TranslateFn,
  item: Pick<WatchtowerItem, "dueLabel">,
  minutes: number,
): string {
  if (minutes > 0) {
    return item.dueLabel ? t("{0} in", item.dueLabel) : t("Due in");
  }
  return item.dueLabel ?? (minutes < 0 ? t("Overdue") : t("Due now"));
}

/** How much of the clock's window is still left, from 1 (four hours or more) to 0. */
export function clockRemaining(minutes: number): number {
  return Math.min(1, Math.max(0, minutes) / CLOCK_WINDOW_MINUTES);
}

function sameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

/**
 * An instant as a fact reads it, on the reader's own clock: "Today 6:00 PM",
 * "Tomorrow 8:00 AM", "Yesterday 1:40 PM", else the date and time.
 */
export function formatFactTime(t: TranslateFn, at: number, now: number, timezone: string): string {
  const when = toUserWallClock(at, timezone);
  const today = toUserWallClock(now, timezone);
  const time = formatUnixTime(at, { timezone });
  if (when !== undefined && today !== undefined) {
    if (sameDay(when, today)) {
      return t("Today {0}", time);
    }
    const tomorrow = new Date(today);
    tomorrow.setDate(today.getDate() + 1);
    if (sameDay(when, tomorrow)) {
      return t("Tomorrow {0}", time);
    }
    const yesterday = new Date(today);
    yesterday.setDate(today.getDate() - 1);
    if (sameDay(when, yesterday)) {
      return t("Yesterday {0}", time);
    }
  }

  return formatUnixDateTimeShort(at, { timezone });
}
