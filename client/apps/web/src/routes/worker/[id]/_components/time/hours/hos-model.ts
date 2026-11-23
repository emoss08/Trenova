import type { WorkerHosDailyLog, WorkerHosLogEntry } from "@/lib/graphql/telematics";
import { fromUserWallClock, toUserWallClock, userWallClockNow } from "@trenova/shared/lib/date";
import { dateTimeFormatter } from "@trenova/shared/i18n/format";
import { addDays, format, startOfDay, subDays } from "date-fns";
import { SECONDS_IN_DAY } from "@/lib/time-units";

export const DAILY_LOG_DAYS = 7;
export const DAY_KEY_FORMAT = "yyyy-MM-dd";

/**
 * How often the live reads (the clocks and today's log) are read again while the Hours view is
 * on screen. TanStack stops the interval while the browser tab is in the background, and the
 * view's observers unsubscribe while `Kept` hides it, so a hidden view reads nothing.
 */
export const HOS_LIVE_REFRESH_MS = 60_000;

/** How far back the violations and driver forms reach. */
export const HOS_RECORDS_DAYS = 30;

/** What the Hours view reads, as whole days in the reader's zone, so it moves only at midnight. */
export type HosWindow = {
  /** Midnight today on the reader's wall clock. */
  today: Date;
  /** The daily logs' first and last day keys (`yyyy-MM-dd`). */
  startDate: string;
  endDate: string;
  /** The records' window in unix seconds: midnight 30 days ago to midnight tonight. */
  since: number;
  until: number;
};

export function hosWindow(nowSec: number): HosWindow {
  const today = startOfDay(toUserWallClock(nowSec) ?? userWallClockNow());
  return {
    today,
    startDate: format(subDays(today, DAILY_LOG_DAYS - 1), DAY_KEY_FORMAT),
    endDate: format(today, DAY_KEY_FORMAT),
    since: fromUserWallClock(subDays(today, HOS_RECORDS_DAYS)) ?? 0,
    until: fromUserWallClock(addDays(today, 1)) ?? 0,
  };
}

/** A log day in full, as the daily log heads it: "Monday, Nov 23". */
export function formatLogDay(date: Date): string {
  return dateTimeFormatter({ weekday: "long", month: "short", day: "numeric" }).format(date);
}

export const LOG_LANE_CODES = ["OFF", "SB", "D", "ON"] as const;

export type LogLane = 0 | 1 | 2 | 3;

const LANE_BY_STATUS: Record<string, LogLane> = {
  offDuty: 0,
  personalConveyance: 0,
  sleeperBed: 1,
  driving: 2,
  onDuty: 3,
  yardMove: 3,
};

export function laneOfStatus(status: string): LogLane {
  return LANE_BY_STATUS[status] ?? 3;
}

export type LogSegment = {
  index: number;
  lane: LogLane;
  status: string;
  startSec: number;
  endSec: number;
  ongoing: boolean;
  vehicleName: string | null;
  remark: string | null;
};

/**
 * The day's duty segments clipped to the day, in time order. An open segment runs to now, or to
 * the end of the day when the day is over, so the line stops where the driver is.
 */
export function buildLogSegments(
  entries: readonly WorkerHosLogEntry[],
  dayStart: number,
  dayEnd: number,
  nowSec: number,
): LogSegment[] {
  const openEnd = Math.min(nowSec, dayEnd);
  const sorted = [...entries].sort((a, b) => a.logStartAt - b.logStartAt);
  const segments: LogSegment[] = [];
  for (const entry of sorted) {
    const startSec = Math.max(entry.logStartAt, dayStart);
    const endSec = Math.min(entry.logEndAt ?? openEnd, dayEnd);
    if (endSec <= startSec) continue;
    segments.push({
      index: segments.length,
      lane: laneOfStatus(entry.hosStatusType),
      status: entry.hosStatusType,
      startSec,
      endSec,
      ongoing: entry.logEndAt == null,
      vehicleName: entry.vehicleName ?? null,
      remark: entry.remark ?? null,
    });
  }
  return segments;
}

export type LaneTotals = readonly [number, number, number, number];

/** Milliseconds in each lane (off, sleeper, driving, on duty), from the segments the line draws. */
export function laneTotalsMs(segments: readonly LogSegment[]): LaneTotals {
  const totals: [number, number, number, number] = [0, 0, 0, 0];
  for (const segment of segments) {
    totals[segment.lane] += (segment.endSec - segment.startSec) * 1000;
  }
  return totals;
}

/** A day the provider summarised but sent no segments for still has lane totals. */
export function dailyLogLaneTotalsMs(dailyLog: WorkerHosDailyLog): LaneTotals {
  return [
    dailyLog.offDutyDurationMs + dailyLog.personalConveyanceDurationMs,
    dailyLog.sleeperBerthDurationMs,
    dailyLog.driveDurationMs,
    dailyLog.onDutyDurationMs + dailyLog.yardMoveDurationMs,
  ];
}

/**
 * The lane totals the grid and the summary strip both show: from the segments when there are any,
 * so the line, the totals and the strip always agree, and from the provider's summary otherwise.
 */
export function dayLaneTotals(
  segments: readonly LogSegment[],
  dailyLog: WorkerHosDailyLog | null,
): LaneTotals {
  if (segments.length > 0 || !dailyLog) return laneTotalsMs(segments);
  return dailyLogLaneTotalsMs(dailyLog);
}

export type LogGridGeometry = {
  /** Left edge of the hour columns. */
  left: number;
  /** Width of the 24 hour columns. */
  width: number;
  /** Top edge of the first lane. */
  top: number;
  laneHeight: number;
};

export function logX(sec: number, dayStart: number, dayEnd: number, geometry: LogGridGeometry) {
  const span = Math.max(dayEnd - dayStart, 1);
  const fraction = Math.min(1, Math.max(0, (sec - dayStart) / span));
  return geometry.left + fraction * geometry.width;
}

export function laneCenterY(lane: LogLane, geometry: LogGridGeometry): number {
  return geometry.top + lane * geometry.laneHeight + geometry.laneHeight / 2;
}

function fixed(value: number): string {
  return String(Math.round(value * 100) / 100);
}

/**
 * The stepped line through the day's duty statuses as an SVG path: across each segment's lane,
 * straight down or up at each change, and a fresh start after a gap with no segment.
 */
export function logPath(
  segments: readonly LogSegment[],
  dayStart: number,
  dayEnd: number,
  geometry: LogGridGeometry,
): string {
  const parts: string[] = [];
  let previous: LogSegment | null = null;
  for (const segment of segments) {
    const x1 = fixed(logX(segment.startSec, dayStart, dayEnd, geometry));
    const x2 = fixed(logX(segment.endSec, dayStart, dayEnd, geometry));
    const y = fixed(laneCenterY(segment.lane, geometry));
    if (previous === null || previous.endSec < segment.startSec) {
      parts.push(`M${x1} ${y}`);
    } else if (previous.lane !== segment.lane) {
      parts.push(`V${y}`);
    }
    parts.push(`H${x2}`);
    previous = segment;
  }
  return parts.join(" ");
}

/** The paper log's hour row: midnight, 1 to 11, noon, 1 to 11, midnight. */
export function logHourLabels(midnight: string, noon: string): string[] {
  return Array.from({ length: 25 }, (_, hour) =>
    hour === 0 || hour === 24 ? midnight : hour === 12 ? noon : String(hour % 12),
  );
}

/** A lane total as the paper log writes it: hours and zero-padded minutes ("4:48"). */
export function formatLaneTotal(ms: number): string {
  const minutes = Math.max(0, Math.floor(ms / 60_000));
  return `${Math.floor(minutes / 60)}:${String(minutes % 60).padStart(2, "0")}`;
}

export type HosDaySlot = {
  key: string;
  date: Date;
  isToday: boolean;
  startAt: number;
  endAt: number;
  dailyLog: WorkerHosDailyLog | null;
};

/** The last seven days in the reader's time zone, oldest first, each with its summary if any. */
export function buildDaySlots(
  dailyLogs: readonly WorkerHosDailyLog[],
  today: Date = startOfDay(userWallClockNow()),
): HosDaySlot[] {
  const byDay = new Map<string, WorkerHosDailyLog>();
  for (const log of dailyLogs) {
    const wall = toUserWallClock(log.startAt);
    if (wall) byDay.set(format(wall, DAY_KEY_FORMAT), log);
  }
  const slots: HosDaySlot[] = [];
  for (let offset = DAILY_LOG_DAYS - 1; offset >= 0; offset -= 1) {
    const date = subDays(today, offset);
    const key = format(date, DAY_KEY_FORMAT);
    const dailyLog = byDay.get(key) ?? null;
    const fallbackStart = fromUserWallClock(date) ?? 0;
    slots.push({
      key,
      date,
      isToday: offset === 0,
      startAt: dailyLog?.startAt ?? fallbackStart,
      endAt: dailyLog?.endAt ?? fallbackStart + SECONDS_IN_DAY,
      dailyLog,
    });
  }
  return slots;
}
