import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";
import {
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "@trenova/shared/types/helpers";

const decimalNumberSchema = z.coerce.number().finite();

export const accountingBasisSchema = z.enum(["Accrual", "Cash"]);
export type AccountingBasis = z.infer<typeof accountingBasisSchema>;

export const revenueRecognitionPolicySchema = z.enum(["OnInvoicePost", "OnCashReceipt"]);
export type RevenueRecognitionPolicy = z.infer<typeof revenueRecognitionPolicySchema>;

export const expenseRecognitionPolicySchema = z.enum(["OnVendorBillPost", "OnCashDisbursement"]);
export type ExpenseRecognitionPolicy = z.infer<typeof expenseRecognitionPolicySchema>;

export const journalPostingModeSchema = z.enum(["Manual", "Automatic"]);
export type JournalPostingMode = z.infer<typeof journalPostingModeSchema>;

export const journalSourceEventSchema = z.enum([
  "InvoicePosted",
  "CreditMemoPosted",
  "DebitMemoPosted",
  "CustomerPaymentPosted",
  "CustomerShortPayRecognized",
  "CustomerPaymentReversed",
  "VendorBillPosted",
  "VendorPaymentPosted",
  "DriverSettlementPosted",
  "DriverSettlementVoided",
  "EscrowInterestAccrued",
  "EscrowAdjusted",
  "EscrowRefunded",
  "CarrierSettlementPosted",
  "CarrierSettlementVoided",
  "CarrierSettlementPaid",
  "DriverSettlementPaid",
  "FactoringFunded",
  "FactoringReserveReleased",
  "FactoringChargedBack",
  "FactoringChargebackReversed",
]);
export type JournalSourceEvent = z.infer<typeof journalSourceEventSchema>;

export const manualJournalEntryPolicySchema = z.enum(["AllowAll", "AdjustmentOnly", "Disallow"]);
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

export const exchangeRateOverridePolicySchema = z.enum(["Allow", "RequireApproval", "Disallow"]);
export type ExchangeRateOverridePolicy = z.infer<typeof exchangeRateOverridePolicySchema>;

const basis = accountingBasisSchema.enum;
const revenue = revenueRecognitionPolicySchema.enum;
const expense = expenseRecognitionPolicySchema.enum;
const event = journalSourceEventSchema.enum;

export type Recognition = {
  revenue: RevenueRecognitionPolicy;
  expense: ExpenseRecognitionPolicy;
};

/**
 * The one revenue and one expense policy each accounting basis allows, as the server holds
 * them (`ValidRevenueRecognitionPolicies` and `ValidExpenseRecognitionPolicies`).
 */
export const RECOGNITION_BY_BASIS: Readonly<Record<AccountingBasis, Recognition>> = {
  [basis.Accrual]: { revenue: revenue.OnInvoicePost, expense: expense.OnVendorBillPost },
  [basis.Cash]: { revenue: revenue.OnCashReceipt, expense: expense.OnCashDisbursement },
};

/** The events automatic posting must take for each policy to recognize anything at all. */
const EVENTS_BY_REVENUE: Readonly<Record<RevenueRecognitionPolicy, readonly JournalSourceEvent[]>> =
  {
    [revenue.OnInvoicePost]: [event.InvoicePosted, event.CreditMemoPosted, event.DebitMemoPosted],
    [revenue.OnCashReceipt]: [event.CustomerPaymentPosted],
  };

const EVENTS_BY_EXPENSE: Readonly<Record<ExpenseRecognitionPolicy, readonly JournalSourceEvent[]>> =
  {
    [expense.OnVendorBillPost]: [event.VendorBillPosted],
    [expense.OnCashDisbursement]: [event.VendorPaymentPosted],
  };

/** The events the server requires among the auto-post events for these policies. */
export function requiredSourceEvents(recognition: Recognition): JournalSourceEvent[] {
  return [...EVENTS_BY_REVENUE[recognition.revenue], ...EVENTS_BY_EXPENSE[recognition.expense]];
}

/**
 * The events automatic posting may not take under a revenue policy: revenue recognized on
 * cash receipt is not also posted from invoices and memos.
 */
export function blockedSourceEvents(policy: RevenueRecognitionPolicy): JournalSourceEvent[] {
  return policy === revenue.OnCashReceipt ? [...EVENTS_BY_REVENUE[revenue.OnInvoicePost]] : [];
}

/**
 * An organization's accounting control, as the server sends it. What the settings require of
 * each other the form checks (`accountingControlFormSchema`).
 */
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
  defaultCashAccountId: nullableStringSchema,
  defaultUnappliedCashAccountId: nullableStringSchema,
  defaultExpenseAccountId: nullableStringSchema,
  defaultArAccountId: nullableStringSchema,
  defaultApAccountId: nullableStringSchema,
  defaultTaxLiabilityAccountId: nullableStringSchema,
  defaultWriteOffAccountId: nullableStringSchema,
  defaultRetainedEarningsAccountId: nullableStringSchema,
  realizedFxGainAccountId: nullableStringSchema,
  realizedFxLossAccountId: nullableStringSchema,

  defaultDriverPayExpenseAccountId: nullableStringSchema,
  defaultPurchasedTransportationAccountId: nullableStringSchema,
  defaultSettlementsPayableAccountId: nullableStringSchema,
  defaultDriverAdvanceAccountId: nullableStringSchema,
  defaultEscrowLiabilityAccountId: nullableStringSchema,
  defaultDriverReimbursementAccountId: nullableStringSchema,
  defaultEscrowInterestExpenseAccountId: nullableStringSchema,
  defaultEscrowAdjustmentAccountId: nullableStringSchema,
});

export type AccountingControl = z.infer<typeof accountingControlSchema>;

type AccountField = keyof Pick<
  AccountingControl,
  | "defaultRevenueAccountId"
  | "defaultExpenseAccountId"
  | "defaultArAccountId"
  | "defaultApAccountId"
  | "defaultCashAccountId"
  | "defaultUnappliedCashAccountId"
  | "realizedFxGainAccountId"
  | "realizedFxLossAccountId"
>;

/**
 * The record as the form checks it before a save, following the server's validator: the
 * recognition policies the basis allows; under automatic posting, at least one event, every
 * event the policies need and none they block, and the accounts journals post to (cash and
 * unapplied cash once customer payments post); both realized FX accounts under several
 * currencies; and a tolerance above zero while reconciliation runs. A value the page hides is
 * not checked here; it is sent as the server requires (`sanitizeAccountingControlForSubmit`).
 * Whether the exchange-rate integration is ready only the server can tell.
 */
export const accountingControlFormSchema = accountingControlSchema.superRefine((values, ctx) => {
  const refuse = (path: keyof AccountingControl, message: string) => {
    ctx.addIssue({ code: "custom", path: [path], message });
  };
  const requireAccount = (path: AccountField, message: string) => {
    if (!values[path]) refuse(path, message);
  };

  const allowed = RECOGNITION_BY_BASIS[values.accountingBasis];
  if (values.revenueRecognitionPolicy !== allowed.revenue) {
    refuse(
      "revenueRecognitionPolicy",
      translate("Choose the revenue recognition this accounting basis allows"),
    );
  }
  if (values.expenseRecognitionPolicy !== allowed.expense) {
    refuse(
      "expenseRecognitionPolicy",
      translate("Choose the expense recognition this accounting basis allows"),
    );
  }

  if (values.journalPostingMode === journalPostingModeSchema.enum.Automatic) {
    const events = values.autoPostSourceEvents;
    if (events.length === 0) {
      refuse("autoPostSourceEvents", translate("Choose at least one event that posts journals"));
    } else {
      const recognition = {
        revenue: values.revenueRecognitionPolicy,
        expense: values.expenseRecognitionPolicy,
      };
      if (requiredSourceEvents(recognition).some((entry) => !events.includes(entry))) {
        refuse(
          "autoPostSourceEvents",
          translate("Include every event your recognition policies depend on"),
        );
      }
      if (
        blockedSourceEvents(values.revenueRecognitionPolicy).some((entry) => events.includes(entry))
      ) {
        refuse(
          "autoPostSourceEvents",
          translate(
            "Invoices and memos cannot post journals while revenue is recognized on cash receipt",
          ),
        );
      }
    }

    requireAccount("defaultRevenueAccountId", translate("Choose a revenue account"));
    requireAccount("defaultExpenseAccountId", translate("Choose an expense account"));
    requireAccount("defaultArAccountId", translate("Choose an accounts receivable account"));
    requireAccount("defaultApAccountId", translate("Choose an accounts payable account"));
    if (
      events.includes(journalSourceEventSchema.enum.CustomerPaymentPosted) ||
      values.revenueRecognitionPolicy === revenueRecognitionPolicySchema.enum.OnCashReceipt
    ) {
      requireAccount("defaultCashAccountId", translate("Choose a cash account"));
      requireAccount(
        "defaultUnappliedCashAccountId",
        translate("Choose an unapplied cash account"),
      );
    }
  }

  if (values.currencyMode === currencyModeSchema.enum.MultiCurrency) {
    requireAccount("realizedFxGainAccountId", translate("Choose a realized FX gain account"));
    requireAccount("realizedFxLossAccountId", translate("Choose a realized FX loss account"));
  }

  if (
    values.reconciliationMode !== reconciliationModeSchema.enum.Disabled &&
    !(values.reconciliationToleranceAmount > 0)
  ) {
    refuse("reconciliationToleranceAmount", translate("Enter a tolerance above zero"));
  }
});
