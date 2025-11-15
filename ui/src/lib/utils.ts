import { ResourcePageInfo, ResourceType, RouteInfo } from "@/types/nav-links";
import {
  faFileAlt,
  faFileContract,
  faFileExcel,
  faFileImage,
  faFilePdf,
  faFileWord,
} from "@fortawesome/pro-solid-svg-icons";
import { clsx, type ClassValue } from "clsx";
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
 * Truncates a string to a given length
 * @param str - The string to truncate
 * @param length - The length to truncate the string to
 * @returns {string}
 */
export function truncateText(str: string, length: number): string {
  return str?.length > length ? `${str.slice(0, length)}...` : str;
}

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

export function formatLocation(location?: LocationSchema) {
  if (!location) {
    return "";
  }

  const { state, addressLine1, addressLine2, city, postalCode } = location;

  const addressLine =
    addressLine1 + ", " + (addressLine2 ? `${addressLine2}, ` : "");
  const cityStateZip = `${city} ${state?.abbreviation}, ${postalCode}`;

  return `${addressLine} ${cityStateZip}`;
}

/** Helper function to safely convert decimal string to number */
export function toNumber(value: any): number {
  const num = Number(value);
  return isNaN(num) ? 0 : num; // Ensure it returns 0 if NaN
}

export const resourcePathMap = new Map<string, ResourcePageInfo>();

// Populate the map with all routes that have links
export function populateResourcePathMap(routeItems: RouteInfo[], prefix = "") {
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
  const pageInfo = resourcePathMap.get(resourceType);

  if (!pageInfo) {
    // Default fallback
    return { path: "/", supportsModal: false };
  }

  return pageInfo;
}

export function formatFileSize(bytes: number) {
  if (bytes === 0) return "0 Bytes";

  const k = 1024;
  const sizes = ["Bytes", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

const imageTypes = [
  ".png",
  "image/png",
  ".jpg",
  "image/jpeg",
  ".gif",
  "image/gif",
  ".webp",
  "image/webp",
];
const documentTypes = [".pdf", "application/pdf"];
const excelTypes = [
  ".xls",
  "application/excel",
  ".xlsx",
  "application/xlsx",
  "csv",
  "application/csv",
];
const wordTypes = [".doc", "application/word", ".docx", "application/docx"];

export function getFileIcon(fileType: string) {
  const type = fileType.toLowerCase();
  if (documentTypes.includes(type)) return faFilePdf;
  if (imageTypes.includes(type)) return faFileImage;
  if (excelTypes.includes(type)) return faFileExcel;
  if (wordTypes.includes(type)) return faFileWord;
  if (type.includes("contract")) return faFileContract;
  return faFileAlt;
}

type FileClass = {
  bgColor: string;
  iconColor: string;
  borderColor: string;
};

export function getFileClass(fileType: string): FileClass {
  const type = fileType.toLowerCase();
  if (documentTypes.includes(type))
    return {
      bgColor: "bg-red-600/20",
      iconColor: "text-red-600",
      borderColor: "border-red-600",
    };
  if (imageTypes.includes(type))
    return {
      bgColor: "bg-blue-600/20",
      iconColor: "text-blue-600",
      borderColor: "border-blue-600",
    };
  return {
    bgColor: "bg-muted-foreground/10",
    iconColor: "text-muted-foreground",
    borderColor: "border-muted-foreground",
  };
}

export function pluralize(word: string, count: number) {
  return count === 1 ? word : `${word}s`;
}

export function formatBytes(bytes: number) {
  if (bytes === 0) return "0 Bytes";

  const k = 1024;
  const sizes = ["Bytes", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

export function clamp(n: number, min: number, max: number) {
  return Math.max(min, Math.min(max, n));
}
