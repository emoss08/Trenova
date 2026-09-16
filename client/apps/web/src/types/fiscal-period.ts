import { z } from "zod";
import { tenantInfoSchema } from "@trenova/shared/types/helpers";

const MAX_OPERATING_PERIOD_NUMBER = 12;
const MAX_ADJUSTING_PERIOD_NUMBER = 14;

const fiscalPeriodStatusSchema = z.enum([
  "Inactive",
  "Open",
  "Locked",
  "Closed",
  "PermanentlyClosed",
]);
export type FiscalPeriodStatus = z.infer<typeof fiscalPeriodStatusSchema>;

const periodTypeSchema = z.enum(["Month", "Quarter", "Week", "Adjusting"]);
export type PeriodType = z.infer<typeof periodTypeSchema>;

export const fiscalPeriodSchema = z
  .object({
    ...tenantInfoSchema.shape,
    fiscalYearId: z.string().min(1),
    status: fiscalPeriodStatusSchema,
    periodType: periodTypeSchema,
    periodNumber: z.number().int().min(1),
    name: z.string().min(1).max(100),
    startDate: z.number().int(),
    endDate: z.number().int(),
    isAdjusting: z.boolean().optional(),
    allowAdjustingEntries: z.boolean().optional(),
    adjustmentDeadline: z.number().int().nullish(),
    lockedAt: z.number().int().nullish(),
    closedAt: z.number().int().nullish(),
    closedById: z.string().nullish(),
  })
  .superRefine((period, ctx) => {
    const max =
      period.isAdjusting || period.periodType === "Adjusting"
        ? MAX_ADJUSTING_PERIOD_NUMBER
        : MAX_OPERATING_PERIOD_NUMBER;

    if (period.periodNumber > max) {
      ctx.addIssue({
        code: "too_big",
        origin: "number",
        maximum: max,
        inclusive: true,
        path: ["periodNumber"],
        message: `Period number must be at most ${max}`,
      });
    }
  });

export type FiscalPeriod = z.infer<typeof fiscalPeriodSchema>;

export const closeBlockerSchema = z.object({
  field: z.string(),
  code: z.string(),
  message: z.string(),
  category: z.string(),
});

export const fiscalPeriodCloseCheckSchema = z.object({
  canClose: z.boolean(),
  blockers: z.array(closeBlockerSchema),
});

export type FiscalPeriodCloseCheck = z.infer<typeof fiscalPeriodCloseCheckSchema>;
