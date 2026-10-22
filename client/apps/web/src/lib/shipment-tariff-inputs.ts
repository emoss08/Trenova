import type { TemplateInputsById } from "@/lib/tariff-inputs-resolver";
import { translate } from "@trenova/shared/i18n/runtime";
import { addTariffInputIssues } from "@trenova/shared/lib/tariff-inputs";
import type { z } from "zod";

type ShipmentTariffValues = {
  formulaTemplateId?: string | null;
  tariffInputs?: Record<string, unknown> | null;
};

/** The tariff a shipment rates through, when it has one; a manual rate runs none. */
export function shipmentTemplateIds(values: ShipmentTariffValues): string[] {
  return values.formulaTemplateId ? [values.formulaTemplateId] : [];
}

/**
 * Checks a shipment's input values against its tariff: each must fit its declared type, and an
 * input with no default must be set, since nothing but the shipment supplies it at re-rate.
 */
export function refineShipmentInputs(inputs: TemplateInputsById) {
  return (shipment: ShipmentTariffValues, ctx: z.RefinementCtx): void => {
    const declarations = shipment.formulaTemplateId
      ? inputs.get(shipment.formulaTemplateId)
      : undefined;
    if (!declarations) return;
    addTariffInputIssues(ctx, ["tariffInputs"], declarations, shipment.tariffInputs, translate, {
      requireUnset: true,
    });
  };
}
