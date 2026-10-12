import type {
  FormulaTemplateFormValues,
  TestExpressionRequest,
} from "@trenova/shared/types/formula-template";

/** The part of a template the engine prices with, as a form or a saved template holds it. */
export type FormulaPricingContent = Pick<
  FormulaTemplateFormValues,
  | "expression"
  | "schemaId"
  | "variableDefinitions"
  | "breakdownDefinitions"
  | "minCharge"
  | "maxCharge"
  | "roundingMode"
  | "roundingPrecision"
>;

export type FormulaPricingInputs = {
  /** The load's own values; null when a saved shipment supplies them. */
  sampleValues: Readonly<Record<string, unknown>> | null;
  /** The template's variables as the person set them, over their defaults. */
  overrides?: Readonly<Record<string, unknown>>;
  /** A saved shipment to price instead of the sample values. */
  shipmentId?: string;
};

/**
 * The request that prices a template's formula: the sample load's values, the
 * template's variables at their defaults with anything the person set over
 * them, and the breakdown, limits and rounding it is saved with. Null when
 * there is no expression to price.
 */
export function formulaTestRequest(
  content: FormulaPricingContent,
  inputs: FormulaPricingInputs,
): TestExpressionRequest | null {
  const expression = content.expression ?? "";
  if (!expression.trim()) return null;

  const usingShipment = Boolean(inputs.shipmentId);
  const variables: Record<string, unknown> =
    usingShipment || !inputs.sampleValues ? {} : { ...inputs.sampleValues };

  for (const variable of content.variableDefinitions ?? []) {
    if (variable.name && variable.defaultValue !== undefined && variable.defaultValue !== null) {
      variables[variable.name] = variable.defaultValue;
    }
  }
  for (const [name, value] of Object.entries(inputs.overrides ?? {})) {
    if (value !== undefined) {
      variables[name] = value;
    }
  }

  const breakdowns = (content.breakdownDefinitions ?? []).filter(
    (item) => item.name?.trim() && item.expression?.trim(),
  );
  const { minCharge, maxCharge, roundingMode, roundingPrecision } = content;

  return {
    expression,
    schemaId: content.schemaId || "shipment",
    variables,
    ...(usingShipment ? { shipmentId: inputs.shipmentId } : {}),
    ...(breakdowns.length > 0 ? { breakdowns } : {}),
    ...(minCharge != null ? { minCharge: String(minCharge) } : {}),
    ...(maxCharge != null ? { maxCharge: String(maxCharge) } : {}),
    ...(roundingMode ? { roundingMode } : {}),
    ...(roundingPrecision != null ? { roundingPrecision } : {}),
  };
}
