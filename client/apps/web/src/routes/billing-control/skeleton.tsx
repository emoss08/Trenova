import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import type { SettingsSectionSpec } from "@/components/settings-form-skeleton";
import {
  useBillingControlPlan,
  useBillingControlTabs,
  type BillingControlPlan,
  type BillingControlTab,
} from "./billing-control-tabs";

const choices = (count: number) => ({ kind: "input", count, narrow: false }) as const;
const figures = (count: number) => ({ kind: "input", count, narrow: true }) as const;

/**
 * Each tab's sections as the form draws them, one row to a setting. Rows that wait on
 * another setting are drawn closed (transfers, drafts and approval by hand, both checks set
 * to block, reminders and statements off), since the outline cannot know the saved values;
 * rows the plan leaves out are left out here too.
 */
function tabSections({
  novaReviews,
  rating,
}: BillingControlPlan): Record<BillingControlTab, readonly SettingsSectionSpec[]> {
  return {
    invoices: [
      { title: "sm", description: 1, fields: [choices(1), "textarea"] },
      { title: "md", description: 1, fields: [{ kind: "switch", count: 2 }, "textarea"] },
    ],
    automation: [
      { title: "md", description: 1, fields: [choices(2)] },
      { title: "md", description: 1, fields: [choices(2)] },
      {
        title: "md",
        description: 2,
        fields: novaReviews ? ["switch", choices(1)] : [choices(1)],
      },
    ],
    exceptions: [
      { title: "sm", description: 1, fields: [choices(2)] },
      { title: "sm", description: 1, fields: ["switch"] },
    ],
    rating: [
      { title: "md", description: 1, fields: [choices(1)] },
      { title: "md", description: 1, fields: rating ? [{ kind: "switch", count: 2 }] : ["switch"] },
    ],
    collections: [
      { title: "sm", description: 2, fields: [choices(1), figures(1)] },
      { title: "md", description: 2, fields: ["switch"] },
      { title: "md", description: 2, fields: ["switch"] },
    ],
  };
}

/** The form's tab row and the open tab's sections, for what this organization's plan holds. */
export function BillingControlSkeleton() {
  return (
    <SettingsPageSkeleton
      tabs={useBillingControlTabs()}
      sections={tabSections(useBillingControlPlan())}
    />
  );
}
