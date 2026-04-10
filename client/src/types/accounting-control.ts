import { z } from "zod";
import {
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

const decimalNumberSchema = z.coerce.number().finite();

export const accountingBasisSchema = z.enum(["Accrual", "Cash"]);
export type AccountingBasis = z.infer<typeof accountingBasisSchema>;

export const revenueRecognitionPolicySchema = z.enum(["OnInvoicePost", "OnCashReceipt"]);
export type RevenueRecognitionPolicy = z.infer<typeof revenueRecognitionPolicySchema>;

export const expenseRecognitionPolicySchema = z.enum([
  "OnVendorBillPost",
  "OnCashDisbursement",
]);
export type ExpenseRecognitionPolicy = z.infer<typeof expenseRecognitionPolicySchema>;

export const journalPostingModeSchema = z.enum(["Manual", "Automatic"]);
export type JournalPostingMode = z.infer<typeof journalPostingModeSchema>;

export const journalSourceEventSchema = z.enum([
  "InvoicePosted",
  "CreditMemoPosted",
  "DebitMemoPosted",
  "CustomerPaymentPosted",
  "VendorBillPosted",
  "VendorPaymentPosted",
]);
export type JournalSourceEvent = z.infer<typeof journalSourceEventSchema>;

export const manualJournalEntryPolicySchema = z.enum([
  "AllowAll",
  "AdjustmentOnly",
  "Disallow",
]);
export type ManualJournalEntryPolicy = z.infer<typeof manualJournalEntryPolicySchema>;

export const journalReversalPolicySchema = z.enum(["Disallow", "NextOpenPeriod"]);
export type JournalReversalPolicy = z.infer<typeof journalReversalPolicySchema>;

export const periodCloseModeSchema = z.enum(["ManualOnly", "SystemScheduled"]);
export type PeriodCloseMode = z.infer<typeof periodCloseModeSchema>;

export const lockedPeriodPostingPolicySchema = z.enum(["BlockSubledgerAllowManualJe"]);
export type LockedPeriodPostingPolicy = z.infer<typeof lockedPeriodPostingPolicySchema>;

export const closedPeriodPostingPolicySchema = z.enum(["RequireReopen", "PostToNextOpen"]);
export type ClosedPeriodPostingPolicy = z.infer<typeof closedPeriodPostingPolicySchema>;

export const reconciliationModeSchema = z.enum(["Disabled", "WarnOnly", "BlockPosting"]);
export type ReconciliationMode = z.infer<typeof reconciliationModeSchema>;

export const currencyModeSchema = z.enum(["SingleCurrency", "MultiCurrency"]);
export type CurrencyMode = z.infer<typeof currencyModeSchema>;

export const exchangeRateDatePolicySchema = z.enum(["DocumentDate", "AccountingDate"]);
export type ExchangeRateDatePolicy = z.infer<typeof exchangeRateDatePolicySchema>;

export const exchangeRateOverridePolicySchema = z.enum([
  "Allow",
  "RequireApproval",
  "Disallow",
]);
export type ExchangeRateOverridePolicy = z.infer<typeof exchangeRateOverridePolicySchema>;

export const accountingControlSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  accountingBasis: accountingBasisSchema,
  revenueRecognitionPolicy: revenueRecognitionPolicySchema,
  expenseRecognitionPolicy: expenseRecognitionPolicySchema,

  journalPostingMode: journalPostingModeSchema,
  autoPostSourceEvents: z.array(journalSourceEventSchema).default([]),
  manualJournalEntryPolicy: manualJournalEntryPolicySchema,
  requireManualJeApproval: z.boolean(),
  journalReversalPolicy: journalReversalPolicySchema,

  periodCloseMode: periodCloseModeSchema,
  requirePeriodCloseApproval: z.boolean(),
  lockedPeriodPostingPolicy: lockedPeriodPostingPolicySchema,
  closedPeriodPostingPolicy: closedPeriodPostingPolicySchema,
  requireReconciliationToClose: z.boolean(),

  reconciliationMode: reconciliationModeSchema,
  reconciliationToleranceAmount: decimalNumberSchema,
  notifyOnReconciliationException: z.boolean(),

  currencyMode: currencyModeSchema,
  functionalCurrencyCode: z.string().min(3).max(3),
  exchangeRateDatePolicy: exchangeRateDatePolicySchema,
  exchangeRateOverridePolicy: exchangeRateOverridePolicySchema,

  defaultRevenueAccountId: nullableStringSchema,
  defaultExpenseAccountId: nullableStringSchema,
  defaultArAccountId: nullableStringSchema,
  defaultApAccountId: nullableStringSchema,
  defaultTaxLiabilityAccountId: nullableStringSchema,
  defaultWriteOffAccountId: nullableStringSchema,
  defaultRetainedEarningsAccountId: nullableStringSchema,
  realizedFxGainAccountId: nullableStringSchema,
  realizedFxLossAccountId: nullableStringSchema,
});

export type AccountingControl = z.infer<typeof accountingControlSchema>;
