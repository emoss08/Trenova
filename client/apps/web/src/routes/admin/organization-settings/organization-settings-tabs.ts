import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { OrganizationSettings } from "@trenova/shared/types/organization";

export type OrganizationSettingsTab = "profile" | "branding" | "operations";

export type OrganizationSettingsField = keyof OrganizationSettings & string;

export type OrganizationSettingsTabSpec = SettingsTabSpec<OrganizationSettingsField> & {
  value: OrganizationSettingsTab;
};

const TAB_LABELS = defineLabels({
  profile: "Profile",
  branding: "Branding",
  operations: "Operations",
});

/**
 * The organization's fields each tab holds, so a refused save marks and opens the tab at
 * fault. The logo and icon save the moment they are chosen, not with the form, but they
 * are the branding tab's all the same.
 */
const TAB_FIELDS: Record<OrganizationSettingsTab, readonly OrganizationSettingsField[]> = {
  profile: [
    "name",
    "timezone",
    "loginSlug",
    "addressLine1",
    "addressLine2",
    "city",
    "stateId",
    "postalCode",
  ],
  branding: ["logoUrl", "iconUrl"],
  operations: ["brokerageEnabled", "assetOperationsEnabled", "scacCode", "dotNumber", "taxId"],
};

const TAB_ORDER: readonly OrganizationSettingsTab[] = ["profile", "branding", "operations"];

/**
 * The organization settings tabs, in order: who the organization is and where it is
 * registered, how it looks on what it sends, and how it operates. Every organization sees
 * all three; the form and its loading state both read this, so the outline has the form's
 * tabs.
 */
export function organizationSettingsTabs(): OrganizationSettingsTabSpec[] {
  return TAB_ORDER.map((tab) => ({ value: tab, label: TAB_LABELS[tab], fields: TAB_FIELDS[tab] }));
}

/** The tabs for the form and its loading state alike. */
export function useOrganizationSettingsTabs(): OrganizationSettingsTabSpec[] {
  return organizationSettingsTabs();
}
