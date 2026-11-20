import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import type { SettingsSectionSpec } from "@/components/settings-form-skeleton";
import { useLeaveControlTabs, type LeaveControlTab } from "./leave-control-tabs";

const choices = (count: number) => ({ kind: "input", count, narrow: false }) as const;
const figures = (count: number) => ({ kind: "input", count, narrow: true }) as const;

/** The one tab's sections as the form draws them, one row to a setting. */
const TAB_SECTIONS: Record<LeaveControlTab, readonly SettingsSectionSpec[]> = {
  policy: [
    { title: "sm", description: 1, fields: [choices(1)] },
    { title: "sm", description: 1, fields: [figures(3)] },
    { title: "sm", description: 1, fields: [figures(2)] },
    { title: "sm", description: 1, fields: [figures(1)] },
  ],
};

/** The form's sections, with no tab row since the page has one tab. */
export function LeaveControlSkeleton() {
  return <SettingsPageSkeleton tabs={useLeaveControlTabs()} sections={TAB_SECTIONS} />;
}
