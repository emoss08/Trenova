import { TimeFormat, type TimeFormatType } from "@trenova/shared/types/user";
import { parseDate } from "@trenova/shared/lib/chrono";
import { endOfDay, startOfDay, startOfMonth } from "date-fns";
import { fromZonedTime, toZonedTime } from "date-fns-tz";

type DateFormatOptions = {
  timeFormat?: TimeFormatType;
  showSeconds?: boolean;
  showTimeZone?: boolean;
  showDate?: boolean;
  showTime?: boolean;
};

export type UserDatePreferences = {
  timezone: string;
  timeFormat: TimeFormatType;
};

const AUTO_TIMEZONE = "auto";

const FALLBACK_PREFERENCES: UserDatePreferences = {
  timezone: AUTO_TIMEZONE,
  timeFormat: TimeFormat.enum["12-hour"],
};

let userDatePreferences: UserDatePreferences = FALLBACK_PREFERENCES;

/**
 * Every timestamp crossing the wire is UTC epoch seconds, so the only place a
 * wall clock exists is at render time. The signed-in user's preferences are
 * mirrored here out of the auth store, which lets every formatter below render
 * in their timezone without each call site threading it through — and makes
 * "the client is timezone aware" a property of this module rather than a rule
 * a few hundred call sites have to remember.
 */
export function setUserDatePreferences(preferences: Partial<UserDatePreferences>): void {
  userDatePreferences = {
    timezone: preferences.timezone || FALLBACK_PREFERENCES.timezone,
    timeFormat: preferences.timeFormat || FALLBACK_PREFERENCES.timeFormat,
  };
}

export function getUserDatePreferences(): UserDatePreferences {
  return userDatePreferences;
}

export function resolveUserTimezone(userTimezone?: string): string {
  const timezone = userTimezone || userDatePreferences.timezone;
  return timezone === AUTO_TIMEZONE ? Intl.DateTimeFormat().resolvedOptions().timeZone : timezone;
}

export function resolveUserTimeFormat(timeFormat?: TimeFormatType): TimeFormatType {
  return timeFormat || userDatePreferences.timeFormat;
}

function isHour12(timeFormat?: TimeFormatType): boolean {
  return resolveUserTimeFormat(timeFormat) === TimeFormat.enum["12-hour"];
}

export type ZonedFormatOptions = Intl.DateTimeFormatOptions & {
  timezone?: string;
  timeFormat?: TimeFormatType;
};

function zonedFormatter({ timezone, timeFormat, ...options }: ZonedFormatOptions) {
  const showsTime =
    options.hour !== undefined ||
    options.minute !== undefined ||
    options.second !== undefined ||
    options.timeStyle !== undefined;

  return new Intl.DateTimeFormat("en-US", {
    timeZone: resolveUserTimezone(timezone),
    ...(showsTime && options.hour12 === undefined ? { hour12: isHour12(timeFormat) } : {}),
    ...options,
  });
}

export function dateToUnixTimestamp(date: Date): number {
  if (!(date instanceof Date) || isNaN(date.getTime())) {
    throw new Error("Invalid date provided to dateToUnixTimestamp");
  }
  return Math.floor(date.getTime() / 1000);
}

export const toDate = (unixTimeStamp: number | undefined): Date | undefined => {
  if (unixTimeStamp === undefined || unixTimeStamp === null || isNaN(unixTimeStamp)) {
    return undefined;
  }
  const date = new Date(unixTimeStamp * 1000);
  return isNaN(date.getTime()) ? undefined : date;
};

export const toDateFromUnixSeconds = (unixSeconds: number) => new Date(unixSeconds * 1000);

export const toUnixTimeStamp = (date: Date | undefined): number | undefined => {
  if (!date || !(date instanceof Date) || isNaN(date.getTime())) {
    return undefined;
  }
  return Math.floor(date.getTime() / 1000);
};

/**
 * The instant a calendar day starts in the user's timezone. Everything that
 * needs "which day is this" — bucketing, spans, countdowns — goes through here
 * so a shipment that arrived at 11pm Denver time is not filed under tomorrow.
 */
function startOfUserDay(date: Date, timezone?: string): Date {
  const zoned = toZonedTime(date, resolveUserTimezone(timezone));
  return new Date(zoned.getFullYear(), zoned.getMonth(), zoned.getDate());
}

export function getTodayDate(timezone?: string): number {
  return getStartOfDay(new Date(), timezone);
}

/**
 * Projects an instant into the user's timezone as a wall-clock Date: the same
 * calendar fields someone standing in that zone reads off a clock. Calendars
 * and time inputs edit wall clocks, so a field bound to a stored timestamp
 * comes through here and goes back out through {@link fromUserWallClock}.
 */
export function toUserWallClock(
  unixSeconds: number | null | undefined,
  timezone?: string,
): Date | undefined {
  const date = toDate(unixSeconds ?? undefined);
  return date ? toZonedTime(date, resolveUserTimezone(timezone)) : undefined;
}

export function userWallClockNow(timezone?: string): Date {
  return toZonedTime(new Date(), resolveUserTimezone(timezone));
}

export function fromUserWallClock(
  date: Date | null | undefined,
  timezone?: string,
): number | undefined {
  if (!(date instanceof Date) || isNaN(date.getTime())) {
    return undefined;
  }
  return dateToUnixTimestamp(fromZonedTime(date, resolveUserTimezone(timezone)));
}

export function formatDateInUserTimezone(
  date: Date | undefined | null,
  options: ZonedFormatOptions,
  fallback = "N/A",
): string {
  if (!(date instanceof Date) || isNaN(date.getTime())) {
    return fallback;
  }
  return zonedFormatter(options).format(date);
}

export function formatUnixInUserTimezone(
  value: number | null | undefined,
  options: ZonedFormatOptions,
  fallback = "N/A",
): string {
  const date = toDate(value ?? undefined);
  return date ? zonedFormatter(options).format(date) : fallback;
}

export function formatToUserTimezone(
  timestamp: number,
  options: DateFormatOptions = {},
  userTimezone?: string,
): string {
  if (timestamp === undefined || timestamp === null || isNaN(timestamp)) {
    return "N/A";
  }

  const date = toDate(timestamp);
  if (!date) {
    return "N/A";
  }

  const {
    timeFormat,
    showSeconds = false,
    showTimeZone = true,
    showDate = true,
    showTime = true,
  } = options;

  const formatOptions: ZonedFormatOptions = {
    timezone: userTimezone,
    timeFormat,
  };

  if (showTime) {
    formatOptions.hour = "2-digit";
    formatOptions.minute = "2-digit";

    if (showSeconds) {
      formatOptions.second = "2-digit";
    }

    if (showTimeZone) {
      formatOptions.timeZoneName = "short";
    }
  }

  if (showDate) {
    formatOptions.year = "numeric";
    formatOptions.month = "2-digit";
    formatOptions.day = "2-digit";
  }

  return zonedFormatter(formatOptions).format(date);
}

export function formatCurrentUserTime(
  date: Date = new Date(),
  timeFormat?: TimeFormatType,
  userTimezone?: string,
): string {
  return formatDateInUserTimezone(date, {
    timezone: userTimezone,
    timeFormat,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    timeZoneName: "shortGeneric",
  });
}

export function formatSplitDateTime(
  timestamp: number | undefined,
  timeFormat?: TimeFormatType,
  userTimezone?: string,
): {
  date: string;
  time: string;
} {
  if (timestamp === undefined || timestamp === null) return { date: "-", time: "" };

  const dateObj = toDate(timestamp);
  if (!dateObj) return { date: "-", time: "" };

  return {
    date: formatDateInUserTimezone(dateObj, {
      timezone: userTimezone,
      day: "numeric",
      month: "short",
      year: "numeric",
    }),
    time: formatDateInUserTimezone(dateObj, {
      timezone: userTimezone,
      timeFormat,
      hour: "2-digit",
      minute: "2-digit",
    }),
  };
}

/**
 * Formats a calendar date the user picked. Local getters are deliberate here:
 * the value is a day on a calendar, not an instant, so re-projecting it into
 * another zone would shift it across the date line.
 */
export function generateDateOnlyString(date: Date): string {
  if (!(date instanceof Date) || isNaN(date.getTime())) {
    throw new Error("Invalid date provided to generateDateOnlyString");
  }
  return new Intl.DateTimeFormat("en-US", {
    month: "2-digit",
    day: "2-digit",
    year: "numeric",
  }).format(date);
}

export function generateDateOnly(date: string): Date | null {
  if (!date || typeof date !== "string") {
    return null;
  }

  const parsed = parseDate(date);
  if (parsed && !isNaN(parsed.getTime())) {
    const normalized = new Date(parsed);
    normalized.setHours(0, 0, 0, 0);
    return normalized;
  }
  return null;
}

const ISO_DATE_PATTERN = /^(\d{4})-(\d{2})-(\d{2})$/;

/**
 * Formats a local calendar date as `YYYY-MM-DD`. Local getters are used
 * deliberately: the value represents the day a user picked, not an instant, so
 * a UTC conversion would shift it across the date line for western timezones.
 */
export function toISODateString(date: Date): string {
  if (!(date instanceof Date) || isNaN(date.getTime())) {
    throw new Error("Invalid date provided to toISODateString");
  }
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${date.getFullYear()}-${month}-${day}`;
}

/**
 * Parses a `YYYY-MM-DD` calendar date into a local midnight Date, rejecting
 * anything that is not a real calendar day (e.g. `2026-02-30`).
 */
export function fromISODateString(value: string): Date | null {
  if (!value || typeof value !== "string") {
    return null;
  }
  const match = ISO_DATE_PATTERN.exec(value);
  if (!match) {
    return null;
  }
  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  const date = new Date(year, month - 1, day);
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) {
    return null;
  }
  return date;
}

export function isValidDateOnlyFormat(dateString: string): boolean {
  if (!dateString || typeof dateString !== "string") {
    return false;
  }
  try {
    const date = new Date(dateString);
    return !isNaN(date.getTime()) && generateDateOnlyString(date) === dateString;
  } catch {
    return false;
  }
}

/**
 * Formats a wall-clock date — one already expressed in the user's timezone by
 * {@link toUserWallClock} or typed into a picker — so no zone projection is
 * applied. Only the 12/24-hour preference is honoured.
 */
export function generateDateTimeString(date: Date, options: DateFormatOptions = {}): string {
  if (!(date instanceof Date) || isNaN(date.getTime())) {
    throw new Error("Invalid date provided to generateDateTimeString");
  }

  const { timeFormat, showSeconds = false } = options;

  return new Intl.DateTimeFormat("en-US", {
    month: "2-digit",
    day: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hour12: isHour12(timeFormat),
    ...(showSeconds ? { second: "2-digit" } : {}),
  }).format(date);
}

export function generateDateTimeStringFromUnixTimestamp(
  timestamp?: number,
  options: DateFormatOptions = {},
): string {
  if (timestamp === undefined || timestamp === null || isNaN(timestamp)) {
    return "N/A";
  }

  const date = toDate(timestamp);
  if (!date) {
    return "-";
  }

  const { timeFormat, showSeconds = false } = options;

  return formatDateInUserTimezone(date, {
    timeFormat,
    month: "2-digit",
    day: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    ...(showSeconds ? { second: "2-digit" } : {}),
  });
}

export function generateDateTime(date: string): Date | null {
  if (!date || typeof date !== "string") {
    return null;
  }

  const parsed = parseDate(date);
  return parsed && !isNaN(parsed.getTime()) ? parsed : null;
}

const dateTimeFormatRegex =
  /^(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s([1-9]|[12]\d|3[01])(st|nd|rd|th)\s\d{4},\s(0\d|1[0-2]):([0-5]\d)\s(AM|PM)$/;

export function isValidDateTimeFormat(dateString: string): boolean {
  if (!dateString || typeof dateString !== "string") {
    return false;
  }
  return dateTimeFormatRegex.test(dateString);
}

export function formatDurationFromSeconds(durationInSeconds: number): string {
  if (
    durationInSeconds === undefined ||
    durationInSeconds === null ||
    isNaN(durationInSeconds) ||
    durationInSeconds < 0
  ) {
    return "0m";
  }

  if (durationInSeconds === 0) {
    return "0m";
  }

  const days = Math.floor(durationInSeconds / 86400);
  const hours = Math.floor((durationInSeconds % 86400) / 3600);
  const minutes = Math.floor(((durationInSeconds % 86400) % 3600) / 60);

  const parts: string[] = [];

  if (days > 0) {
    parts.push(`${days}d`);
  }
  if (hours > 0) {
    parts.push(`${hours}h`);
  }
  if (minutes > 0) {
    parts.push(`${minutes}m`);
  }

  if (parts.length === 0) {
    return "< 1m";
  }

  return parts.join(" ");
}

/**
 * How long ago something happened, at the resolution a person cares about: a
 * few seconds reads as "just now", minutes and hours are rounded down.
 */
export function formatSecondsAgo(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 5) {
    return "just now";
  }

  if (seconds < 60) {
    return `${Math.floor(seconds)}s ago`;
  }

  if (seconds < 3600) {
    return `${Math.floor(seconds / 60)}m ago`;
  }

  return `${Math.floor(seconds / 3600)}h ago`;
}

export function formatDurationMs(durationInMs: number): string {
  if (durationInMs === undefined || durationInMs === null || isNaN(durationInMs)) {
    return "0m";
  }
  return formatDurationFromSeconds(Math.floor(durationInMs / 1000));
}

export function formatClockDurationMs(durationInMs: number): string {
  if (durationInMs === undefined || durationInMs === null || isNaN(durationInMs)) {
    return "0:00";
  }
  const totalMinutes = Math.max(0, Math.floor(durationInMs / 60_000));
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  return `${hours}:${String(minutes).padStart(2, "0")}`;
}

const MS_PER_DAY = 86_400_000;

export const daysUntil = (unixSeconds: number, timezone?: string) => {
  const target = startOfUserDay(toDateFromUnixSeconds(unixSeconds), timezone);
  const today = startOfUserDay(new Date(), timezone);
  return Math.round((target.getTime() - today.getTime()) / MS_PER_DAY);
};

export const inclusiveDays = (startUnix: number, endUnix: number, timezone?: string) => {
  const s = startOfUserDay(toDateFromUnixSeconds(startUnix), timezone);
  const e = startOfUserDay(toDateFromUnixSeconds(endUnix), timezone);
  return Math.max(1, Math.floor((e.getTime() - s.getTime()) / MS_PER_DAY) + 1);
};

export const formatRange = (startUnix: number, endUnix: number, timezone?: string) => {
  const resolved = resolveUserTimezone(timezone);
  const s = toDateFromUnixSeconds(startUnix);
  const e = toDateFromUnixSeconds(endUnix);

  const zonedStart = toZonedTime(s, resolved);
  const zonedEnd = toZonedTime(e, resolved);
  const zonedNow = toZonedTime(new Date(), resolved);

  const sameYear = zonedStart.getFullYear() === zonedEnd.getFullYear();
  const sameMonth = sameYear && zonedStart.getMonth() === zonedEnd.getMonth();
  const sameDay = sameMonth && zonedStart.getDate() === zonedEnd.getDate();

  const showYear =
    !sameYear ||
    zonedStart.getFullYear() !== zonedNow.getFullYear() ||
    zonedEnd.getFullYear() !== zonedNow.getFullYear();

  const sFmt = formatDateInUserTimezone(s, {
    timezone: resolved,
    month: "short",
    day: "numeric",
    ...(showYear ? { year: "numeric" } : {}),
  });

  if (sameDay) return sFmt;

  const eFmt = formatDateInUserTimezone(e, {
    timezone: resolved,
    ...(sameMonth ? {} : { month: "short" }),
    day: "numeric",
    ...(showYear ? { year: "numeric" } : {}),
  });

  return `${sFmt}–${eFmt}`;
};

export type DateRangePreset = {
  label: string;
  getValue: () => { startDate: number; endDate: number };
};

const startOfWallClockDay = (wallClock: Date, timezone: string) =>
  dateToUnixTimestamp(fromZonedTime(startOfDay(wallClock), timezone));

const endOfWallClockDay = (wallClock: Date, timezone: string) =>
  dateToUnixTimestamp(fromZonedTime(endOfDay(wallClock), timezone));

export function getStartOfDay(date: Date = new Date(), timezone?: string): number {
  const resolved = resolveUserTimezone(timezone);
  return startOfWallClockDay(toZonedTime(date, resolved), resolved);
}

export function getEndOfDay(date: Date = new Date(), timezone?: string): number {
  const resolved = resolveUserTimezone(timezone);
  return endOfWallClockDay(toZonedTime(date, resolved), resolved);
}

export function getStartOfMonth(date: Date = new Date(), timezone?: string): number {
  const resolved = resolveUserTimezone(timezone);
  return dateToUnixTimestamp(fromZonedTime(startOfMonth(toZonedTime(date, resolved)), resolved));
}

export function getEndOfMonth(date: Date = new Date(), timezone?: string): number {
  const resolved = resolveUserTimezone(timezone);
  const wallClock = toZonedTime(date, resolved);
  const lastDay = new Date(wallClock.getFullYear(), wallClock.getMonth() + 1, 0);
  return endOfWallClockDay(lastDay, resolved);
}

export function getStartOfYear(timezone?: string): number {
  const resolved = resolveUserTimezone(timezone);
  const zonedNow = toZonedTime(new Date(), resolved);
  return dateToUnixTimestamp(
    fromZonedTime(new Date(zonedNow.getFullYear(), 0, 1, 0, 0, 0, 0), resolved),
  );
}

export function getEndOfYear(timezone?: string): number {
  const resolved = resolveUserTimezone(timezone);
  const zonedNow = toZonedTime(new Date(), resolved);
  return dateToUnixTimestamp(
    fromZonedTime(new Date(zonedNow.getFullYear(), 11, 31, 23, 59, 59, 0), resolved),
  );
}

function getStartOfQuarter(date: Date): Date {
  const quarter = Math.floor(date.getMonth() / 3) * 3;
  return new Date(date.getFullYear(), quarter, 1);
}

function getEndOfQuarter(date: Date): Date {
  const quarter = Math.floor(date.getMonth() / 3) * 3 + 2;
  return new Date(date.getFullYear(), quarter + 1, 0);
}

function addDays(date: Date, days: number): Date {
  const result = new Date(date);
  result.setDate(result.getDate() + days);
  return result;
}

const NUMERIC_DATE: Intl.DateTimeFormatOptions = {
  month: "numeric",
  day: "numeric",
  year: "numeric",
};

const MEDIUM_DATE: Intl.DateTimeFormatOptions = {
  month: "short",
  day: "numeric",
  year: "numeric",
};

const MONTH_DAY: Intl.DateTimeFormatOptions = { month: "short", day: "numeric" };

const CLOCK_TIME: Intl.DateTimeFormatOptions = { hour: "numeric", minute: "2-digit" };

const CLOCK_TIME_WITH_SECONDS: Intl.DateTimeFormatOptions = {
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
};

/**
 * Display options every unix formatter accepts. `fallback` is what an unset
 * timestamp reads as — the reason a screen never has to hand-roll its own
 * `if (!value) return "—"` wrapper around a formatter.
 */
export type UnixFormatOptions = {
  timezone?: string;
  timeFormat?: TimeFormatType;
  fallback?: string;
};

function formatUnixShape(
  value: number | null | undefined,
  shape: Intl.DateTimeFormatOptions,
  { timezone, timeFormat, fallback = "N/A" }: UnixFormatOptions = {},
): string {
  if (!value) {
    return fallback;
  }
  return formatUnixInUserTimezone(value, { ...shape, timezone, timeFormat }, fallback);
}

export function formatUnixDate(value: number | null | undefined, options?: UnixFormatOptions) {
  return formatUnixShape(value, NUMERIC_DATE, options);
}

export function formatUnixDateMedium(
  value: number | null | undefined,
  options?: UnixFormatOptions,
) {
  return formatUnixShape(value, MEDIUM_DATE, options);
}

export function formatUnixMonthDay(value: number | null | undefined, options?: UnixFormatOptions) {
  return formatUnixShape(value, MONTH_DAY, options);
}

export function formatUnixWeekday(value: number | null | undefined, options?: UnixFormatOptions) {
  return formatUnixShape(value, { weekday: "long" }, options);
}

export function formatUnixTime(value: number | null | undefined, options?: UnixFormatOptions) {
  return formatUnixShape(value, CLOCK_TIME, options);
}

export function formatUnixTimeWithSeconds(
  value: number | null | undefined,
  options?: UnixFormatOptions,
) {
  return formatUnixShape(value, CLOCK_TIME_WITH_SECONDS, options);
}

export function formatUnixDateTime(value: number | null | undefined, options?: UnixFormatOptions) {
  return formatUnixShape(value, { ...NUMERIC_DATE, ...CLOCK_TIME }, options);
}

export function formatUnixDateTimeMedium(
  value: number | null | undefined,
  options?: UnixFormatOptions,
) {
  return formatUnixShape(value, { ...MEDIUM_DATE, ...CLOCK_TIME }, options);
}

export function formatUnixDateTimeShort(
  value: number | null | undefined,
  options?: UnixFormatOptions,
) {
  return formatUnixShape(value, { ...MONTH_DAY, ...CLOCK_TIME }, options);
}

export function formatUnixDateTimeOrDash(
  value: number | null | undefined,
  options?: UnixFormatOptions,
) {
  return formatUnixDateTime(value, { fallback: "-", ...options });
}

export function formatUnixDateOrDash(
  value: number | null | undefined,
  options?: UnixFormatOptions,
) {
  return formatUnixDateMedium(value, { fallback: "-", ...options });
}

/**
 * The settlement surfaces (driver and carrier statements, batches, cost
 * events, invoice matching) all render dates as a medium date with an em-dash
 * fallback for unset timestamps — one formatter so they cannot drift apart.
 */
export function formatSettlementDate(
  value: number | null | undefined,
  options?: UnixFormatOptions,
) {
  return formatUnixDateMedium(value, { fallback: "—", ...options });
}

/** Compact month/day variant of {@link formatSettlementDate} for dense rails and lists. */
export function formatSettlementMonthDay(
  value: number | null | undefined,
  options?: UnixFormatOptions,
) {
  return formatUnixMonthDay(value, { fallback: "—", ...options });
}

/**
 * Presets are anchored to *today in the user's timezone*, so "Today" means the
 * day they are living in rather than the day the browser happens to be in.
 * Every boundary below is computed on wall-clock dates and converted back to an
 * instant exactly once.
 */
export function getCommonDatePresets(timezone?: string): DateRangePreset[] {
  const resolved = resolveUserTimezone(timezone);
  const today = () => toZonedTime(new Date(), resolved);

  return [
    {
      label: "Today",
      getValue: () => {
        const now = today();
        return {
          startDate: startOfWallClockDay(now, resolved),
          endDate: endOfWallClockDay(now, resolved),
        };
      },
    },
    {
      label: "Next 7 days",
      getValue: () => {
        const now = today();
        return {
          startDate: startOfWallClockDay(now, resolved),
          endDate: endOfWallClockDay(addDays(now, 6), resolved),
        };
      },
    },
    {
      label: "Next 30 days",
      getValue: () => {
        const now = today();
        return {
          startDate: startOfWallClockDay(now, resolved),
          endDate: endOfWallClockDay(addDays(now, 29), resolved),
        };
      },
    },
    {
      label: "This month",
      getValue: () => {
        const now = today();
        return {
          startDate: startOfWallClockDay(startOfMonth(now), resolved),
          endDate: endOfWallClockDay(new Date(now.getFullYear(), now.getMonth() + 1, 0), resolved),
        };
      },
    },
    {
      label: "Next month",
      getValue: () => {
        const now = today();
        const firstDayNextMonth = new Date(now.getFullYear(), now.getMonth() + 1, 1);
        return {
          startDate: startOfWallClockDay(firstDayNextMonth, resolved),
          endDate: endOfWallClockDay(new Date(now.getFullYear(), now.getMonth() + 2, 0), resolved),
        };
      },
    },
    {
      label: "This Quarter",
      getValue: () => {
        const now = today();
        return {
          startDate: startOfWallClockDay(getStartOfQuarter(now), resolved),
          endDate: endOfWallClockDay(getEndOfQuarter(now), resolved),
        };
      },
    },
  ];
}
