import { fromUserWallClock, toUserWallClock, userWallClockNow } from "@trenova/shared/lib/date";

/** The snoozes offered without picking a time: a few hours, the next morning, next week. */
export type SnoozePresetKey = "later" | "tomorrow" | "nextWeek";

export type SnoozePreset = { key: SnoozePresetKey; until: number };

const LATER_HOURS = 3;
const MORNING_HOUR = 8;
const MONDAY = 1;
const DAYS_IN_WEEK = 7;

function morningOf(wallClock: Date, timezone: string): number {
  const morning = new Date(wallClock);
  morning.setHours(MORNING_HOUR, 0, 0, 0);
  return fromUserWallClock(morning, timezone) ?? 0;
}

/**
 * When each preset ends, on the person's own calendar: three hours from now,
 * eight the next morning, and eight on the coming Monday.
 */
export function snoozePresets(nowSeconds: number, timezone: string): SnoozePreset[] {
  const wallClock = toUserWallClock(nowSeconds, timezone) ?? userWallClockNow(timezone);

  const tomorrow = new Date(wallClock);
  tomorrow.setDate(tomorrow.getDate() + 1);

  const monday = new Date(wallClock);
  const ahead = (MONDAY - monday.getDay() + DAYS_IN_WEEK) % DAYS_IN_WEEK || DAYS_IN_WEEK;
  monday.setDate(monday.getDate() + ahead);

  return [
    { key: "later", until: nowSeconds + LATER_HOURS * 3600 },
    { key: "tomorrow", until: morningOf(tomorrow, timezone) },
    { key: "nextWeek", until: morningOf(monday, timezone) },
  ];
}

const LOCAL_DATE_TIME = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/u;

/**
 * A `datetime-local` value read as the person's own wall clock, which may be
 * in a different zone from the browser's. Null for anything else.
 */
export function wallClockInputToUnix(value: string, timezone: string): number | null {
  const match = LOCAL_DATE_TIME.exec(value);
  if (!match) {
    return null;
  }
  const [, year, month, day, hour, minute] = match;
  const wallClock = new Date(
    Number(year),
    Number(month) - 1,
    Number(day),
    Number(hour),
    Number(minute),
  );

  return fromUserWallClock(wallClock, timezone) ?? null;
}
