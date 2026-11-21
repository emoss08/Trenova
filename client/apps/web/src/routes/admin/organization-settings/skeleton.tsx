import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import type { SettingsSectionSpec } from "@/components/settings-form-skeleton";
import {
  useOrganizationSettingsTabs,
  type OrganizationSettingsTab,
} from "./organization-settings-tabs";

const text = { kind: "input", narrow: false } as const;
const code = { kind: "input", narrow: true } as const;

/**
 * Each tab's sections as the form draws them, one row to a setting. The operating model is
 * drawn as an organization whose plan includes brokerage sees it: the preset and both
 * switches. The branding preview is drawn as one picture of its height.
 */
const TAB_SECTIONS: Record<OrganizationSettingsTab, readonly SettingsSectionSpec[]> = {
  profile: [
    { title: "sm", description: 1, fields: [text, text, text] },
    { title: "md", description: 1, fields: [text, text, text, text, code] },
  ],
  branding: [{ title: "md", description: 2, fields: ["preview"] }],
  operations: [
    { title: "md", description: 2, fields: [text, "switch", "switch"] },
    { title: "sm", description: 1, fields: [code, code, text] },
  ],
};

/** The form's tab row and the open tab's sections. */
export function OrganizationSettingsSkeleton() {
  return <SettingsPageSkeleton tabs={useOrganizationSettingsTabs()} sections={TAB_SECTIONS} />;
}
