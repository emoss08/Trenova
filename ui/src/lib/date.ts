import { TimeFormat } from "@/types/user";
import * as chrono from "chrono-node";
import { format, fromUnixTime } from "date-fns";

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
};

const TIME_FORMAT_24 = "HH:mm";
const TIME_FORMAT_24_WITH_SECONDS = "HH:mm:ss";
const DATE_FORMAT = "MM/dd/yyyy";
const DATE_TIME_FORMAT_24 = `${DATE_FORMAT} ${TIME_FORMAT_24}`;
const DATE_TIME_FORMAT_24_WITH_SECONDS = `${DATE_FORMAT} ${TIME_FORMAT_24_WITH_SECONDS}`;

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
 * Formats a Unix timestamp into separated date and time parts using 24-hour format.
 * Date is formatted as "MMM d" (e.g., "Jan 23")
 * Time is formatted as "HH:mm" (e.g., "14:30")
 *
 * @param timestamp Unix timestamp in seconds
 * @returns Object containing formatted date and time strings, or default values if timestamp is invalid
 */
export function formatSplitDateTime(timestamp: number | undefined): {
  date: string;
  time: string;
} {
  if (!timestamp) return { date: "-", time: "" };

  const dateObj = toDate(timestamp);
  if (!dateObj) return { date: "-", time: "" };

  return {
    date: format(dateObj, "d MMM yyyy"),
    time: format(dateObj, "HH:mm"),
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
 * Generates a date and time string from a Date object using 24-hour format.
 *
 * @param date The Date object to format.
 * @param showSeconds Whether to include seconds in the output.
 * @returns A formatted date and time string.
 * @throws {Error} If the input date is invalid
 */
export function generateDateTimeString(
  date: Date,
  showSeconds = false,
): string {
  if (!(date instanceof Date) || isNaN(date.getTime())) {
    throw new Error("Invalid date provided to generateDateTimeString");
  }
  return format(
    date,
    showSeconds ? DATE_TIME_FORMAT_24_WITH_SECONDS : DATE_TIME_FORMAT_24,
  );
}

export function generateDateTimeStringFromUnixTimestamp(
  timestamp: number,
  showSeconds = false,
): string {
  const date = toDate(timestamp);
  if (!date) {
    return "-";
  }

  return generateDateTimeString(date, showSeconds);
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
 * @returns Formatted date string
 */
export function formatToUserTimezone(
  timestamp: number,
  options: DateFormatOptions = {},
): string {
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

  if (!timestamp || isNaN(timestamp)) {
    return "N/A";
  }

  const { showSeconds = false, showTimeZone = true, showDate = true } = options;

  const date = fromUnixTime(timestamp);

  if (isNaN(date.getTime())) {
    return "N/A";
  }

  const formatOptions: Intl.DateTimeFormatOptions = {
    hour: "2-digit",
    minute: "2-digit",
    timeZone: timezone,
    hour12: false, // Always use 24-hour format
  };

  if (showSeconds) {
    formatOptions.second = "2-digit";
  }

  if (showTimeZone) {
    formatOptions.timeZoneName = "short";
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
