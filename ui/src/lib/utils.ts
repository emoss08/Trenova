import { Resource } from "@/types/audit-entry";
import { ResourcePageInfo, ResourceType, RouteInfo } from "@/types/nav-links";
import { clsx, type ClassValue } from "clsx";
import { RefObject, useEffect } from "react";
import { twMerge } from "tailwind-merge";
import { LocationSchema } from "./schemas/location-schema";

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
 * Converts a decimal string to a USD string
 * @param value - The decimal string to convert
 * @returns {string}
 */
export function ConvertDecimalToUSD(value: string): string {
  if (value === "") {
    return "";
  }

  const num = parseFloat(value);

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
 * List of words that should remain lowercase in titles
 * unless they are the first or last word
 */
const LOWERCASE_WORDS = new Set([
  "a",
  "an",
  "the",
  "and",
  "but",
  "or",
  "for",
  "nor",
  "in",
  "on",
  "at",
  "to",
  "by",
  "of",
]);

/**
 * Converts a string to title case with special handling for technical terms
 * @param str - The input string to format
 * @returns Formatted string in title case
 */
export function toTitleCase(str: string): string {
  // First, handle technical terms and special cases
  const technicalTerms: Record<string, string> = {
    id: "ID",
    url: "URL",
    uri: "URI",
    api: "API",
    ui: "UI",
    ux: "UX",
    ip: "IP",
    sql: "SQL",
  };

  // Split the input string by common delimiters
  const words = str
    .replace(/_/g, " ") // Replace underscores with spaces
    .replace(/([A-Z])/g, " $1") // Add space before capital letters
    .toLowerCase() // Convert to lowercase
    .replace(/\s+/g, " ") // Replace multiple spaces with single space
    .trim() // Remove leading/trailing spaces
    .split(" "); // Split into array of words

  return words
    .map((word, index, arr) => {
      // Check if it's a known technical term
      if (technicalTerms[word]) {
        return technicalTerms[word];
      }

      // Special handling for "At" in timestamps
      if (
        word === "at" &&
        (arr[index - 1]?.toLowerCase().includes("created") ||
          arr[index - 1]?.toLowerCase().includes("updated"))
      ) {
        return "At";
      }

      // Always capitalize first and last words
      if (index === 0 || index === arr.length - 1) {
        return word.charAt(0).toUpperCase() + word.slice(1);
      }

      // Keep lowercase words in lowercase unless they're after a colon or period
      if (
        LOWERCASE_WORDS.has(word) &&
        arr[index - 1]?.slice(-1) !== ":" &&
        arr[index - 1]?.slice(-1) !== "."
      ) {
        return word;
      }

      // Capitalize the first letter of other words
      return word.charAt(0).toUpperCase() + word.slice(1);
    })
    .join(" ");
}

export function toSentenceCase(str: string) {
  return str
    .replace(/_/g, " ")
    .replace(/([A-Z])/g, " $1")
    .toLowerCase()
    .replace(/^\w/, (c) => c.toUpperCase())
    .replace(/\s+/g, " ")
    .trim();
}

/**
 * @see https://github.com/radix-ui/primitives/blob/main/packages/core/primitive/src/primitive.tsx
 */
export function composeEventHandlers<E>(
  originalEventHandler?: (event: E) => void,
  ourEventHandler?: (event: E) => void,
  { checkForDefaultPrevented = true } = {},
) {
  return function handleEvent(event: E) {
    originalEventHandler?.(event);

    if (
      checkForDefaultPrevented === false ||
      !(event as unknown as Event).defaultPrevented
    ) {
      return ourEventHandler?.(event);
    }
  };
}

export function formatLocation(location?: LocationSchema) {
  if (!location) {
    return "";
  }

  const { state, addressLine1, addressLine2, city, postalCode } = location;

  const addressLine = addressLine1 + (addressLine2 ? `, ${addressLine2}` : "");
  const cityStateZip = `${city} ${state?.abbreviation}, ${postalCode}`;

  return `${addressLine} ${cityStateZip}`;
}

/** Helper function to safely convert decimal string to number */
export function toNumber(value: any): number {
  const num = Number(value);
  return isNaN(num) ? 0 : num; // Ensure it returns 0 if NaN
}

export function resourceToPage(resource: Resource) {
  switch (resource) {
    case Resource.LocationCategory:
      return "/dispatch/configurations/location-categories";
    case Resource.Location:
      return "/dispatch/configurations/locations";
    case Resource.FleetCode:
      return "/dispatch/configurations/fleet-codes";
    case Resource.Worker:
      return "/dispatch/configurations/workers";
    case Resource.Tractor:
      return "/dispatch/configurations/tractors";
    case Resource.Trailer:
      return "/dispatch/configurations/trailers";
    case Resource.Shipment:
      return "/shipments/management";
    case Resource.ShipmentType:
      return "/shipments/configurations/shipment-types";
    case Resource.ServiceType:
      return "/shipments/configurations/service-types";
    case Resource.HazardousMaterial:
      return "/shipments/configurations/hazardous-materials";
    case Resource.Commodity:
      return "/shipments/configurations/commodities";
    case Resource.Assignment:
      return "/shipments/assignments";
    case Resource.ShipmentMove:
      return "/shipments/moves";
    case Resource.Stop:
      return "/shipments/stops";
    case Resource.Customer:
      return "/customers";
    case Resource.Invoice:
      return "/invoices";
    case Resource.Dispatch:
      return "/dispatch/management";
    case Resource.Report:
      return "/reports";
    case Resource.AuditEntries:
      return "/audit-entries";
    case Resource.TableConfiguration:
      return "/dispatch/configurations/table-configurations";
    case Resource.Integration:
      return "/dispatch/integrations";
    case Resource.Setting:
      return "/dispatch/settings";
    case Resource.Template:
      return "/dispatch/templates";
    case Resource.Backup:
      return "/dispatch/backups";
    default:
      return "/";
  }
}

export const resourcePathMap = new Map<string, ResourcePageInfo>();

// Populate the map with all routes that have links
export function populateResourcePathMap(routeItems: RouteInfo[], prefix = "") {
  console.log("routeItems", routeItems);
  for (const route of routeItems) {
    if (route.link) {
      const resourceKey = route.key.toLowerCase();
      resourcePathMap.set(resourceKey, {
        path: route.link,
        supportsModal: route.supportsModal ?? false,
      });
    }

    if (route.tree) {
      populateResourcePathMap(route.tree, prefix);
    }
  }
}

/**
 * Get page info for a specific resource type from our navigation structure
 */
export function getRoutePageInfo(resourceType: ResourceType): ResourcePageInfo {
  console.log("resourceType", resourceType);
  const pageInfo = resourcePathMap.get(resourceType);

  console.log("pageInfo", pageInfo);

  if (!pageInfo) {
    // Default fallback
    return { path: "/", supportsModal: false };
  }

  return pageInfo;
}

export function convertValueToDisplay(value: any) {
  if (typeof value === "boolean") {
    return value ? "true" : "false";
  }

  if (typeof value === "object" && value !== null) {
    return JSON.stringify(value);
  }

  return value;
}
