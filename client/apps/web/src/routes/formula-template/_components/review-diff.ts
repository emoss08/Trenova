import { formulaTemplateRoutes } from "@/lib/formula-template-routes";
import type { FieldChange } from "@trenova/shared/types/formula-template";

export type ChangedFieldRow = {
  path: string;
  label: string;
  summary: string;
};

const FIELD_LABELS: Record<string, string> = {
  name: "Name",
  description: "Description",
  type: "Type",
  schemaId: "Schema",
  minCharge: "Minimum charge",
  maxCharge: "Maximum charge",
  roundingMode: "Rounding mode",
  roundingPrecision: "Rounding precision",
  variableDefinitions: "Variables",
  breakdownDefinitions: "Breakdown",
  metadata: "Metadata",
};

const SEGMENT_LABELS: Record<string, string> = {
  variableDefinitions: "Variable",
  breakdownDefinitions: "Breakdown item",
  defaultValue: "default value",
  expression: "expression",
  name: "name",
  label: "label",
  description: "description",
  type: "type",
  required: "required",
};

function formatSide(value: unknown): string {
  if (value === null || value === undefined || value === "") return "empty";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean" || typeof value === "bigint") {
    return String(value);
  }
  return JSON.stringify(value);
}

function labelFor(path: string): string {
  if (FIELD_LABELS[path]) return FIELD_LABELS[path];

  const segments = path.split(".");
  const words: string[] = [];
  for (let i = 0; i < segments.length; i++) {
    const segment = segments[i];
    if (/^\d+$/.test(segment)) {
      words.push(String(Number(segment) + 1));
      continue;
    }
    words.push(SEGMENT_LABELS[segment] ?? segment);
  }
  const [head, ...rest] = words;
  return [head, ...rest].join(" ");
}

/**
 * Turns the server's change map into rows a reviewer can scan. The expression
 * is left out because it gets a proper diff viewer of its own; everything
 * else is one line of "was → now".
 */
export function describeChangedFields(changes: Record<string, FieldChange>): ChangedFieldRow[] {
  return Object.entries(changes)
    .filter(([path]) => path !== "expression")
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([path, change]) => ({
      path,
      label: labelFor(path),
      summary: `${formatSide(change.from)} → ${formatSide(change.to)}`,
    }));
}

/** The studio, opened straight onto the approve step. */
export function reviewLinkFor(templateId: string): string {
  return `${formulaTemplateRoutes.edit(templateId)}?review=approve`;
}
