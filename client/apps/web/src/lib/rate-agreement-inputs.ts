import type { TemplateInputsById } from "@/lib/tariff-inputs-resolver";
import { translate } from "@trenova/shared/i18n/runtime";
import { addTariffInputIssues, sharedTariffInputs } from "@trenova/shared/lib/tariff-inputs";
import type { RateAgreement, RateAgreementRule } from "@trenova/shared/types/rate";
import type { TariffInputDeclaration, TariffInputValues } from "@trenova/shared/types/tariff";
import type { z } from "zod";

/** The formula templates an agreement's lanes price through, each once, in lane order. */
export function rateAgreementTemplateIds(
  rules: readonly Pick<RateAgreementRule, "formulaTemplateId">[] | null | undefined,
): string[] {
  return [...new Set((rules ?? []).map((rule) => rule.formulaTemplateId).filter(Boolean))];
}

/**
 * The inputs an agreement may set for every lane: the ones every lane's tariff declares with
 * the same type, since a value the agreement sets is checked against each lane's tariff.
 * `complete` is false while some lane's tariff has not loaded, and the list is then empty
 * rather than wrong.
 */
export function agreementSharedInputs(
  rules: readonly Pick<RateAgreementRule, "formulaTemplateId">[] | null | undefined,
  inputs: TemplateInputsById,
): { declarations: TariffInputDeclaration[]; complete: boolean } {
  const templateIds = rateAgreementTemplateIds(rules);
  const sets = templateIds.map((templateId) => inputs.get(templateId));
  if (sets.some((set) => set === undefined)) return { declarations: [], complete: false };
  return {
    declarations: sharedTariffInputs(sets as (readonly TariffInputDeclaration[])[]),
    complete: true,
  };
}

/**
 * Checks every lane's input values against its tariff and the agreement's own values against
 * the inputs its lanes share: a lane's required input is satisfied by the agreement, and the
 * agreement sets nothing a lane's tariff does not declare. Lanes whose tariff has not loaded are
 * left to the server's check.
 */
export function refineRateAgreementInputs(inputs: TemplateInputsById) {
  return (agreement: RateAgreement, ctx: z.RefinementCtx): void => {
    const agreementValues = agreement.inputs as TariffInputValues;
    (agreement.rules ?? []).forEach((rule, index) => {
      const declarations = inputs.get(rule.formulaTemplateId);
      if (!declarations) return;
      addTariffInputIssues(ctx, ["rules", index, "inputs"], declarations, rule.inputs, translate, {
        inherited: agreementValues,
        requireUnset: true,
        currency: rule.currency || agreement.currency,
      });
    });

    if (Object.keys(agreement.inputs ?? {}).length === 0) return;
    const shared = agreementSharedInputs(agreement.rules, inputs);
    if (!shared.complete) return;
    addTariffInputIssues(ctx, ["inputs"], shared.declarations, agreement.inputs, translate, {
      scope: "agreement",
      currency: agreement.currency,
    });
  };
}
