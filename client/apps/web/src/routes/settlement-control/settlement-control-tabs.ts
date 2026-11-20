import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import type { SettlementControl } from "@/types/settlement-control";
import { defineLabels } from "@trenova/shared/i18n/labels";

export type SettlementControlTab = "payPeriod" | "review" | "deductions";

export type SettlementControlField = keyof SettlementControl & string;

export type SettlementControlTabSpec = SettingsTabSpec<SettlementControlField> & {
  value: SettlementControlTab;
};

const TAB_LABELS = defineLabels({
  payPeriod: "Pay period",
  review: "Review & approval",
  deductions: "Advances & escrow",
});

/** The form fields each tab holds, so a refused save marks and opens the tab at fault. */
const TAB_FIELDS: Record<SettlementControlTab, readonly SettlementControlField[]> = {
  payPeriod: [
    "payPeriodFrequency",
    "periodEndDayOfWeek",
    "payDelayDays",
    "autoGenerateBatches",
    "payTrigger",
    "autoAttachAccruals",
  ],
  review: [
    "varianceThresholdPct",
    "varianceLookbackWeeks",
    "autoApproveClean",
    "requireSeparateApprover",
    "autoPostOnApprove",
  ],
  deductions: ["allowNegativeNet", "defaultEscrowInterestRate", "escrowInterestFrequencyMonths"],
};

const TAB_ORDER: readonly SettlementControlTab[] = ["payPeriod", "review", "deductions"];

/**
 * The tabs of settlement control, in order: when drivers are paid and for what, how a
 * settlement is checked and approved, and what is held back from it. Every organization that
 * reaches the page (one that runs a fleet) uses all three, so none is hidden.
 */
export function settlementControlTabs(): SettlementControlTabSpec[] {
  return TAB_ORDER.map((tab) => ({
    value: tab,
    label: TAB_LABELS[tab],
    fields: TAB_FIELDS[tab],
  }));
}

/** The tabs the form and its loading state both draw. */
export function useSettlementControlTabs(): SettlementControlTabSpec[] {
  return settlementControlTabs();
}
