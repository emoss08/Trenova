import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import type { SettingsSectionSpec } from "@/components/settings-form-skeleton";
import { useSettlementControlTabs, type SettlementControlTab } from "./settlement-control-tabs";

const choices = (count: number) => ({ kind: "input", count, narrow: false }) as const;
const figures = (count: number) => ({ kind: "input", count, narrow: true }) as const;

/**
 * Each tab's sections as the form draws them, one row to a setting. The average under the
 * net pay check is drawn as it stands on a new organization's settings, where the check is
 * on (a 25% threshold), since the outline cannot know the saved values.
 */
const TAB_SECTIONS: Record<SettlementControlTab, readonly SettingsSectionSpec[]> = {
  payPeriod: [
    { title: "sm", description: 1, fields: [choices(2), figures(1), "switch"] },
    { title: "sm", description: 1, fields: [choices(1), "switch"] },
  ],
  review: [
    { title: "sm", description: 1, fields: [figures(2)] },
    { title: "sm", description: 1, fields: [{ kind: "switch", count: 3 }] },
  ],
  deductions: [
    { title: "md", description: 1, fields: ["switch"] },
    { title: "md", description: 1, fields: [figures(1), choices(1)] },
  ],
};

/** The form's tab row and the open tab's sections. */
export function SettlementControlSkeleton() {
  return <SettingsPageSkeleton tabs={useSettlementControlTabs()} sections={TAB_SECTIONS} />;
}
