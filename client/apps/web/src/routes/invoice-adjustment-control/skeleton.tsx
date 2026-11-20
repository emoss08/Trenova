import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import type { SettingsSectionSpec } from "@/components/settings-form-skeleton";
import {
  useInvoiceAdjustmentControlTabs,
  type InvoiceAdjustmentControlTab,
} from "./invoice-adjustment-control-tabs";

const choices = (count: number) => ({ kind: "input", count, narrow: false }) as const;

/**
 * Each tab's sections as the form draws them, one row to a setting. Rows that wait on
 * another setting (the closed-period policy, the approval amounts, credit beyond what is
 * owed, the rerate tolerance) are drawn closed, since the outline cannot know the saved
 * values.
 */
const TAB_SECTIONS: Record<InvoiceAdjustmentControlTab, readonly SettingsSectionSpec[]> = {
  eligibility: [
    { title: "md", description: 2, fields: [choices(3)] },
    { title: "sm", description: 1, fields: [choices(1)] },
  ],
  approval: [
    { title: "sm", description: 1, fields: [choices(2)] },
    { title: "sm", description: 1, fields: [choices(1)] },
    { title: "sm", description: 1, fields: ["switch"] },
  ],
  credits: [
    { title: "sm", description: 1, fields: [choices(1)] },
    { title: "sm", description: 1, fields: [choices(2)] },
  ],
};

/** The form's tab row and the open tab's sections. */
export function InvoiceAdjustmentControlSkeleton() {
  return <SettingsPageSkeleton tabs={useInvoiceAdjustmentControlTabs()} sections={TAB_SECTIONS} />;
}
