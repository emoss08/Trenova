import {
  buildDayColumnsForRange,
  buildHourTicksForRange,
  canvasWidthForRange,
  formatRangeLabelForDays,
  getBarGeometryForRange,
  getRangeForDays,
  secondsToXForRange,
  shiftAnchorByDays,
  type BarGeometry,
  type DayColumn,
  type HourTick,
  type TimeRange,
} from "@/lib/timeline/time-scale";
import type { TimelineZoom } from "../url-state";

export {
  isTodayAnchor,
  packLanes,
  parseAnchorDate,
  rangeDurationSeconds,
  serializeAnchorDate,
  type BarGeometry,
  type DayColumn,
  type HourTick,
  type LanePacking,
  type TimeRange,
} from "@/lib/timeline/time-scale";

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

export function getTimelineRange(anchor: Date, zoom: TimelineZoom): TimeRange {
  return getRangeForDays(anchor, ZOOM_CONFIGS[zoom].days);
}

export function shiftAnchor(anchor: Date, zoom: TimelineZoom, direction: 1 | -1): Date {
  return shiftAnchorByDays(anchor, ZOOM_CONFIGS[zoom].days, direction);
}

export function formatRangeLabel(anchor: Date, zoom: TimelineZoom): string {
  return formatRangeLabelForDays(anchor, ZOOM_CONFIGS[zoom].days);
}

export function canvasWidthPx(range: TimeRange, zoom: TimelineZoom): number {
  return canvasWidthForRange(range, ZOOM_CONFIGS[zoom].pxPerHour);
}

export function secondsToX(seconds: number, range: TimeRange, zoom: TimelineZoom): number {
  return secondsToXForRange(seconds, range, ZOOM_CONFIGS[zoom].pxPerHour);
}

export function buildDayColumns(range: TimeRange, zoom: TimelineZoom): DayColumn[] {
  return buildDayColumnsForRange(range, ZOOM_CONFIGS[zoom].pxPerHour);
}

export function buildHourTicks(range: TimeRange, zoom: TimelineZoom): HourTick[] {
  const config = ZOOM_CONFIGS[zoom];
  return buildHourTicksForRange(range, config.pxPerHour, config.hourTickStep);
}

export function getBarGeometry(
  start: number,
  end: number,
  range: TimeRange,
  zoom: TimelineZoom,
): BarGeometry {
  return getBarGeometryForRange(start, end, range, ZOOM_CONFIGS[zoom].pxPerHour);
}
