import {
  CURRENT_FORMULA_LANGUAGE_VERSION,
  DEFAULT_ROUNDING_MODE,
  DEFAULT_ROUNDING_PRECISION,
  type FormulaTemplate,
  type FormulaTemplateFormValues,
  type StandardTemplate,
} from "@trenova/shared/types/formula-template";
import type { UseFormSetValue } from "react-hook-form";

/** The part of a template that computes a charge, with nothing that names or tracks it. */
export type TemplateContentValues = Pick<
  FormulaTemplateFormValues,
  | "schemaId"
  | "languageVersion"
  | "expression"
  | "variableDefinitions"
  | "breakdownDefinitions"
  | "minCharge"
  | "maxCharge"
  | "roundingMode"
  | "roundingPrecision"
>;

const CONTENT_KEYS: readonly (keyof TemplateContentValues)[] = [
  "schemaId",
  "languageVersion",
  "expression",
  "variableDefinitions",
  "breakdownDefinitions",
  "minCharge",
  "maxCharge",
  "roundingMode",
  "roundingPrecision",
];

export function starterValuesFrom(standard: StandardTemplate): TemplateContentValues {
  return {
    schemaId: standard.schemaId,
    languageVersion: CURRENT_FORMULA_LANGUAGE_VERSION,
    expression: standard.expression,
    variableDefinitions: standard.variableDefinitions,
    breakdownDefinitions: [],
    minCharge: null,
    maxCharge: null,
    roundingMode: DEFAULT_ROUNDING_MODE,
    roundingPrecision: DEFAULT_ROUNDING_PRECISION,
  };
}

export function copyValuesFrom(source: FormulaTemplate): TemplateContentValues {
  return {
    schemaId: source.schemaId,
    languageVersion: source.languageVersion,
    expression: source.expression,
    variableDefinitions: source.variableDefinitions,
    breakdownDefinitions: source.breakdownDefinitions,
    minCharge: source.minCharge,
    maxCharge: source.maxCharge,
    roundingMode: source.roundingMode,
    roundingPrecision: source.roundingPrecision,
  };
}

export function applyTemplateValues(
  setValue: UseFormSetValue<FormulaTemplateFormValues>,
  values: TemplateContentValues,
) {
  for (const key of CONTENT_KEYS) {
    setValue(key, values[key], { shouldDirty: true });
  }
}
