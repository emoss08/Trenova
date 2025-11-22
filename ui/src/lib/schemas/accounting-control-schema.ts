import * as z from "zod";
import { glAccountSchema } from "./gl-account-schema";
import {
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const JournalEntryCriteriaSchema = z.enum([
  "InvoicePosted",
  "BillPosted",
  "PaymentReceived",
  "PaymentMade",
  "DeliveryComplete",
  "ShipmentDispatched",
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
  "OnBilling",
]);

export const accountingControlSchema = z
  .object({
    id: optionalStringSchema,
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,

    // Default GL Accounts
    defaultRevenueAccountId: nullableStringSchema,
    defaultExpenseAccountId: nullableStringSchema,
    defaultArAccountId: nullableStringSchema,
    defaultApAccountId: nullableStringSchema,
    defaultTaxAccountId: nullableStringSchema,

    // Journal Entry Automation
    autoCreateJournalEntries: z.boolean(),
    journalEntryCriteria: JournalEntryCriteriaSchema,

    // Journal Entry Controls
    restrictManualJournalEntries: z.boolean(),
    requireJournalEntryApproval: z.boolean(),
    enableJournalEntryReversal: z.boolean(),

    // Period Controls
    allowPostingToClosedPeriods: z.boolean(),
    requirePeriodEndApproval: z.boolean(),
    autoClosePeriods: z.boolean(),

    // Reconciliation Settings
    enableReconciliation: z.boolean(),
    reconciliationThreshold: z
      .number()
      .min(1, { error: "Reconciliation threshold must be at least 1" })
      .max(10000, { error: "Reconciliation threshold cannot exceed 10,000" }),
    reconciliationThresholdAction: ThresholdActionSchema,
    haltOnPendingReconciliation: z.boolean(),
    enableReconciliationNotifications: z.boolean(),

    // Revenue Recognition
    revenueRecognitionMethod: RevenueRecognitionSchema,
    deferRevenueUntilPaid: z.boolean(),

    // Expense Recognition
    expenseRecognitionMethod: ExpenseRecognitionSchema,
    accrueExpenses: z.boolean(),

    // Tax Settings
    enableAutomaticTaxCalculation: z.boolean(),

    // Audit & Compliance
    requireDocumentAttachment: z.boolean(),
    retainDeletedEntries: z.boolean(),

    // Multi-Currency (for future expansion)
    enableMultiCurrency: z.boolean(),
    defaultCurrencyCode: z
      .string()
      .min(3, { error: "Default currency code must be 3 characters" })
      .max(3, { error: "Default currency code must be 3 characters" }),
    currencyGainAccountId: nullableStringSchema,
    currencyLossAccountId: nullableStringSchema,

    // * Relationships
    defaultRevenueAccount: glAccountSchema.nullish(),
    defaultExpenseAccount: glAccountSchema.nullish(),
    defaultArAccount: glAccountSchema.nullish(),
    defaultApAccount: glAccountSchema.nullish(),
    defaultTaxAccount: glAccountSchema.nullish(),
    currencyGainAccount: glAccountSchema.nullish(),
    currencyLossAccount: glAccountSchema.nullish(),
  })
  .refine(
    (data) => {
      if (!data.autoCreateJournalEntries && data.restrictManualJournalEntries) {
        return false;
      }

      return true;
    },
    {
      message:
        "No journal entries can be created with this configuration. You must either enable auto-creation or allow manual entries.",
      path: ["restrictManualJournalEntries"],
    },
  );

export type AccountingControlSchema = z.infer<typeof accountingControlSchema>;

export type JournalEntryCriteriaType = z.infer<
  typeof JournalEntryCriteriaSchema
>;
