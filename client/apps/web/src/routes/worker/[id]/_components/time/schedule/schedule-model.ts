import { dateTimeFormatter } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { dayMaskToDays } from "@trenova/shared/lib/scheduling";
import type { AvailabilityPreferenceValue } from "@trenova/shared/types/scheduling";

const MINUTES_IN_DAY = 1440;
const DAY_SECONDS = 86_400;

/** The week as the schedule draws it, Monday first; values are Sunday-based day indexes. */
export const WEEK_ORDER = [1, 2, 3, 4, 5, 6, 0] as const;

function sampleDay(day: number): Date {
  return new Date(2026, 0, 4 + day);
}

/** A Sunday-based day index as the reader's language shortens it ("Mon"). */
export function shortWeekday(day: number): string {
  return dateTimeFormatter({ weekday: "short" }).format(sampleDay(day));
}

/** A Sunday-based day index in full ("Monday"). */
export function longWeekday(day: number): string {
  return dateTimeFormatter({ weekday: "long" }).format(sampleDay(day));
}

export type DayBar = { startPercent: number; widthPercent: number };

export type WeekDay = {
  day: number;
  bars: DayBar[];
  /** Minutes the pattern starts on this day, 0 when the day is off. */
  minutes: number;
};

/**
 * A shift pattern laid on a week, Monday first: each working day's bar from its start minute,
 * and a shift that runs past midnight carried onto the next day's bar.
 */
export function weekBars(mask: string, startMinute: number, durationMinutes: number): WeekDay[] {
  const working = new Set(dayMaskToDays(mask));
  const bars = new Map<number, DayBar[]>(WEEK_ORDER.map((day) => [day, []]));
  const start = Math.max(0, Math.min(MINUTES_IN_DAY, startMinute));
  const duration = Math.max(0, Math.min(MINUTES_IN_DAY, durationMinutes));
  const end = start + duration;
  for (const day of working) {
    const firstPart = Math.min(end, MINUTES_IN_DAY) - start;
    if (firstPart > 0) {
      bars.get(day)?.push({
        startPercent: (start / MINUTES_IN_DAY) * 100,
        widthPercent: (firstPart / MINUTES_IN_DAY) * 100,
      });
    }
    if (end > MINUTES_IN_DAY) {
      bars.get((day + 1) % 7)?.push({
        startPercent: 0,
        widthPercent: ((end - MINUTES_IN_DAY) / MINUTES_IN_DAY) * 100,
      });
    }
  }
  return WEEK_ORDER.map((day) => ({
    day,
    bars: bars.get(day) ?? [],
    minutes: working.has(day) ? duration : 0,
  }));
}

export type StatedPreference = { dayOfWeek: number; preference: AvailabilityPreferenceValue };

/** The working days of a pattern on which the worker said they can't work. */
export function shiftClashes(mask: string, preferences: readonly StatedPreference[]): number[] {
  const unavailable = new Set(
    preferences.filter((row) => row.preference === "Unavailable").map((row) => row.dayOfWeek),
  );
  return WEEK_ORDER.filter((day) => dayMaskToDays(mask).includes(day) && unavailable.has(day));
}

function runs(days: readonly number[], shortDay: (day: number) => string, t: TranslateFn): string {
  const ordered = WEEK_ORDER.filter((day) => days.includes(day));
  const positions = ordered.map((day) => WEEK_ORDER.indexOf(day));
  const parts: string[] = [];
  let index = 0;
  while (index < positions.length) {
    let last = index;
    while (last + 1 < positions.length && positions[last + 1] === positions[last] + 1) last += 1;
    const first = shortDay(ordered[index]);
    const final = shortDay(ordered[last]);
    if (last - index >= 2) parts.push(t("{0} to {1}", first, final));
    else if (last > index) parts.push(t("{0}, {1}", first, final));
    else parts.push(first);
    index = last + 1;
  }
  return parts.join(", ");
}

/** What the worker said in one line: "Prefers Tue to Thu · not weekends". */
export function availabilitySummary(
  t: TranslateFn,
  preferences: readonly StatedPreference[],
  shortDay: (day: number) => string,
): string {
  if (preferences.length === 0) return t("Nothing said yet");
  const preferred = preferences.filter((row) => row.preference === "Preferred");
  const unavailable = preferences
    .filter((row) => row.preference === "Unavailable")
    .map((row) => row.dayOfWeek);
  const parts: string[] = [];
  if (preferred.length > 0) {
    parts.push(
      t(
        "Prefers {0}",
        runs(
          preferred.map((row) => row.dayOfWeek),
          shortDay,
          t,
        ),
      ),
    );
  }
  if (unavailable.length === 0) {
    parts.push(t("any day"));
  } else if (unavailable.length === 2 && unavailable.includes(0) && unavailable.includes(6)) {
    parts.push(t("not weekends"));
  } else {
    parts.push(t("not {0}", runs(unavailable, shortDay, t)));
  }
  return parts.join(" · ");
}

/** Whole days since the worker was last home, or null when nobody has recorded it. */
export function daysOut(lastHomeAt: number | null | undefined, nowSec: number): number | null {
  if (!lastHomeAt) return null;
  return Math.max(0, Math.floor((nowSec - lastHomeAt) / DAY_SECONDS));
}

export type TimeOffSpan = { status: string; startDate: number; endDate: number };

/** The next approved time off that has not ended yet, earliest first. */
export function nextApprovedTimeOff<T extends TimeOffSpan>(
  entries: readonly T[],
  nowSec: number,
): T | null {
  let next: T | null = null;
  for (const entry of entries) {
    if (entry.status !== "Approved" || entry.endDate < nowSec) continue;
    if (next === null || entry.startDate < next.startDate) next = entry;
  }
  return next;
}
