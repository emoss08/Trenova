import type {
  EDIDiagnostic,
  EDITemplateElement,
  EDITemplateElementBaseSource,
  EDITemplateSegment,
  EDITemplateStatus,
  EDITemplateTransformStep,
  EDITemplateVersion,
} from "@/types/edi";

export type EDIDocumentContextFilters = {
  transactionSet?: string;
  direction?: string;
  status?: string;
  query?: string;
  limit?: number;
};

export const functionalGroupByTransactionSet: Record<string, string> = {
  "204": "SM",
  "210": "IM",
  "214": "QM",
  "990": "GF",
  "997": "FA",
  "999": "FA",
};

export function functionalGroupForTransactionSet(transactionSet?: string | null) {
  return functionalGroupByTransactionSet[transactionSet ?? ""] ?? "SM";
}

export function buildEDIDocumentContextQuery(filters: EDIDocumentContextFilters) {
  const params = new URLSearchParams({ limit: String(filters.limit ?? 100) });
  if (filters.transactionSet) params.set("transactionSet", filters.transactionSet);
  if (filters.direction) params.set("direction", filters.direction);
  if (filters.status) params.set("status", filters.status);
  if (filters.query?.trim()) params.set("search", filters.query.trim());
  return `?${params.toString()}`;
}

export type TransformArgumentDefinition = {
  key: string;
  label: string;
  kind: "string" | "number" | "boolean" | "json" | "path-list";
  required?: boolean;
  placeholder?: string;
};

export type TransformOperationDefinition = {
  operation: string;
  label: string;
  description: string;
  arguments: TransformArgumentDefinition[];
};

export type ConditionDraft =
  | { mode: "none" }
  | { mode: "truthy"; path: string }
  | { mode: "falsey"; path: string }
  | { mode: "comparison"; path: string; operator: "==" | "!="; value: string }
  | { mode: "starlarkFunction"; functionName: string }
  | { mode: "inlineStarlark"; script: string };

export const editableTemplateStatuses: EDITemplateStatus[] = ["Draft"];

export const transformOperationDefinitions: TransformOperationDefinition[] = [
  {
    operation: "trim",
    label: "Trim",
    description: "Remove leading and trailing spaces.",
    arguments: [],
  },
  {
    operation: "upper",
    label: "Uppercase",
    description: "Convert text to uppercase.",
    arguments: [],
  },
  {
    operation: "lower",
    label: "Lowercase",
    description: "Convert text to lowercase.",
    arguments: [],
  },
  {
    operation: "concat",
    label: "Concat",
    description: "Append literal values or $path references.",
    arguments: [
      { key: "values", label: "Values", kind: "path-list", placeholder: "ABC, $shipment.bol" },
      { key: "separator", label: "Separator", kind: "string", placeholder: "-" },
    ],
  },
  {
    operation: "substring",
    label: "Substring",
    description: "Return a zero-based character slice.",
    arguments: [
      { key: "start", label: "Start", kind: "number", required: true },
      { key: "end", label: "End", kind: "number" },
    ],
  },
  {
    operation: "left_pad",
    label: "Left Pad",
    description: "Pad to a fixed length on the left.",
    arguments: [
      { key: "length", label: "Length", kind: "number", required: true },
      { key: "pad", label: "Pad", kind: "string", placeholder: "0" },
    ],
  },
  {
    operation: "right_pad",
    label: "Right Pad",
    description: "Pad to a fixed length on the right.",
    arguments: [
      { key: "length", label: "Length", kind: "number", required: true },
      { key: "pad", label: "Pad", kind: "string", placeholder: " " },
    ],
  },
  {
    operation: "truncate",
    label: "Truncate",
    description: "Limit a value to a maximum length.",
    arguments: [{ key: "length", label: "Length", kind: "number", required: true }],
  },
  {
    operation: "replace",
    label: "Replace",
    description: "Replace occurrences of one value with another.",
    arguments: [
      { key: "old", label: "Old", kind: "string", required: true },
      { key: "new", label: "New", kind: "string", required: true },
      { key: "count", label: "Count", kind: "number" },
    ],
  },
  {
    operation: "contains",
    label: "Contains",
    description: "Return true when text contains a substring.",
    arguments: [{ key: "value", label: "Value", kind: "string", required: true }],
  },
  {
    operation: "starts_with",
    label: "Starts With",
    description: "Return true when text starts with a prefix.",
    arguments: [{ key: "value", label: "Value", kind: "string", required: true }],
  },
  {
    operation: "ends_with",
    label: "Ends With",
    description: "Return true when text ends with a suffix.",
    arguments: [{ key: "value", label: "Value", kind: "string", required: true }],
  },
  {
    operation: "coalesce",
    label: "Coalesce",
    description: "Use the first non-empty fallback value.",
    arguments: [{ key: "values", label: "Values", kind: "path-list" }],
  },
  {
    operation: "default",
    label: "Default",
    description: "Use a fallback when the current value is empty.",
    arguments: [{ key: "value", label: "Value", kind: "string", required: true }],
  },
  {
    operation: "empty_if_none",
    label: "Empty If None",
    description: "Return an empty string for null.",
    arguments: [],
  },
  {
    operation: "required",
    label: "Required",
    description: "Raise a validation error when empty.",
    arguments: [{ key: "message", label: "Message", kind: "string" }],
  },
  {
    operation: "format_date",
    label: "Format Date",
    description: "Format a timestamp/date value.",
    arguments: [{ key: "layout", label: "Layout", kind: "string", placeholder: "20060102" }],
  },
  {
    operation: "format_time",
    label: "Format Time",
    description: "Format a timestamp time value.",
    arguments: [{ key: "layout", label: "Layout", kind: "string", placeholder: "1504" }],
  },
  {
    operation: "format_decimal",
    label: "Format Decimal",
    description: "Format a decimal with fixed places.",
    arguments: [{ key: "places", label: "Places", kind: "number", placeholder: "2" }],
  },
  {
    operation: "format_int",
    label: "Format Int",
    description: "Round and format as an integer.",
    arguments: [],
  },
  {
    operation: "normalize_phone",
    label: "Normalize Phone",
    description: "Strip phone punctuation.",
    arguments: [],
  },
  {
    operation: "normalize_state",
    label: "Normalize State",
    description: "Uppercase and trim to two characters.",
    arguments: [],
  },
  {
    operation: "normalize_postal",
    label: "Normalize Postal",
    description: "Uppercase alphanumeric postal code.",
    arguments: [],
  },
  {
    operation: "qualifier",
    label: "Qualifier",
    description: "Map a source value through a JSON object.",
    arguments: [
      {
        key: "mapping",
        label: "Mapping",
        kind: "json",
        required: true,
        placeholder: '{"LTL":"M"}',
      },
      { key: "fallback", label: "Fallback", kind: "string" },
    ],
  },
  {
    operation: "conditional",
    label: "Conditional",
    description: "Choose a then/else value from a condition.",
    arguments: [
      { key: "when", label: "When", kind: "string", required: true, placeholder: "$shipment.bol" },
      { key: "rule", label: "Rule", kind: "string", placeholder: "truthy" },
      { key: "value", label: "Compare Value", kind: "string" },
      { key: "then", label: "Then", kind: "string" },
      { key: "else", label: "Else", kind: "string" },
    ],
  },
];

export function isTemplateVersionEditable(version?: Pick<EDITemplateVersion, "status"> | null) {
  return !!version && editableTemplateStatuses.includes(version.status);
}

export function getReadOnlyReason(
  version?: Pick<EDITemplateVersion, "status" | "isActive"> | null,
) {
  if (!version) return "Select a template version before editing.";
  if (version.status === "Draft") return "";
  if (version.status === "Certified")
    return "Certified versions are locked. Create a new draft to edit.";
  if (version.status === "Active" || version.isActive)
    return "Active versions are locked. Clone a draft for changes.";
  if (version.status === "Superseded") return "Superseded versions are preserved for history.";
  if (version.status === "Deprecated") return "Deprecated versions are read-only.";
  if (version.status === "Archived") return "Archived versions are read-only.";
  return "This version is read-only.";
}

export function buildConditionString(draft: ConditionDraft): string {
  switch (draft.mode) {
    case "none":
      return "";
    case "truthy":
      return draft.path.trim();
    case "falsey":
      return draft.path.trim() ? `!${draft.path.trim()}` : "";
    case "comparison":
      return draft.path.trim() && draft.value.trim()
        ? `${draft.path.trim()} ${draft.operator} ${JSON.stringify(draft.value)}`
        : "";
    case "starlarkFunction":
      return draft.functionName.trim() ? `starlark:${draft.functionName.trim()}` : "";
    case "inlineStarlark":
      return draft.script.trim() ? `starlark:${draft.script.trim()}` : "";
  }
}

export function parseConditionString(condition?: string | null): ConditionDraft {
  const trimmed = condition?.trim() ?? "";
  if (!trimmed) return { mode: "none" };
  if (trimmed.startsWith("starlark:")) {
    const script = trimmed.slice("starlark:".length).trim();
    const functionReference = script.match(/^([A-Za-z_][A-Za-z0-9_]*)(?:\(\))?$/);
    if (functionReference) {
      return { mode: "starlarkFunction", functionName: functionReference[1] };
    }
    return { mode: "inlineStarlark", script };
  }
  const comparison = trimmed.match(/^(.+?)\s*(==|!=)\s*"([^"]*)"$/);
  if (comparison) {
    return {
      mode: "comparison",
      path: comparison[1].trim(),
      operator: comparison[2] as "==" | "!=",
      value: comparison[3],
    };
  }
  if (trimmed.startsWith("!")) return { mode: "falsey", path: trimmed.slice(1).trim() };
  return { mode: "truthy", path: trimmed };
}

export function insertPathReference(current: string, path: string, asReference = true) {
  const value = asReference ? `$${path}` : path;
  if (!current.trim()) return value;
  return `${current.trim()}, ${value}`;
}

export function getTransformOperationDefinition(operation: string) {
  return transformOperationDefinitions.find((definition) => definition.operation === operation);
}

export function createTransformStep(operation: string): EDITemplateTransformStep {
  return { operation, arguments: {} };
}

export function diagnosticKey(diagnostic: EDIDiagnostic) {
  return `${diagnostic.segmentId ?? ""}:${diagnostic.elementPosition}:${diagnostic.path ?? ""}`;
}

export function diagnosticsForSegment(diagnostics: EDIDiagnostic[], segment: EDITemplateSegment) {
  return diagnostics.filter((diagnostic) => diagnostic.segmentId === segment.segmentId);
}

export function diagnosticsForElement(
  diagnostics: EDIDiagnostic[],
  segment: EDITemplateSegment,
  element: EDITemplateElement,
) {
  return diagnostics.filter(
    (diagnostic) =>
      diagnostic.segmentId === segment.segmentId && diagnostic.elementPosition === element.position,
  );
}

export function cloneSegments(segments: EDITemplateSegment[]) {
  return segments.map((segment) => ({
    ...segment,
    elements: segment.elements.map((element) => ({
      ...element,
      baseSource: cloneBaseSource(element.baseSource),
      transformPipeline: element.transformPipeline.map((step) => ({
        operation: step.operation,
        arguments: { ...step.arguments },
      })),
      validation: { ...element.validation },
    })),
  }));
}

function cloneBaseSource(source: EDITemplateElementBaseSource | null | undefined) {
  return source ? { ...source } : null;
}
