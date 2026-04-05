import type { Location } from "@/types/location";
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

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
    .replace(/([A-Z]+)([A-Z][a-z])/g, "$1 $2") // Split runs like "XMLParser" → "XML Parser"
    .replace(/([a-z\d])([A-Z])/g, "$1 $2") // Split camelCase like "firstName" → "first Name"
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

export function pluralize(word: string, count: number) {
  return count === 1 ? word : `${word}s`;
}

export function upperFirst(str: string): string {
  if (!str) return "";
  return str.charAt(0).toUpperCase() + str.slice(1);
}

export function truncateText(str: string, length: number): string {
  return str?.length > length ? `${str.slice(0, length)}...` : str;
}

export function formatCurrency(num: number, currency: string = "USD"): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: currency,
  }).format(num);
}

export function formatLocation(location?: Location) {
  if (!location) {
    return "";
  }

  const { state, addressLine1, addressLine2, city, postalCode } = location;

  const parts: string[] = [addressLine1];
  if (addressLine2) parts.push(addressLine2);
  const stateAbbr = state?.abbreviation;
  const cityPart = city || "";
  const cityState = stateAbbr ? (cityPart ? `${cityPart}, ${stateAbbr}` : stateAbbr) : cityPart;
  const zip = postalCode || "";
  const lastPart = cityState && zip ? `${cityState} ${zip}` : cityState || zip;
  if (lastPart) parts.push(lastPart);
  return parts.filter(Boolean).join(", ");
}

export const initials = (first?: string, last?: string) =>
  `${(first?.[0] ?? "").toUpperCase()}${(last?.[0] ?? "").toUpperCase()}`.trim() || "•";

export function getNameInitials(name?: string, fallback = "U") {
  if (!name) {
    return fallback;
  }

  const letters = name
    .split(" ")
    .map((part) => part[0])
    .filter(Boolean)
    .join("")
    .toUpperCase()
    .slice(0, 2);

  return letters || fallback;
}

export function isAbsoluteUrl(value?: string | null) {
  return Boolean(value) && (value!.startsWith("http://") || value!.startsWith("https://"));
}

export function findDuplicateIds<T>(items: T[], getId: (item: T) => string | undefined): Set<string> {
  const seen = new Set<string>();
  const dupes = new Set<string>();
  for (const item of items) {
    const id = getId(item);
    if (!id) continue;
    if (seen.has(id)) {
      dupes.add(id);
    } else {
      seen.add(id);
    }
  }
  return dupes;
}
