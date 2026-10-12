import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";
import { nonNegativeDecimalString, optionalNonNegativeDecimalString } from "./decimal";
import { iftaFuelTypeSchema, iftaQuarterSchema } from "./fuel-ifta-enums";

export const IFTA_RATE_SCALE = 4;
export const IFTA_MIN_YEAR = 2000;
export const IFTA_MAX_YEAR = 2100;

const RATE_MESSAGE = "Enter a rate with up to four decimals";

export const iftaYearSchema = z
  .number()
  .int({ error: () => translate("Year is a whole number") })
  .min(IFTA_MIN_YEAR, { error: () => translate("Year must be {0} or later", IFTA_MIN_YEAR) })
  .max(IFTA_MAX_YEAR, { error: () => translate("Year must be {0} or earlier", IFTA_MAX_YEAR) });

export const iftaPeriodFormSchema = z.object({
  year: iftaYearSchema,
  quarter: iftaQuarterSchema,
});

export type IftaPeriodFormValues = z.infer<typeof iftaPeriodFormSchema>;

export const iftaTaxRateFormSchema = z.object({
  jurisdictionId: z
    .string()
    .min(1, { error: () => translate("Choose the jurisdiction the rate applies to") }),
  year: iftaYearSchema,
  quarter: iftaQuarterSchema,
  fuelType: iftaFuelTypeSchema,
  ratePerGallon: nonNegativeDecimalString(IFTA_RATE_SCALE, RATE_MESSAGE),
  surchargeRatePerGallon: optionalNonNegativeDecimalString(IFTA_RATE_SCALE, RATE_MESSAGE),
  sourceNote: z
    .string()
    .max(500, { error: () => translate("Keep the note under 500 characters") })
    .nullable(),
  sourceUrl: z
    .string()
    .nullable()
    .transform((value) => {
      const trimmed = value?.trim() ?? "";
      return trimmed === "" ? null : trimmed;
    })
    .pipe(
      z
        .string()
        .url({ error: () => translate("Enter a full web address") })
        .nullable(),
    ),
});

export type IftaTaxRateFormValues = z.infer<typeof iftaTaxRateFormSchema>;
