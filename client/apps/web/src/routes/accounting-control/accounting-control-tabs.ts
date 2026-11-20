import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import type { AccountingControl } from "@/types/accounting-control";
import { PlanFeature } from "@/lib/plan-feature";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import {
  hasOrganizationCapability,
  OrganizationCapability,
} from "@trenova/shared/types/organization-capability";
import { defineLabels } from "@trenova/shared/i18n/labels";

export type AccountingControlTab = "posting" | "accounts" | "driverPay" | "close" | "currency";

export type AccountingControlField = keyof AccountingControl & string;

export type AccountingControlTabSpec = SettingsTabSpec<AccountingControlField> & {
  value: AccountingControlTab;
};

const TAB_LABELS = defineLabels({
  posting: "Posting",
  accounts: "Default accounts",
  driverPay: "Driver pay",
  close: "Close & reconciliation",
  currency: "Currency",
});

/** The form fields each tab holds, so a refused save marks and opens the tab at fault. */
const TAB_FIELDS: Record<AccountingControlTab, readonly AccountingControlField[]> = {
  posting: [
    "accountingBasis",
    "revenueRecognitionPolicy",
    "expenseRecognitionPolicy",
    "journalPostingMode",
    "autoPostSourceEvents",
    "manualJournalEntryPolicy",
    "requireManualJeApproval",
    "journalReversalPolicy",
  ],
  accounts: [
    "defaultRevenueAccountId",
    "defaultArAccountId",
    "defaultCashAccountId",
    "defaultUnappliedCashAccountId",
    "defaultWriteOffAccountId",
    "defaultTaxLiabilityAccountId",
    "defaultExpenseAccountId",
    "defaultApAccountId",
    "defaultRetainedEarningsAccountId",
  ],
  driverPay: [
    "defaultDriverPayExpenseAccountId",
    "defaultPurchasedTransportationAccountId",
    "defaultDriverReimbursementAccountId",
    "defaultSettlementsPayableAccountId",
    "defaultDriverAdvanceAccountId",
    "defaultEscrowLiabilityAccountId",
    "defaultEscrowInterestExpenseAccountId",
    "defaultEscrowAdjustmentAccountId",
  ],
  close: [
    "periodCloseMode",
    "requirePeriodCloseApproval",
    "lockedPeriodPostingPolicy",
    "closedPeriodPostingPolicy",
    "reconciliationMode",
    "reconciliationToleranceAmount",
    "requireReconciliationToClose",
    "notifyOnReconciliationException",
  ],
  currency: [
    "currencyMode",
    "functionalCurrencyCode",
    "exchangeRateDatePolicy",
    "exchangeRateOverridePolicy",
    "realizedFxGainAccountId",
    "realizedFxLossAccountId",
  ],
};

const TAB_ORDER: readonly AccountingControlTab[] = [
  "posting",
  "accounts",
  "driverPay",
  "close",
  "currency",
];

/** The tabs this organization sees, for the form and its loading state alike. */
export function useAccountingControlTabs(): AccountingControlTabSpec[] {
  const capabilities = useOrgCapabilities();
  const driverPay = usePlanFeature(PlanFeature.DriverPay).available;

  return accountingControlTabs({
    runsAssets: hasOrganizationCapability(capabilities, OrganizationCapability.AssetOperations),
    driverPay,
  });
}

/**
 * The tabs an organization sees, in order. Driver pay is offered only to one that runs a
 * fleet and whose plan includes driver pay: a brokerage settles no drivers. The form and
 * its loading state both read this, so the outline has the form's tabs.
 */
export function accountingControlTabs({
  runsAssets,
  driverPay,
}: {
  runsAssets: boolean;
  driverPay: boolean;
}): AccountingControlTabSpec[] {
  return TAB_ORDER.filter((tab) => tab !== "driverPay" || (runsAssets && driverPay)).map((tab) => ({
    value: tab,
    label: TAB_LABELS[tab],
    fields: TAB_FIELDS[tab],
  }));
}
