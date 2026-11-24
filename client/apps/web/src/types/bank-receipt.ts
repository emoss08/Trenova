import { z } from "zod";
import {
  nullableStringSchema,
  optionalStringSchema,
  tenantInfoSchema,
} from "@trenova/shared/types/helpers";

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

/** One application of the matched payment to an invoice; the number is empty when the reader may not read that invoice. */
export const bankReceiptCardApplicationSchema = z.object({
  invoiceId: z.string(),
  invoiceNumber: z.string(),
  appliedAmountMinor: z.number().int(),
  shortPayAmountMinor: z.number().int(),
});
export type BankReceiptCardApplication = z.infer<typeof bankReceiptCardApplicationSchema>;

export const bankReceiptCardPaymentSchema = z.object({
  id: z.string(),
  customerId: z.string(),
  customerName: z.string(),
  paymentMethod: z.string(),
  referenceNumber: z.string(),
  paymentDate: z.number().int(),
  status: z.string(),
  currencyCode: z.string(),
  amountMinor: z.number().int(),
  appliedAmountMinor: z.number().int(),
  unappliedAmountMinor: z.number().int(),
  applications: z.array(bankReceiptCardApplicationSchema),
});
export type BankReceiptCardPayment = z.infer<typeof bankReceiptCardPaymentSchema>;

export const bankReceiptCardWorkItemSchema = z.object({
  id: z.string(),
  status: z.string(),
  assignedTo: z.string(),
  assignedAt: z.number().int().nullish(),
  resolutionType: z.string(),
  resolutionNote: z.string(),
});
export type BankReceiptCardWorkItem = z.infer<typeof bankReceiptCardWorkItemSchema>;

export const bankReceiptCandidateKindSchema = z.enum(["CustomerPayment", "Invoice"]);
export type BankReceiptCandidateKind = z.infer<typeof bankReceiptCandidateKindSchema>;

export const bankReceiptCandidateStandingSchema = z.enum(["Exact", "SameAmount", "Short", "Over"]);
export type BankReceiptCandidateStanding = z.infer<typeof bankReceiptCandidateStandingSchema>;

/**
 * An open item the receipt most plausibly pays, from the stored projection: the invoice
 * or posted payment, why (the evidence codes), and the receipt's amount against its open amount.
 * Empty strings stand for what the reader may not read.
 */
export const bankReceiptCardCandidateSchema = z.object({
  rank: z.number().int(),
  kind: bankReceiptCandidateKindSchema,
  invoiceId: z.string(),
  invoiceNumber: z.string(),
  paymentId: z.string(),
  paymentReference: z.string(),
  customerName: z.string(),
  standing: bankReceiptCandidateStandingSchema,
  evidence: z.array(z.string()),
  openMinor: z.number().int(),
  differenceMinor: z.number().int(),
});
export type BankReceiptCardCandidate = z.infer<typeof bankReceiptCardCandidateSchema>;

/** A bank receipt as the Desk card reads it, in one request (GET /accounting/bank-receipts/:id/card/). */
export const bankReceiptCardSchema = z.object({
  receipt: bankReceiptSchema,
  payment: bankReceiptCardPaymentSchema.nullable(),
  paymentHidden: z.boolean(),
  workItem: bankReceiptCardWorkItemSchema.nullable(),
  workItemShown: z.boolean(),
  candidates: z.array(bankReceiptCardCandidateSchema),
  candidatesWithheld: z.array(bankReceiptCandidateKindSchema),
});
export type BankReceiptCard = z.infer<typeof bankReceiptCardSchema>;
