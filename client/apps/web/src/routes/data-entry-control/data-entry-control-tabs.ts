import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import type { DataEntryControl } from "@/types/data-entry-control";
import { defineLabels } from "@trenova/shared/i18n/labels";

export type DataEntryControlTab = "formatting";

export type DataEntryControlField = keyof DataEntryControl & string;

export type DataEntryControlTabSpec = SettingsTabSpec<DataEntryControlField> & {
  value: DataEntryControlTab;
};

const TAB_LABELS = defineLabels({
  formatting: "Case formatting",
});

/** The form fields each tab holds, so a refused save marks and opens the tab at fault. */
const TAB_FIELDS: Record<DataEntryControlTab, readonly DataEntryControlField[]> = {
  formatting: ["codeCase", "nameCase", "emailCase", "cityCase"],
};

const TAB_ORDER: readonly DataEntryControlTab[] = ["formatting"];

/**
 * The page's tabs. It has one job, how entered text is capitalized, so it has one tab and
 * draws no tab row; every organization sees it.
 */
export function dataEntryControlTabs(): DataEntryControlTabSpec[] {
  return TAB_ORDER.map((tab) => ({ value: tab, label: TAB_LABELS[tab], fields: TAB_FIELDS[tab] }));
}

/** The tabs this organization sees, for the form and its loading state alike. */
export function useDataEntryControlTabs(): DataEntryControlTabSpec[] {
  return dataEntryControlTabs();
}
