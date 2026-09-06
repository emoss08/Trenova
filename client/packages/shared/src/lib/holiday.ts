import { fromZonedTime, toZonedTime } from "date-fns-tz";
import { resolveUserTimezone } from "./date";

/** The shape both the TMS calendar and the driver portal read holidays through. */
export type HolidayLike = {
  id: string;
  name: string;
  /** Midnight UTC of the calendar day, as the server stores it. */
  holidayDate: number;
  kind: "Holiday" | "Blackout";
  recursAnnually: boolean;
};

export type HolidayOccurrence<T extends HolidayLike = HolidayLike> = {
  entry: T;
  /** Midnight UTC of the day this entry lands on in the projected year. */
  date: number;
  year: number;
  /** 0-based month, matching `Date`. */
  month: number;
  day: number;
  /** `YYYY-MM-DD`, handy as a lookup key for day cells. */
  isoDate: string;
};

const DAY_SECONDS = 86_400;

export function utcMidnight(year: number, month: number, day: number): number {
  return Date.UTC(year, month, day) / 1000;
}

export function utcDateParts(unixSeconds: number): { year: number; month: number; day: number } {
  const date = new Date(unixSeconds * 1000);
  return { year: date.getUTCFullYear(), month: date.getUTCMonth(), day: date.getUTCDate() };
}

export function isoDateKey(year: number, month: number, day: number): string {
  return `${year}-${String(month + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
}

export function daysInMonth(year: number, month: number): number {
  return new Date(Date.UTC(year, month + 1, 0)).getUTCDate();
}

const KIND_ORDER: Record<HolidayLike["kind"], number> = { Holiday: 0, Blackout: 1 };

/**
 * Lands every entry on the requested year: recurring entries move to that
 * year (a 29 February becomes the 28th when the year has none), one-offs are
 * kept only when they are dated in it. Sorted by date, holidays before
 * blackouts, then name.
 */
export function projectHolidaysOntoYear<T extends HolidayLike>(
  entries: readonly T[],
  year: number,
): HolidayOccurrence<T>[] {
  const occurrences: HolidayOccurrence<T>[] = [];
  for (const entry of entries) {
    const parts = utcDateParts(entry.holidayDate);
    if (!entry.recursAnnually && parts.year !== year) continue;
    const day = Math.min(parts.day, daysInMonth(year, parts.month));
    occurrences.push({
      entry,
      date: utcMidnight(year, parts.month, day),
      year,
      month: parts.month,
      day,
      isoDate: isoDateKey(year, parts.month, day),
    });
  }
  occurrences.sort(
    (a, b) =>
      a.date - b.date ||
      KIND_ORDER[a.entry.kind] - KIND_ORDER[b.entry.kind] ||
      a.entry.name.localeCompare(b.entry.name),
  );
  return occurrences;
}

/** Twelve buckets, January first, so a calendar can render empty months too. */
export function groupOccurrencesByMonth<T extends HolidayLike>(
  occurrences: readonly HolidayOccurrence<T>[],
): HolidayOccurrence<T>[][] {
  const months: HolidayOccurrence<T>[][] = Array.from({ length: 12 }, () => []);
  for (const occurrence of occurrences) {
    months[occurrence.month].push(occurrence);
  }
  return months;
}

/** Formats a UTC-midnight day in UTC so it never drifts into the day before. */
export function formatUtcDate(
  unixSeconds: number,
  options: Intl.DateTimeFormatOptions = {},
): string {
  return new Intl.DateTimeFormat("en-US", {
    timeZone: "UTC",
    month: "short",
    day: "numeric",
    year: "numeric",
    ...options,
  }).format(new Date(unixSeconds * 1000));
}

/**
 * Collapses an instant the user picked on a calendar to the UTC midnight of
 * the calendar day they saw, which is how holidays are stored.
 */
export function toUtcDateOnly(unixSeconds: number, timezone?: string): number {
  const zoned = toZonedTime(new Date(unixSeconds * 1000), resolveUserTimezone(timezone));
  return utcMidnight(zoned.getFullYear(), zoned.getMonth(), zoned.getDate());
}

/** The inverse of {@link toUtcDateOnly}: the instant that day starts for the user. */
export function fromUtcDateOnly(unixSeconds: number, timezone?: string): number {
  const parts = utcDateParts(unixSeconds);
  return Math.floor(
    fromZonedTime(
      new Date(parts.year, parts.month, parts.day),
      resolveUserTimezone(timezone),
    ).getTime() / 1000,
  );
}

export function addUtcDays(unixSeconds: number, days: number): number {
  return unixSeconds + days * DAY_SECONDS;
}
