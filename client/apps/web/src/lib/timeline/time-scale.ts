import {
  fromUserWallClock,
  getTodayDate,
  toUserWallClock,
  userWallClockNow,
} from "@trenova/shared/lib/date";
import { addDays, addHours, format, isValid, parse, startOfDay } from "date-fns";

/**
 * Pure time-scale math shared by every horizontal timeline in the app (shipment
 * command center, dispatch console). Positions are computed from absolute epoch
 * seconds so day/hour boundaries stay correct across DST transitions — a "day"
 * column is however many real seconds that day contains, and pixel positions
 * follow.
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

export function formatRangeLabelForDays(anchor: Date, days: number): string {
  const start = startOfDay(anchor);
  if (days === 1) return format(start, "EEE, MMM d, yyyy");
  const end = addDays(start, days - 1);
  const sameMonth = start.getMonth() === end.getMonth();
  const startLabel = format(start, "MMM d");
  const endLabel = format(end, sameMonth ? "d, yyyy" : "MMM d, yyyy");
  return `${startLabel} – ${endLabel}`;
}

export function rangeDurationSeconds(range: TimeRange): number {
  return range.end - range.start;
}

export function canvasWidthForRange(range: TimeRange, pxPerHour: number): number {
  return Math.round((rangeDurationSeconds(range) / 3600) * pxPerHour);
}

export function secondsToXForRange(seconds: number, range: TimeRange, pxPerHour: number): number {
  const width = canvasWidthForRange(range, pxPerHour);
  const clamped = Math.min(Math.max(seconds, range.start), range.end);
  return ((clamped - range.start) / rangeDurationSeconds(range)) * width;
}

export type DayColumn = {
  start: number;
  end: number;
  x: number;
  width: number;
  label: string;
  isToday: boolean;
  isWeekend: boolean;
};

export function buildDayColumnsForRange(range: TimeRange, pxPerHour: number): DayColumn[] {
  const columns: DayColumn[] = [];
  const todayStart = getTodayDate();
  let cursor = startOfDay(toUserWallClock(range.start) ?? new Date(range.start * 1000));

  while ((fromUserWallClock(cursor) ?? 0) < range.end) {
    const next = addDays(cursor, 1);
    const start = fromUserWallClock(cursor) ?? 0;
    const end = Math.min(fromUserWallClock(next) ?? range.end, range.end);
    const x = secondsToXForRange(start, range, pxPerHour);
    const day = cursor.getDay();
    columns.push({
      start,
      end,
      x,
      width: secondsToXForRange(end, range, pxPerHour) - x,
      label: format(cursor, "EEE, MMM d"),
      isToday: start === todayStart,
      isWeekend: day === 0 || day === 6,
    });
    cursor = next;
  }

  return columns;
}

export type HourTick = {
  time: number;
  x: number;
  label: string;
};

export function buildHourTicksForRange(
  range: TimeRange,
  pxPerHour: number,
  hourTickStep: number,
): HourTick[] {
  if (hourTickStep === 0) return [];

  const ticks: HourTick[] = [];
  let cursor = startOfDay(toUserWallClock(range.start) ?? new Date(range.start * 1000));

  while ((fromUserWallClock(cursor) ?? 0) < range.end) {
    const dayEnd = addDays(cursor, 1);
    let hour = addHours(cursor, hourTickStep);
    while (hour < dayEnd) {
      const time = fromUserWallClock(hour) ?? 0;
      if (time >= range.end) break;
      ticks.push({
        time,
        x: secondsToXForRange(time, range, pxPerHour),
        label: format(hour, "HH:mm"),
      });
      hour = addHours(hour, hourTickStep);
    }
    cursor = dayEnd;
  }

  return ticks;
}

export type BarGeometry = {
  left: number;
  width: number;
  clippedStart: boolean;
  clippedEnd: boolean;
};

const MIN_BAR_WIDTH_PX = 14;

export function getBarGeometryForRange(
  start: number,
  end: number,
  range: TimeRange,
  pxPerHour: number,
): BarGeometry {
  const left = secondsToXForRange(start, range, pxPerHour);
  const right = secondsToXForRange(end, range, pxPerHour);
  return {
    left,
    width: Math.max(right - left, MIN_BAR_WIDTH_PX),
    clippedStart: start < range.start,
    clippedEnd: end > range.end,
  };
}

export type LanePacking = {
  laneByIndex: number[];
  laneCount: number;
};

/**
 * Greedy interval partitioning: assigns each span the lowest lane whose last
 * occupant ends before the span starts. Input spans must be sorted by start.
 */
export function packLanes(spans: readonly { start: number; end: number }[]): LanePacking {
  const laneEnds: number[] = [];
  const laneByIndex: number[] = [];

  for (const span of spans) {
    let lane = laneEnds.findIndex((laneEnd) => span.start >= laneEnd);
    if (lane === -1) {
      lane = laneEnds.length;
      laneEnds.push(span.end);
    } else {
      laneEnds[lane] = span.end;
    }
    laneByIndex.push(lane);
  }

  return { laneByIndex, laneCount: Math.max(laneEnds.length, 1) };
}
