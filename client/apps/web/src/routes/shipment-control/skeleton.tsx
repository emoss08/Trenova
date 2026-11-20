import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import type { SettingsSectionSpec } from "@/components/settings-form-skeleton";
import { useShipmentControlTabs, type ShipmentControlTab } from "./shipment-control-tabs";

const figure = { kind: "input", narrow: true } as const;

/**
 * Each tab's sections as the form draws them, one row to a setting. Rows that wait on a
 * switch are drawn closed and the flat detention section open (detention policies off), since
 * the outline cannot know the saved values.
 */
const TAB_SECTIONS: Record<ShipmentControlTab, readonly SettingsSectionSpec[]> = {
  entry: [
    { title: "sm", description: 1, fields: [figure, { kind: "switch", count: 2 }] },
    { title: "md", description: 1, fields: ["switch"] },
  ],
  automation: [
    { title: "sm", description: 1, fields: ["switch"] },
    { title: "md", description: 1, fields: ["switch"] },
  ],
  detention: [
    { title: "md", description: 2, fields: ["switch"] },
    { title: "sm", description: 1, fields: ["switch"] },
  ],
};

/** The form's tab row and the open tab's sections, for the tabs this organization sees. */
export function ShipmentControlSkeleton() {
  return <SettingsPageSkeleton tabs={useShipmentControlTabs()} sections={TAB_SECTIONS} />;
}
