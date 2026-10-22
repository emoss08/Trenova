/**
 * The report display contract as the browser holds it, and the parts of rendering it that stay
 * native. Every rule (rounding, grouping, bands, tones, resolution) runs in the rules module
 * through `@trenova/shared/rules/report`; what is left here is the contract's types, writing a
 * date the module hands back in the report's time zone with `Intl`, float coercion for charts
 * and sorting, and the alignment check.
 */

export type ReportDisplayStyle =
  | ""
  | "number"
  | "currency"
  | "percent"
  | "duration"
  | "date"
  | "bool"
  | "text";

export type ReportNegativeStyle = "" | "minus" | "parentheses";

export type ReportNotation = "" | "standard" | "compact";

export type ReportDateStyle =
  | ""
  | "date"
  | "dateLong"
  | "dateTime"
  | "time"
  | "monthYear"
  | "quarter"
  | "year"
  | "weekday"
  | "iso";

export type ReportBoolStyle = "" | "yesNo" | "trueFalse" | "onOff" | "check" | "oneZero";

export type ReportDurationUnit = "" | "seconds" | "minutes" | "hours" | "days";

export type ReportDurationStyle = "" | "clock" | "compact" | "verbose" | "decimalHours";

export type ReportDisplayTone = "" | "positive" | "warning" | "negative" | "neutral";

/** A decimal as the contract carries it: a GraphQL float, or the module's exact string. */
type ContractDecimal = number | string;

/** A tone rule. The module writes its bounds as exact decimal strings. */
export type ReportDisplayRule = {
  op: string;
  value: ContractDecimal;
  upper?: ContractDecimal | null;
  tone: string;
};

/**
 * The ranges a banded dimension groups into: a fixed `width`, or ascending `edges` whose last
 * range is open-ended. The row carries a range's lower bound, which the module renders as the
 * range it stands for.
 */
export type ReportBand = {
  width?: ContractDecimal | null;
  edges?: readonly ContractDecimal[] | null;
};

/**
 * A resolved display, as the server sends it with every column (GraphQL) and as
 * `report.resolve` returns it (with `band` and `rules` absent when there are none).
 */
export type ReportColumnDisplay = {
  style: ReportDisplayStyle;
  decimals: number;
  grouping: boolean;
  currency: string;
  negative: ReportNegativeStyle;
  notation: ReportNotation;
  prefix: string;
  suffix: string;
  dateStyle: ReportDateStyle;
  boolStyle: ReportBoolStyle;
  durationUnit: ReportDurationUnit;
  durationStyle: ReportDurationStyle;
  nullText: string;
  rules?: readonly ReportDisplayRule[] | null;
  band?: ReportBand | null;
};

/** The same contract as GraphQL types it, with every enum-ish field a plain string. */
export type ReportColumnDisplayLike = {
  [K in keyof ReportColumnDisplay]: ReportColumnDisplay[K] extends string
    ? string
    : ReportColumnDisplay[K];
};

/**
 * Report values cross the wire as strings when they are decimals, so anything plotting or
 * comparing them has to coerce first.
 */
export function reportNumericValue(value: unknown): number | null {
  if (typeof value === "number") return Number.isFinite(value) ? value : null;
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : null;
  }
  return null;
}

/** Numeric-leaning columns are right-aligned and tabular in grids. */
export function isNumericDisplay(display: ReportColumnDisplayLike | null | undefined): boolean {
  if (!display) return false;
  return (
    display.style === "number" ||
    display.style === "currency" ||
    display.style === "percent" ||
    display.style === "duration"
  );
}

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

const WEEKDAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

function pad2(value: number): string {
  return value < 10 ? `0${value}` : String(value);
}

type DateParts = {
  year: number;
  month: number;
  day: number;
  hour: number;
  minute: number;
  weekday: number;
};

// The server writes dates in the organization's time zone; the preview has to agree with the
// export, so the same zone is applied here when one is known.
function dateParts(date: Date, timeZone: string | undefined): DateParts {
  if (!timeZone) {
    return {
      year: date.getFullYear(),
      month: date.getMonth(),
      day: date.getDate(),
      hour: date.getHours(),
      minute: date.getMinutes(),
      weekday: date.getDay(),
    };
  }

  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    weekday: "short",
    hour12: false,
  }).formatToParts(date);

  const lookup = (type: Intl.DateTimeFormatPartTypes) =>
    parts.find((part) => part.type === type)?.value ?? "";

  return {
    year: Number(lookup("year")),
    month: Number(lookup("month")) - 1,
    day: Number(lookup("day")),
    hour: Number(lookup("hour")) % 24,
    minute: Number(lookup("minute")),
    weekday: Math.max(WEEKDAYS.indexOf(lookup("weekday")), 0),
  };
}

/**
 * Writes a date the module handed back as whole epoch seconds, in the column's date style and
 * the report's time zone, the way the server's Go layouts write it.
 */
export function formatReportDate(
  epochSeconds: number,
  dateStyle: string,
  timeZone?: string,
): string {
  const date = new Date(epochSeconds * 1000);
  if (Number.isNaN(date.getTime())) return "";

  const { year, month, day, hour, minute, weekday } = dateParts(date, timeZone);
  const clock = `${pad2(hour)}:${pad2(minute)}`;
  const isoDate = `${year}-${pad2(month + 1)}-${pad2(day)}`;

  switch (dateStyle) {
    case "date":
      return isoDate;
    case "dateLong":
      return `${MONTHS[month]} ${day}, ${year}`;
    case "time":
      return clock;
    case "monthYear":
      return `${MONTHS[month]} ${year}`;
    case "quarter":
      return `Q${Math.floor(month / 3) + 1} ${year}`;
    case "year":
      return String(year);
    case "weekday":
      return `${WEEKDAYS[weekday]} ${isoDate}`;
    case "iso":
      return date.toISOString();
    default:
      return `${isoDate} ${clock}`;
  }
}
