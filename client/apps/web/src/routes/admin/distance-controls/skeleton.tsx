import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import type { SettingsSectionSpec } from "@/components/settings-form-skeleton";
import { useDistanceControlTabs, type DistanceControlTab } from "./distance-control-tabs";

const choices = (count: number) => ({ kind: "input", count, narrow: false }) as const;

/**
 * The one tab's sections as the form draws it, one row to a setting. City matching waits on
 * the two stored mileage switches and is drawn closed, since the outline cannot know the
 * saved values.
 */
const TAB_SECTIONS: Record<DistanceControlTab, readonly SettingsSectionSpec[]> = {
  mileage: [
    { title: "sm", description: 1, fields: [choices(3)] },
    { title: "md", description: 1, fields: [choices(3)] },
    { title: "md", description: 1, fields: [{ kind: "switch", count: 2 }] },
    { title: "md", description: 1, fields: ["switch"] },
    { title: "sm", description: 1, fields: [choices(1)] },
  ],
};

/** The form's sections, with no tab row since the page has one tab. */
export function DistanceControlSkeleton() {
  return <SettingsPageSkeleton tabs={useDistanceControlTabs()} sections={TAB_SECTIONS} />;
}
