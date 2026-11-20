import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import type { SettingsSectionSpec } from "@/components/settings-form-skeleton";
import { useCostControlTabs, type CostControlTab } from "./cost-control-tabs";

const figure = { kind: "input", narrow: true } as const;
const choice = { kind: "input", narrow: false } as const;

/** A category's rows: its switch, and under it the choice of where its rate comes from. */
const categories = (count: number) =>
  Array.from({ length: count }, () => ["switch", choice] as const).flat();

/**
 * Each tab's sections as the form draws them, one row to a setting. Rows that wait on
 * another setting are drawn as they stand on a new organization's settings (live fuel price
 * off, GL actuals off, every category active on its benchmark), since the outline cannot
 * know the saved values: six variable and four fixed categories (costingcontrol/defaults.go).
 */
const TAB_SECTIONS: Record<CostControlTab, readonly SettingsSectionSpec[]> = {
  basis: [
    { title: "sm", description: 1, fields: [figure, "switch"] },
    { title: "sm", description: 1, fields: ["switch", figure] },
  ],
  rates: [
    { title: "md", description: 1, fields: categories(6) },
    { title: "sm", description: 1, fields: categories(4) },
  ],
  actuals: [{ title: "sm", description: 2, fields: ["switch"] }],
};

/** The form's tab row and the open tab's sections, for the tabs this organization sees. */
export function CostControlSkeleton() {
  return <SettingsPageSkeleton tabs={useCostControlTabs()} sections={TAB_SECTIONS} />;
}
