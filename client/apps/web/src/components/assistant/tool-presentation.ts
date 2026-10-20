import { translate } from "@trenova/shared/i18n/runtime";
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
import { defineLabels } from "@trenova/shared/i18n/labels";

export { humanizeKey, recordLabel };

export const WEB_SEARCH_TOOL = "web_search";
export const WEB_READ_TOOL = "web_read";

/** A tool that brings back pages from the public web through an extension. */
export function isWebTool(name: string): boolean {
  return name === WEB_SEARCH_TOOL || name === WEB_READ_TOOL;
}

/**
 * How a known tool reads to a person. Anything not listed falls back to its
 * humanized name, so a new tool is never shown as an identifier, just less
 * warmly than a known one.
 */
const TOOL_TITLES: Record<string, string> = defineLabels({
  agent_list_memories: "Recall what was recorded",
  home_get_layout: "Read your home page",
  home_list_widgets: "List home page widgets",
  home_add_widget: "Add to your home page",
  home_remove_widget: "Remove from your home page",
  home_arrange_layout: "Rearrange your home page",
  remember: "Record for later",
  agent_retire_memory: "Retire a memory",
  factoring_list_submissions: "List factoring schedules",
  factoring_get_submission: "Read factoring schedule",
  factoring_create_submission: "Send invoices to the factor",
  factoring_retry_submission: "Retry factoring schedule",
  factoring_cancel_submission: "Cancel factoring schedule",
  insight_list: "Review insights",
  insight_get: "Read insight",
  insight_dismiss: "Dismiss insight",
  receivable_list_bank_receipt_exceptions: "List unmatched receipts",
  receivable_get_bank_receipt: "Look up bank receipt",
  receivable_list_customer_payments: "List customer payments",
  receivable_match_bank_receipt: "Match bank receipt",
  receivable_post_customer_payment: "Record customer payment",
  receivable_resolve_bank_receipt_work_item: "Close reconciliation item",
  shipment_get: "Look up shipment",
  shipment_list: "Find shipments",
  worker_get: "Look up driver",
  worker_list_credentials: "Check credentials",
  worker_list: "Find drivers",
  equipment_list_tractors: "List tractors",
  equipment_list_trailers: "List trailers",
  customer_list: "List customers",
  location_list: "List locations",
  invoice_list: "List invoices",
  carrier_list: "List carriers",
  equipment_list_types: "List equipment types",
  masterdata_list_fleet_codes: "List fleet codes",
  masterdata_list_service_types: "List service types",
  masterdata_list_shipment_types: "List shipment types",
  masterdata_list_commodities: "List commodities",
  masterdata_list_hazardous_materials: "List hazardous materials",
  masterdata_list_hold_reasons: "List hold reasons",
  shipment_add_comment: "Add a shipment note",
  shipment_place_hold: "Place a hold",
  shipment_release_hold: "Release a hold",
  shipment_cancel: "Cancel shipment",
  dispatch_record_stop_actual: "Record stop arrival",
  customer_get: "Look up customer",
  carrier_get: "Look up carrier",
  equipment_get_tractor: "Look up tractor",
  equipment_get_trailer: "Look up trailer",
  invoice_get: "Look up invoice",
  masterdata_list_accessorial_charges: "List accessorial charges",
  document_list_types: "List document types",
  location_list_categories: "List location categories",
  formula_list_templates: "List rating methods",
  ask_user: "Ask you to choose",
  find_tools: "Look for a tool",
  agent_find_page: "Search the product guide",
  agent_read_attachment: "Read an attached file",
  agent_open_page: "Open a page",
  billing_transfer: "Transfer to billing",
  billing_assign_biller: "Assign biller",
  billing_start_review: "Send to review",
  intake_accept_field: "Accept a field",
  intake_accept_confident: "Accept confident fields",
  intake_set_field: "Set a field",
  intake_set_required_field: "Choose a required record",
  intake_set_stop_location: "Match a stop to a location",
  intake_set_stop_schedule: "Schedule a stop",
  location_create: "Create location",
  formula_describe_schema: "Read the formula reference",
  formula_test_expression: "Price a formula",
  formula_propose: "Propose a formula",
  report_compose_table_view: "Build a table",
  report_compare_runs: "Compare report runs",
  publish_artifact: "Publish a document",
  report_list: "Browse reports",
  report_run: "Start report",
  report_get_run: "Check report run",
  report_get: "Read report",
  report_list_datasets: "Browse datasets",
  report_get_dataset: "Read dataset",
  report_preview: "Preview report",
  report_create: "Create report",
  report_update: "Change report",
  report_fork: "Copy report",
  shipment_get_tracking: "Track shipment",
  equipment_list_positions: "Locate trucks",
  worker_get_hos: "Check driver hours",
  dispatch_get_board: "Read the board",
  exception_list_service_failures: "List service failures",
  exception_list_service_failure_reason_codes: "List failure reasons",
  detention_list_occurrences: "Read detention desk",
  insight_list_weather_alerts: "Check weather alerts",
  exception_evaluate_service_failures: "Check for service failures",
  exception_resolve_service_failure: "Resolve service failure",
  worker_send_message: "Message driver",
  customer_send_email: "Email customer",
  detention_send_notice: "Send detention notice",
  detention_waive: "Waive detention",
  detention_approve: "Approve detention",
  inbox_list_email_profiles: "List email profiles",
  agent_raise_exception: "Flag for manual review",
  shipment_request_documents: "Request missing documents",
  attach_document_to_bqi: "Attach document to billing item",
  reassign_move: "Reassign move",
  worker_list_time_off: "List time off",
  intake_get_draft: "Read document draft",
  rate_quote: "Quote shipment",
  carrier_shop: "Shop carriers",
  dispatch_rank_candidates: "Rank drivers for move",
  dispatch_plan: "Plan dispatch",
  shipment_create: "Enter shipment",
  shipment_update: "Change shipment",
  dispatch_create_tender_from_routing_guide: "Tender via routing guide",
  dispatch_create_tender: "Tender to carriers",
  equipment_set_tractor_status: "Change tractor status",
  equipment_set_trailer_status: "Change trailer status",
  web_search: "Search the web",
  web_read: "Read a web page",
  web_list_portal_sites: "List portal sites",
  web_read_portal: "Look something up on a portal",
  web_use_portal: "Do something on a portal",
  report_analyze_data: "Analyze data",
  worker_approve_pto: "Approve time off",
  worker_reject_pto: "Decline time off",
  worker_cancel_pto: "Cancel time off",
  accounting_get_connection: "Check accounting connection",
  accounting_check_connection: "Test accounting connection",
  accounting_list_mapping_gaps: "List accounting mapping gaps",
  accounting_get_mapping: "Look up accounting mapping",
  accounting_set_mapping: "Set accounting mapping",
  accounting_clear_mapping: "Clear accounting mapping",
  accounting_create_reference_record: "Create accounting record",
  accounting_refresh_reference_data: "Refresh accounting records",
  accounting_list_sync_records: "List accounting sync records",
  accounting_get_sync_record: "Look up accounting sync record",
  accounting_retry_sync_record: "Retry accounting sync",
  accounting_skip_sync_record: "Skip accounting sync",
  accounting_pause_sync: "Pause accounting sync",
  accounting_resume_sync: "Resume accounting sync",
  accounting_request_backfill: "Request accounting backfill",
});

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

  // Each shape is one message, so a language can put the field after its condition.
  if (typeof filter.days === "number") {
    return operator === "lastndays"
      ? translate("{0} last {1}d", field, filter.days)
      : translate("{0} next {1}d", field, filter.days);
  }
  if (Array.isArray(filter.values)) {
    return `${field} ${filter.values.join("/")}`;
  }
  if (typeof filter.value === "string" || typeof filter.value === "number") {
    return `${field} ${filter.value}`;
  }
  if (operator === "isnull") {
    return translate("{0} empty", field);
  }
  if (operator === "isnotnull") {
    return translate("{0} set", field);
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
