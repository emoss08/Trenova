import { useAuthStore } from "@/stores/user-store";
import * as chrono from "chrono-node";
import { format, fromUnixTime } from "date-fns";

// Helper function to convert a Date to Unix timestamp
export function dateToUnixTimestamp(date: Date): number {
  return Math.floor(date.getTime() / 1000);
}

// Gets today's date as a Unix timestamp
export function getTodayDate(): number {
  const date = new Date();
  date.setUTCHours(0, 0, 0, 0);

  return dateToUnixTimestamp(date);
}

// Converts a Unix timestamp to a Date
export const toDate = (unixTimeStamp: number | undefined) => {
  return unixTimeStamp ? new Date(unixTimeStamp * 1000) : undefined;
};

// Converts a Date to a Unix timestamp
export const toUnixTimeStamp = (date: Date | undefined) => {
  if (!date) return undefined;

  return date ? Math.floor(date.getTime() / 1000) : undefined;
};

// Date-only utilities
export function generateDateOnlyString(date: Date) {
  return format(date, "MMM do yyyy");
}

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

// DateTime utilities (original functionality)
export function generateDateTimeString(date: Date) {
  return format(date, "MMM do yyyy, hh:mm a");
}

export function generateDateTime(date: string) {
  return chrono.parseDate(date);
}

const dateTimeFormatRegex =
  /^(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s([1-9]|[12]\d|3[01])(st|nd|rd|th)\s\d{4},\s(0\d|1[0-2]):([0-5]\d)\s(AM|PM)$/;

export function isValidDateTimeFormat(dateString: string) {
  return dateTimeFormatRegex.test(dateString);
}

export function formatToUserTimezone(timestamp: number) {
  const { user } = useAuthStore();

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
    timeZone: user?.timezone,
    timeZoneName: "short",
  });
}
