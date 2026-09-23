import { isAppPath } from "@/lib/app-path";
import { formatMetricValue } from "@/routes/home/_components/widgets/insight-presentation";
import { insightUnitSchema } from "@/types/insight";
import { formatNumber } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { phaseTone, type StatusPhase } from "@trenova/shared/lib/status-phase";
import { formatCurrency, formatPercent } from "@trenova/shared/lib/utils";
import type { BadgeTone } from "@trenova/shared/types/badge";

/*
 * What a person reads of a record the assistant looked at.
 *
 * A tool's result is written for the model, which needs every record's id, the
 * tenant, the version a write checks and which way a metric counts as worse. A
 * person needs none of it, and shown it they got a column of
 * "inst_01M37R101VKZTB7TSKR30FJ0AT", a cell of raw JSON and a detected date of
 * 1790187600.
 *
 * The server projects a new artifact before it stores it
 * (assistantservice/artifact_display.go). These are the same rules, for what was
 * stored before that and for the conversation's step details, which read the
 * model's own copy. Change one side and change the other.
 */

/** How a column or a field reads. The set matches `assistantartifact.AllDisplayTypes`. */
export const DISPLAY_TYPES = [
  "text",
  "longText",
  "date",
  "datetime",
  "money",
  "number",
  "percent",
  "enum",
  "status",
  "boolean",
  "flag",
  "metrics",
  "links",
] as const;

export type DisplayType = (typeof DISPLAY_TYPES)[number];

export type DisplayColumn = { key: string; label: string; type: DisplayType };

export type DisplayField = DisplayColumn & { value: unknown };

export type DisplayMetric = { label: string; value: string | number; unit: string };

export type DisplayLink = { label: string; path: string; count: number };

/** Types read in a row's detail or below a card's fields rather than in a cell. */
export function isDetailType(type: DisplayType): boolean {
  return type === "longText" || type === "metrics" || type === "links";
}

/** Types whose values line up on the right, as figures do. */
export function isFigureType(type: DisplayType): boolean {
  return type === "money" || type === "number" || type === "percent";
}

/** Where a projected row keeps the id its link is built from. Never a column. */
export const RECORD_ID_KEY = "id";

const LONG_TEXT_LENGTH = 120;
const MAX_CODE_LENGTH = 40;

const HIDDEN_KEYS = new Set([
  "id",
  "version",
  "organizationid",
  "businessunitid",
  "tenantid",
  "direction",
  "dedupekey",
  "narrated",
  "modelidentifier",
  "truncated",
  "hasmore",
  "nextoffset",
  "offset",
]);

const FLAG_KEYS = new Set(["stale"]);

/** Keys that name a record, most specific first; a table leads with them. */
const LEAD_KEYS = [
  "proNumber",
  "invoiceNumber",
  "number",
  "referenceNumber",
  "name",
  "fullName",
  "displayName",
  "title",
  "headline",
  "subject",
  "label",
  "code",
  "unitNumber",
  "licenseNumber",
];

/** Keys tried in order for the words that name a record. */
const RECORD_LABEL_KEYS = [
  "proNumber",
  "invoiceNumber",
  "referenceNumber",
  "name",
  "displayName",
  "fullName",
  "code",
  "unitNumber",
  "licenseNumber",
  "number",
  "title",
  "label",
  "headline",
  "subject",
];

const ID_WORDS = ["Id", "ID", "Ids", "IDs"];
const DATE_TIME_WORDS = [
  "At",
  "Time",
  "Timestamp",
  "Start",
  "End",
  "Arrival",
  "Departure",
  "Eta",
  "Since",
  "Until",
  "Cutoff",
  "For",
];
const DATE_WORDS = [
  "Date",
  "Expiry",
  "Expires",
  "On",
  "Due",
  "Check",
  "From",
  "To",
  "Through",
  "Deadline",
  "Of",
  "Dob",
];
const STATUS_WORDS = ["Status", "Severity", "Priority", "Standing"];
const ENUM_WORDS = [
  "Category",
  "Type",
  "Kind",
  "Method",
  "Class",
  "Mode",
  "Side",
  "Source",
  "Channel",
  "Tier",
  "Level",
  "Role",
  "Unit",
];
const MONEY_WORDS = [
  "Amount",
  "Charge",
  "Charges",
  "Cost",
  "Revenue",
  "Price",
  "Balance",
  "Fee",
  "Fees",
  "Pay",
];
const DECIMAL_MONEY_WORDS = ["Total", "Subtotal"];
const PERCENT_WORDS = ["Percent", "Percentage", "Pct"];
const LABEL_NUMBER_WORDS = ["Year", "Number", "Code", "Zip", "PostalCode", "Phone"];
const PROSE_KEYS = new Set([
  "narrative",
  "recommendation",
  "body",
  "content",
  "explanation",
  "rationale",
  "instructions",
]);

/** Initialisms a label keeps in capitals. */
const ACRONYMS = new Set([
  "id",
  "url",
  "uri",
  "api",
  "ui",
  "ux",
  "ip",
  "sql",
  "edi",
  "scac",
  "dot",
  "vin",
  "pto",
  "gl",
  "pdf",
  "csv",
  "json",
  "bol",
  "cdl",
  "mvr",
  "twic",
  "eta",
  "hos",
  "mc",
  "un",
]);

/** A record's identifier: a lowercase prefix, an underscore and a ULID. */
const PULID = /^[a-z][a-z0-9]{0,10}_[0-9A-HJKMNP-TV-Z]{26}$/;
const DECIMAL = /^[+-]?(?:\d+(?:\.\d*)?|\.\d+)$/;
const WRITTEN_DATE = /^\d{4}-\d{2}-\d{2}/;
const CODE = /^[A-Za-z][A-Za-z0-9_]*$/;
const UPPER_CODE = /^[A-Z0-9]+$/;
const EARLIEST_INSTANT = 946_684_800;
const LATEST_INSTANT = 4_102_444_800;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/** Whether a value is a record's id, which a person never needs to read. */
export function isRecordId(value: unknown): boolean {
  return typeof value === "string" && PULID.test(value.trim());
}

/** Whether key is one of words, or ends in one as a camel-case word of its own. */
function endsWithWord(key: string, words: readonly string[]): boolean {
  const lowered = key.toLowerCase();
  for (const word of words) {
    if (lowered === word.toLowerCase()) {
      return true;
    }
    const start = key.length - word.length;
    if (start < 1 || !key.endsWith(word)) {
      continue;
    }
    if (/[a-z0-9]/.test(key.charAt(start - 1))) {
      return true;
    }
  }

  return false;
}

/** Keys that say nothing a person can use: ids, the tenant, a write's version. */
export function isHiddenKey(key: string): boolean {
  const lowered = key.toLowerCase();

  return (
    HIDDEN_KEYS.has(lowered) ||
    endsWithWord(key, ID_WORDS) ||
    lowered.endsWith("_id") ||
    lowered.endsWith("_ids")
  );
}

/** Whether a key names an instant, and whether its hour matters. */
export function datedKey(key: string): "date" | "datetime" | null {
  if (endsWithWord(key, DATE_TIME_WORDS)) return "datetime";
  if (endsWithWord(key, DATE_WORDS)) return "date";
  return null;
}

function isPlausibleInstant(value: number): boolean {
  return Number.isInteger(value) && value >= EARLIEST_INSTANT && value <= LATEST_INSTANT;
}

/** A value with nothing to read. Under a date's name a zero means "not set". */
function isEmptyValue(value: unknown, dated: boolean): boolean {
  if (value === null || value === undefined) return true;
  if (typeof value === "string") return value.trim() === "";
  if (Array.isArray(value)) return value.length === 0;
  if (isRecord(value)) return Object.keys(value).length === 0;
  if (typeof value === "number") return dated && value === 0;
  return false;
}

type Shape = "scalar" | "bool" | "list" | "object";

function shapeOf(values: readonly unknown[]): Shape {
  let shape: Shape = "scalar";
  for (const [index, value] of values.entries()) {
    const current: Shape =
      typeof value === "boolean"
        ? "bool"
        : Array.isArray(value)
          ? "list"
          : isRecord(value)
            ? "object"
            : "scalar";
    if (index > 0 && current !== shape) {
      return "scalar";
    }
    shape = current;
  }

  return shape;
}

function isMetric(value: unknown): value is Record<string, unknown> {
  return isRecord(value) && stringOf(value.label) !== "" && "value" in value;
}

function isLink(value: unknown): value is Record<string, unknown> {
  return isRecord(value) && stringOf(value.label) !== "" && stringOf(value.path) !== "";
}

function listType(values: readonly unknown[]): DisplayType | null {
  let metrics = true;
  let links = true;
  let words = true;
  for (const value of values) {
    for (const entry of Array.isArray(value) ? value : []) {
      metrics &&= isMetric(entry);
      links &&= isLink(entry);
      words &&= typeof entry === "string" || typeof entry === "number";
    }
  }
  if (metrics) return "metrics";
  if (links) return "links";
  if (words) return "text";
  return null;
}

function stringOf(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function numericOf(value: unknown): string | number | null {
  if (typeof value === "number") return Number.isFinite(value) ? value : null;
  if (typeof value === "string") {
    const text = value.trim();
    return DECIMAL.test(text) ? text : null;
  }
  return null;
}

function allInstants(values: readonly unknown[]): boolean {
  let stamped = false;
  let written = true;
  for (const value of values) {
    if (typeof value === "number") {
      if (!isPlausibleInstant(value)) return false;
      stamped = true;
    } else if (typeof value === "string") {
      const text = value.trim();
      if (DECIMAL.test(text)) return false;
      written &&= WRITTEN_DATE.test(text);
    } else {
      return false;
    }
  }

  return stamped || written;
}

function isMoneyKey(key: string, values: readonly unknown[], amountIsMoney: boolean): boolean {
  if (endsWithWord(key, DECIMAL_MONEY_WORDS)) {
    return values.every((value) => typeof value === "string");
  }
  if (!endsWithWord(key, MONEY_WORDS)) {
    return false;
  }

  return amountIsMoney || key.toLowerCase() !== "amount";
}

function isCode(value: unknown): boolean {
  return typeof value === "string" && value.length <= MAX_CODE_LENGTH && CODE.test(value);
}

function isProse(key: string, values: readonly unknown[]): boolean {
  if (PROSE_KEYS.has(key.toLowerCase())) return true;
  return values.some(
    (value) =>
      typeof value === "string" &&
      value.length > LONG_TEXT_LENGTH &&
      [...value].length > LONG_TEXT_LENGTH,
  );
}

/**
 * How a column reads, from its key and every value in it, or null when it is
 * not shown: an id under any name, a column of nothing but ids, the tenant, a
 * flag that never holds, nested records with no name, or a column with nothing
 * in it.
 */
export function classifyValues(
  key: string,
  values: readonly unknown[],
  amountIsMoney = true,
): DisplayType | null {
  if (isHiddenKey(key)) return null;

  const dated = datedKey(key);
  const present = values.filter((value) => !isEmptyValue(value, dated !== null));
  if (present.length === 0 || present.every(isRecordId)) return null;

  if (FLAG_KEYS.has(key.toLowerCase())) {
    return present.some((value) => value === true) ? "flag" : null;
  }

  switch (shapeOf(present)) {
    case "bool":
      return "boolean";
    case "list":
      return listType(present);
    case "object":
      return present.every((value) => isRecord(value) && recordLabel(value) !== "") ? "text" : null;
    case "scalar":
      break;
  }

  const numeric = present.every((value) => numericOf(value) !== null);
  if (dated !== null && allInstants(present)) return dated;
  if (numeric && isMoneyKey(key, present, amountIsMoney)) return "money";
  if (numeric && endsWithWord(key, PERCENT_WORDS)) return "percent";
  if (endsWithWord(key, STATUS_WORDS) && present.every(isCode)) return "status";
  if (endsWithWord(key, ENUM_WORDS) && present.every(isCode)) return "enum";
  if (
    present.every((value) => typeof value === "number") &&
    !endsWithWord(key, LABEL_NUMBER_WORDS)
  ) {
    return "number";
  }
  if (isProse(key, present)) return "longText";
  return "text";
}

/** A string a person can read: never a record's id. */
function readableString(value: unknown): string {
  const text = stringOf(value);
  return isRecordId(text) ? "" : text;
}

/** A value as words: a number as written, a list as its members, a record as its name. */
function textOf(value: unknown): string {
  if (typeof value === "string") return readableString(value);
  if (typeof value === "number") return Number.isFinite(value) ? String(value) : "";
  if (Array.isArray(value)) {
    return value
      .filter((entry) => typeof entry === "string" || typeof entry === "number")
      .map(textOf)
      .filter((text) => text !== "")
      .join(", ");
  }
  if (isRecord(value)) return recordLabel(value);
  return "";
}

function metricsOf(value: unknown): DisplayMetric[] {
  return (Array.isArray(value) ? value : []).flatMap((entry) => {
    if (!isMetric(entry)) return [];
    const figure = numericOf(entry.value);
    return figure === null
      ? []
      : [{ label: stringOf(entry.label), value: figure, unit: stringOf(entry.unit) }];
  });
}

function linksOf(value: unknown): DisplayLink[] {
  return (Array.isArray(value) ? value : []).flatMap((entry) => {
    if (!isLink(entry)) return [];
    const path = stringOf(entry.path);
    if (!isAppPath(path)) return [];
    const count = typeof entry.count === "number" && entry.count > 0 ? entry.count : 0;
    return [{ label: stringOf(entry.label), path, count }];
  });
}

/**
 * One value as its type draws it, or undefined when there is nothing to draw.
 * It reads a value the server already projected the same way, so a stored row
 * and a raw one come out alike.
 */
export function projectValue(type: DisplayType, value: unknown): unknown {
  switch (type) {
    case "text": {
      const text = textOf(value);
      return text === "" ? undefined : text;
    }
    case "longText":
    case "enum":
    case "status": {
      const text = readableString(value);
      return text === "" ? undefined : text;
    }
    case "date":
    case "datetime":
      if (typeof value === "number") return isPlausibleInstant(value) ? value : undefined;
      return stringOf(value) === "" ? undefined : stringOf(value);
    case "money":
    case "percent":
    case "number":
      return numericOf(value) ?? undefined;
    case "boolean":
      return typeof value === "boolean" ? value : undefined;
    case "flag":
      return value === true ? true : undefined;
    case "metrics": {
      const metrics = metricsOf(value);
      return metrics.length === 0 ? undefined : metrics;
    }
    case "links": {
      const links = linksOf(value);
      return links.length === 0 ? undefined : links;
    }
  }
}

/** The keys that name a record first, the rest where the record put them. */
export function leadFirst(keys: readonly string[]): string[] {
  const present = new Set(keys);
  const leads = LEAD_KEYS.filter((key) => present.has(key));
  const leadSet = new Set(leads);

  return [...leads, ...keys.filter((key) => !leadSet.has(key))];
}

/**
 * A key as a label: "customerId" reads "Customer ID", "proNumber" reads "Pro
 * number", "dotNumber" reads "DOT number". Sentence case, initialisms kept.
 */
export function humanizeKey(key: string): string {
  const words = key
    .replace(/[_.-]+/g, " ")
    .replace(/([A-Z]+)([A-Z][a-z])/g, "$1 $2")
    .replace(/([a-z\d])([A-Z])/g, "$1 $2")
    .trim()
    .split(/\s+/)
    .filter((word) => word !== "");

  return words
    .map((word, index) => {
      const lowered = word.toLowerCase();
      if (ACRONYMS.has(lowered)) return lowered.toUpperCase();
      return index === 0 ? lowered.charAt(0).toUpperCase() + lowered.slice(1) : lowered;
    })
    .join(" ");
}

/**
 * A column head. A date says what happened rather than when: "detectedOn"
 * reads "Detected", "createdAt" reads "Created".
 */
export function displayLabel(key: string, type: DisplayType): string {
  if (type === "date" || type === "datetime") {
    for (const word of ["At", "On"]) {
      if (key.length > word.length && endsWithWord(key, [word])) {
        return humanizeKey(key.slice(0, -word.length));
      }
    }
  }

  return humanizeKey(key);
}

/**
 * A member of a set as words: "InTransit" reads "In transit". A code that is
 * all capitals — "OTR", "W2" — is already how people say it.
 */
export function humanizeCode(value: string): string {
  return UPPER_CODE.test(value) ? value : humanizeKey(value);
}

/** What a record is called, or "" when nothing names it. Never its id. */
export function recordLabel(record: unknown): string {
  if (!isRecord(record)) {
    return typeof record === "string" ? readableString(record) : "";
  }
  for (const key of RECORD_LABEL_KEYS) {
    const value = readableString(record[key]);
    if (value !== "") return value;
  }

  return `${readableString(record.firstName)} ${readableString(record.lastName)}`.trim();
}

/*
 * Where a status sits in its lifecycle, so its badge takes the phase's tone
 * rather than a colour chosen for it. Keys are lowered with separators removed.
 * A value not listed is neutral: a badge that guesses a severity says
 * something false.
 */
const STATUS_PHASES: Readonly<Record<string, StatusPhase>> = {
  draft: "draft",
  new: "draft",
  unassigned: "draft",
  info: "draft",
  planned: "queued",
  assigned: "queued",
  queued: "queued",
  scheduled: "queued",
  readytoinvoice: "queued",
  active: "active",
  intransit: "active",
  inprogress: "active",
  dispatched: "active",
  processing: "active",
  started: "active",
  enroute: "active",
  pending: "awaiting",
  submitted: "awaiting",
  inreview: "awaiting",
  requested: "awaiting",
  tendered: "awaiting",
  awaitingapproval: "awaiting",
  pendingapproval: "awaiting",
  warning: "attention",
  delayed: "attention",
  late: "attention",
  hold: "attention",
  onhold: "attention",
  disputed: "attention",
  overdue: "attention",
  exception: "attention",
  blocked: "attention",
  atrisk: "attention",
  expiringsoon: "attention",
  completed: "complete",
  complete: "complete",
  delivered: "complete",
  paid: "complete",
  posted: "complete",
  billed: "complete",
  invoiced: "complete",
  resolved: "complete",
  approved: "complete",
  compliant: "complete",
  qualified: "complete",
  matched: "complete",
  reconciled: "complete",
  sent: "complete",
  closed: "closed",
  superseded: "closed",
  dismissed: "closed",
  archived: "closed",
  inactive: "closed",
  terminated: "closed",
  cancelled: "failed",
  canceled: "failed",
  rejected: "failed",
  declined: "failed",
  expired: "failed",
  failed: "failed",
  error: "failed",
  void: "failed",
  voided: "failed",
  noncompliant: "failed",
  suspended: "failed",
  critical: "failed",
};

/** The phase a status value sits in, or null when it is not one we know. */
export function statusPhase(value: string): StatusPhase | null {
  return STATUS_PHASES[value.toLowerCase().replace(/[\s_-]+/g, "")] ?? null;
}

/** A status value's badge tone, from its phase. */
export function statusTone(value: string): BadgeTone {
  const phase = statusPhase(value);
  return phase === null ? "neutral" : phaseTone(phase);
}

/** Phases most in need of a person first, for sorting a status column. */
const PHASE_RANK: Readonly<Record<StatusPhase, number>> = {
  failed: 0,
  attention: 1,
  awaiting: 2,
  active: 3,
  queued: 4,
  draft: 5,
  complete: 6,
  closed: 7,
};

/** A metric's value in its unit: "3", "0d", "$1,240". */
export function formatMetric(metric: DisplayMetric): string {
  const unit = insightUnitSchema.safeParse(metric.unit);
  if (unit.success) {
    return formatMetricValue({
      key: metric.label,
      label: metric.label,
      value: String(metric.value),
      unit: unit.data,
      direction: "Neutral",
      baseline: null,
      baselineLabel: "",
    });
  }
  const figure = Number(metric.value);
  const shown = Number.isFinite(figure)
    ? formatNumber(figure, { maximumFractionDigits: 2 })
    : String(metric.value);

  return metric.unit === "" ? shown : `${shown} ${metric.unit.toLowerCase()}`;
}

/**
 * A projected value as one line of text, for a place with no room to draw it:
 * a sort, a filter, a step's details. Dates are the reader's own timezone.
 */
export function formatDisplayValue(type: DisplayType, value: unknown, t: TranslateFn): string {
  switch (type) {
    case "enum":
    case "status":
      return typeof value === "string" ? humanizeCode(value) : "";
    case "date":
      return typeof value === "number" ? formatUnixDateMedium(value) : stringOf(value);
    case "datetime":
      return typeof value === "number" ? formatUnixDateTimeMedium(value) : stringOf(value);
    case "money":
    case "percent":
    case "number": {
      const figure = Number(value);
      if (value === null || value === undefined || !Number.isFinite(figure)) return "";
      if (type === "money") return formatCurrency(figure);
      if (type === "percent") return formatPercent(figure, Number.isInteger(figure) ? 0 : 1);
      return formatNumber(figure, { maximumFractionDigits: 2 });
    }
    case "boolean":
      return value === true ? t("Yes") : value === false ? t("No") : "";
    case "flag":
      return value === true ? t("Yes") : "";
    case "metrics":
      return (Array.isArray(value) ? (value as DisplayMetric[]) : [])
        .map((metric) => `${metric.label} ${formatMetric(metric)}`)
        .join(" · ");
    case "links":
      return (Array.isArray(value) ? (value as DisplayLink[]) : [])
        .map((link) => (link.count > 0 ? `${link.label} (${link.count})` : link.label))
        .join(", ");
    case "text":
    case "longText":
      return typeof value === "string" ? value : textOf(value);
  }
}

/** What a column sorts by: figures and instants as numbers, statuses by urgency. */
export function sortKey(type: DisplayType, value: unknown): number | string | null {
  if (value === undefined || value === null) return null;
  switch (type) {
    case "money":
    case "percent":
    case "number": {
      const figure = Number(value);
      return Number.isFinite(figure) ? figure : null;
    }
    case "date":
    case "datetime":
      if (typeof value === "number") return value;
      return WRITTEN_DATE.test(stringOf(value)) ? stringOf(value) : null;
    case "status": {
      const phase = typeof value === "string" ? statusPhase(value) : null;
      return phase === null ? 8 : PHASE_RANK[phase];
    }
    case "boolean":
    case "flag":
      return value === true ? 1 : 0;
    case "metrics":
    case "links":
      return Array.isArray(value) ? value.length : null;
    default:
      return textOf(value).toLowerCase();
  }
}

function isDisplayType(value: unknown): value is DisplayType {
  return typeof value === "string" && (DISPLAY_TYPES as readonly string[]).includes(value);
}

/** Columns the server projected, or null when the payload predates the projection. */
export function readDisplayColumns(value: unknown): DisplayColumn[] | null {
  if (!Array.isArray(value) || value.length === 0 || !value.every(isRecord)) {
    return null;
  }

  return value.flatMap((entry) => {
    const key = stringOf(entry.key);
    if (key === "" || !isDisplayType(entry.type) || isHiddenKey(key)) return [];
    return [
      { key, label: stringOf(entry.label) || displayLabel(key, entry.type), type: entry.type },
    ];
  });
}

/** Fields the server projected for a card, or null when the payload predates it. */
export function readDisplayFields(value: unknown): DisplayField[] | null {
  if (!Array.isArray(value)) {
    return null;
  }

  return value.flatMap((entry) => {
    if (!isRecord(entry)) return [];
    const key = stringOf(entry.key);
    if (key === "" || !isDisplayType(entry.type) || isHiddenKey(key)) return [];
    const projected = projectValue(entry.type, entry.value);
    if (projected === undefined) return [];
    return [
      {
        key,
        label: stringOf(entry.label) || displayLabel(key, entry.type),
        type: entry.type,
        value: projected,
      },
    ];
  });
}

/**
 * A list result the server stored before the projection, read by the same
 * rules: the declared columns a person can read, in the order that leads with
 * the record's name.
 */
export function projectColumns(
  declared: readonly string[],
  rows: readonly Record<string, unknown>[],
): DisplayColumn[] {
  const amountIsMoney = !declared.includes("method");

  return leadFirst(declared).flatMap((key) => {
    const type = classifyValues(
      key,
      rows.map((row) => row[key]),
      amountIsMoney,
    );
    return type === null ? [] : [{ key, label: displayLabel(key, type), type }];
  });
}

/**
 * A record as labelled values, by the same rules: its name first, the rest in
 * its own order; a nested record by its name, a nested set of sentences field
 * by field, anything else nested left out.
 */
export function projectRecord(record: Record<string, unknown>): DisplayField[] {
  const amountIsMoney = record.method === undefined || record.method === null;
  const fields: DisplayField[] = [];
  const add = (key: string, value: unknown) => {
    const type = classifyValues(key, [value], amountIsMoney);
    if (type === null) return;
    const projected = projectValue(type, value);
    if (projected === undefined) return;
    fields.push({ key, label: displayLabel(key, type), type, value: projected });
  };

  for (const key of leadFirst(Object.keys(record))) {
    const value = record[key];
    if (isRecord(value) && !isHiddenKey(key) && recordLabel(value) === "") {
      for (const child of Object.keys(value).sort()) {
        if (typeof value[child] === "string") {
          add(`${key}${child.charAt(0).toUpperCase()}${child.slice(1)}`, value[child]);
        }
      }
      continue;
    }
    add(key, value);
  }

  return fields;
}
