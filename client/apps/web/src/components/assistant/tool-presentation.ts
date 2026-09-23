import { humanizeToolName } from "./proposal-state";
import {
  classifyValues,
  displayLabel,
  humanizeKey,
  isHiddenKey,
  isRecordId,
  leadFirst,
  projectValue,
  recordLabel,
  type DisplayType,
} from "./readable-values";

export { humanizeKey, recordLabel };

/**
 * How a known tool reads to a person. Anything not listed falls back to its
 * humanized name, so a new tool is never shown as an identifier, just less
 * warmly than a known one.
 */
const TOOL_TITLES: Record<string, string> = {
  recall_memory: "Recall what was recorded",
  get_my_home_layout: "Read your home page",
  list_home_widgets: "List home page widgets",
  add_home_widget: "Add to your home page",
  remove_home_widget: "Remove from your home page",
  arrange_home_layout: "Rearrange your home page",
  remember: "Record for later",
  forget_memory: "Retire a memory",
  list_insights: "Review insights",
  get_insight: "Read insight",
  dismiss_insight: "Dismiss insight",
  list_bank_receipt_exceptions: "List unmatched receipts",
  get_bank_receipt: "Look up bank receipt",
  list_customer_payments: "List customer payments",
  match_bank_receipt: "Match bank receipt",
  post_customer_payment: "Record customer payment",
  resolve_bank_receipt_work_item: "Close reconciliation item",
  get_shipment: "Look up shipment",
  search_shipments: "Search shipments",
  get_worker: "Look up driver",
  search_worker: "Search drivers",
  list_expiring_credentials: "Check expiring credentials",
  list_workers: "List drivers",
  list_shipments: "List shipments",
  list_tractors: "List tractors",
  list_trailers: "List trailers",
  list_customers: "List customers",
  list_locations: "List locations",
  list_invoices: "List invoices",
  list_carriers: "List carriers",
  list_equipment_types: "List equipment types",
  list_fleet_codes: "List fleet codes",
  list_service_types: "List service types",
  list_shipment_types: "List shipment types",
  list_commodities: "List commodities",
  list_hazardous_materials: "List hazardous materials",
  list_hold_reasons: "List hold reasons",
  add_shipment_comment: "Add a shipment note",
  place_shipment_hold: "Place a hold",
  release_shipment_hold: "Release a hold",
  cancel_shipment: "Cancel shipment",
  record_stop_actual: "Record stop arrival",
  get_customer: "Look up customer",
  get_carrier: "Look up carrier",
  get_tractor: "Look up tractor",
  get_trailer: "Look up trailer",
  get_invoice: "Look up invoice",
  list_accessorial_charges: "List accessorial charges",
  list_document_types: "List document types",
  list_location_categories: "List location categories",
  ask_user: "Ask you to choose",
  find_tools: "Look for a tool",
  find_in_trenova: "Search the product guide",
  open_page: "Open a page",
  compose_table_view: "Build a table",
  compare_report_runs: "Compare report runs",
  publish_artifact: "Publish a document",
  list_reports: "Browse reports",
  run_report: "Start report",
  get_report_run: "Check report run",
  describe_report: "Read report",
  list_report_datasets: "Browse datasets",
  describe_report_dataset: "Read dataset",
  preview_report: "Preview report",
  create_report: "Create report",
  update_report: "Change report",
  fork_report: "Copy report",
  get_shipment_tracking: "Track shipment",
  list_vehicle_positions: "Locate trucks",
  get_worker_hos: "Check driver hours",
  get_dispatch_board: "Read the board",
  list_service_failures: "List service failures",
  list_service_failure_reason_codes: "List failure reasons",
  list_detention_desk: "Read detention desk",
  list_weather_alerts: "Check weather alerts",
  evaluate_service_failures: "Check for service failures",
  resolve_service_failure: "Resolve service failure",
  notify_driver: "Message driver",
  email_customer: "Email customer",
  send_detention_notice: "Send detention notice",
  waive_detention: "Waive detention",
  list_email_profiles: "List email profiles",
  flag_for_manual_review: "Flag for manual review",
  request_missing_docs: "Request missing documents",
  attach_document_to_bqi: "Attach document to billing item",
  reassign_move: "Reassign move",
  list_time_off: "List time off",
  get_shipment_draft: "Read document draft",
  quote_shipment: "Quote shipment",
  shop_carriers: "Shop carriers",
  rank_move_candidates: "Rank drivers for move",
  plan_dispatch: "Plan dispatch",
  create_shipment: "Enter shipment",
  update_shipment: "Change shipment",
  tender_move_to_routing_guide: "Tender via routing guide",
  tender_move_to_carriers: "Tender to carriers",
  update_tractor_status: "Change tractor status",
  update_trailer_status: "Change trailer status",
  approve_worker_pto: "Approve time off",
  reject_worker_pto: "Decline time off",
  cancel_worker_pto: "Cancel time off",
};

/** Argument keys that name the record a tool was about, most specific first. */
const SUBJECT_KEYS = [
  "need",
  "customerId",
  "carrierId",
  "tractorId",
  "trailerId",
  "invoiceId",
  "moveId",
  "proNumber",
  "reportKey",
  "definitionId",
  "runId",
  "serviceFailureId",
  "insightId",
  "bankReceiptId",
  "workItemId",
  "customerPaymentId",
  "occurrenceId",
  "workerNumber",
  "ptoId",
  "dataset",
  "page",
  "question",
  "query",
  "search",
  "shipmentId",
  "workerId",
  "id",
  "number",
  "name",
  "title",
];

export type ToolCallDescription = {
  title: string;
  subject: string;
};

/** How many filters fit on one line before the rest become a count. */
const VISIBLE_FILTERS = 2;

type ToolFilter = {
  field?: unknown;
  operator?: unknown;
  value?: unknown;
  values?: unknown;
  days?: unknown;
};

/**
 * What a filter was asking for, in as few words as it takes to recognize it.
 *
 * A list tool carries its whole question inside `filters`, so without this the
 * call reads as a bare "List shipments" and the reader cannot tell whether it
 * asked for yesterday's deliveries or everything.
 */
function describeFilter(filter: ToolFilter): string {
  const field = typeof filter.field === "string" ? filter.field : "";
  if (field === "") {
    return "";
  }

  const operator = typeof filter.operator === "string" ? filter.operator : "";

  if (typeof filter.days === "number") {
    const span = operator === "lastndays" ? "last" : "next";
    return `${field} ${span} ${filter.days}d`;
  }
  if (Array.isArray(filter.values)) {
    return `${field} ${filter.values.join("/")}`;
  }
  if (typeof filter.value === "string" || typeof filter.value === "number") {
    return `${field} ${filter.value}`;
  }
  if (operator === "isnull") {
    return `${field} empty`;
  }
  if (operator === "isnotnull") {
    return `${field} set`;
  }

  return `${field} ${operator}`.trim();
}

function describeFilters(raw: unknown): string {
  if (!Array.isArray(raw) || raw.length === 0) {
    return "";
  }

  const parts = raw
    .filter((entry): entry is ToolFilter => typeof entry === "object" && entry !== null)
    .map(describeFilter)
    .filter((part) => part !== "");
  if (parts.length === 0) {
    return "";
  }

  const shown = parts.slice(0, VISIBLE_FILTERS).join(", ");
  const hidden = parts.length - VISIBLE_FILTERS;

  return hidden > 0 ? `${shown} +${hidden}` : shown;
}

/**
 * The one line a reader sees for a tool call: what was done, and to what.
 *
 * A search shows the text it searched for in quotes; a lookup shows what the
 * record is called; anything else shows its first scalar argument, which is
 * the best guess at what the call was about without pretending to understand
 * it. A record's id is never the subject — "Read insight
 * inst_01M37R101VKZTB7TSKR30FJ0AT" names nothing a person can use, so the line
 * says "Read insight" and stops.
 */
export function describeToolCall(
  name: string,
  args: Record<string, unknown> | null | undefined,
): ToolCallDescription {
  const title = TOOL_TITLES[name] ?? humanizeToolName(name);
  const values = args ?? {};

  const filters = describeFilters(values.filters);
  if (filters !== "") {
    return { title, subject: filters };
  }

  for (const key of SUBJECT_KEYS) {
    const value = values[key];
    if (typeof value === "string" && value.trim() !== "" && !isRecordId(value)) {
      return {
        title,
        subject: key === "query" || key === "search" || key === "question" ? `“${value}”` : value,
      };
    }
    // A number under "page" is a page of results, not a page of the app.
    if (typeof value === "number" && key !== "page") {
      return { title, subject: String(value) };
    }
  }

  const firstScalar = Object.values(values).find(
    (value) => typeof value === "string" && value.trim() !== "" && !isRecordId(value),
  );
  if (typeof firstScalar === "string") {
    return { title, subject: firstScalar };
  }

  return { title, subject: definitionDataset(values.definition) };
}

/**
 * A report definition names no record; its whole question is the object. The
 * dataset it is built on is the one word that says what it is about.
 */
function definitionDataset(definition: unknown): string {
  if (typeof definition !== "object" || definition === null) {
    return "";
  }
  const entity = (definition as { entity?: unknown }).entity;

  return typeof entity === "string" ? entity.trim() : "";
}

export type ParsedToolResult =
  | { kind: "json"; value: unknown; truncated: boolean }
  | { kind: "text"; text: string; truncated: boolean }
  | { kind: "error"; message: string };

const FENCE_OPEN = "<untrusted_data>";
const FENCE_CLOSE = "</untrusted_data>";
// What the server appends when it cuts a result short (agentruntime/fence.go).
// Matched on its opening rather than in full: the notice is a paragraph of
// instruction to the model and will be reworded, while the bracketed opening is
// what identifies it. The older "…(truncated)" marker is still recognised so a
// thread saved before the notice changed still reads correctly.
const TRUNCATION_MARKS = ["[This result was cut off here:", "…(truncated)"];
const FAILURE_PREFIX = /^Tool "[^"]*" failed: /u;

/**
 * Takes the record back out of a saved tool result.
 *
 * The server fences results as untrusted data for the model's benefit and may
 * cut a long one short. A person wants the record itself, and wants to be told
 * when they are not seeing all of it; JSON that was cut short is shown as text
 * rather than failing to parse.
 */
/** Where the truncation notice begins, or -1 when the result is whole. */
function truncationStart(body: string): number {
  for (const mark of TRUNCATION_MARKS) {
    const at = body.lastIndexOf(mark);
    if (at !== -1) {
      return at;
    }
  }

  return -1;
}

export function parseToolResult(content: string): ParsedToolResult {
  const failure = FAILURE_PREFIX.exec(content);
  if (failure) {
    return { kind: "error", message: content.slice(failure[0].length).trim() };
  }

  const open = content.indexOf(FENCE_OPEN);
  const close = content.lastIndexOf(FENCE_CLOSE);
  if (open === -1 || close === -1 || close < open) {
    return { kind: "text", text: content, truncated: false };
  }

  let body = content.slice(open + FENCE_OPEN.length, close).trim();
  const cut = truncationStart(body);
  const truncated = cut !== -1;
  if (truncated) {
    body = body.slice(0, cut).trim();
  }

  if (!truncated) {
    try {
      return { kind: "json", value: JSON.parse(body), truncated: false };
    } catch {
      // Not JSON after all; shown as the text it is.
    }
  }

  return { kind: "text", text: body, truncated };
}

/**
 * A value as the details show it: plain words, a typed value drawn the way its
 * type reads (a date in the reader's timezone, a status as a badge, a list of
 * measurements as label and value), or the size of something nested — never
 * its JSON.
 */
export type ReadableValue =
  | { kind: "text"; text: string }
  | { kind: "value"; type: DisplayType; value: unknown }
  | { kind: "items"; count: number }
  | { kind: "fields"; count: number };

export type ReadableEntry = { key: string; label: string; value: ReadableValue };

/**
 * One value by the rules an artifact reads by. An id — under any name, or
 * shaped like one under any other — is left out, as are the tenant and a
 * write's version; an epoch reads as a date; a list of measurements reads as
 * measurements. What those rules have no reading for is described by its
 * size, and the literal stays behind Details.
 */
function readableValue(key: string, value: unknown): ReadableValue | null {
  if (value === null || value === undefined || value === "" || isHiddenKey(key)) {
    return null;
  }
  if (isRecordId(value)) {
    return null;
  }
  if (key === "filters" && Array.isArray(value)) {
    const filters = describeFilters(value);
    if (filters !== "") {
      return { kind: "text", text: filters };
    }
  }

  const type = classifyValues(key, [value]);
  if (type !== null) {
    const projected = projectValue(type, value);
    if (projected === undefined) {
      return null;
    }
    return type === "text" && typeof projected === "string"
      ? { kind: "text", text: projected }
      : { kind: "value", type, value: projected };
  }
  if (Array.isArray(value)) {
    return { kind: "items", count: value.length };
  }
  if (typeof value === "object") {
    return { kind: "fields", count: Object.keys(value).length };
  }

  return null;
}

/** How many labelled values a record shows before the rest are left to Details. */
export const READABLE_LIMIT = 10;

/**
 * An object as labelled values, its name first and the rest in its own
 * order, with empty values, ids and bookkeeping left out and nested values
 * reduced to their shape, so a person reads a record rather than a wall of
 * JSON. The whole object stays behind Details.
 *
 * `leadWithName` puts the keys that name a record first, for a result; what a
 * call was asked keeps the order it was asked in.
 */
export function readableEntries(
  value: Record<string, unknown> | null | undefined,
  limit: number = READABLE_LIMIT,
  leadWithName = false,
): { entries: ReadableEntry[]; hidden: number } {
  const record = value ?? {};
  const keys = leadWithName ? leadFirst(Object.keys(record)) : Object.keys(record);
  const all: ReadableEntry[] = [];
  for (const key of keys) {
    const readable = readableValue(key, record[key]);
    if (readable !== null) {
      all.push({
        key,
        label: readable.kind === "value" ? displayLabel(key, readable.type) : humanizeKey(key),
        value: readable,
      });
    }
  }

  return { entries: all.slice(0, limit), hidden: Math.max(0, all.length - limit) };
}

/** What a tool returned, shaped for reading. */
export type ReadableResult =
  | { kind: "list"; count: number; more: boolean; labels: string[] }
  | { kind: "record"; entries: ReadableEntry[]; hidden: number }
  | { kind: "text"; text: string };

/** How many of a list's records are named before the rest are a count. */
const LIST_LABELS = 5;

function readableList(items: unknown[], count: number, more: boolean): ReadableResult {
  return {
    kind: "list",
    count,
    more,
    labels: items
      .map(recordLabel)
      .filter((label) => label !== "")
      .slice(0, LIST_LABELS),
  };
}

export function readableResult(value: unknown): ReadableResult {
  if (Array.isArray(value)) {
    return readableList(value, value.length, false);
  }
  if (typeof value !== "object" || value === null) {
    return { kind: "text", text: value === null || value === undefined ? "" : String(value) };
  }
  const record = value as Record<string, unknown>;
  if (Array.isArray(record.items)) {
    const declared = typeof record.count === "number" ? record.count : record.items.length;
    return readableList(record.items, declared, record.hasMore === true);
  }

  return { kind: "record", ...readableEntries(record, READABLE_LIMIT, true) };
}
