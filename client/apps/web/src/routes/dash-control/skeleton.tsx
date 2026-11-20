import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import type { SettingsSectionSpec } from "@/components/settings-form-skeleton";
import { useDashControlTabs, type DashControlTab } from "./dash-control-tabs";

const switches = (count: number) => ({ kind: "switch", count }) as const;

/**
 * Each tab's sections as the form draws them, one row to a setting. Rows that wait on a
 * switch are drawn closed, as they stand with that switch off, since the outline cannot know
 * the saved values.
 */
const TAB_SECTIONS: Record<DashControlTab, readonly SettingsSectionSpec[]> = {
  loads: [
    { title: "sm", description: 1, fields: [switches(1)] },
    { title: "sm", description: 1, fields: [switches(3)] },
    { title: "sm", description: 1, fields: [switches(1)] },
  ],
  pay: [
    { title: "sm", description: 1, fields: [switches(1)] },
    { title: "md", description: 1, fields: [switches(2)] },
  ],
  profile: [
    { title: "sm", description: 1, fields: [switches(2)] },
    { title: "sm", description: 1, fields: [switches(1)] },
  ],
  reminders: [{ title: "md", description: 1, fields: [switches(1)] }],
};

/** The form's tab row and the open tab's sections, for the tabs this organization sees. */
export function DashControlSkeleton() {
  return <SettingsPageSkeleton tabs={useDashControlTabs()} sections={TAB_SECTIONS} />;
}
