import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import {
  addDecimalStrings,
  compareDecimalStrings,
  isDecimalString,
  multiplyDecimalStrings,
} from "@trenova/shared/types/decimal";

export type CarrierIntelUsageDayRow = {
  day: number;
  calls: number;
  billableUnits: number;
  estimatedCost: string;
};

export type SpendCapState = "uncapped" | "within" | "soft" | "exceeded";

export type SpendCapProgress = {
  state: SpendCapState;
  percent: number;
  softCapAmount: string | null;
};

export function utcDayKeyToUnix(dayKey: number): number {
  const year = Math.floor(dayKey / 10000);
  const month = Math.floor((dayKey % 10000) / 100);
  const day = dayKey % 100;
  return Math.floor(Date.UTC(year, month - 1, day) / 1000);
}

export function summarizeUsageByDay(
  rows: readonly {
    day: number;
    calls: number;
    billableUnits: number;
    estimatedCost: string;
  }[],
): CarrierIntelUsageDayRow[] {
  const byDay = new Map<number, { calls: number; billableUnits: number; costs: string[] }>();
  for (const row of rows) {
    const entry = byDay.get(row.day) ?? { calls: 0, billableUnits: 0, costs: [] };
    entry.calls += row.calls;
    entry.billableUnits += row.billableUnits;
    entry.costs.push(row.estimatedCost);
    byDay.set(row.day, entry);
  }

  return [...byDay.entries()]
    .map(([day, entry]) => ({
      day,
      calls: entry.calls,
      billableUnits: entry.billableUnits,
      estimatedCost: addDecimalStrings(entry.costs, 2),
    }))
    .sort((left, right) => right.day - left.day);
}

export function spendCapProgress(
  monthToDate: string,
  cap: string | null | undefined,
  softCapPercent: number,
): SpendCapProgress {
  if (!cap || !isDecimalString(cap) || compareDecimalStrings(cap, "0") <= 0) {
    return { state: "uncapped", percent: 0, softCapAmount: null };
  }

  const spent = isDecimalString(monthToDate) ? monthToDate : "0";
  const softCapAmount = multiplyDecimalStrings(cap, (softCapPercent / 100).toString(), 2);
  const percent = Math.min(100, Math.max(0, (Number(spent) / Number(cap)) * 100));

  let state: SpendCapState = "within";
  if (compareDecimalStrings(spent, cap) >= 0) {
    state = "exceeded";
  } else if (compareDecimalStrings(spent, softCapAmount) >= 0) {
    state = "soft";
  }

  return { state, percent, softCapAmount };
}

export function recentUsageMonths(nowSeconds: number, count: number): number[] {
  const now = new Date(nowSeconds * 1000);
  const year = now.getUTCFullYear();
  const month = now.getUTCMonth();
  const months = new Array<number>(Math.max(count, 0));
  for (let offset = 0; offset < months.length; offset++) {
    months[offset] = Math.floor(Date.UTC(year, month - offset, 1) / 1000);
  }
  return months;
}

export function formatUsageMonth(monthStart: number): string {
  return formatUnixInUserTimezone(monthStart, { month: "long", year: "numeric", timezone: "UTC" });
}

const SECONDS_PER_UTC_DAY = 86_400;

export function shiftUsageMonth(monthStart: number, delta: number): number {
  const date = new Date(monthStart * 1000);
  return Math.floor(Date.UTC(date.getUTCFullYear(), date.getUTCMonth() + delta, 1) / 1000);
}

export type CarrierIntelUsageDayPoint = {
  day: number;
  start: number;
  calls: number;
  billableUnits: number;
  cost: number;
};

function unixToUtcDayKey(unixSeconds: number): number {
  const date = new Date(unixSeconds * 1000);
  return date.getUTCFullYear() * 10000 + (date.getUTCMonth() + 1) * 100 + date.getUTCDate();
}

export function dailyUsageSeries(
  rows: readonly {
    day: number;
    calls: number;
    billableUnits: number;
    estimatedCost: string;
  }[],
  monthStart: number,
  nowSeconds: number,
): CarrierIntelUsageDayPoint[] {
  const totals = new Map(summarizeUsageByDay(rows).map((row) => [row.day, row]));
  const monthEnd = shiftUsageMonth(monthStart, 1);
  const lastDay = Math.min(monthEnd - SECONDS_PER_UTC_DAY, nowSeconds);
  const points: CarrierIntelUsageDayPoint[] = [];
  for (let start = monthStart; start <= lastDay; start += SECONDS_PER_UTC_DAY) {
    const day = unixToUtcDayKey(start);
    const total = totals.get(day);
    points.push({
      day,
      start,
      calls: total?.calls ?? 0,
      billableUnits: total?.billableUnits ?? 0,
      cost: total ? Number(total.estimatedCost) : 0,
    });
  }
  return points;
}
