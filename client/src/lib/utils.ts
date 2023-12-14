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
import { clsx, type ClassValue } from "clsx";
import { RefObject, useEffect } from "react";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * Formats a date string into a human readable format
 * @param dateStr - The date string to format
 * @returns {string}
 */
export function upperFirst(str: string): string {
  if (!str) return "";
  return str.charAt(0).toUpperCase() + str.slice(1);
}

/**
 * Formats a number into a USD string
 * @param num - The number to format
 * @returns {string}
 */
export function USDollarFormat(num: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(num);
}

/**
 * Formats a number into a USD string
 * @param num - The number to format
 * @returns {string}
 */
export function USDollarFormatString(num: string): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(parseFloat(num));
}

/**
 * Truncates a string to a given length
 * @param str - The string to truncate
 * @param length - The length to truncate the string to
 * @returns {string}
 */
export function truncateText(str: string, length: number): string {
  return str.length > length ? str.substring(0, length) + "..." : str;
}

/**
 * Returns a random integer between min (inclusive) and max (inclusive).
 * @param min - The minimum value to return
 * @param max - The maximum value to return
 * @returns {number}
 */
export const useClickOutside = <T extends HTMLElement>(
  ref: RefObject<T>,
  handler: (event: MouseEvent | TouchEvent) => void,
): void => {
  useEffect(() => {
    let startedInside: boolean | null = false;
    let startedWhenMounted = false;

    const listener = (event: MouseEvent | TouchEvent) => {
      // Do nothing if `mousedown` or `touchstart` started inside ref element
      if (startedInside || !startedWhenMounted) return;
      // Do nothing if clicking ref's element or descendent elements
      if (!ref.current || ref.current.contains(event.target as Node)) return;

      handler(event);
    };

    const validateEventStart = (event: MouseEvent | TouchEvent) => {
      startedWhenMounted = !!ref.current;
      startedInside = ref.current && ref.current.contains(event.target as Node);
    };

    document.addEventListener("mousedown", validateEventStart);
    document.addEventListener("touchstart", validateEventStart);
    document.addEventListener("click", listener);

    return () => {
      document.removeEventListener("mousedown", validateEventStart);
      document.removeEventListener("touchstart", validateEventStart);
      document.removeEventListener("click", listener);
    };
  }, [ref, handler]);
};

/**
 * Validates a decimal value with a given number of decimal places
 * @param value - The value to validate
 * @param decimalPlaces - The number of decimal places to allow
 * @returns {boolean}
 */
export function validateDecimal(value: string, decimalPlaces: number): boolean {
  const regex = new RegExp(`^\\d+(\\.\\d{1,${decimalPlaces}})?$`);
  return regex.test(value);
}

/**
 * Formats a duration string into a human readable format
 * @param durationStr - The duration string to format
 * @returns {string}
 */
export function formatDuration(durationStr: string): string {
  if (!durationStr) return "";

  const parts = durationStr.split(" ");
  const days = parseInt(parts[0]);
  const timeParts = parts[1].split(":");
  const hours = parseInt(timeParts[0]);
  const minutes = parseInt(timeParts[1]);
  const seconds = parseInt(timeParts[2]);

  let result = "";
  if (days > 0) result += `${days} day${days > 1 ? "s" : ""}, `;
  if (hours > 0) result += `${hours} hour${hours > 1 ? "s" : ""}, `;
  if (minutes > 0) result += `${minutes} minute${minutes > 1 ? "s" : ""}, `;
  if (seconds > 0) result += `${seconds} second${seconds > 1 ? "s" : ""}`;

  return result.replace(/, $/, ""); // Remove trailing comma
}

/**
 * Sanitizes query params by removing nullish and empty string values
 * @param queryParams
 * @returns {Record<string, string>}
 */
function sanitizeQueryParams(
  queryParams: Record<string, string | number | boolean>,
): Record<string, string> {
  return Object.entries(queryParams).reduce(
    (acc, [key, value]) => {
      // Check for nullish or empty string values
      if (value !== null && value !== undefined && value !== "") {
        // Ensure the key and value are properly encoded for URL usage
        acc[encodeURIComponent(key)] = encodeURIComponent(value.toString());
      }
      return acc;
    },
    {} as Record<string, string>,
  );
}

type PopoutWindowParams = {
  width?: number;
  height?: number;
  left?: number;
  top?: number;
  hideHeader?: boolean;
};

/**
 * Opens a new window with the given path and query params
 * @param path - The path to open the new window to
 * @param incomingQueryParams - The query params to pass to the new window
 * @returns {void}
 */
export function PopoutWindow(
  path: string,
  incomingQueryParams?: Record<string, string | number | boolean>,
  {
    width = 1280,
    height = 720,
    left = window.screen.width / 2 - width / 2,
    top = window.screen.height / 2 - height / 2,
    hideHeader = true,
  }: PopoutWindowParams = {},
): void {
  const extendedQueryParams = sanitizeQueryParams({
    ...incomingQueryParams,
    width: width.toString(),
    height: height.toString(),
    left: left.toString(),
    top: top.toString(),
    hideHeader: hideHeader.toString(),
  });

  const url = `${path}?${new URLSearchParams(extendedQueryParams).toString()}`;

  window.open(
    url,
    "",
    `toolbar=no, location=no, directories=no, status=no, menubar=no, scrollbars=no, resizable=no, copyhistory=no, width=${width}, height=${height}, top=${top}, left=${left}`,
  );
}

/**
 * Removes all undefined, null, and empty string values from an object
 * @param obj
 * @returns {Record<string, any>}
 */
export const cleanObject = (obj: Record<string, any>): Record<string, any> => {
  const cleanedObj: Record<string, any> = {};
  Object.keys(obj).forEach((key) => {
    if (obj[key] !== undefined && obj[key] !== "" && obj[key] !== null) {
      cleanedObj[key] = obj[key];
    }
  });
  return cleanedObj;
};

/**
 * Converts a camelCase string to a readable string
 * @param str
 * @returns {string}
 */
export const convertCamelCaseToReadable = (str: string): string => {
  return str
    .replace(/([A-Z])/g, " $1")
    .replace(/^./, (str) => str.toUpperCase());
};
