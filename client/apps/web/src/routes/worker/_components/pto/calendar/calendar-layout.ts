import { toDateFromUnixSeconds } from "@trenova/shared/lib/date";
import type { WorkerPTO } from "@trenova/shared/types/worker";

export const DAY_SECONDS = 86_400;
export const WEEK_DAYS = 7;

export type CalendarDay = {
  key: string;
  unix: number;
  date: number;
  month: number;
  year: number;
  inMonth: boolean;
  isToday: boolean;
  isWeekend: boolean;
};

export type CalendarWeek = {
  key: string;
  days: CalendarDay[];
};

export type CalendarSegment<T extends PTOSpan = PTOSpan> = {
  item: T;
  startCol: number;
  endCol: number;
  lane: number;
  continuesBefore: boolean;
  continuesAfter: boolean;
};

export type PTOSpan = Pick<WorkerPTO, "id" | "startDate" | "endDate">;

function localMidnight(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function unixOf(date: Date): number {
  return Math.floor(date.getTime() / 1000);
}

function dayKey(date: Date): string {
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${date.getFullYear()}-${month}-${day}`;
}

export function monthOf(unix: number): { year: number; month: number } {
  const date = toDateFromUnixSeconds(unix);
  return { year: date.getFullYear(), month: date.getMonth() };
}

export function monthStartUnix(year: number, month: number): number {
  return unixOf(new Date(year, month, 1));
}

export function monthEndUnix(year: number, month: number): number {
  return unixOf(new Date(year, month + 1, 0, 23, 59, 59));
}

export function shiftMonth(year: number, month: number, delta: number) {
  const date = new Date(year, month + delta, 1);
  return { year: date.getFullYear(), month: date.getMonth() };
}

export function buildMonthGrid(
  year: number,
  month: number,
  today: Date = new Date(),
): CalendarWeek[] {
  const first = new Date(year, month, 1);
  const gridStart = new Date(first);
  gridStart.setDate(first.getDate() - first.getDay());
  const todayKey = dayKey(localMidnight(today));

  const weeks: CalendarWeek[] = [];
  const cursor = new Date(gridStart);
  do {
    const days: CalendarDay[] = [];
    for (let i = 0; i < WEEK_DAYS; i += 1) {
      const date = new Date(cursor);
      const key = dayKey(date);
      days.push({
        key,
        unix: unixOf(date),
        date: date.getDate(),
        month: date.getMonth(),
        year: date.getFullYear(),
        inMonth: date.getMonth() === month && date.getFullYear() === year,
        isToday: key === todayKey,
        isWeekend: date.getDay() === 0 || date.getDay() === 6,
      });
      cursor.setDate(cursor.getDate() + 1);
    }
    weeks.push({ key: days[0].key, days });
  } while (cursor.getMonth() === month && cursor.getFullYear() === year);

  return weeks;
}

function dayIndex(unix: number): number {
  return Math.floor(localMidnight(toDateFromUnixSeconds(unix)).getTime() / DAY_SECONDS / 1000);
}

export function buildWeekSegments<T extends PTOSpan>(
  items: readonly T[],
  week: CalendarWeek,
): CalendarSegment<T>[] {
  const weekStartIdx = dayIndex(week.days[0].unix);
  const weekEndIdx = weekStartIdx + WEEK_DAYS - 1;

  const candidates: Omit<CalendarSegment<T>, "lane">[] = [];
  for (const item of items) {
    const startIdx = dayIndex(item.startDate);
    const endIdx = dayIndex(item.endDate);
    if (endIdx < weekStartIdx || startIdx > weekEndIdx) continue;
    candidates.push({
      item,
      startCol: Math.max(0, startIdx - weekStartIdx),
      endCol: Math.min(WEEK_DAYS - 1, endIdx - weekStartIdx),
      continuesBefore: startIdx < weekStartIdx,
      continuesAfter: endIdx > weekEndIdx,
    });
  }

  candidates.sort((a, b) => {
    if (a.startCol !== b.startCol) return a.startCol - b.startCol;
    const aLen = a.endCol - a.startCol;
    const bLen = b.endCol - b.startCol;
    if (aLen !== bLen) return bLen - aLen;
    return a.item.id?.localeCompare(b.item.id ?? "") ?? 0;
  });

  const laneEnds: number[] = [];
  const segments: CalendarSegment<T>[] = [];
  for (const candidate of candidates) {
    let lane = laneEnds.findIndex((end) => end < candidate.startCol);
    if (lane === -1) {
      lane = laneEnds.length;
      laneEnds.push(candidate.endCol);
    } else {
      laneEnds[lane] = candidate.endCol;
    }
    segments.push({ ...candidate, lane });
  }

  return segments;
}

export function laneCount(segments: readonly CalendarSegment[]): number {
  let max = 0;
  for (const segment of segments) {
    if (segment.lane + 1 > max) max = segment.lane + 1;
  }
  return max;
}

export function selectionRange(anchor: number, focus: number): { start: number; end: number } {
  return anchor <= focus ? { start: anchor, end: focus } : { start: focus, end: anchor };
}

export function isWithin(unix: number, range: { start: number; end: number } | null): boolean {
  return !!range && unix >= range.start && unix <= range.end;
}
