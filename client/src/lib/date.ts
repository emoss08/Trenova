import { TimeFormat, type TimeFormatType } from "@/types/user";
import { parseDate } from "@/lib/chrono";
import { endOfDay, startOfDay, startOfMonth } from "date-fns";
import { fromZonedTime, toZonedTime } from "date-fns-tz";

type DateFormatOptions = {
  timeFormat?: TimeFormatType;
  showSeconds?: boolean;
  showTimeZone?: boolean;
  showDate?: boolean;
  showTime?: boolean;
};

function resolveTimezone(userTimezone: string): string {
  return userTimezone === "auto" ? Intl.DateTimeFormat().resolvedOptions().timeZone : userTimezone;
}

export function dateToUnixTimestamp(date: Date): number {
  if (!(date instanceof Date) || isNaN(date.getTime())) {
    throw new Error("Invalid date provided to dateToUnixTimestamp");
  }
  return Math.floor(date.getTime() / 1000);
}

export function getTodayDate(): number {
  const date = new Date();
  date.setUTCHours(0, 0, 0, 0);
  return dateToUnixTimestamp(date);
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

export function formatToUserTimezone(
  timestamp: number,
  options: DateFormatOptions = {},
  userTimezone: string = "auto",
): string {
  if (timestamp === undefined || timestamp === null || isNaN(timestamp)) {
    return "N/A";
  }

  const date = toDate(timestamp);
  if (!date) {
    return "N/A";
  }

  const timezone = resolveTimezone(userTimezone);
  const {
    timeFormat = TimeFormat.enum["24-hour"],
    showSeconds = false,
    showTimeZone = true,
    showDate = true,
    showTime = true,
  } = options;

  const formatOptions: Intl.DateTimeFormatOptions = {
    timeZone: timezone,
  };

  if (showTime) {
    formatOptions.hour = "2-digit";
    formatOptions.minute = "2-digit";
    formatOptions.hour12 = timeFormat === TimeFormat.enum["12-hour"];

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

  return new Intl.DateTimeFormat("en-US", formatOptions).format(date);
}

export function formatSplitDateTime(
  timestamp: number | undefined,
  timeFormat: TimeFormatType = TimeFormat.enum["12-hour"],
  userTimezone: string = "auto",
): {
  date: string;
  time: string;
} {
  if (timestamp === undefined || timestamp === null) return { date: "-", time: "" };

  const dateObj = toDate(timestamp);
  if (!dateObj) return { date: "-", time: "" };

  const timezone = resolveTimezone(userTimezone);

  const dateStr = new Intl.DateTimeFormat("en-US", {
    timeZone: timezone,
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(dateObj);

  const timeStr = new Intl.DateTimeFormat("en-US", {
    timeZone: timezone,
    hour: "2-digit",
    minute: "2-digit",
    hour12: timeFormat === TimeFormat.enum["12-hour"],
  }).format(dateObj);

  return {
    date: dateStr,
    time: timeStr,
  };
}

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

export function generateDateTimeString(date: Date, options: DateFormatOptions = {}): string {
  if (!(date instanceof Date) || isNaN(date.getTime())) {
    throw new Error("Invalid date provided to generateDateTimeString");
  }

  const { timeFormat = TimeFormat.enum["24-hour"], showSeconds = false } = options;

  const formatOptions: Intl.DateTimeFormatOptions = {
    month: "2-digit",
    day: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hour12: timeFormat === TimeFormat.enum["12-hour"],
  };

  if (showSeconds) {
    formatOptions.second = "2-digit";
  }

  return new Intl.DateTimeFormat("en-US", formatOptions).format(date);
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

  return generateDateTimeString(date, options);
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

const startOfLocalDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate());

export const inclusiveDays = (startUnix: number, endUnix: number) => {
  const s = startOfLocalDay(toDateFromUnixSeconds(startUnix));
  const e = startOfLocalDay(toDateFromUnixSeconds(endUnix));
  const MS_PER_DAY = 86_400_000;
  return Math.max(1, Math.floor((e.getTime() - s.getTime()) / MS_PER_DAY) + 1);
};

export const formatRange = (startUnix: number, endUnix: number) => {
  const s = toDateFromUnixSeconds(startUnix);
  const e = toDateFromUnixSeconds(endUnix);

  const sameYear = s.getFullYear() === e.getFullYear();
  const sameMonth = sameYear && s.getMonth() === e.getMonth();
  const sameDay = sameMonth && s.getDate() === e.getDate();
  const now = new Date();

  const showYear =
    !sameYear || s.getFullYear() !== now.getFullYear() || e.getFullYear() !== now.getFullYear();

  const sFmt = new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    ...(showYear ? { year: "numeric" } : {}),
  }).format(s);

  if (sameDay) return sFmt;

  const eFmt = new Intl.DateTimeFormat(undefined, {
    ...(sameMonth ? {} : { month: "short" }),
    day: "numeric",
    ...(showYear ? { year: "numeric" } : {}),
  }).format(e);

  return `${sFmt}–${eFmt}`;
};

export type DateRangePreset = {
  label: string;
  getValue: () => { startDate: number; endDate: number };
};

export function getStartOfDay(date: Date = new Date(), timezone?: string): number {
  if (!timezone) {
    return dateToUnixTimestamp(startOfDay(date));
  }

  const zonedDate = toZonedTime(date, timezone);
  const startOfDayInZone = startOfDay(zonedDate);
  const utcDate = fromZonedTime(startOfDayInZone, timezone);
  return dateToUnixTimestamp(utcDate);
}

export function getEndOfDay(date: Date = new Date(), timezone?: string): number {
  if (!timezone) {
    return dateToUnixTimestamp(endOfDay(date));
  }

  const zonedDate = toZonedTime(date, timezone);
  const endOfDayInZone = endOfDay(zonedDate);
  const utcDate = fromZonedTime(endOfDayInZone, timezone);
  return dateToUnixTimestamp(utcDate);
}

export function getStartOfMonth(date: Date = new Date(), timezone?: string): number {
  if (!timezone) {
    const start = startOfMonth(date);
    return dateToUnixTimestamp(start);
  }

  const year = date.getFullYear();
  const month = date.getMonth();

  const noonUTC = Date.UTC(year, month, 1, 12, 0, 0, 0);
  const noonDate = new Date(noonUTC);

  const dateInTimezone = toZonedTime(noonDate, timezone);

  const midnightLocal = new Date(
    dateInTimezone.getFullYear(),
    dateInTimezone.getMonth(),
    1,
    0,
    0,
    0,
    0,
  );

  const utcDate = fromZonedTime(midnightLocal, timezone);

  return dateToUnixTimestamp(utcDate);
}

export function getEndOfMonth(date: Date = new Date(), timezone?: string): number {
  const year = date.getFullYear();
  const month = date.getMonth();

  if (!timezone) {
    const lastDay = new Date(year, month + 1, 0);
    const endOfMonthDate = new Date(
      lastDay.getFullYear(),
      lastDay.getMonth(),
      lastDay.getDate(),
      23,
      59,
      59,
      999,
    );
    return dateToUnixTimestamp(endOfMonthDate);
  }

  const lastDayOfMonth = new Date(Date.UTC(year, month + 1, 0)).getUTCDate();

  const noonOnLastDay = new Date(Date.UTC(year, month, lastDayOfMonth, 12, 0, 0, 0));

  const lastDayInTimezone = toZonedTime(noonOnLastDay, timezone);

  const endOfDayUTC = Date.UTC(
    lastDayInTimezone.getFullYear(),
    lastDayInTimezone.getMonth(),
    lastDayInTimezone.getDate(),
    23,
    59,
    59,
    999,
  );
  const endOfDayDate = new Date(endOfDayUTC);

  const utcDate = fromZonedTime(endOfDayDate, timezone);

  return dateToUnixTimestamp(utcDate);
}

export function getStartOfYear(): number {
  return Math.floor(new Date(new Date().getFullYear(), 0, 1).getTime() / 1000);
}

export function getEndOfYear(): number {
  return Math.floor(new Date(new Date().getFullYear(), 11, 31, 23, 59, 59).getTime() / 1000);
}

function getStartOfQuarter(date: Date = new Date()): Date {
  const quarter = Math.floor(date.getMonth() / 3) * 3;
  return new Date(date.getFullYear(), quarter, 1);
}

function getEndOfQuarter(date: Date = new Date()): Date {
  const quarter = Math.floor(date.getMonth() / 3) * 3 + 2;
  return new Date(date.getFullYear(), quarter + 1, 0);
}

function addDays(date: Date, days: number): Date {
  const result = new Date(date);
  result.setDate(result.getDate() + days);
  return result;
}

export function getCommonDatePresets(timezone?: string): DateRangePreset[] {
  return [
    {
      label: "Today",
      getValue: () => {
        const today = new Date();
        return {
          startDate: getStartOfDay(today, timezone),
          endDate: getEndOfDay(today, timezone),
        };
      },
    },
    {
      label: "Next 7 days",
      getValue: () => {
        const today = new Date();
        const endDate = addDays(today, 6);
        return {
          startDate: getStartOfDay(today, timezone),
          endDate: getEndOfDay(endDate, timezone),
        };
      },
    },
    {
      label: "Next 30 days",
      getValue: () => {
        const today = new Date();
        const endDate = addDays(today, 29);
        return {
          startDate: getStartOfDay(today, timezone),
          endDate: getEndOfDay(endDate, timezone),
        };
      },
    },
    {
      label: "This month",
      getValue: () => {
        const today = new Date();
        return {
          startDate: getStartOfMonth(today, timezone),
          endDate: getEndOfMonth(today, timezone),
        };
      },
    },
    {
      label: "Next month",
      getValue: () => {
        const today = new Date();
        const firstDayNextMonth = new Date(today.getFullYear(), today.getMonth() + 1, 1);
        return {
          startDate: getStartOfMonth(firstDayNextMonth, timezone),
          endDate: getEndOfMonth(firstDayNextMonth, timezone),
        };
      },
    },
    {
      label: "This Quarter",
      getValue: () => {
        const today = new Date();
        return {
          startDate: getStartOfDay(getStartOfQuarter(today), timezone),
          endDate: getEndOfDay(getEndOfQuarter(today), timezone),
        };
      },
    },
  ];
}
