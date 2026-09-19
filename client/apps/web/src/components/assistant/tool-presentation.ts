import { humanizeToolName } from "./proposal-state";

/**
 * How a known tool reads to a person. Anything not listed falls back to its
 * humanized name, so a new tool is never shown as an identifier, just less
 * warmly than a known one.
 */
const TOOL_TITLES: Record<string, string> = {
  get_shipment: "Look up shipment",
  search_shipments: "Search shipments",
  get_worker: "Look up driver",
  search_worker: "Search drivers",
  flag_for_manual_review: "Flag for manual review",
  request_missing_docs: "Request missing documents",
  attach_document_to_bqi: "Attach document to billing item",
  reassign_move: "Reassign move",
};

/** Argument keys that name the record a tool was about, most specific first. */
const SUBJECT_KEYS = [
  "proNumber",
  "workerNumber",
  "query",
  "search",
  "shipmentId",
  "workerId",
  "id",
  "number",
  "name",
];

export type ToolCallDescription = {
  title: string;
  subject: string;
};

/**
 * The one line a reader sees for a tool call: what was done, and to what.
 *
 * A search shows the text it searched for in quotes; a lookup shows the
 * identifier; anything else shows its first scalar argument, which is the best
 * guess at what the call was about without pretending to understand it.
 */
export function describeToolCall(
  name: string,
  args: Record<string, unknown> | null | undefined,
): ToolCallDescription {
  const title = TOOL_TITLES[name] ?? humanizeToolName(name);
  const values = args ?? {};

  for (const key of SUBJECT_KEYS) {
    const value = values[key];
    if (typeof value === "string" && value.trim() !== "") {
      return {
        title,
        subject: key === "query" || key === "search" ? `“${value}”` : value,
      };
    }
    if (typeof value === "number") {
      return { title, subject: String(value) };
    }
  }

  const firstScalar = Object.values(values).find(
    (value) => typeof value === "string" && value.trim() !== "",
  );

  return { title, subject: typeof firstScalar === "string" ? firstScalar : "" };
}

export type ParsedToolResult =
  | { kind: "json"; value: unknown; truncated: boolean }
  | { kind: "text"; text: string; truncated: boolean }
  | { kind: "error"; message: string };

const FENCE_OPEN = "<untrusted_data>";
const FENCE_CLOSE = "</untrusted_data>";
const TRUNCATION_MARK = "…(truncated)";
const FAILURE_PREFIX = /^Tool "[^"]*" failed: /u;

/**
 * Takes the record back out of a saved tool result.
 *
 * The server fences results as untrusted data for the model's benefit and may
 * cut a long one short. A person wants the record itself, and wants to be told
 * when they are not seeing all of it; JSON that was cut short is shown as text
 * rather than failing to parse.
 */
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
  const truncated = body.endsWith(TRUNCATION_MARK);
  if (truncated) {
    body = body.slice(0, -TRUNCATION_MARK.length).trim();
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
