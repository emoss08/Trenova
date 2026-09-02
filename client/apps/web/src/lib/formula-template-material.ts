import type {
  FormulaTemplate,
  FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";

type ContentSnapshot = {
  expression: string;
  schemaId: string;
  type: string;
  minCharge: number | null;
  maxCharge: number | null;
  roundingMode: string;
  roundingPrecision: number;
  variableDefinitions: string;
  breakdownDefinitions: string;
};

function numberOrNull(value: unknown): number | null {
  if (value === null || value === undefined || value === "") return null;
  const parsed = typeof value === "number" ? value : Number(value);
  return Number.isNaN(parsed) ? null : parsed;
}

function stableJson(value: unknown): string {
  return JSON.stringify(value ?? [], (_key, item: unknown) => {
    if (item && typeof item === "object" && !Array.isArray(item)) {
      const record = item as Record<string, unknown>;
      return Object.keys(record)
        .sort()
        .reduce<Record<string, unknown>>((sorted, key) => {
          if (record[key] !== undefined) sorted[key] = record[key];
          return sorted;
        }, {});
    }
    return item;
  });
}

function snapshot(source: FormulaTemplate | FormulaTemplateFormValues): ContentSnapshot {
  return {
    expression: source.expression ?? "",
    schemaId: source.schemaId || "shipment",
    type: source.type,
    minCharge: numberOrNull(source.minCharge),
    maxCharge: numberOrNull(source.maxCharge),
    roundingMode: source.roundingMode ?? "HalfUp",
    roundingPrecision: source.roundingPrecision ?? 2,
    variableDefinitions: stableJson(source.variableDefinitions),
    breakdownDefinitions: stableJson(source.breakdownDefinitions),
  };
}

/**
 * Mirrors the server's HasMaterialChange: a change to what the template
 * computes (expression, schema, type, guardrails, rounding, variables,
 * breakdowns) is material and returns an approved template to Draft. Name,
 * description, and metadata edits are not.
 */
export function hasMaterialChange(
  template: FormulaTemplate,
  values: FormulaTemplateFormValues,
): boolean {
  const before = snapshot(template);
  const after = snapshot(values);
  return (Object.keys(before) as Array<keyof ContentSnapshot>).some(
    (key) => before[key] !== after[key],
  );
}

/** Whether saving a material change would take this template out of production. */
export function saveDemotesToDraft(
  template: FormulaTemplate | null,
  values: FormulaTemplateFormValues,
): boolean {
  if (!template) return false;
  if (template.status !== "Active" && template.status !== "InReview") return false;
  return hasMaterialChange(template, values);
}
