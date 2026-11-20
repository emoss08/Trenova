import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import { PlanFeature } from "@/lib/plan-feature";
import type { DashControl } from "@/types/dash-control";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import { defineLabels } from "@trenova/shared/i18n/labels";

export type DashControlTab = "loads" | "pay" | "profile" | "reminders";

export type DashControlField = keyof DashControl & string;

export type DashControlTabSpec = SettingsTabSpec<DashControlField> & { value: DashControlTab };

const TAB_LABELS = defineLabels({
  loads: "Loads",
  pay: "Pay & expenses",
  profile: "Profile & time off",
  reminders: "Reminders",
});

/** The form fields each tab holds, so a refused save marks and opens the tab at fault. */
const TAB_FIELDS: Record<DashControlTab, readonly DashControlField[]> = {
  loads: [
    "requireLoadAcknowledgment",
    "allowLoadRefusals",
    "allowStopActions",
    "allowLoadDocumentUpload",
    "allowLoadComments",
    "enableDetentionAlerts",
    "detentionAlertThresholdMinutes",
  ],
  pay: [
    "showLoadPay",
    "showPayEstimates",
    "allowExpenseSubmission",
    "requireExpenseReceipt",
    "allowSettlementDisputes",
  ],
  profile: [
    "allowProfileDocumentUpload",
    "allowContactInfoEdit",
    "requireContactChangeApproval",
    "allowPtoRequests",
  ],
  reminders: ["sendCredentialReminders", "driverDigestCadence", "driverDigestWeekday"],
};

const TAB_ORDER: readonly DashControlTab[] = ["loads", "pay", "profile", "reminders"];

export type DashControlPlan = {
  /** Driver settlements: the pay, expenses and disputes the Pay tab shows drivers. */
  driverPay: boolean;
  /** Credentials or training: what the reminders are about. */
  reminders: boolean;
};

/** What the organization's plan includes that decides which tabs it sees. */
export function useDashControlPlan(): DashControlPlan {
  const driverPay = usePlanFeature(PlanFeature.DriverPay).available;
  const compliance = usePlanFeature(PlanFeature.WorkforceCompliance).available;
  const training = usePlanFeature(PlanFeature.WorkforceTalent).available;

  return { driverPay, reminders: compliance || training };
}

/**
 * The tabs an organization sees, in order. Pay & expenses needs driver settlements, which
 * hold the pay, reimbursements and statements those settings show drivers; Reminders needs
 * credentials or training to remind drivers of. A tab left out still saves its values as
 * they were. The form and its loading state both read this.
 */
export function dashControlTabs(plan: DashControlPlan): DashControlTabSpec[] {
  return TAB_ORDER.filter(
    (tab) => (tab !== "pay" || plan.driverPay) && (tab !== "reminders" || plan.reminders),
  ).map((tab) => ({ value: tab, label: TAB_LABELS[tab], fields: TAB_FIELDS[tab] }));
}

/** The tabs this organization sees, for the form and its loading state alike. */
export function useDashControlTabs(): DashControlTabSpec[] {
  return dashControlTabs(useDashControlPlan());
}
