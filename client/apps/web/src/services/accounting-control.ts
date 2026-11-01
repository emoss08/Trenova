import {
  AccountingControlDocument,
  UpdateAccountingControlDocument,
  type UpdateAccountingControlInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { accountingControlSchema, type AccountingControl } from "@/types/accounting-control";

export function toAccountingControlInput(data: AccountingControl): UpdateAccountingControlInput {
  return {
    version: data.version ?? 0,
    accountingBasis: data.accountingBasis,
    revenueRecognitionPolicy: data.revenueRecognitionPolicy,
    expenseRecognitionPolicy: data.expenseRecognitionPolicy,
    journalPostingMode: data.journalPostingMode,
    autoPostSourceEvents: data.autoPostSourceEvents,
    manualJournalEntryPolicy: data.manualJournalEntryPolicy,
    requireManualJeApproval: data.requireManualJeApproval,
    journalReversalPolicy: data.journalReversalPolicy,
    periodCloseMode: data.periodCloseMode,
    requirePeriodCloseApproval: data.requirePeriodCloseApproval,
    lockedPeriodPostingPolicy: data.lockedPeriodPostingPolicy,
    closedPeriodPostingPolicy: data.closedPeriodPostingPolicy,
    requireReconciliationToClose: data.requireReconciliationToClose,
    reconciliationMode: data.reconciliationMode,
    reconciliationToleranceAmount: String(data.reconciliationToleranceAmount ?? 0),
    notifyOnReconciliationException: data.notifyOnReconciliationException,
    currencyMode: data.currencyMode,
    functionalCurrencyCode: data.functionalCurrencyCode,
    exchangeRateDatePolicy: data.exchangeRateDatePolicy,
    exchangeRateOverridePolicy: data.exchangeRateOverridePolicy,
    defaultRevenueAccountId: data.defaultRevenueAccountId || null,
    defaultCashAccountId: data.defaultCashAccountId || null,
    defaultUnappliedCashAccountId: data.defaultUnappliedCashAccountId || null,
    defaultExpenseAccountId: data.defaultExpenseAccountId || null,
    defaultArAccountId: data.defaultArAccountId || null,
    defaultApAccountId: data.defaultApAccountId || null,
    defaultTaxLiabilityAccountId: data.defaultTaxLiabilityAccountId || null,
    defaultWriteOffAccountId: data.defaultWriteOffAccountId || null,
    defaultRetainedEarningsAccountId: data.defaultRetainedEarningsAccountId || null,
    realizedFxGainAccountId: data.realizedFxGainAccountId || null,
    realizedFxLossAccountId: data.realizedFxLossAccountId || null,
    defaultDriverPayExpenseAccountId: data.defaultDriverPayExpenseAccountId || null,
    defaultPurchasedTransportationAccountId: data.defaultPurchasedTransportationAccountId || null,
    defaultSettlementsPayableAccountId: data.defaultSettlementsPayableAccountId || null,
    defaultDriverAdvanceAccountId: data.defaultDriverAdvanceAccountId || null,
    defaultEscrowLiabilityAccountId: data.defaultEscrowLiabilityAccountId || null,
    defaultDriverReimbursementAccountId: data.defaultDriverReimbursementAccountId || null,
    defaultEscrowInterestExpenseAccountId: data.defaultEscrowInterestExpenseAccountId || null,
  };
}

export class AccountingControlService {
  public async get(options?: { signal?: AbortSignal }) {
    const response = await requestGraphQL({
      document: AccountingControlDocument,
      operationName: "AccountingControl",
      signal: options?.signal,
    });

    return safeParse(accountingControlSchema, response.accountingControl, "Accounting Control");
  }

  public async update(data: AccountingControl) {
    const response = await requestGraphQL({
      document: UpdateAccountingControlDocument,
      operationName: "UpdateAccountingControl",
      variables: { input: toAccountingControlInput(data) },
    });

    return safeParse(
      accountingControlSchema,
      response.updateAccountingControl,
      "Accounting Control",
    );
  }
}
