import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import type { CarrierSettlementControl } from "@/types/carrier-settlement-control";
import { defineLabels } from "@trenova/shared/i18n/labels";

export type CarrierSettlementControlTab = "payPeriod" | "matching" | "approval";

export type CarrierSettlementControlField = keyof CarrierSettlementControl & string;

export type CarrierSettlementControlTabSpec = SettingsTabSpec<CarrierSettlementControlField> & {
  value: CarrierSettlementControlTab;
};

const TAB_LABELS = defineLabels({
  payPeriod: "Pay period",
  matching: "Invoice matching",
  approval: "Approval & posting",
});

/** The form fields each tab holds, so a refused save marks and opens the tab at fault. */
const TAB_FIELDS: Record<CarrierSettlementControlTab, readonly CarrierSettlementControlField[]> = {
  payPeriod: [
    "payPeriodFrequency",
    "periodEndDayOfWeek",
    "payDelayDays",
    "autoGenerateBatches",
    "payTrigger",
  ],
  matching: [
    "varianceToleranceMinor",
    "autoMatchInboundInvoices",
    "autoAcceptWithinTolerance",
    "holdUntilInvoiceMatched",
  ],
  approval: [
    "requireSeparateApprover",
    "autoPostOnApprove",
    "defaultApAccountId",
    "defaultPurchasedTransportationAccountId",
  ],
};

const TAB_ORDER: readonly CarrierSettlementControlTab[] = ["payPeriod", "matching", "approval"];

/**
 * The tabs of carrier settlement control, in order: when carriers are paid and for what, how
 * their invoices are checked against the buy rate, and how a settlement is approved and
 * posted. Every organization that reaches the page (a brokerage) uses all three, so none is
 * hidden.
 */
export function carrierSettlementControlTabs(): CarrierSettlementControlTabSpec[] {
  return TAB_ORDER.map((tab) => ({
    value: tab,
    label: TAB_LABELS[tab],
    fields: TAB_FIELDS[tab],
  }));
}

/** The tabs the form and its loading state both draw. */
export function useCarrierSettlementControlTabs(): CarrierSettlementControlTabSpec[] {
  return carrierSettlementControlTabs();
}
