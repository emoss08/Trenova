import type { ProposalField } from "@/types/assistant";

/** The form's state: every field as the text a person edits. */
export type ProposalDraft = Record<string, string>;

export type DraftErrors = Record<string, string>;

type Parsed = { value: unknown } | { error: string };

/**
 * A record subset keeps at least one of its records: the server refuses an
 * empty set whether the parameter is required or not.
 */
const SUBSET_EMPTY_ERROR = "Keep at least one, or reject the proposal instead.";

/**
 * Starts the form from what the agent proposed. Every value becomes text in
 * the shape its control edits: a number as digits, a flag as yes or no, a
 * list as comma-separated items, an object as indented JSON.
 */
export function draftFromArguments(
  fields: readonly ProposalField[],
  args: Record<string, unknown> | null | undefined,
): ProposalDraft {
  const draft: ProposalDraft = {};
  for (const field of fields) {
    draft[field.name] = textFor(field, args?.[field.name]);
  }

  return draft;
}

function textFor(field: ProposalField, value: unknown): string {
  if (value === null || value === undefined) {
    return "";
  }
  switch (field.kind) {
    case "Boolean":
      return value === true ? "true" : value === false ? "false" : "";
    case "List":
    case "RecordSubset":
      return Array.isArray(value) ? value.map(scalarText).join(", ") : scalarText(value);
    case "JSON":
      return typeof value === "string" ? value : JSON.stringify(value, null, 2);
    default:
      return scalarText(value);
  }
}

/** A scalar as its text; anything structured as JSON, never "[object Object]". */
function scalarText(value: unknown): string {
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return JSON.stringify(value);
}

/**
 * Reads one field's text back into the value the tool takes. An empty
 * optional field is left out rather than sent as an empty string, since the
 * tool's schema may refuse the empty string where it accepts absence.
 */
export function parseDraftValue(field: ProposalField, raw: string): Parsed {
  const text = raw.trim();
  if (text === "" && field.kind === "RecordSubset") {
    return { error: SUBSET_EMPTY_ERROR };
  }
  if (text === "") {
    return field.required ? { error: "This value is required" } : { value: undefined };
  }

  switch (field.kind) {
    case "Integer":
    case "Number": {
      const number = Number(text);
      if (!Number.isFinite(number) || (field.kind === "Integer" && !Number.isInteger(number))) {
        return { error: field.kind === "Integer" ? "Must be a whole number" : "Must be a number" };
      }
      if (outOfBounds(field, number)) {
        return { error: boundsMessage(field) };
      }
      return { value: number };
    }
    case "Boolean": {
      const lower = text.toLowerCase();
      if (["true", "yes", "on", "1"].includes(lower)) return { value: true };
      if (["false", "no", "off", "0"].includes(lower)) return { value: false };
      return { error: "Must be yes or no" };
    }
    case "Choice":
      if (field.options.length > 0 && !field.options.includes(text)) {
        return { error: "Choose one of the listed values" };
      }
      return { value: text };
    case "List":
    case "RecordSubset": {
      const items = text
        .split(",")
        .map((item) => item.trim())
        .filter((item) => item !== "");
      if (field.kind === "RecordSubset" && items.length === 0) {
        return { error: SUBSET_EMPTY_ERROR };
      }
      if (field.options.length > 0) {
        const stray = items.find((item) => !field.options.includes(item));
        if (stray !== undefined) {
          return { error: "Choose only from the listed values" };
        }
      }
      return { value: items };
    }
    case "JSON":
      try {
        return { value: JSON.parse(text) };
      } catch {
        return { error: "Must be valid JSON" };
      }
    default:
      if (field.maxLength && text.length > field.maxLength) {
        return { error: `At most ${field.maxLength} characters` };
      }
      return { value: text };
  }
}

function outOfBounds(field: ProposalField, number: number): boolean {
  const min = field.minimum ?? null;
  const max = field.maximum ?? null;
  return (min !== null && number < min) || (max !== null && number > max);
}

function boundsMessage(field: ProposalField): string {
  const min = field.minimum ?? null;
  const max = field.maximum ?? null;
  if (min !== null && max !== null) return `Must be between ${min} and ${max}`;
  if (min !== null) return `Must be at least ${min}`;
  return `Must be at most ${max}`;
}

/** Every field's problem, keyed by name. Empty when the draft would be accepted. */
export function validateDraft(fields: readonly ProposalField[], draft: ProposalDraft): DraftErrors {
  const errors: DraftErrors = {};
  for (const field of fields) {
    const parsed = parseDraftValue(field, draft[field.name] ?? "");
    if ("error" in parsed) {
      errors[field.name] = parsed.error;
    }
  }

  return errors;
}

/**
 * What the person actually changed: each field whose parsed value differs
 * from the proposal. A form returns every field; the decision records only
 * the changes, and a value equal to the proposal is not one. Values compare
 * as JSON so 3 and 3.0 agree and key order does not matter.
 */
export function changedValues(
  fields: readonly ProposalField[],
  proposed: Record<string, unknown> | null | undefined,
  draft: ProposalDraft,
): Record<string, unknown> {
  const changed: Record<string, unknown> = {};
  for (const field of fields) {
    const parsed = parseDraftValue(field, draft[field.name] ?? "");
    if ("error" in parsed) {
      continue;
    }
    const before = proposed?.[field.name];
    const after = parsed.value;
    if (after === undefined) {
      if (before !== undefined && before !== null) {
        changed[field.name] = null;
      }
      continue;
    }
    if (!sameJson(before, after)) {
      changed[field.name] = after;
    }
  }

  return changed;
}

/**
 * A parameter path as the steps into a field's value: "shipment.moves[0].bol"
 * is the field "shipment" and the steps ["moves", 0, "bol"]. Null when the
 * path is not one the editor can follow.
 */
export function parseParamPath(
  param: string,
): { field: string; steps: (string | number)[] } | null {
  const pattern = /^([A-Za-z_$][\w$]*)((?:\.[A-Za-z_$][\w$]*|\[\d+\])*)$/u;
  const match = pattern.exec(param.trim());
  if (match === null) {
    return null;
  }
  const steps: (string | number)[] = [];
  const rest = match[2] ?? "";
  for (const step of rest.matchAll(/\.([A-Za-z_$][\w$]*)|\[(\d+)\]/gu) ?? []) {
    steps.push(step[1] !== undefined ? step[1] : Number(step[2]));
  }

  return { field: match[1] ?? "", steps };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function parsedJson(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return undefined;
  }
}

function valueAt(value: unknown, steps: readonly (string | number)[]): unknown {
  let current = value;
  for (const step of steps) {
    if (typeof step === "number") {
      if (!Array.isArray(current)) return undefined;
      current = current[step];
    } else {
      if (!isRecord(current)) return undefined;
      current = current[step];
    }
  }

  return current;
}

/**
 * The value at a path inside a JSON field's draft, as text for one input.
 * Undefined when the draft is not valid JSON or the path leads nowhere a
 * value could sit; an absent value is "".
 */
export function readNested(
  draft: ProposalDraft,
  field: string,
  steps: readonly (string | number)[],
): string | undefined {
  const value = parsedJson(draft[field] ?? "");
  if (value === undefined) {
    return undefined;
  }
  const parent = valueAt(value, steps.slice(0, -1));
  const last = steps.at(-1);
  if (
    last === undefined ||
    (typeof last === "number" ? !Array.isArray(parent) : !isRecord(parent))
  ) {
    return undefined;
  }
  const at = valueAt(value, steps);

  return at === undefined || at === null ? "" : scalarText(at);
}

/**
 * The JSON field's draft with the value at a path replaced by what a person
 * typed: a number where a number was, otherwise the text. An emptied input
 * drops the value, since "none" is what an empty BOL means.
 */
export function writeNested(
  draft: ProposalDraft,
  field: string,
  steps: readonly (string | number)[],
  text: string,
): ProposalDraft {
  const value = parsedJson(draft[field] ?? "");
  const last = steps.at(-1);
  if (value === undefined || last === undefined) {
    return draft;
  }
  const next = structuredClone(value);
  const parent = valueAt(next, steps.slice(0, -1));
  if (typeof last === "number" ? !Array.isArray(parent) : !isRecord(parent)) {
    return draft;
  }
  const before = valueAt(next, steps);
  const trimmed = text.trim();
  let replacement: unknown = text;
  if (trimmed === "") {
    replacement = undefined;
  } else if (typeof before === "number" && Number.isFinite(Number(trimmed))) {
    replacement = Number(trimmed);
  }
  if (Array.isArray(parent) && typeof last === "number") {
    parent[last] = replacement === undefined ? null : replacement;
  } else if (isRecord(parent) && typeof last === "string") {
    if (replacement === undefined) {
      Reflect.deleteProperty(parent, last);
    } else {
      parent[last] = replacement;
    }
  }

  return { ...draft, [field]: JSON.stringify(next, null, 2) };
}

function sameJson(a: unknown, b: unknown): boolean {
  return canonical(a) === canonical(b);
}

function canonical(value: unknown): string {
  if (value === undefined) return "undefined";
  return JSON.stringify(sortKeys(value));
}

function sortKeys(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(sortKeys);
  if (value && typeof value === "object") {
    const entries = Object.entries(value as Record<string, unknown>).sort(([a], [b]) =>
      a.localeCompare(b),
    );
    return Object.fromEntries(entries.map(([key, inner]) => [key, sortKeys(inner)]));
  }
  return value;
}
