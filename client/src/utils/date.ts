/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import {
  format,
  formatDistanceToNow,
  parseISO,
  differenceInSeconds,
  differenceInMinutes,
  differenceInHours,
  differenceInDays,
} from "date-fns";

/**
 * Formats the given date string into the format "yyyy/MM/dd HH:mm".
 * @param date - The date string to format.
 * @returns A string representing the date in "yyyy/MM/dd HH:mm" format.
 */
export function formatDate(date: string): string {
  const parsedDate = parseISO(date);
  return format(parsedDate, "yyyy/MM/dd HH:mm");
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
  } if (diffInMinutes < 60) {
    return `${diffInMinutes} min${diffInMinutes === 1 ? "" : "s"} ago`;
  } if (diffInHours < 24) {
    return `${diffInHours} hr${diffInHours === 1 ? "" : "s"} ago`;
  } 
  return `${diffInDays} day${diffInDays === 1 ? "" : "s"} ago`;
  
}
