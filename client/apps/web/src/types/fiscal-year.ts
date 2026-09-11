import { z } from "zod";
import { fiscalPeriodSchema } from "./fiscal-period";
import {
  nullableIntegerSchema,
  optionalStringSchema,
  tenantInfoSchema,
} from "@trenova/shared/types/helpers";

const fiscalYearStatusSchema = z.enum(["Draft", "Open", "Closed", "PermanentlyClosed"]);
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

const closeBlockerSchema = z.object({
  field: z.string(),
  code: z.string(),
  message: z.string(),
  category: z.string(),
});

export type FiscalYearCloseBlocker = z.infer<typeof closeBlockerSchema>;

const closePlanLineSchema = z.object({
  glAccountId: z.string(),
  accountCode: z.string(),
  accountName: z.string(),
  accountCategory: z.string(),
  debitMinor: z.number().int(),
  creditMinor: z.number().int(),
  isRetainedEarnings: z.boolean(),
});

export type FiscalYearClosePlanLine = z.infer<typeof closePlanLineSchema>;

const closePlanEntrySchema = z.object({
  kind: z.enum(["Closing", "Opening"]),
  fiscalYearId: z.string(),
  fiscalPeriodId: z.string(),
  fiscalPeriodName: z.string(),
  createsPeriod: z.boolean(),
  accountingDate: z.number().int(),
  description: z.string(),
  totalDebitMinor: z.number().int(),
  totalCreditMinor: z.number().int(),
  lines: z.array(closePlanLineSchema),
});

export type FiscalYearClosePlanEntry = z.infer<typeof closePlanEntrySchema>;

const subledgerCheckSchema = z.object({
  key: z.string(),
  label: z.string(),
  glAccountId: z.string(),
  accountCode: z.string(),
  accountName: z.string(),
  glBalanceMinor: z.number().int(),
  subledgerBalanceMinor: z.number().int(),
  differenceMinor: z.number().int(),
  toleranceMinor: z.number().int(),
  reconciled: z.boolean(),
  enforced: z.boolean(),
});

export type FiscalYearSubledgerCheck = z.infer<typeof subledgerCheckSchema>;

/**
 * What closing a fiscal year would post, computed without writing anything: the
 * closing entry that empties the income statement into retained earnings, the
 * opening entry that carries the balance sheet into the next year, and every
 * reason the close cannot run yet.
 */
export const fiscalYearClosePlanSchema = z.object({
  fiscalYearId: z.string(),
  fiscalYearName: z.string(),
  nextFiscalYearId: z.string().nullish(),
  nextFiscalYearName: z.string().nullish(),
  retainedEarningsAccountId: z.string().nullish(),
  retainedEarningsAccountCode: z.string().nullish(),
  retainedEarningsAccountName: z.string().nullish(),
  revenueMinor: z.number().int(),
  costOfRevenueMinor: z.number().int(),
  operatingExpenseMinor: z.number().int(),
  netIncomeMinor: z.number().int(),
  closingEntry: closePlanEntrySchema.nullish(),
  openingEntry: closePlanEntrySchema.nullish(),
  subledgerChecks: z.array(subledgerCheckSchema),
  revision: z.number().int(),
  canClose: z.boolean(),
  blockers: z.array(closeBlockerSchema),
});

export type FiscalYearClosePlan = z.infer<typeof fiscalYearClosePlanSchema>;
