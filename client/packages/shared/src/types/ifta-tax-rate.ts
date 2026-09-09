import { z } from "zod";
import { nonNegativeDecimalString, optionalNonNegativeDecimalString } from "./decimal";
import { iftaFuelTypeSchema, iftaQuarterSchema } from "./fuel-ifta-enums";

export const IFTA_RATE_SCALE = 4;
export const IFTA_MIN_YEAR = 2000;
export const IFTA_MAX_YEAR = 2100;

const RATE_MESSAGE = "Enter a rate with up to four decimals";

export const iftaYearSchema = z
  .number()
  .int({ message: "Year is a whole number" })
  .min(IFTA_MIN_YEAR, { message: `Year must be ${IFTA_MIN_YEAR} or later` })
  .max(IFTA_MAX_YEAR, { message: `Year must be ${IFTA_MAX_YEAR} or earlier` });

export const iftaPeriodFormSchema = z.object({
  year: iftaYearSchema,
  quarter: iftaQuarterSchema,
});

export type IftaPeriodFormValues = z.infer<typeof iftaPeriodFormSchema>;

export const iftaTaxRateFormSchema = z.object({
  jurisdictionId: z.string().min(1, { message: "Choose the jurisdiction the rate applies to" }),
  year: iftaYearSchema,
  quarter: iftaQuarterSchema,
  fuelType: iftaFuelTypeSchema,
  ratePerGallon: nonNegativeDecimalString(IFTA_RATE_SCALE, RATE_MESSAGE),
  surchargeRatePerGallon: optionalNonNegativeDecimalString(IFTA_RATE_SCALE, RATE_MESSAGE),
  sourceNote: z.string().max(500, { message: "Keep the note under 500 characters" }).nullable(),
  sourceUrl: z
    .string()
    .nullable()
    .transform((value) => {
      const trimmed = value?.trim() ?? "";
      return trimmed === "" ? null : trimmed;
    })
    .pipe(z.string().url({ message: "Enter a full web address" }).nullable()),
});

export type IftaTaxRateFormValues = z.infer<typeof iftaTaxRateFormSchema>;
