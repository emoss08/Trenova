import { useUserStore } from "@/stores/AuthStore";
import {
  differenceInDays,
  differenceInHours,
  differenceInMinutes,
  differenceInSeconds,
  formatDistanceToNow,
  parseISO,
} from "date-fns";

/**
 * Formats the given date string into the format "yyyy/MM/dd HH:mm".
 * @param date - The date string to format.
 * @returns A string representing the date in "yyyy/MM/dd HH:mm" format.
 */
export function formatToUserTimezone(date: string): string {
  // Get the user timezone from state
  const user = useUserStore.get("user");

  // Parse the date string into a Date object.
  const parsedDate = parseISO(date);

  // Format the date into the desired format.
  return parsedDate.toLocaleString("en-US", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: user.timezone,
  });
}

/**
 * Converts the given date string into a human-readable format relative to the current time.
 * @param date - The date string to convert.
 * @returns A string representing the date in a human-readable format relative to the current time.
 */
export function formatDateToHumanReadable(date: string): string {
  const parsedDate = parseISO(date);
  return formatDistanceToNow(parsedDate, { addSuffix: true });
}

/**
 * Converts a timestamp into a human-readable format indicating the time elapsed since the timestamp.
 * @param timestamp - The timestamp to convert.
 * @returns A string indicating the time elapsed since the timestamp in seconds, minutes, hours, or days as appropriate.
 */
export function formatTimestamp(timestamp: string) {
  const date = parseISO(timestamp);
  const now = new Date();

  const diffInSeconds = differenceInSeconds(now, date);
  const diffInMinutes = differenceInMinutes(now, date);
  const diffInHours = differenceInHours(now, date);
  const diffInDays = differenceInDays(now, date);

  if (diffInSeconds < 60) {
    return `${diffInSeconds} sec${diffInSeconds === 1 ? "" : "s"} ago`;
  }
  if (diffInMinutes < 60) {
    return `${diffInMinutes} min${diffInMinutes === 1 ? "" : "s"} ago`;
  }
  if (diffInHours < 24) {
    return `${diffInHours} hr${diffInHours === 1 ? "" : "s"} ago`;
  }
  return `${diffInDays} day${diffInDays === 1 ? "" : "s"} ago`;
}
