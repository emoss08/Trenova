import { z } from "zod";
import {
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const accountingMethodSchema = z.enum(["Accrual", "Cash", "Hybrid"]);

export type AccountingMethod = z.infer<typeof accountingMethodSchema>;

export const journalEntryCriteriaSchema = z.enum([
  "InvoicePosted",
  "BillPosted",
  "PaymentReceived",
  "PaymentMade",
  "DeliveryComplete",
  "ShipmentDispatched",
]);

export type JournalEntryCriteria = z.infer<typeof journalEntryCriteriaSchema>;

export const thresholdActionSchema = z.enum(["Warn", "Block", "Notify"]);

export type ThresholdAction = z.infer<typeof thresholdActionSchema>;

export const revenueRecognitionSchema = z.enum([
  "OnDelivery",
  "OnBilling",
  "OnPayment",
  "OnPickup",
]);

export type RevenueRecognition = z.infer<typeof revenueRecognitionSchema>;

export const expenseRecognitionSchema = z.enum(["OnIncurrence", "OnAccrual", "OnPayment"]);

export type ExpenseRecognition = z.infer<typeof expenseRecognitionSchema>;

export const accountingControlSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  accountingMethod: accountingMethodSchema,

  defaultRevenueAccountId: nullableStringSchema,
  defaultExpenseAccountId: nullableStringSchema,
  defaultArAccountId: nullableStringSchema,
  defaultApAccountId: nullableStringSchema,
  defaultTaxAccountId: nullableStringSchema,
  defaultDeferredRevenueAccountId: nullableStringSchema,
  defaultCostOfServiceAccountId: nullableStringSchema,
  defaultRetainedEarningsAccountId: nullableStringSchema,

  autoCreateJournalEntries: z.boolean(),
  journalEntryCriteria: z.array(journalEntryCriteriaSchema).default([]),

  restrictManualJournalEntries: z.boolean(),
  requireJournalEntryApproval: z.boolean(),
  enableJournalEntryReversal: z.boolean(),

  allowPostingToClosedPeriods: z.boolean(),
  requirePeriodEndApproval: z.boolean(),
  autoClosePeriods: z.boolean(),

  enableReconciliation: z.boolean(),
  reconciliationThreshold: z.coerce.number().max(10000, {
    message: "Reconciliation threshold cannot exceed 10,000",
  }),
  reconciliationThresholdAction: thresholdActionSchema,
  haltOnPendingReconciliation: z.boolean(),
  enableReconciliationNotifications: z.boolean(),

  revenueRecognitionMethod: revenueRecognitionSchema,
  deferRevenueUntilPaid: z.boolean(),

  expenseRecognitionMethod: expenseRecognitionSchema,
  accrueExpenses: z.boolean(),

  enableAutomaticTaxCalculation: z.boolean(),

  requireDocumentAttachment: z.boolean(),
  retainDeletedEntries: z.boolean(),

  enableMultiCurrency: z.boolean(),
  defaultCurrencyCode: z
    .string()
    .min(3, { message: "Currency code must be 3 characters" })
    .max(3, { message: "Currency code must be 3 characters" }),
  currencyGainAccountId: nullableStringSchema,
  currencyLossAccountId: nullableStringSchema,
});

export type AccountingControl = z.infer<typeof accountingControlSchema>;
