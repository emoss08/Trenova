import {
  currencyModeSchema,
  exchangeRateOverridePolicySchema,
  journalPostingModeSchema,
  manualJournalEntryPolicySchema,
  periodCloseModeSchema,
  reconciliationModeSchema,
  type AccountingControl,
} from "@/types/accounting-control";

/**
 * The record as it is saved: a setting the page hides because another one makes it
 * meaningless is sent the way the server requires it then, so a save is never refused for a
 * field the person cannot see.
 */
export function sanitizeAccountingControlForSubmit(values: AccountingControl): AccountingControl {
  const next = { ...values };

  if (next.currencyMode === currencyModeSchema.enum.SingleCurrency) {
    next.exchangeRateOverridePolicy = exchangeRateOverridePolicySchema.enum.Disallow;
    next.realizedFxGainAccountId = null;
    next.realizedFxLossAccountId = null;
  }

  if (next.journalPostingMode === journalPostingModeSchema.enum.Manual) {
    next.autoPostSourceEvents = [];
  }

  if (next.manualJournalEntryPolicy === manualJournalEntryPolicySchema.enum.Disallow) {
    next.requireManualJeApproval = false;
  }

  if (next.periodCloseMode === periodCloseModeSchema.enum.SystemScheduled) {
    next.requirePeriodCloseApproval = false;
  }

  if (next.reconciliationMode === reconciliationModeSchema.enum.Disabled) {
    next.reconciliationToleranceAmount = 0;
    next.requireReconciliationToClose = false;
  }

  return next;
}
