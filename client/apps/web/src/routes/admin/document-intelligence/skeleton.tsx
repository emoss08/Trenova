import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import type { SettingsSectionSpec } from "@/components/settings-form-skeleton";
import {
  useDocumentIntelligenceTabs,
  type DocumentIntelligenceTab,
} from "./document-intelligence-tabs";

/**
 * Each tab's sections as the form draws them, one row to a setting. Rows that wait on a
 * switch are drawn closed, as they stand with that switch off, and the notices that depend
 * on saved values or the server are left out, since the outline cannot know either.
 */
const TAB_SECTIONS: Record<DocumentIntelligenceTab, readonly SettingsSectionSpec[]> = {
  reading: [{ title: "sm", description: 1, fields: ["switch"] }],
  filing: [
    { title: "md", description: 1, fields: ["switch"] },
    { title: "md", description: 1, fields: ["switch"] },
  ],
  drafts: [
    { title: "md", description: 1, fields: ["switch"] },
    { title: "sm", description: 1, fields: ["switch"] },
  ],
};

/** The form's tab row and the open tab's sections. */
export function DocumentIntelligenceSkeleton() {
  return <SettingsPageSkeleton tabs={useDocumentIntelligenceTabs()} sections={TAB_SECTIONS} />;
}
