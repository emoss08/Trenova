import type { Location } from "@trenova/shared/types/location";
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { StoreApi, UseBoundStore } from "zustand";

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

export function parseCommaSeparatedList(value: string): string[] {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

export function parseWhitespaceSeparatedList(value: string): string[] {
  return value
    .split(/\s+/)
    .map((item) => item.trim())
    .filter(Boolean);
}

export function formatIdentityProviderName(name: string): string {
  return name
    .replace(/azure\s*ad/gi, "Entra ID")
    .replace(/microsoft entra id/gi, "Microsoft Entra ID");
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

export function formatCompactCurrency(num: number, currency: string = "USD"): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: currency,
    notation: "compact",
    maximumFractionDigits: 1,
  }).format(num);
}

export function formatPercent(value: number, digits: number = 1): string {
  return `${value.toFixed(digits)}%`;
}

export function formatPerMile(value: number, digits: number = 2, currency: string = "USD"): string {
  const formatted = new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: currency,
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(value);
  return `${formatted}/mi`;
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

export function downloadTextFile(filename: string, contents: string, type = "text/plain"): void {
  const blob = new Blob([contents], { type });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

export function downloadJsonFile(filename: string, data: unknown): void {
  downloadTextFile(filename, JSON.stringify(data, null, 2), "application/json");
}

export function formatFileSize(bytes: number): string {
  if (bytes <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / 1024 ** exponent;
  return `${value >= 100 || exponent === 0 ? Math.round(value) : value.toFixed(1)} ${units[exponent]}`;
}

export function findDuplicateIds<T>(
  items: T[],
  getId: (item: T) => string | undefined,
): Set<string> {
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

type WithSelectors<S> = S extends { getState: () => infer T }
  ? S & { use: { [K in keyof T]: () => T[K] } }
  : never;

export const createSelectors = <S extends UseBoundStore<StoreApi<object>>>(_store: S) => {
  const store = _store as WithSelectors<typeof _store>;
  store.use = {};
  for (const k of Object.keys(store.getState())) {
    (store.use as any)[k] = () => store((s) => s[k as keyof typeof s]);
  }

  return store;
};
