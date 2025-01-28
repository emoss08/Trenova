import * as chrono from "chrono-node";
import { format, fromUnixTime } from "date-fns";

/**
 * Converts a Date object to a Unix timestamp.
 * The timestamp represents the number of seconds since the Unix epoch (January 1, 1970, 00:00:00 UTC).
 *
 * @param date The Date object to convert.
 * @returns A Unix timestamp representing the input date.
 */
export function dateToUnixTimestamp(date: Date): number {
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
 * Converts a Unix timestamp to a Date object.
 * Handles undefined input gracefully.
 *
 * @param unixTimeStamp The Unix timestamp to convert, or undefined.
 * @returns A Date object representing the timestamp, or undefined if the input is undefined.
 */
export const toDate = (unixTimeStamp: number | undefined) => {
  return unixTimeStamp ? new Date(unixTimeStamp * 1000) : undefined;
};

/**
 * Converts a Unix timestamp to a Date object.
 * Handles undefined input gracefully.
 *
 * @param unixTimeStamp The Unix timestamp to convert, or undefined.
 * @returns A Date object representing the timestamp, or undefined if the input is undefined.
 */
export const toUnixTimeStamp = (date: Date | undefined) => {
  if (!date) return undefined;

  return date ? Math.floor(date.getTime() / 1000) : undefined;
};

/**
 * Generates a date string from a Date object.
 * Formats the date using date-fns in the format "MMM do yyyy".
 *
 * @param date The Date object to format.
 * @returns A formatted date string.
 */
export function generateDateOnlyString(date: Date) {
  return format(date, "MMM do yyyy");
}

/**
 * Generates a Date object with the time set to midnight (00:00:00) from a date string.
 * Parses the input string using chrono-node and normalizes the time to midnight.
 * @param date The date string to parse.
 * @returns A Date object representing the parsed date at midnight, or null if parsing fails.
 */
export function generateDateOnly(date: string) {
  const parsed = chrono.parseDate(date);
  if (parsed) {
    const normalized = new Date(parsed);
    normalized.setHours(0, 0, 0, 0);
    return normalized;
  }
  return null;
}

const dateOnlyFormatRegex =
  /^(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s([1-9]|[12]\d|3[01])(st|nd|rd|th)\s\d{4}$/;

export function isValidDateOnlyFormat(dateString: string) {
  return dateOnlyFormatRegex.test(dateString);
}

/**
 * Generates a date and time string from a Date object.
 * Formats the date using date-fns in the format "MMM do yyyy, hh:mm a".
 *
 * @param date The Date object to format.
 * @returns A formatted date and time string.
 */
export function generateDateTimeString(date: Date) {
  return format(date, "MMM do yyyy, hh:mm a");
}

/**
 * Generates a Date object from a date and time string.
 * Parses the input string using chrono-node.
 *
 * @param date The date and time string to parse.
 * @returns A Date object representing the parsed date and time, or null if parsing fails.
 */
export function generateDateTime(date: string) {
  return chrono.parseDate(date);
}

const dateTimeFormatRegex =
  /^(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s([1-9]|[12]\d|3[01])(st|nd|rd|th)\s\d{4},\s(0\d|1[0-2]):([0-5]\d)\s(AM|PM)$/;

/**
 * Checks if a date string is in a valid date and time format.
 * Uses a regular expression to validate the format "MMM do yyyy, hh:mm a".
 *
 * @param dateString The date string to validate.
 * @returns True if the date string is in the valid format, false otherwise.
 */
export function isValidDateTimeFormat(dateString: string) {
  return dateTimeFormatRegex.test(dateString);
}

/**
 * Formats a Unix timestamp to a date and time string in the specified timezone.
 * Converts the timestamp to a Date object and formats it using the provided timezone.
 *
 * @param timestamp The Unix timestamp to format.
 * @param timezone The timezone to format the date in
 * @returns A formatted date and time string in the specified timezone, or "N/A" if the date is invalid.
 */
export function formatToUserTimezone(timestamp: number, timezone?: string) {
  // Convert Unix timestamp to Date object
  const date = fromUnixTime(timestamp);

  // Check if the date is valid
  if (isNaN(date.getTime())) {
    return "N/A";
  }

  return date.toLocaleString("en-US", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: timezone || "UTC",
    timeZoneName: "short",
  });
}
