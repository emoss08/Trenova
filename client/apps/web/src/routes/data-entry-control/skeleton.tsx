import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import type { SettingsSectionSpec } from "@/components/settings-form-skeleton";
import { useDataEntryControlTabs, type DataEntryControlTab } from "./data-entry-control-tabs";

const choices = (count: number) => ({ kind: "input", count, narrow: false }) as const;

/** The one tab's section as the form draws it: a row for each kind of text. */
const TAB_SECTIONS: Record<DataEntryControlTab, readonly SettingsSectionSpec[]> = {
  formatting: [{ title: "sm", description: 1, fields: [choices(4)] }],
};

/** The form's section, with no tab row since the page has one tab. */
export function DataEntryControlSkeleton() {
  return <SettingsPageSkeleton tabs={useDataEntryControlTabs()} sections={TAB_SECTIONS} />;
}
