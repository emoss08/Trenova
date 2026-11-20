import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { LeaveControl } from "@trenova/shared/types/worker-leave";

export type LeaveControlTab = "policy";

export type LeaveControlField = keyof LeaveControl & string;

export type LeaveControlTabSpec = SettingsTabSpec<LeaveControlField> & {
  value: LeaveControlTab;
};

const TAB_LABELS = defineLabels({
  policy: "Leave policy",
});

/** The form fields each tab holds, so a refused save marks and opens the tab at fault. */
const TAB_FIELDS: Record<LeaveControlTab, readonly LeaveControlField[]> = {
  policy: [
    "measurementMethod",
    "entitlementWeeks",
    "militaryCaregiverWeeks",
    "workweekHours",
    "eligibilityMonths",
    "eligibilityHours",
    "certificationDueDays",
  ],
};

const TAB_ORDER: readonly LeaveControlTab[] = ["policy"];

/**
 * The page's tabs. It has one job, the leave policy every employee is held to, so it has one
 * tab and draws no tab row; every organization that reaches the page sees it.
 */
export function leaveControlTabs(): LeaveControlTabSpec[] {
  return TAB_ORDER.map((tab) => ({ value: tab, label: TAB_LABELS[tab], fields: TAB_FIELDS[tab] }));
}

/** The tabs the form and its loading state both draw. */
export function useLeaveControlTabs(): LeaveControlTabSpec[] {
  return leaveControlTabs();
}
