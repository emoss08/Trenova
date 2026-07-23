import { addDays, addHours, format, isValid, parse, startOfDay } from "date-fns";
import type { TimelineZoom } from "../url-state";

/**
 * Pure time-scale math for the dispatch timeline. Positions are computed from
 * absolute epoch seconds so day/hour boundaries stay correct across DST
 * transitions — a "day" column is however many real seconds that local day
 * contains, and pixel positions follow.
 */

export type TimeRange = {
  /** Inclusive start, unix seconds. */
  start: number;
  /** Exclusive end, unix seconds. */
  end: number;
};

export type ZoomConfig = {
  days: number;
  pxPerHour: number;
  /** Hour step between labeled ticks inside a day (0 = day labels only). */
  hourTickStep: number;
};

export const ZOOM_CONFIGS: Record<TimelineZoom, ZoomConfig> = {
  day: { days: 1, pxPerHour: 64, hourTickStep: 2 },
  "3day": { days: 3, pxPerHour: 26, hourTickStep: 6 },
  week: { days: 7, pxPerHour: 11, hourTickStep: 0 },
};

export const ZOOM_OPTIONS: readonly { id: TimelineZoom; label: string }[] = [
  { id: "day", label: "Day" },
  { id: "3day", label: "3 Days" },
  { id: "week", label: "Week" },
] as const;

const ANCHOR_DATE_FORMAT = "yyyy-MM-dd";

export function parseAnchorDate(value: string | null): Date {
  if (value) {
    const parsed = parse(value, ANCHOR_DATE_FORMAT, new Date());
    if (isValid(parsed)) return startOfDay(parsed);
  }
  return startOfDay(new Date());
}

export function serializeAnchorDate(date: Date): string {
  return format(date, ANCHOR_DATE_FORMAT);
}

export function isTodayAnchor(anchor: Date): boolean {
  return anchor.getTime() === startOfDay(new Date()).getTime();
}

export function getTimelineRange(anchor: Date, zoom: TimelineZoom): TimeRange {
  const start = startOfDay(anchor);
  const end = addDays(start, ZOOM_CONFIGS[zoom].days);
  return {
    start: Math.floor(start.getTime() / 1000),
    end: Math.floor(end.getTime() / 1000),
  };
}

export function shiftAnchor(anchor: Date, zoom: TimelineZoom, direction: 1 | -1): Date {
  return addDays(startOfDay(anchor), ZOOM_CONFIGS[zoom].days * direction);
}

export function formatRangeLabel(anchor: Date, zoom: TimelineZoom): string {
  const start = startOfDay(anchor);
  if (zoom === "day") return format(start, "EEE, MMM d, yyyy");
  const end = addDays(start, ZOOM_CONFIGS[zoom].days - 1);
  const sameMonth = start.getMonth() === end.getMonth();
  const startLabel = format(start, "MMM d");
  const endLabel = format(end, sameMonth ? "d, yyyy" : "MMM d, yyyy");
  return `${startLabel} – ${endLabel}`;
}

export function rangeDurationSeconds(range: TimeRange): number {
  return range.end - range.start;
}

export function canvasWidthPx(range: TimeRange, zoom: TimelineZoom): number {
  return Math.round((rangeDurationSeconds(range) / 3600) * ZOOM_CONFIGS[zoom].pxPerHour);
}

export function secondsToX(seconds: number, range: TimeRange, zoom: TimelineZoom): number {
  const width = canvasWidthPx(range, zoom);
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

export function buildDayColumns(range: TimeRange, zoom: TimelineZoom): DayColumn[] {
  const columns: DayColumn[] = [];
  const todayStart = Math.floor(startOfDay(new Date()).getTime() / 1000);
  let cursor = new Date(range.start * 1000);

  while (Math.floor(cursor.getTime() / 1000) < range.end) {
    const next = addDays(cursor, 1);
    const start = Math.floor(cursor.getTime() / 1000);
    const end = Math.min(Math.floor(next.getTime() / 1000), range.end);
    const x = secondsToX(start, range, zoom);
    const day = cursor.getDay();
    columns.push({
      start,
      end,
      x,
      width: secondsToX(end, range, zoom) - x,
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

export function buildHourTicks(range: TimeRange, zoom: TimelineZoom): HourTick[] {
  const step = ZOOM_CONFIGS[zoom].hourTickStep;
  if (step === 0) return [];

  const ticks: HourTick[] = [];
  let cursor = new Date(range.start * 1000);

  while (Math.floor(cursor.getTime() / 1000) < range.end) {
    const dayEnd = addDays(cursor, 1);
    let hour = addHours(cursor, step);
    while (hour < dayEnd) {
      const time = Math.floor(hour.getTime() / 1000);
      if (time >= range.end) break;
      ticks.push({
        time,
        x: secondsToX(time, range, zoom),
        label: format(hour, "HH:mm"),
      });
      hour = addHours(hour, step);
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

export function getBarGeometry(
  start: number,
  end: number,
  range: TimeRange,
  zoom: TimelineZoom,
): BarGeometry {
  const left = secondsToX(start, range, zoom);
  const right = secondsToX(end, range, zoom);
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
