import { getLocale } from "@trenova/shared/i18n/runtime";
import { fromUserWallClock, getStartOfDay, userWallClockNow } from "@trenova/shared/lib/date";
import { addDays, format, isValid, parse, startOfDay } from "date-fns";

/**
 * Pure time-scale math shared by every horizontal timeline in the app (the
 * shipments board and the dispatch console). Positions are computed from absolute
 * epoch seconds so day boundaries stay correct across DST transitions: a day is
 * however many real seconds it contains, and pixel positions follow.
 *
 * A `Date` in this module is always a *wall clock in the user's timezone*, not
 * an instant: anchors, day columns and hour ticks are the days and hours the
 * dispatcher is living in. Crossing to epoch seconds always goes through
 * `fromUserWallClock` / `toUserWallClock`, which is what keeps a Denver board
 * from starting its day at 10pm because the server sent UTC.
 */

export type TimeRange = {
  /** Inclusive start, unix seconds. */
  start: number;
  /** Exclusive end, unix seconds. */
  end: number;
};

const ANCHOR_DATE_FORMAT = "yyyy-MM-dd";

export function parseAnchorDate(value: string | null): Date {
  if (value) {
    const parsed = parse(value, ANCHOR_DATE_FORMAT, new Date());
    if (isValid(parsed)) return startOfDay(parsed);
  }
  return startOfDay(userWallClockNow());
}

export function serializeAnchorDate(date: Date): string {
  return format(date, ANCHOR_DATE_FORMAT);
}

export function isTodayAnchor(anchor: Date): boolean {
  return anchor.getTime() === startOfDay(userWallClockNow()).getTime();
}

export function getRangeForDays(anchor: Date, days: number): TimeRange {
  const start = startOfDay(anchor);
  return {
    start: fromUserWallClock(start) ?? 0,
    end: fromUserWallClock(addDays(start, days)) ?? 0,
  };
}

export function shiftAnchorByDays(anchor: Date, days: number, direction: 1 | -1): Date {
  return addDays(startOfDay(anchor), days * direction);
}

/**
 * The window's label in the reader's language: "Wed, Nov 5, 2026" for a day, a range the
 * locale writes its own way for longer. Intl spells it, so no date-fns locale bundle rides
 * along.
 */
export function formatRangeLabelForDays(
  anchor: Date,
  days: number,
  locale: string = getLocale(),
): string {
  const start = startOfDay(anchor);
  if (days === 1) {
    return new Intl.DateTimeFormat(locale, {
      weekday: "short",
      month: "short",
      day: "numeric",
      year: "numeric",
    }).format(start);
  }
  const end = addDays(start, days - 1);
  return new Intl.DateTimeFormat(locale, {
    month: "short",
    day: "numeric",
    year: "numeric",
  }).formatRange(start, end);
}

const HOUR = 3_600;
const DAY = 86_400;

/**
 * The start of each day in the range in `timezone`, plus the end of the last day: the
 * boundaries a timeline's axis is drawn on. Each is found from that day's noon, so a day that
 * gains or loses an hour to daylight saving still starts at its own midnight.
 */
export function dayBoundaries(firstDayStart: number, days: number, timezone: string): number[] {
  const boundaries: number[] = [];
  for (let i = 0; i <= days; i++) {
    const noon = firstDayStart + i * DAY + 12 * HOUR;
    boundaries.push(getStartOfDay(new Date(noon * 1000), timezone));
  }
  return boundaries;
}
