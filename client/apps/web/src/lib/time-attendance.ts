import type { OpenTimeEntryRow, TimesheetRow } from "@/lib/graphql/timesheet";
import { elapsedMinutes } from "@trenova/shared/lib/timesheet";

export type SheetTotals = Pick<
  TimesheetRow,
  "workerId" | "regularMinutes" | "overtimeMinutes" | "paidLeaveMinutes"
>;

export type SheetSummary = {
  count: number;
  /** Distinct people behind the weeks, which is the number a manager thinks in. */
  workers: number;
  regularMinutes: number;
  overtimeMinutes: number;
  paidLeaveMinutes: number;
  totalMinutes: number;
  /** Weeks carrying any overtime at all, the ones a payroll clerk double-checks. */
  overtimeWeeks: number;
};

export function summarizeSheets(rows: readonly SheetTotals[]): SheetSummary {
  const workers = new Set<string>();
  const summary: SheetSummary = {
    count: rows.length,
    workers: 0,
    regularMinutes: 0,
    overtimeMinutes: 0,
    paidLeaveMinutes: 0,
    totalMinutes: 0,
    overtimeWeeks: 0,
  };
  for (const row of rows) {
    workers.add(row.workerId);
    summary.regularMinutes += row.regularMinutes;
    summary.overtimeMinutes += row.overtimeMinutes;
    summary.paidLeaveMinutes += row.paidLeaveMinutes;
    if (row.overtimeMinutes > 0) summary.overtimeWeeks += 1;
  }
  summary.workers = workers.size;
  summary.totalMinutes =
    summary.regularMinutes + summary.overtimeMinutes + summary.paidLeaveMinutes;
  return summary;
}

export type QueueSegment = "Submitted" | "Open" | "Approved" | "Locked";

/**
 * The queue's four views and the statuses behind each. A week sent back is
 * still a week somebody has to finish, so it lives with the open ones rather
 * than vanishing between "open" and "awaiting approval".
 */
export const QUEUE_SEGMENT_STATUSES: Record<QueueSegment, readonly string[]> = {
  Submitted: ["Submitted"],
  Open: ["Open", "Rejected"],
  Approved: ["Approved"],
  Locked: ["Locked"],
};

/** Every status the live views read together, so one request serves three segments. */
export const ACTIVE_QUEUE_STATUSES = ["Submitted", "Open", "Rejected", "Approved"] as const;

export function queueSegmentOf(status: string): QueueSegment | null {
  for (const segment of Object.keys(QUEUE_SEGMENT_STATUSES) as QueueSegment[]) {
    if (QUEUE_SEGMENT_STATUSES[segment].includes(status)) return segment;
  }
  return null;
}

export function countBySegment(rows: readonly { status: string }[]): Record<QueueSegment, number> {
  const counts: Record<QueueSegment, number> = { Submitted: 0, Open: 0, Approved: 0, Locked: 0 };
  for (const row of rows) {
    const segment = queueSegmentOf(row.status);
    if (segment) counts[segment] += 1;
  }
  return counts;
}

export function sheetsInSegment<T extends { status: string }>(
  rows: readonly T[],
  segment: QueueSegment,
): T[] {
  return rows.filter((row) => queueSegmentOf(row.status) === segment);
}

export type RunningEntry<T extends Pick<OpenTimeEntryRow, "clockedInAt"> = OpenTimeEntryRow> = {
  entry: T;
  runningMinutes: number;
};

/** Everyone on the clock, longest-running first: the one to worry about is the one still going. */
export function rankRunning<T extends Pick<OpenTimeEntryRow, "clockedInAt">>(
  entries: readonly T[],
  now: number,
): RunningEntry<T>[] {
  return entries
    .map((entry) => ({ entry, runningMinutes: elapsedMinutes(entry.clockedInAt, now) }))
    .sort((a, b) => b.runningMinutes - a.runningMinutes);
}

export function longestRunning<T extends Pick<OpenTimeEntryRow, "clockedInAt">>(
  entries: readonly T[],
  now: number,
): RunningEntry<T> | null {
  return rankRunning(entries, now)[0] ?? null;
}

const LONG_SHIFT_MINUTES = 12 * 60;

/** A punch that has run past a working day is more likely forgotten than worked. */
export function isOverlong(runningMinutes: number): boolean {
  return runningMinutes >= LONG_SHIFT_MINUTES;
}

export type DayEntry = {
  id: string;
  clockedInAt: number;
  clockedOutAt?: number | null;
  paidMinutes: number;
  breakMinutes: number;
};

export type EntryDay<T extends DayEntry = DayEntry> = {
  /** Calendar day in the reader's zone, as YYYY-MM-DD. */
  key: string;
  /** The first punch of the day, for a heading. */
  startsAt: number;
  entries: T[];
  paidMinutes: number;
  breakMinutes: number;
  /** Whether a punch on the day is still open. */
  running: boolean;
};

const DAY_KEY_FORMATS = new Map<string, Intl.DateTimeFormat>();

function dayKeyFormat(timezone: string): Intl.DateTimeFormat {
  let format = DAY_KEY_FORMATS.get(timezone);
  if (!format) {
    format = new Intl.DateTimeFormat("en-CA", {
      timeZone: timezone,
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    });
    DAY_KEY_FORMATS.set(timezone, format);
  }
  return format;
}

export function dayKeyFor(unixSeconds: number, timezone: string): string {
  return dayKeyFormat(timezone).format(new Date(unixSeconds * 1000));
}

/**
 * Punches by the calendar day they started on, newest day first and newest
 * punch first inside it. A punch is a wall-clock event to the person who made
 * it, so the day is the reader's, not UTC's.
 */
export function groupEntriesByDay<T extends DayEntry>(
  entries: readonly T[],
  timezone: string,
): EntryDay<T>[] {
  const days = new Map<string, EntryDay<T>>();
  for (const entry of entries) {
    const key = dayKeyFor(entry.clockedInAt, timezone);
    let day = days.get(key);
    if (!day) {
      day = {
        key,
        startsAt: entry.clockedInAt,
        entries: [],
        paidMinutes: 0,
        breakMinutes: 0,
        running: false,
      };
      days.set(key, day);
    }
    day.entries.push(entry);
    day.startsAt = Math.min(day.startsAt, entry.clockedInAt);
    day.paidMinutes += entry.paidMinutes;
    day.breakMinutes += entry.breakMinutes;
    if (entry.clockedOutAt == null) day.running = true;
  }
  const result = Array.from(days.values());
  for (const day of result) {
    day.entries.sort((a, b) => b.clockedInAt - a.clockedInAt);
  }
  return result.sort((a, b) => b.startsAt - a.startsAt);
}

const MINUTES_IN_DAY = 24 * 60;
const SECONDS_IN_DAY = 86_400;
const DAYS_IN_WEEK = 7;

const MINUTE_OF_DAY_FORMATS = new Map<string, Intl.DateTimeFormat>();

function minuteOfDayFormat(timezone: string): Intl.DateTimeFormat {
  let format = MINUTE_OF_DAY_FORMATS.get(timezone);
  if (!format) {
    format = new Intl.DateTimeFormat("en-US", {
      timeZone: timezone,
      hourCycle: "h23",
      hour: "2-digit",
      minute: "2-digit",
    });
    MINUTE_OF_DAY_FORMATS.set(timezone, format);
  }
  return format;
}

/** Minutes since midnight in the reader's zone, so a punch lands where the clock on the wall put it. */
export function minuteOfDay(unixSeconds: number, timezone: string): number {
  let hour = 0;
  let minute = 0;
  for (const part of minuteOfDayFormat(timezone).formatToParts(new Date(unixSeconds * 1000))) {
    if (part.type === "hour") hour = Number(part.value);
    else if (part.type === "minute") minute = Number(part.value);
  }
  return hour * 60 + minute;
}

export type TrackSpan = {
  id: string;
  /** Where on a 24-hour track the span starts and ends, from 0 to 1. */
  start: number;
  end: number;
  running: boolean;
};

/**
 * A day's punches laid on a 24-hour track. An open punch runs to now; a
 * punch that crosses midnight is cut at the end of the day it started on,
 * which is the day the grouping already put it in.
 */
export function dayTrackSpans(
  entries: readonly Pick<DayEntry, "id" | "clockedInAt" | "clockedOutAt">[],
  now: number,
  timezone: string,
): TrackSpan[] {
  return entries
    .map((entry) => {
      const start = minuteOfDay(entry.clockedInAt, timezone);
      const finish = entry.clockedOutAt ?? now;
      const length = Math.max(0, Math.floor((finish - entry.clockedInAt) / 60));
      const end = Math.min(MINUTES_IN_DAY, start + length);
      return {
        id: entry.id,
        start: start / MINUTES_IN_DAY,
        end: Math.max(end, start + 1) / MINUTES_IN_DAY,
        running: entry.clockedOutAt == null,
      };
    })
    .sort((a, b) => a.start - b.start);
}

export type WeekCardDay = {
  /** Midnight UTC of the rota day, the way the week itself is keyed. */
  startsAt: number;
  paidMinutes: number;
  breakMinutes: number;
  punches: number;
  running: boolean;
};

/**
 * A week's punches as the seven cells of a time card. The rota week is keyed
 * on UTC days, and so is the card, so its Monday is the week's Monday.
 */
export function weekCardDays(
  entries: readonly DayEntry[],
  periodStart: number,
  periodEnd: number = periodStart + DAYS_IN_WEEK * SECONDS_IN_DAY,
): WeekCardDay[] {
  const count = Math.max(1, Math.round((periodEnd - periodStart) / SECONDS_IN_DAY));
  const days: WeekCardDay[] = Array.from({ length: count }, (_, index) => ({
    startsAt: periodStart + index * SECONDS_IN_DAY,
    paidMinutes: 0,
    breakMinutes: 0,
    punches: 0,
    running: false,
  }));
  for (const entry of entries) {
    const index = Math.floor((entry.clockedInAt - periodStart) / SECONDS_IN_DAY);
    const day = days[index];
    if (!day) continue;
    day.paidMinutes += entry.paidMinutes;
    day.breakMinutes += entry.breakMinutes;
    day.punches += 1;
    if (entry.clockedOutAt == null) day.running = true;
  }
  return days;
}

/** Whole days a handed-over week has sat waiting; never negative. */
export function waitingDays(submittedAt: number, now: number): number {
  return Math.max(0, Math.floor((now - submittedAt) / SECONDS_IN_DAY));
}

export type OvertimeHeadroom = {
  /** Minutes left before the week tips into overtime. */
  remaining: number;
  /** Minutes already past the threshold. */
  over: number;
  /** How much of the threshold is used, from 0 to 1. */
  share: number;
};

export function overtimeHeadroom(totalMinutes: number, thresholdMinutes: number): OvertimeHeadroom {
  if (thresholdMinutes <= 0) return { remaining: 0, over: Math.max(0, totalMinutes), share: 1 };
  return {
    remaining: Math.max(0, thresholdMinutes - totalMinutes),
    over: Math.max(0, totalMinutes - thresholdMinutes),
    share: Math.min(1, Math.max(0, totalMinutes / thresholdMinutes)),
  };
}
