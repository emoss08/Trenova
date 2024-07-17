/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */



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
 * Formats the given date string into the user's timezone.
 * @param date - The date string to format.
 * @returns A string representing the date in the user's timezone.
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
 * Formats the given date string into a human-readable format relative to now.
 * @param date - The date string to format.
 * @returns A string representing the date relative to now.
 */
export function formatDateRelativeToNow(date: string): string {
  const user = useUserStore.get("user");
  const parsedDate = parseISO(date).toLocaleString("en-US", {
    timeZone: user.timezone,
  });

  return formatDistanceToNow(parseISO(parsedDate), { addSuffix: true });
}

// Gets today's date in YYYY-MM-DD format
export function getTodayDate() {
  const date = new Date();
  date.setUTCHours(0, 0, 0, 0); // Set to midnight UTC
  return date.toISOString();
}

// Gets the date N days ago from today in YYYY-MM-DD format
export function getDateNDaysAgo(days: number) {
  const date = new Date();
  date.setUTCHours(0, 0, 0, 0); // Set to midnight UTC
  date.setUTCDate(date.getUTCDate() - days);
  return date.toISOString();
}

// Converts ISO string into a date string in the format "M/DD"
// Example: "2021-05-01T00:00:00.000Z" -> "May 1"
export function getMonthDayString(date: string) {
  return parseISO(date).toLocaleString("en-US", {
    month: "short",
    day: "numeric",
  });
}

// takes two dates and get the days between them
export function getDaysBetweenDates(date1: string, date2: string) {
  const startDate = parseISO(date1);
  const endDate = parseISO(date2);
  return differenceInDays(endDate, startDate);
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
