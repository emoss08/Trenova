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



import { ShipmentStatus } from "@/types/shipment";
import { clsx, type ClassValue } from "clsx";
import { RefObject, useEffect } from "react";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * Formats a date string into a human readable format
 * @returns {string}
 * @param str
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
 * Truncates a string to a given length
 * @param str - The string to truncate
 * @param length - The length to truncate the string to
 * @returns {string}
 */
export function truncateText(str: string, length: number): string {
  return str?.length > length ? `${str.slice(0, length)}...` : str;
}

/**
 * Returns a random integer between min (inclusive) and max (inclusive).
 * @returns {number}
 * @param ref
 * @param handler
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

/**
 * Opens a new window with the given path and query params
 * @param path - The path to open the new window to
 * @param incomingQueryParams - The query params to pass to the new window
 * @param width - The width of the new window
 * @param height - The height of the new window
 * @param left - The left position of the new window
 * @param top - The top position of the new window
 * @param hideHeader - Whether or not to hide the header of the new window
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
  }: {
    width?: number;
    height?: number;
    left?: number;
    top?: number;
    hideHeader?: boolean;
  } = {},
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
 * Function to convert shipment status to human readable format
 * @param status - The status to convert
 */
export function shipmentStatusToReadable(status: ShipmentStatus) {
  switch (status) {
    case "New":
      return "New";
    case "InProgress":
      return "In Progress";
    case "Completed":
      return "Completed";
    case "Hold":
      return "On Hold";
    case "Billed":
      return "Billed";
    case "Voided":
      return "Voided";
    default:
      return "Unknown";
  }
}

export const toTitleCase = (value: string) => {
  return value
    .toLowerCase()
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
};

export const focusRing = [
  // base
  "outline outline-offset-2 outline-0 focus-visible:outline-2",
  // outline color
  "outline-blue-500 dark:outline-blue-500",
];
