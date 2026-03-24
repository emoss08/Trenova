import { z } from "zod";
import { tenantInfoSchema } from "./helpers";

const fiscalPeriodStatusSchema = z.enum(["Open", "Closed", "Locked"]);
export type FiscalPeriodStatus = z.infer<typeof fiscalPeriodStatusSchema>;

const periodTypeSchema = z.enum(["Month", "Quarter", "Year"]);
export type PeriodType = z.infer<typeof periodTypeSchema>;

export const fiscalPeriodSchema = z.object({
  ...tenantInfoSchema.shape,
  fiscalYearId: z.string().min(1),
  status: fiscalPeriodStatusSchema,
  periodType: periodTypeSchema,
  periodNumber: z.number().int().min(1).max(12),
  name: z.string().min(1).max(100),
  startDate: z.number().int(),
  endDate: z.number().int(),
  closedAt: z.number().int().nullish(),
  closedById: z.string().nullish(),
});

export type FiscalPeriod = z.infer<typeof fiscalPeriodSchema>;
