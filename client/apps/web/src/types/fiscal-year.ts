import { z } from "zod";
import { fiscalPeriodSchema } from "./fiscal-period";
import {
  nullableIntegerSchema,
  optionalStringSchema,
  tenantInfoSchema,
} from "./helpers";

const fiscalYearStatusSchema = z.enum(["Draft", "Open", "Closed", "Locked"]);
export type FiscalYearStatus = z.infer<typeof fiscalYearStatusSchema>;

export const fiscalYearSchema = z.object({
  ...tenantInfoSchema.shape,
  status: fiscalYearStatusSchema,
  year: z.number().int().min(1900).max(2100),
  name: z.string().min(1).max(100),
  description: optionalStringSchema,
  startDate: z.number().int(),
  endDate: z.number().int(),
  taxYear: nullableIntegerSchema,
  budgetAmount: nullableIntegerSchema,
  adjustmentDeadline: nullableIntegerSchema,
  isCurrent: z.boolean().optional(),
  isCalendarYear: z.boolean().optional(),
  allowAdjustingEntries: z.boolean().optional(),
  closedAt: z.number().int().nullish(),
  lockedAt: z.number().int().nullish(),
  closedById: z.string().nullish(),
  lockedById: z.string().nullish(),
  periods: z.array(fiscalPeriodSchema).optional(),
});

export type FiscalYear = z.infer<typeof fiscalYearSchema>;
