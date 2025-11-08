import * as z from "zod";
import {
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const JournalEntryCriteriaSchema = z.enum([
  "ShipmentBilled",
  "PaymentReceived",
  "ExpenseRecognized",
  "DeliveryComplete",
]);

export const ThresholdActionSchema = z.enum(["Warn", "Block", "Notify"]);

export const RevenueRecognitionSchema = z.enum([
  "OnDelivery",
  "OnBilling",
  "OnPayment",
  "OnPickup",
]);

export const ExpenseRecognitionSchema = z.enum([
  "OnIncurrence",
  "OnAccrual",
  "OnPayment",
]);

export const accountingControlSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  autoCreateJournalEntries: z.boolean(),
  journalEntryCriteria: JournalEntryCriteriaSchema,
  defaultRevenueAccountId: nullableStringSchema,
  defaultExpenseAccountId: nullableStringSchema,
  restrictManualJournalEntries: z.boolean(),
  requireJournalEntryApproval: z.boolean(),
  enableJournalEntryReversal: z.boolean(),
  allowPostingToClosedPeriods: z.boolean(),
  requirePeriodEndApproval: z.boolean(),
  autoClosePeriods: z.boolean(),
  enableReconciliation: z.boolean(),
  reconciliationThreshold: z
    .number()
    .min(1, { error: "Reconciliation threshold must be at least 1" })
    .max(10000, { error: "Reconciliation threshold cannot exceed 10,000" }),
  reconciliationThresholdAction: ThresholdActionSchema,
  haltOnPendingReconciliation: z.boolean(),
  enableReconciliationNotifications: z.boolean(),
  revenueRecognitionMethod: RevenueRecognitionSchema,
  deferRevenueUntilPaid: z.boolean(),
  expenseRecognitionMethod: ExpenseRecognitionSchema,
  accrueExpenses: z.boolean(),
  enableAutomaticTaxCalculation: z.boolean(),
  defaultTaxAccountId: nullableStringSchema,
  requireDocumentAttachment: z.boolean(),
  retainDeletedEntries: z.boolean(),
  enableMultiCurrency: z.boolean(),
  defaultCurrencyCode: z
    .string()
    .min(3, { error: "Default currency code must be 3 characters" })
    .max(3, { error: "Default currency code must be 3 characters" }),
  currencyGainAccountId: nullableStringSchema,
  currencyLossAccountId: nullableStringSchema,
});

export type AccountingControlSchema = z.infer<typeof accountingControlSchema>;
