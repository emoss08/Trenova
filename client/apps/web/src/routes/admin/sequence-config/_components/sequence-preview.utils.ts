import {
  identifierCharacters,
  type LocationCodeComponent,
  type LocationCodeStrategy,
  type SequenceConfig,
} from "@/types/sequence-config";
import { drawsRecordCodes } from "./sequence-config-constants";

/** What a piece of a sample number is, so the preview can name it and open its setting. */
export type SequencePart =
  | "prefix"
  | "businessUnit"
  | "year"
  | "month"
  | "week"
  | "day"
  | "location"
  | "counter"
  | "random"
  | "checkDigit"
  | "component"
  | "text"
  | "separator";

export type PreviewSegment = {
  part: SequencePart;
  text: string;
  /** The setting that puts this piece there, relative to its sequence (`prefix`). */
  field?: string;
  /** A location code's part, for its name. */
  component?: LocationCodeComponent;
};

/** The codes and digits a sample stands in with, since the real ones exist only when issued. */
export const SAMPLE = {
  businessUnitCode: "HQ",
  locationCode: "DAL",
  randomDigits: "4821739056",
  location: {
    name: "Acme Warehouse",
    city: "Dallas",
    state: "TX",
    postal_code: "75201",
  } satisfies Record<LocationCodeComponent, string>,
} as const;

const pad = (value: number, width: number) => String(value).padStart(width, "0");

function yearText(digits: number, now: Date): string {
  return digits === 2 ? pad(now.getUTCFullYear() % 100, 2) : pad(now.getUTCFullYear(), 4);
}

/** The ISO 8601 week of a date in UTC, as Go's `Time.ISOWeek` gives it. */
export function isoWeek(now: Date): number {
  const day = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()));
  const weekday = day.getUTCDay() || 7;
  day.setUTCDate(day.getUTCDate() + 4 - weekday);
  const yearStart = Date.UTC(day.getUTCFullYear(), 0, 1);
  return Math.ceil(((day.getTime() - yearStart) / 86_400_000 + 1) / 7);
}

/** The Luhn check digit over the digits of a value, as the server computes it. */
export function luhnCheckDigit(input: string): number {
  let sum = 0;
  let double = true;
  for (let index = input.length - 1; index >= 0; index -= 1) {
    const char = input.charCodeAt(index);
    if (char < 48 || char > 57) continue;
    let digit = char - 48;
    if (double) {
      digit *= 2;
      if (digit > 9) digit -= 9;
    }
    sum += digit;
    double = !double;
  }
  return (10 - (sum % 10)) % 10;
}

function counterText(digits: number): string {
  return "1".padStart(Math.max(1, digits || 1), "0");
}

function joinWith(segments: PreviewSegment[], separator: string): PreviewSegment[] {
  if (!separator) return segments;
  return segments.flatMap((segment, index) =>
    index === 0 ? [segment] : [{ part: "separator", text: separator }, segment],
  );
}

export function segmentsText(segments: readonly PreviewSegment[]): string {
  return segments.map((segment) => segment.text).join("");
}

/**
 * The first number a sequence issues in the month of `now`, piece by piece, in the order
 * and shape the server's generator writes it (`pkg/seqgen`).
 */
export function sequenceSegments(cfg: SequenceConfig, now: Date): PreviewSegment[] {
  if (cfg.sequenceType === "location_code") {
    return locationCodeSegments(cfg.locationCodeStrategy);
  }
  if (cfg.allowCustomFormat && cfg.customFormat.trim()) {
    return patternSegments(cfg, now);
  }

  const codes = drawsRecordCodes(cfg.sequenceType);
  const parts: PreviewSegment[] = [];
  if (cfg.prefix) parts.push({ part: "prefix", text: cfg.prefix, field: "prefix" });
  if (codes && cfg.includeBusinessUnitCode) {
    parts.push({
      part: "businessUnit",
      text: SAMPLE.businessUnitCode,
      field: "includeBusinessUnitCode",
    });
  }
  if (cfg.includeYear) {
    parts.push({ part: "year", text: yearText(cfg.yearDigits, now), field: "includeYear" });
  }
  if (cfg.includeWeekNumber) {
    parts.push({ part: "week", text: pad(isoWeek(now), 2), field: "includeWeekNumber" });
  } else if (cfg.includeMonth) {
    parts.push({ part: "month", text: pad(now.getUTCMonth() + 1, 2), field: "includeMonth" });
  }
  if (cfg.includeDay) {
    parts.push({ part: "day", text: pad(now.getUTCDate(), 2), field: "includeDay" });
  }
  if (codes && cfg.includeLocationCode) {
    parts.push({ part: "location", text: SAMPLE.locationCode, field: "includeLocationCode" });
  }
  parts.push({ part: "counter", text: counterText(cfg.sequenceDigits), field: "sequenceDigits" });
  if (cfg.includeRandomDigits && cfg.randomDigitsCount > 0) {
    parts.push({
      part: "random",
      text: SAMPLE.randomDigits.slice(0, cfg.randomDigitsCount),
      field: "randomDigitsCount",
    });
  }

  const separator = cfg.useSeparators ? cfg.separatorChar : "";
  const joined = joinWith(parts, separator);
  if (!cfg.includeCheckDigit) return joined;

  const check: PreviewSegment = {
    part: "checkDigit",
    text: String(luhnCheckDigit(segmentsText(joined))),
    field: "includeCheckDigit",
  };
  return separator
    ? [...joined, { part: "separator", text: separator }, check]
    : [...joined, check];
}

const tokenPattern = /\{([PBYMWDLSRC])\}/g;

function patternSegments(cfg: SequenceConfig, now: Date): PreviewSegment[] {
  const codes = drawsRecordCodes(cfg.sequenceType);
  const token = (letter: string): PreviewSegment => {
    switch (letter) {
      case "P":
        return { part: "prefix", text: cfg.prefix, field: "prefix" };
      case "B":
        return { part: "businessUnit", text: codes ? SAMPLE.businessUnitCode : "" };
      case "Y":
        return { part: "year", text: yearText(cfg.yearDigits, now), field: "yearDigits" };
      case "M":
        return { part: "month", text: pad(now.getUTCMonth() + 1, 2) };
      case "W":
        return { part: "week", text: pad(isoWeek(now), 2) };
      case "D":
        return { part: "day", text: pad(now.getUTCDate(), 2) };
      case "L":
        return { part: "location", text: codes ? SAMPLE.locationCode : "" };
      case "S":
        return { part: "counter", text: counterText(cfg.sequenceDigits), field: "sequenceDigits" };
      case "R":
        return {
          part: "random",
          text:
            cfg.includeRandomDigits && cfg.randomDigitsCount > 0
              ? SAMPLE.randomDigits.slice(0, cfg.randomDigitsCount)
              : "",
          field: "includeRandomDigits",
        };
      default:
        return { part: "checkDigit", text: "", field: "includeCheckDigit" };
    }
  };

  const segments: PreviewSegment[] = [];
  let last = 0;
  for (const match of cfg.customFormat.matchAll(tokenPattern)) {
    if (match.index > last) {
      segments.push({ part: "text", text: cfg.customFormat.slice(last, match.index) });
    }
    segments.push(token(match[1]!));
    last = match.index + match[0].length;
  }
  if (last < cfg.customFormat.length) {
    segments.push({ part: "text", text: cfg.customFormat.slice(last) });
  }

  if (cfg.includeCheckDigit) {
    const digit = String(luhnCheckDigit(segmentsText(segments)));
    for (const segment of segments) {
      if (segment.part === "checkDigit") segment.text = digit;
    }
  }

  return segments
    .filter((segment) => segment.text !== "")
    .map((segment) => (segment.part === "text" ? { ...segment, field: "customFormat" } : segment));
}

function normalizedToken(value: string, casing: LocationCodeStrategy["casing"]): string {
  const token = identifierCharacters(value).toUpperCase();
  return casing === "lower" ? token.toLowerCase() : token;
}

const truncate = (value: string, width: number) => Array.from(value).slice(0, width).join("");

/**
 * The first code issued for the sample location, as the server's location code generator
 * builds it: each chosen part cut to its width, a fallback for a part the location lacks,
 * then the counter.
 */
export function locationCodeSegments(
  strategy: LocationCodeStrategy | null | undefined,
): PreviewSegment[] {
  if (!strategy) return [];
  const width = Math.max(1, strategy.componentWidth || 1);
  const fallback = normalizedToken(strategy.fallbackPrefix.trim(), strategy.casing);
  const parts: PreviewSegment[] = [];
  let lastWasFallback = false;

  for (const component of strategy.components) {
    const token = normalizedToken(SAMPLE.location[component], strategy.casing);
    const usedFallback = token === "";
    if (usedFallback && lastWasFallback) continue;
    parts.push({
      part: "component",
      component,
      text: truncate(usedFallback ? fallback : token, width),
      field: "locationCodeStrategy.components",
    });
    lastWasFallback = usedFallback;
  }
  parts.push({
    part: "counter",
    text: counterText(strategy.sequenceDigits),
    field: "locationCodeStrategy.sequenceDigits",
  });

  return joinWith(parts, strategy.separator);
}

/**
 * Whether two numbers of this sequence can come out the same. The server starts every
 * counter again at 1 each calendar month, so a number is unique only while it carries the
 * year and the month; a week in the month's place does not do, since a week can span two
 * months. A location code's counter never restarts.
 */
export function numbersCanRepeat(cfg: SequenceConfig): boolean {
  if (cfg.sequenceType === "location_code") return false;
  if (cfg.allowCustomFormat && cfg.customFormat.trim()) {
    return !cfg.customFormat.includes("{Y}") || !cfg.customFormat.includes("{M}");
  }
  return !cfg.includeYear || !cfg.includeMonth || cfg.includeWeekNumber;
}
