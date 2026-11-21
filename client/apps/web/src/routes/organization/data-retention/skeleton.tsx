import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import type { SettingsSectionSpec } from "@/components/settings-form-skeleton";
import { useDataRetentionTabs, type DataRetentionTab } from "./data-retention-tabs";

const periods = (count: number) => ({ kind: "input", count, narrow: true }) as const;

/**
 * The one tab's sections as the form draws them, one row to a period. The purges that only
 * run once they are given a period are drawn with their switch off, as a new organization's
 * EDI purges stand, since the outline cannot know the saved values.
 */
const TAB_SECTIONS: Record<DataRetentionTab, readonly SettingsSectionSpec[]> = {
  retention: [
    { title: "sm", description: 2, fields: [periods(4)] },
    { title: "sm", description: 2, fields: ["switch", "switch"] },
    { title: "sm", description: 2, fields: [periods(1)] },
    { title: "md", description: 2, fields: [periods(3), "switch"] },
  ],
};

/** The form's sections, with no tab row since the page has one tab. */
export function DataRetentionSkeleton() {
  return <SettingsPageSkeleton tabs={useDataRetentionTabs()} sections={TAB_SECTIONS} />;
}
