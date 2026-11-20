import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import type { SettingsSectionSpec } from "@/components/settings-form-skeleton";
import { useAccountingControlTabs, type AccountingControlTab } from "./accounting-control-tabs";

const choices = (count: number) => ({ kind: "input", count, narrow: false }) as const;

/**
 * Each tab's sections as the form draws them, one row to a setting. Rows that wait on
 * another setting are drawn as they stand on a new organization's settings (manual posting,
 * manual close, reconciliation off, a single currency), since the outline cannot know the
 * saved values.
 */
const TAB_SECTIONS: Record<AccountingControlTab, readonly SettingsSectionSpec[]> = {
  posting: [
    { title: "sm", description: 2, fields: [choices(3)] },
    { title: "sm", description: 1, fields: [choices(2), "switch", choices(1)] },
  ],
  accounts: [
    { title: "md", description: 1, fields: [choices(6)] },
    { title: "md", description: 1, fields: [choices(2)] },
    { title: "sm", description: 1, fields: [choices(1)] },
  ],
  driverPay: [
    { title: "sm", description: 2, fields: [choices(3)] },
    { title: "md", description: 1, fields: [choices(5)] },
  ],
  close: [
    { title: "sm", description: 1, fields: [choices(1), "switch", choices(2)] },
    { title: "md", description: 1, fields: [choices(1)] },
  ],
  currency: [{ title: "sm", description: 1, fields: [choices(2)] }],
};

/** The form's tab row and the open tab's sections, for the tabs this organization sees. */
export function AccountingControlSkeleton() {
  return <SettingsPageSkeleton tabs={useAccountingControlTabs()} sections={TAB_SECTIONS} />;
}
