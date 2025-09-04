/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { TimeFormat } from "@/types/user";
import * as chrono from "chrono-node";
import {
  endOfDay,
  endOfMonth,
  format,
  fromUnixTime,
  startOfDay,
  startOfMonth,
} from "date-fns";
import { fromZonedTime, toZonedTime } from "date-fns-tz";

type DateFormatOptions = {
  /**
   * The time format to use (12-hour or 24-hour)
   * @default '24-hour'
   */
  timeFormat?: TimeFormat;

  /**
   * Whether to show seconds
   * @default false
   */
  showSeconds?: boolean;

  /**
   * Whether to show the timezone name
   * @default true
   */
  showTimeZone?: boolean;

  /**
   * Whether to show the date
   * @default true
   */
  showDate?: boolean;

  /**
   * Whether to show the time
   * @default true
   */
  showTime?: boolean;
};

const TIME_FORMAT_24 = "HH:mm";
const TIME_FORMAT_24_WITH_SECONDS = "HH:mm:ss";
const TIME_FORMAT_12 = "h:mm a";
const TIME_FORMAT_12_WITH_SECONDS = "h:mm:ss a";
const DATE_FORMAT = "MM/dd/yyyy";
const DATE_TIME_FORMAT_24 = `${DATE_FORMAT} ${TIME_FORMAT_24}`;
const DATE_TIME_FORMAT_24_WITH_SECONDS = `${DATE_FORMAT} ${TIME_FORMAT_24_WITH_SECONDS}`;
const DATE_TIME_FORMAT_12 = `${DATE_FORMAT} ${TIME_FORMAT_12}`;
const DATE_TIME_FORMAT_12_WITH_SECONDS = `${DATE_FORMAT} ${TIME_FORMAT_12_WITH_SECONDS}`;

/**
 * Converts a Date object to a Unix timestamp.
 * The timestamp represents the number of seconds since the Unix epoch (January 1, 1970, 00:00:00 UTC).
 *
 * @param date The Date object to convert.
 * @returns A Unix timestamp representing the input date.
 * @throws {Error} If the input date is invalid
 */
export function dateToUnixTimestamp(date: Date): number {
  if (!(date instanceof Date) || isNaN(date.getTime())) {
    throw new Error("Invalid date provided to dateToUnixTimestamp");
  }
  return Math.floor(date.getTime() / 1000);
}

/**
 * Gets today's date as a Unix timestamp.
 * The time is set to midnight (00:00:00 UTC).
 *
 * @returns A Unix timestamp representing today's date at midnight UTC.
 */
export function getTodayDate(): number {
  const date = new Date();
  date.setUTCHours(0, 0, 0, 0);
  return dateToUnixTimestamp(date);
}

/**
 * Formats a Unix timestamp into separated date and time parts.
 * Date is formatted as "MMM d" (e.g., "Jan 23")
 * Time is formatted according to the specified time format
 *
 * @param timestamp Unix timestamp in seconds
 * @param timeFormat The time format to use (12-hour or 24-hour)
 * @returns Object containing formatted date and time strings, or default values if timestamp is invalid
 */
export function formatSplitDateTime(
  timestamp: number | undefined,
  timeFormat: TimeFormat = TimeFormat.TimeFormat24Hour,
): {
  date: string;
  time: string;
} {
  if (!timestamp) return { date: "-", time: "" };

  const dateObj = toDate(timestamp);
  if (!dateObj) return { date: "-", time: "" };

  const timeFormatString =
    timeFormat === TimeFormat.TimeFormat12Hour
      ? TIME_FORMAT_12
      : TIME_FORMAT_24;

  return {
    date: format(dateObj, "d MMM yyyy"),
    time: format(dateObj, timeFormatString),
  };
}

export function formatDate(date: Date): string {
  return format(date, "d MMM yyyy");
}

/**
 * Converts a Unix timestamp to a Date object.
 * Handles undefined input gracefully.
 *
 * @param unixTimeStamp The Unix timestamp to convert, or undefined.
 * @returns A Date object representing the timestamp, or undefined if the input is undefined.
 */
export const toDate = (unixTimeStamp: number | undefined): Date | undefined => {
  if (!unixTimeStamp || isNaN(unixTimeStamp)) {
    return undefined;
  }
  const date = new Date(unixTimeStamp * 1000);
  return isNaN(date.getTime()) ? undefined : date;
};

/**
 * Converts a Date object to a Unix timestamp.
 * Handles undefined input gracefully.
 *
 * @param date The Date object to convert, or undefined.
 * @returns A Unix timestamp representing the date, or undefined if the input is undefined.
 */
export const toUnixTimeStamp = (date: Date | undefined): number | undefined => {
  if (!date || !(date instanceof Date) || isNaN(date.getTime())) {
    return undefined;
  }
  return Math.floor(date.getTime() / 1000);
};

/**
 * Generates a date string from a Date object.
 * Formats the date using date-fns in the format "dd MMM yyyy".
 *
 * @param date The Date object to format.
 * @returns A formatted date string.
 * @throws {Error} If the input date is invalid
 */
export function generateDateOnlyString(date: Date): string {
  if (!(date instanceof Date) || isNaN(date.getTime())) {
    throw new Error("Invalid date provided to generateDateOnlyString");
  }
  return format(date, DATE_FORMAT);
}

/**
 * Generates a Date object with the time set to midnight (00:00:00) from a date string.
 * Parses the input string using chrono-node and normalizes the time to midnight.
 *
 * @param date The date string to parse.
 * @returns A Date object representing the parsed date at midnight, or null if parsing fails.
 */
export function generateDateOnly(date: string): Date | null {
  if (!date || typeof date !== "string") {
    return null;
  }

  const parsed = chrono.parseDate(date);
  if (parsed && !isNaN(parsed.getTime())) {
    const normalized = new Date(parsed);
    normalized.setHours(0, 0, 0, 0);
    return normalized;
  }
  return null;
}

// const dateOnlyFormatRegex =
//   /^(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s([1-9]|[12]\d|3[01])(st|nd|rd|th)\s\d{4}$/;

/**
 * Checks if a date string matches the expected format.
 *
 * @param dateString The date string to validate.
 * @returns True if the date string matches the format, false otherwise.
 */
export function isValidDateOnlyFormat(dateString: string): boolean {
  if (!dateString || typeof dateString !== "string") {
    return false;
  }
  try {
    const date = new Date(dateString);
    return !isNaN(date.getTime()) && format(date, DATE_FORMAT) === dateString;
  } catch {
    return false;
  }
}

/**
 * Generates a date and time string from a Date object.
 *
 * @param date The Date object to format.
 * @param options Formatting options including time format and whether to show seconds
 * @returns A formatted date and time string.
 * @throws {Error} If the input date is invalid
 */
export function generateDateTimeString(
  date: Date,
  options: DateFormatOptions = {},
): string {
  if (!(date instanceof Date) || isNaN(date.getTime())) {
    throw new Error("Invalid date provided to generateDateTimeString");
  }

  const { timeFormat = TimeFormat.TimeFormat24Hour, showSeconds = false } =
    options;

  let formatString: string;
  if (timeFormat === TimeFormat.TimeFormat12Hour) {
    formatString = showSeconds
      ? DATE_TIME_FORMAT_12_WITH_SECONDS
      : DATE_TIME_FORMAT_12;
  } else {
    formatString = showSeconds
      ? DATE_TIME_FORMAT_24_WITH_SECONDS
      : DATE_TIME_FORMAT_24;
  }

  return format(date, formatString);
}

export function generateDateTimeStringFromUnixTimestamp(
  timestamp: number,
  options: DateFormatOptions = {},
): string {
  const date = toDate(timestamp);
  if (!date) {
    return "-";
  }

  return generateDateTimeString(date, options);
}

/**
 * Generates a Date object from a date and time string.
 *
 * @param date The date and time string to parse.
 * @returns A Date object representing the parsed date and time, or null if parsing fails.
 */
export function generateDateTime(date: string): Date | null {
  if (!date || typeof date !== "string") {
    return null;
  }

  const parsed = chrono.parseDate(date);
  return parsed && !isNaN(parsed.getTime()) ? parsed : null;
}

const dateTimeFormatRegex =
  /^(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s([1-9]|[12]\d|3[01])(st|nd|rd|th)\s\d{4},\s(0\d|1[0-2]):([0-5]\d)\s(AM|PM)$/;

/**
 * Checks if a date string is in a valid date and time format.
 *
 * @param dateString The date string to validate.
 * @returns True if the date string is in the valid format, false otherwise.
 */
export function isValidDateTimeFormat(dateString: string): boolean {
  if (!dateString || typeof dateString !== "string") {
    return false;
  }
  return dateTimeFormatRegex.test(dateString);
}

/**
 * Formats a Unix timestamp to a localized date string based on user preferences
 *
 * @param timestamp - Unix timestamp in seconds
 * @param options - Formatting options
 * @param userTimezone - User's preferred timezone, or "auto" for browser detection
 * @returns Formatted date string
 */
export function formatToUserTimezone(
  timestamp: number,
  options: DateFormatOptions = {},
  userTimezone: string = "auto",
): string {
  const timezone =
    userTimezone === "auto"
      ? Intl.DateTimeFormat().resolvedOptions().timeZone
      : userTimezone;

  if (!timestamp || isNaN(timestamp)) {
    return "N/A";
  }

  const {
    timeFormat = TimeFormat.TimeFormat24Hour,
    showSeconds = false,
    showTimeZone = true,
    showDate = true,
    showTime = true,
  } = options;

  const date = fromUnixTime(timestamp);

  if (isNaN(date.getTime())) {
    return "N/A";
  }

  const formatOptions: Intl.DateTimeFormatOptions = {
    timeZone: timezone,
  };

  if (showTime) {
    formatOptions.hour = "2-digit";
    formatOptions.minute = "2-digit";
    formatOptions.hour12 = timeFormat === TimeFormat.TimeFormat12Hour;

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

/**
 * Validates if a given value is a valid Date object
 *
 * @param date - Value to check
 * @returns boolean indicating if the value is a valid Date
 */
export function isValidDate(date: unknown): date is Date {
  return date instanceof Date && !isNaN(date.getTime());
}

/**
 * Formats a duration in seconds into a human-readable string (e.g., "4d 5h", "5h", "5m").
 *
 * @param durationInSeconds The duration in seconds.
 * @returns A formatted string representing the duration.
 */
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
    // Duration is > 0 but < 60 seconds, display as "1m"
    return "1m";
  }

  return parts.join(" ");
}

export const toDateFromUnixSeconds = (unixSeconds: number) =>
  new Date(unixSeconds * 1000);

const startOfLocalDay = (d: Date) =>
  new Date(d.getFullYear(), d.getMonth(), d.getDate());

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
  const now = new Date();

  const showYear =
    !sameYear ||
    s.getFullYear() !== now.getFullYear() ||
    e.getFullYear() !== now.getFullYear();

  const sFmt = new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    ...(showYear ? { year: "numeric" } : {}),
  }).format(s);

  const eFmt = new Intl.DateTimeFormat(undefined, {
    ...(sameMonth ? {} : { month: "short" }),
    day: "numeric",
    ...(showYear ? { year: "numeric" } : {}),
  }).format(e);

  return sFmt === eFmt ? sFmt : `${sFmt}–${eFmt}`;
};

/**
 * Date range preset configuration for filter components
 */
export type DateRangePreset = {
  label: string;
  getValue: () => { startDate: number; endDate: number };
};

/**
 * Gets the start of day in a specific timezone as Unix timestamp
 * @param date Date object (defaults to today)
 * @param timezone IANA timezone string (e.g., "America/New_York")
 * @returns Unix timestamp for start of day (00:00:00.000) in the specified timezone
 */
export function getStartOfDay(
  date: Date = new Date(),
  timezone?: string,
): number {
  if (!timezone) {
    // Fallback to browser local timezone
    return dateToUnixTimestamp(startOfDay(date));
  }

  // Convert the date to the target timezone
  const zonedDate = toZonedTime(date, timezone);
  // Get start of day in that timezone
  const startOfDayInZone = startOfDay(zonedDate);
  // Convert back to UTC for Unix timestamp
  const utcDate = fromZonedTime(startOfDayInZone, timezone);
  return dateToUnixTimestamp(utcDate);
}

/**
 * Gets the end of day in a specific timezone as Unix timestamp
 * @param date Date object (defaults to today)
 * @param timezone IANA timezone string (e.g., "America/New_York")
 * @returns Unix timestamp for end of day (23:59:59.999) in the specified timezone
 */
export function getEndOfDay(
  date: Date = new Date(),
  timezone?: string,
): number {
  if (!timezone) {
    // Fallback to browser local timezone
    return dateToUnixTimestamp(endOfDay(date));
  }

  // Convert the date to the target timezone
  const zonedDate = toZonedTime(date, timezone);
  // Get end of day in that timezone
  const endOfDayInZone = endOfDay(zonedDate);
  // Convert back to UTC for Unix timestamp
  const utcDate = fromZonedTime(endOfDayInZone, timezone);
  return dateToUnixTimestamp(utcDate);
}

/**
 * Gets the start of month in a specific timezone as Unix timestamp
 * @param date Date object to get month from (defaults to today)
 * @param timezone IANA timezone string (e.g., "America/New_York")
 * @returns Unix timestamp for first day of month at 00:00:00.000 in the specified timezone
 */
export function getStartOfMonth(
  date: Date = new Date(),
  timezone?: string,
): number {
  if (!timezone) {
    return dateToUnixTimestamp(startOfMonth(date));
  }

  // Convert the date to the target timezone
  const zonedDate = toZonedTime(date, timezone);
  // Get start of month in that timezone
  const startOfMonthInZone = startOfMonth(zonedDate);
  // Convert back to UTC for Unix timestamp
  const utcDate = fromZonedTime(startOfMonthInZone, timezone);
  return dateToUnixTimestamp(utcDate);
}

/**
 * Gets the end of month in a specific timezone as Unix timestamp
 * @param date Date object to get month from (defaults to today)
 * @param timezone IANA timezone string (e.g., "America/New_York")
 * @returns Unix timestamp for last day of month at 23:59:59.999 in the specified timezone
 */
export function getEndOfMonth(
  date: Date = new Date(),
  timezone?: string,
): number {
  if (!timezone) {
    return dateToUnixTimestamp(endOfMonth(date));
  }

  // Convert the date to the target timezone
  const zonedDate = toZonedTime(date, timezone);
  // Get end of month in that timezone
  const endOfMonthInZone = endOfMonth(zonedDate);
  // Convert back to UTC for Unix timestamp
  const utcDate = fromZonedTime(endOfMonthInZone, timezone);
  return dateToUnixTimestamp(utcDate);
}

/**
 * Gets the start of quarter in a specific timezone as Unix timestamp
 * @param date Date object to get quarter from (defaults to today)
 * @param timezone IANA timezone string (e.g., "America/New_York")
 * @returns Unix timestamp for first day of quarter at 00:00:00.000 in the specified timezone
 */
export function getStartOfQuarter(date: Date = new Date()): Date {
  const month = date.getMonth();
  const quarter = Math.floor(month / 3) * 3; // 0 for Q1, 3 for Q2, 6 for Q3, 9 for Q4
  return new Date(date.getFullYear(), quarter, 1);
}

/**
 * Gets the end of quarter in a specific timezone as Unix timestamp
 * @param date Date object to get quarter from (defaults to today)
 * @param timezone IANA timezone string (e.g., "America/New_York")
 * @returns Unix timestamp for last day of quarter at 23:59:59.999 in the specified timezone
 */
export function getEndOfQuarter(date: Date = new Date()): Date {
  const month = date.getMonth();
  const quarter = Math.floor(month / 3) * 3 + 2; // 2 for Q1, 5 for Q2, 8 for Q3, 11 for Q4
  const lastDay = new Date(date.getFullYear(), quarter + 1, 0); // Day 0 of next month = last day of current month
  return lastDay;
}

/**
 * Adds days to a date
 * @param date Base date
 * @param days Number of days to add
 * @returns New Date object with days added
 */
export function addDays(date: Date, days: number): Date {
  const result = new Date(date);
  result.setDate(result.getDate() + days);
  return result;
}

/**
 * Common date range presets for PTO and other date filters
 * Uses the specified timezone to ensure proper date boundaries
 * @param timezone IANA timezone string (e.g., "America/New_York") - if not provided, uses browser's local timezone
 * @returns Array of date range preset configurations
 */
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
        const endDate = addDays(today, 6); // Include today
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
        const endDate = addDays(today, 29); // Include today
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
        const nextMonth = new Date(
          today.getFullYear(),
          today.getMonth() + 1,
          1,
        );
        return {
          startDate: getStartOfMonth(nextMonth, timezone),
          endDate: getEndOfMonth(nextMonth, timezone),
        };
      },
    },
    {
      label: "This Quarter",
      getValue: () => {
        const today = new Date();
        const startOfQuarter = getStartOfQuarter(today);
        const endOfQuarter = getEndOfQuarter(today);
        return {
          startDate: getStartOfDay(startOfQuarter, timezone),
          endDate: getEndOfDay(endOfQuarter, timezone),
        };
      },
    },
  ];
}
