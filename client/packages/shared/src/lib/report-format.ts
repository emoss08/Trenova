/**
 * Client-side mirror of the server's `pkg/reportfmt` rendering rules so a
 * builder preview, a CSV export, and a scheduled XLSX all show a column the
 * same way. The resolved display contract arrives with every report column;
 * this module only renders it.
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

export type ReportDisplayRule = {
  op: string;
  value: number;
  upper: number;
  tone: string;
};

/**
 * The ranges a banded dimension groups into. The row carries a range's lower
 * bound, never a value on its own, so the label is rebuilt from these bounds —
 * see `pkg/reportfmt/band.go`, which owns the same boundaries the query used.
 */
export type ReportBand = {
  width: number;
  edges: readonly number[];
};

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

/**
 * The same contract as it arrives from GraphQL, where every enum-ish field is
 * typed as a plain string.
 */
export type ReportColumnDisplayLike = {
  [K in keyof ReportColumnDisplay]: ReportColumnDisplay[K] extends string
    ? string
    : ReportColumnDisplay[K];
};

export type ReportDisplayOverrides = {
  style?: ReportDisplayStyle;
  decimals?: number;
  grouping?: boolean;
  currency?: string;
  negative?: ReportNegativeStyle;
  notation?: ReportNotation;
  prefix?: string;
  suffix?: string;
  dateStyle?: ReportDateStyle;
  boolStyle?: ReportBoolStyle;
  durationUnit?: ReportDurationUnit;
  durationStyle?: ReportDurationStyle;
  nullText?: string;
};

export type ResolveDisplayInput = {
  /** The value type the column emits: string, int, decimal, epoch, bool, … */
  type: string;
  /** Catalog format hint: money, weight, percent, duration, distance, count. */
  hint?: string;
  /** Default date style implied by a dimension's date bucket. */
  dateStyle?: ReportDateStyle;
  /** The ranges a banded dimension groups into, if it is banded. */
  band?: ReportBand | null;
  overrides?: ReportDisplayOverrides;
};

const CURRENCY_SYMBOLS: Record<string, string> = {
  USD: "$",
  CAD: "C$",
  MXN: "MX$",
  EUR: "€",
  GBP: "£",
  AUD: "A$",
  JPY: "¥",
};

const COMPACT_STEPS: { threshold: number; suffix: string }[] = [
  { threshold: 1e12, suffix: "T" },
  { threshold: 1e9, suffix: "B" },
  { threshold: 1e6, suffix: "M" },
  { threshold: 1e3, suffix: "K" },
];

const DURATION_UNIT_SECONDS: Record<ReportDurationUnit, number> = {
  "": 1,
  seconds: 1,
  minutes: 60,
  hours: 3600,
  days: 86400,
};

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

const WEEKDAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

export function currencySymbol(code: string): string {
  return CURRENCY_SYMBOLS[code] ?? (code ? `${code} ` : "$");
}

/**
 * Report values cross the wire as strings when they are decimals, so anything
 * plotting or comparing them has to coerce first.
 */
export function reportNumericValue(value: unknown): number | null {
  return toNumber(value);
}

function toNumber(value: unknown): number | null {
  if (typeof value === "number") return Number.isFinite(value) ? value : null;
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : null;
  }
  return null;
}

function group(digits: string, grouping: boolean): string {
  if (!grouping) return digits;
  const [intPart, fracPart] = digits.split(".");
  const grouped = intPart.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  return fracPart === undefined ? grouped : `${grouped}.${fracPart}`;
}

function signed(body: string, negative: boolean, style: ReportNegativeStyle): string {
  if (!negative || body === "") return body;
  return style === "parentheses" ? `(${body})` : `-${body}`;
}

function pad2(value: number): string {
  return value < 10 ? `0${value}` : String(value);
}

function formatDuration(value: number, display: ReportColumnDisplay): string {
  const seconds = value * (DURATION_UNIT_SECONDS[display.durationUnit] ?? 1);
  const negative = seconds < 0;
  const total = Math.abs(seconds);

  let body: string;
  switch (display.durationStyle) {
    case "decimalHours":
      body = `${group((total / 3600).toFixed(Math.max(display.decimals, 1)), display.grouping)} h`;
      break;
    case "compact":
      body = humanizeDuration(total, false);
      break;
    case "verbose":
      body = humanizeDuration(total, true);
      break;
    default: {
      const hours = Math.floor(total / 3600);
      const minutes = Math.floor((total % 3600) / 60);
      body = `${hours}:${pad2(minutes)}`;
    }
  }

  return signed(`${display.prefix}${body}${display.suffix}`, negative, display.negative);
}

function humanizeDuration(seconds: number, verbose: boolean): string {
  const units: { value: number; short: string; long: string }[] = [
    { value: Math.floor(seconds / 86400), short: "d", long: "day" },
    { value: Math.floor((seconds % 86400) / 3600), short: "h", long: "hour" },
    { value: Math.floor((seconds % 3600) / 60), short: "m", long: "minute" },
  ];

  const parts = units
    .filter((unit) => unit.value !== 0)
    .map((unit) =>
      verbose
        ? `${unit.value} ${unit.long}${unit.value === 1 ? "" : "s"}`
        : `${unit.value}${unit.short}`,
    );

  if (parts.length === 0) return verbose ? "0 minutes" : "0m";
  return parts.join(" ");
}

function formatNumeric(value: number, display: ReportColumnDisplay): string {
  if (display.style === "duration") return formatDuration(value, display);

  const scaled = display.style === "percent" ? value * 100 : value;
  const negative = scaled < 0;
  let magnitude = Math.abs(scaled);
  let compactSuffix = "";

  if (display.notation === "compact") {
    const step = COMPACT_STEPS.find((candidate) => magnitude >= candidate.threshold);
    if (step) {
      magnitude /= step.threshold;
      compactSuffix = step.suffix;
    }
  }

  const digits = group(magnitude.toFixed(display.decimals), display.grouping);
  const symbol = display.style === "currency" ? currencySymbol(display.currency) : "";
  const percentSign = display.style === "percent" ? "%" : "";
  const body = `${display.prefix}${symbol}${digits}${compactSuffix}${percentSign}${display.suffix}`;

  return signed(body, negative, display.negative);
}

type DateParts = {
  year: number;
  month: number;
  day: number;
  hour: number;
  minute: number;
  weekday: number;
};

// The server renders dates in the organization's timezone; the preview has to
// agree with the export, so the same zone is applied here when one is known.
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

function formatDate(
  epochSeconds: number,
  display: ReportColumnDisplay,
  timeZone: string | undefined,
): string {
  const date = new Date(epochSeconds * 1000);
  if (Number.isNaN(date.getTime())) return "";

  const { year, month, day, hour, minute, weekday } = dateParts(date, timeZone);
  const clock = `${pad2(hour)}:${pad2(minute)}`;
  const isoDate = `${year}-${pad2(month + 1)}-${pad2(day)}`;

  switch (display.dateStyle) {
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

function formatBool(value: boolean, style: ReportBoolStyle): string {
  switch (style) {
    case "trueFalse":
      return value ? "True" : "False";
    case "onOff":
      return value ? "On" : "Off";
    case "check":
      return value ? "✓" : "✗";
    case "oneZero":
      return value ? "1" : "0";
    default:
      return value ? "Yes" : "No";
  }
}

/**
 * Renders one cell. `display` is the resolved contract from the server; when it
 * is missing the value is rendered raw rather than guessed at.
 */
export function formatReportValue(
  value: unknown,
  raw: ReportColumnDisplayLike | null | undefined,
  timeZone?: string,
): string {
  const display = raw as ReportColumnDisplay | null | undefined;
  if (value === null || value === undefined) return display?.nullText ?? "";
  if (!display || display.style === "") return rawValue(value);

  if (display.style === "date") {
    const epoch = toNumber(value);
    return epoch === null ? rawValue(value) : formatDate(epoch, display, timeZone);
  }

  if (display.style === "bool") {
    if (typeof value === "boolean") return formatBool(value, display.boolStyle);
    if (value === "true" || value === "false") {
      return formatBool(value === "true", display.boolStyle);
    }
    return rawValue(value);
  }

  if (display.style === "text") {
    const text = rawValue(value);
    if (text === "") return display.nullText;
    return `${display.prefix}${text}${display.suffix}`;
  }

  const numeric = toNumber(value);
  if (numeric === null) return rawValue(value);
  if (bandIsSet(display.band)) return formatBand(numeric, display);
  return formatNumeric(numeric, display);
}

function bandIsSet(band: ReportBand | null | undefined): band is ReportBand {
  return !!band && (band.width > 0 || band.edges.length > 1);
}

/**
 * Rebuilds the range a grouped lower bound stands for. Both ends render through
 * the column's own display, so a banded currency reads "$0.00 – $1,000.00" and
 * a banded duration reads "0:00 – 2:00".
 */
function formatBand(value: number, display: ReportColumnDisplay): string {
  const [lower, upper] = bandBounds(value, display.band as ReportBand);
  // The last range is open-ended, which is both what the query did and the
  // honest reading of a top bucket.
  if (upper === null) return `${formatNumeric(lower, display)}+`;

  // A trailing unit covers the whole range, so it is written once: "10,000 –
  // 15,000 lbs", not "10,000 lbs – 15,000 lbs". A leading symbol leads each
  // end, which is how money ranges are conventionally written.
  const low = formatNumeric(lower, { ...display, suffix: "" });
  return `${low}${RANGE_SEPARATOR}${formatNumeric(upper, display)}`;
}

const RANGE_SEPARATOR = " – ";

function bandBounds(value: number, band: ReportBand): [number, number | null] {
  if (band.width > 0) return [value, value + band.width];

  let index = 0;
  for (let i = 0; i < band.edges.length; i++) {
    if (value >= band.edges[i]) index = i;
  }
  if (index === band.edges.length - 1) return [band.edges[index], null];
  return [band.edges[index], band.edges[index + 1]];
}

function rawValue(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return JSON.stringify(value);
}

const BASELINE: ReportColumnDisplay = {
  style: "text",
  decimals: 0,
  grouping: true,
  currency: "USD",
  negative: "minus",
  notation: "standard",
  prefix: "",
  suffix: "",
  dateStyle: "dateTime",
  boolStyle: "yesNo",
  durationUnit: "seconds",
  durationStyle: "clock",
  nullText: "",
};

function applyHint(display: ReportColumnDisplay, hint: string | undefined): ReportColumnDisplay {
  switch (hint) {
    case "money":
      return { ...display, style: "currency", decimals: 2, negative: "parentheses" };
    case "percent":
      return { ...display, style: "percent", decimals: 1 };
    case "duration":
      return { ...display, style: "duration", decimals: 0 };
    case "distance":
      return { ...display, decimals: 1, suffix: " mi" };
    case "weight":
      return { ...display, decimals: 0, suffix: " lbs" };
    case "count":
      return { ...display, decimals: 0 };
    default:
      return display;
  }
}

function restyle(display: ReportColumnDisplay, style: ReportDisplayStyle): ReportColumnDisplay {
  switch (style) {
    case "currency":
      return { ...display, style, decimals: 2, negative: "parentheses" };
    case "percent":
      return { ...display, style, decimals: 1 };
    case "duration":
      return { ...display, style, decimals: 0 };
    case "number":
      return { ...display, style, suffix: "" };
    default:
      return { ...display, style };
  }
}

/**
 * Mirrors reportfmt.Resolve so the builder can show what a column will look
 * like before the server round-trips it.
 */
export function resolveReportDisplay(input: ResolveDisplayInput): ReportColumnDisplay {
  let display: ReportColumnDisplay = { ...BASELINE };

  switch (input.type) {
    case "epoch":
      display = { ...display, style: "date" };
      break;
    case "bool":
      display = { ...display, style: "bool" };
      break;
    case "int":
      display = applyHint({ ...display, style: "number", decimals: 0 }, input.hint);
      break;
    case "decimal":
      display = applyHint({ ...display, style: "number", decimals: 2 }, input.hint);
      break;
    default:
      break;
  }

  if (input.dateStyle && display.style === "date") {
    display = { ...display, dateStyle: input.dateStyle };
  }
  if (bandIsSet(input.band)) {
    display = { ...display, band: input.band };
  }

  const overrides = input.overrides;
  if (!overrides) return display;

  if (overrides.style && overrides.style !== display.style) {
    display = restyle(display, overrides.style);
  }
  if (overrides.decimals !== undefined) {
    display = { ...display, decimals: Math.min(Math.max(overrides.decimals, 0), 10) };
  }
  if (overrides.grouping !== undefined) display = { ...display, grouping: overrides.grouping };
  if (overrides.currency) display = { ...display, currency: overrides.currency.toUpperCase() };
  if (overrides.negative) display = { ...display, negative: overrides.negative };
  if (overrides.notation) display = { ...display, notation: overrides.notation };
  if (overrides.prefix) display = { ...display, prefix: overrides.prefix };
  if (overrides.suffix) display = { ...display, suffix: overrides.suffix };
  if (overrides.dateStyle) display = { ...display, dateStyle: overrides.dateStyle };
  if (overrides.boolStyle) display = { ...display, boolStyle: overrides.boolStyle };
  if (overrides.durationUnit) display = { ...display, durationUnit: overrides.durationUnit };
  if (overrides.durationStyle) display = { ...display, durationStyle: overrides.durationStyle };
  if (overrides.nullText) display = { ...display, nullText: overrides.nullText };

  return display;
}

function ruleMatches(rule: ReportDisplayRule, value: number): boolean {
  switch (rule.op) {
    case "gt":
      return value > rule.value;
    case "gte":
      return value >= rule.value;
    case "lt":
      return value < rule.value;
    case "lte":
      return value <= rule.value;
    case "eq":
      return value === rule.value;
    case "between":
      return value >= rule.value && value <= rule.upper;
    default:
      return false;
  }
}

/**
 * The emphasis a value earns from the column's rules, mirroring
 * `reportfmt.Resolved.ToneFor` so the grid and the export agree. First match
 * wins; only numeric cells are banded.
 */
export function reportValueTone(
  value: unknown,
  raw: ReportColumnDisplayLike | null | undefined,
): ReportDisplayTone {
  const display = raw as ReportColumnDisplay | null | undefined;
  const rules = display?.rules;
  if (!rules || rules.length === 0) return "";

  const numeric = toNumber(value);
  if (numeric === null) return "";
  // Percent columns are stored as a fraction and shown times 100, so the
  // threshold the author typed is on the displayed scale.
  const scaled = display?.style === "percent" ? numeric * 100 : numeric;

  for (const rule of rules) {
    if (ruleMatches(rule, scaled)) return rule.tone as ReportDisplayTone;
  }
  return "";
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
