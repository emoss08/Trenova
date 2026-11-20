import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import { PlanFeature } from "@/lib/plan-feature";
import type { CostControlFormValues } from "@/types/cost-control";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import { defineLabels } from "@trenova/shared/i18n/labels";

export type CostControlTab = "basis" | "rates" | "actuals";

export type CostControlField = keyof CostControlFormValues & string;

export type CostControlTabSpec = SettingsTabSpec<CostControlField> & {
  value: CostControlTab;
};

const TAB_LABELS = defineLabels({
  basis: "Cost basis",
  rates: "Cost rates",
  actuals: "GL actuals",
});

/** The form fields each tab holds, so a refused save marks and opens the tab at fault. */
const TAB_FIELDS: Record<CostControlTab, readonly CostControlField[]> = {
  basis: [
    "milesPerGallon",
    "useLiveFuelPrice",
    "fuelIndexId",
    "includeDeadheadMiles",
    "targetMarginPercent",
  ],
  rates: ["categories"],
  actuals: ["glActualsEnabled", "glRollingMonths", "plannedMonthlyMiles"],
};

const TAB_ORDER: readonly CostControlTab[] = ["basis", "rates", "actuals"];

/** The tabs this organization sees, for the form and its loading state alike. */
export function useCostControlTabs(): CostControlTabSpec[] {
  return costControlTabs({ accounting: useCostControlAccounting() });
}

/** Whether the plan includes accounting, the ledger GL actuals read their rates from. */
export function useCostControlAccounting(): boolean {
  return usePlanFeature(PlanFeature.Accounting).available;
}

/**
 * The tabs an organization sees, in order. GL actuals derive rates from posted expenses, so
 * they are offered only to an organization whose plan includes accounting; their settings
 * still round-trip when the tab is hidden. The form and its loading state both read this,
 * so the outline has the form's tabs.
 */
export function costControlTabs({ accounting }: { accounting: boolean }): CostControlTabSpec[] {
  return TAB_ORDER.filter((tab) => tab !== "actuals" || accounting).map((tab) => ({
    value: tab,
    label: TAB_LABELS[tab],
    fields: TAB_FIELDS[tab],
  }));
}
