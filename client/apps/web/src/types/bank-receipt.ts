import { z } from "zod";
import { nullableStringSchema, optionalStringSchema, tenantInfoSchema } from "./helpers";

export const bankReceiptStatusSchema = z.enum(["Imported", "Matched", "Exception"]);
export type BankReceiptStatus = z.infer<typeof bankReceiptStatusSchema>;

export const bankReceiptSchema = z.object({
  ...tenantInfoSchema.shape,
  receiptDate: z.number().int(),
  amountMinor: z.number().int(),
  referenceNumber: z.string(),
  memo: optionalStringSchema,
  status: bankReceiptStatusSchema,
  matchedCustomerPaymentId: nullableStringSchema,
  matchedAt: z.number().int().nullish(),
  matchedById: nullableStringSchema,
  exceptionReason: optionalStringSchema,
  importBatchId: nullableStringSchema,
  createdById: optionalStringSchema,
  updatedById: optionalStringSchema,
});
export type BankReceipt = z.infer<typeof bankReceiptSchema>;

export const matchSuggestionSchema = z.object({
  customerPaymentId: z.string(),
  referenceNumber: z.string(),
  amountMinor: z.number().int(),
  customerId: z.string(),
  score: z.number().int(),
  reason: z.string(),
});
export type MatchSuggestion = z.infer<typeof matchSuggestionSchema>;

export const exceptionAgingSchema = z.object({
  currentCount: z.number().int(),
  days1To3Count: z.number().int(),
  days4To7Count: z.number().int(),
  daysOver7Count: z.number().int(),
});
export type ExceptionAging = z.infer<typeof exceptionAgingSchema>;

export const reconciliationSummarySchema = z.object({
  asOfDate: z.number().int(),
  importedCount: z.number().int(),
  importedAmount: z.number().int(),
  matchedCount: z.number().int(),
  matchedAmount: z.number().int(),
  exceptionCount: z.number().int(),
  exceptionAmount: z.number().int(),
  activeWorkItemCount: z.number().int(),
  assignedWorkItemCount: z.number().int(),
  inReviewWorkItemCount: z.number().int(),
  exceptionAging: exceptionAgingSchema,
});
export type ReconciliationSummary = z.infer<typeof reconciliationSummarySchema>;
