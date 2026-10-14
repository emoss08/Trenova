import type { FactoringAgreement } from "@/lib/graphql/factoring";
import {
  basisPointsToPercent,
  formatMinutesOfDay,
  parseMinutesOfDay,
  percentToBasisPoints,
} from "@/lib/factoring";
import type { FactoringAgreementInput } from "@trenova/graphql/generated/graphql";
import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";

export const MAX_FEE_TIERS = 20;
export const MAX_CHARGEBACK_WINDOW_DAYS = 730;
export const MAX_FUNDING_DAYS = 60;
export const MAX_AGREEMENT_NOTES = 2000;

const feeTierSchema = z.object({
  daysTo: z
    .number()
    .int({ error: () => translate("Enter whole days") })
    .nullable(),
  ratePercent: z
    .number({ error: () => translate("Fee rate is required") })
    .min(0, { error: () => translate("Fee rate must be between 0% and 100%") })
    .max(100, { error: () => translate("Fee rate must be between 0% and 100%") }),
  flatMinor: z
    .number()
    .int()
    .min(0, { error: () => translate("Flat fee cannot be negative") }),
});

const documentTypeValue = z.union([z.string(), z.object({ id: z.string().optional() }).loose()]);

export const factoringAgreementFormSchema = z
  .object({
    effectiveFrom: z.number({ error: () => translate("Start date is required") }),
    effectiveTo: z.number().nullable(),
    advanceRatePercent: z
      .number({ error: () => translate("Advance rate is required") })
      .gt(0, { error: () => translate("Advance rate must be greater than zero") })
      .max(100, { error: () => translate("Advance rate cannot be more than 100%") }),
    reserveRatePercent: z
      .number({ error: () => translate("Reserve rate is required") })
      .min(0, { error: () => translate("Reserve rate cannot be negative") })
      .max(100, { error: () => translate("Reserve rate cannot be more than 100%") }),
    minimumFeeMinor: z
      .number()
      .int()
      .min(0, { error: () => translate("Minimum fee cannot be negative") }),
    chargebackWindowDays: z
      .number({ error: () => translate("Chargeback window is required") })
      .int({ error: () => translate("Enter whole days") })
      .min(0, { error: () => translate("Chargeback window cannot be negative") })
      .max(MAX_CHARGEBACK_WINDOW_DAYS, {
        error: () => translate("Chargeback window cannot be longer than 730 days"),
      }),
    recourse: z.enum(["Recourse", "NonRecourse"]),
    accountingTreatment: z.enum(["Sale", "SecuredBorrowing"]),
    fundingDays: z
      .number({ error: () => translate("Funding days are required") })
      .int({ error: () => translate("Enter whole days") })
      .min(0, { error: () => translate("Funding days cannot be negative") })
      .max(MAX_FUNDING_DAYS, { error: () => translate("Funding days cannot be more than 60") }),
    submissionTime: z.string().refine((value) => parseMinutesOfDay(value) !== null, {
      error: () => translate("Enter a time of day"),
    }),
    notes: z.string().max(MAX_AGREEMENT_NOTES, {
      error: () => translate("Notes cannot be longer than 2000 characters"),
    }),
    feeTiers: z
      .array(feeTierSchema)
      .min(1, { error: () => translate("Add at least one fee tier") })
      .max(MAX_FEE_TIERS, { error: () => translate("An agreement can have at most 20 fee tiers") }),
    documentTypes: z.array(documentTypeValue),
  })
  .superRefine((values, ctx) => {
    if (values.advanceRatePercent + values.reserveRatePercent > 100) {
      ctx.addIssue({
        code: "custom",
        path: ["reserveRatePercent"],
        message: translate("Advance and reserve together cannot be more than 100%"),
      });
    }
    if (values.effectiveTo != null && values.effectiveTo <= values.effectiveFrom) {
      ctx.addIssue({
        code: "custom",
        path: ["effectiveTo"],
        message: translate("End date must be after the start date"),
      });
    }
    let start = 0;
    values.feeTiers.forEach((tier, index) => {
      const last = index === values.feeTiers.length - 1;
      if (tier.daysTo == null) {
        if (!last) {
          ctx.addIssue({
            code: "custom",
            path: ["feeTiers", index, "daysTo"],
            message: translate("Only the last fee tier can be open-ended"),
          });
        }
        return;
      }
      if (tier.daysTo <= start) {
        ctx.addIssue({
          code: "custom",
          path: ["feeTiers", index, "daysTo"],
          message: translate("A fee tier must end after it starts"),
        });
      }
      start = tier.daysTo;
    });
  });

export type FactoringAgreementFormValues = z.infer<typeof factoringAgreementFormSchema>;
export type FactoringFeeTierFormValues = FactoringAgreementFormValues["feeTiers"][number];

/** The day each tier starts on: day 0 for the first, the day the one before it ends for the rest. */
export function feeTierStarts(tiers: readonly Pick<FactoringFeeTierFormValues, "daysTo">[]) {
  const starts: number[] = [];
  let start = 0;
  for (const tier of tiers) {
    starts.push(start);
    start = tier.daysTo ?? start;
  }
  return starts;
}

export function emptyAgreementValues(nowUnix: number): FactoringAgreementFormValues {
  return {
    effectiveFrom: nowUnix,
    effectiveTo: null,
    advanceRatePercent: 90,
    reserveRatePercent: 10,
    minimumFeeMinor: 0,
    chargebackWindowDays: 90,
    recourse: "Recourse",
    accountingTreatment: "Sale",
    fundingDays: 1,
    submissionTime: "09:00",
    notes: "",
    feeTiers: [
      { daysTo: 30, ratePercent: 3, flatMinor: 0 },
      { daysTo: null, ratePercent: 5, flatMinor: 0 },
    ],
    documentTypes: [],
  };
}

export function agreementToFormValues(agreement: FactoringAgreement): FactoringAgreementFormValues {
  const tiers = [...agreement.feeTiers].sort((a, b) => a.daysFrom - b.daysFrom);
  return {
    effectiveFrom: agreement.effectiveFrom,
    effectiveTo: agreement.effectiveTo ?? null,
    advanceRatePercent: basisPointsToPercent(agreement.advanceRateBp),
    reserveRatePercent: basisPointsToPercent(agreement.reserveRateBp),
    minimumFeeMinor: agreement.minimumFeeMinor,
    chargebackWindowDays: agreement.chargebackWindowDays,
    recourse: agreement.recourse,
    accountingTreatment: agreement.accountingTreatment,
    fundingDays: agreement.fundingDays,
    submissionTime: formatMinutesOfDay(agreement.submissionTime),
    notes: agreement.notes ?? "",
    feeTiers: tiers.map((tier) => ({
      daysTo: tier.daysTo ?? null,
      ratePercent: basisPointsToPercent(tier.rateBp),
      flatMinor: tier.flatMinor,
    })),
    documentTypes: agreement.documentTypes.map((documentType) => documentType.id),
  };
}

function documentTypeId(value: FactoringAgreementFormValues["documentTypes"][number]): string {
  return typeof value === "string" ? value : (value.id ?? "");
}

export function agreementFormToInput(
  factoringCompanyId: string,
  values: FactoringAgreementFormValues,
): FactoringAgreementInput {
  const starts = feeTierStarts(values.feeTiers);
  const notes = values.notes.trim();
  return {
    factoringCompanyId,
    effectiveFrom: values.effectiveFrom,
    effectiveTo: values.effectiveTo,
    advanceRateBp: percentToBasisPoints(values.advanceRatePercent),
    reserveRateBp: percentToBasisPoints(values.reserveRatePercent),
    minimumFeeMinor: values.minimumFeeMinor,
    chargebackWindowDays: values.chargebackWindowDays,
    recourse: values.recourse,
    accountingTreatment: values.accountingTreatment,
    fundingDays: values.fundingDays,
    submissionTime: parseMinutesOfDay(values.submissionTime) ?? 0,
    notes: notes === "" ? null : notes,
    feeTiers: values.feeTiers.map((tier, index) => ({
      daysFrom: starts[index],
      daysTo: tier.daysTo,
      rateBp: percentToBasisPoints(tier.ratePercent),
      flatMinor: tier.flatMinor,
    })),
    documentTypeIds: values.documentTypes.map(documentTypeId).filter((id) => id !== ""),
  };
}
