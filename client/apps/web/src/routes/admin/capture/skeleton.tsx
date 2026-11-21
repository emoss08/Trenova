import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import {
  SettingsTabsSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";
import { useCaptureTabs, type CaptureTab } from "./capture-tabs";

/**
 * Each tab as it lands, one row to a setting or a record. The settings are drawn with
 * scanning and printing on, as a new organization starts and as nearly every one runs it, so
 * the form's filing and desktop app sections land where the outline drew them.
 */
const TAB_SECTIONS: Record<CaptureTab, readonly SettingsSectionSpec[]> = {
  settings: [
    { title: "sm", description: 1, fields: ["switch"] },
    { title: "sm", description: 1, fields: ["switch", { kind: "input", narrow: true }] },
    { title: "md", description: 1, fields: ["input", "switch"] },
  ],
  profiles: [{ title: "md", description: 1, fields: [{ kind: "input", count: 3 }] }],
  computers: [
    { title: "lg", description: 0, fields: ["input"] },
    { title: "md", description: 1, fields: [{ kind: "input", count: 3 }] },
  ],
};

/** The page's tab row and the open tab's sections. */
export function CaptureSkeleton() {
  return <SettingsPageSkeleton tabs={useCaptureTabs()} sections={TAB_SECTIONS} />;
}

/** The settings form alone, while its record is read: the sections under the page's tab row. */
export function CaptureSettingsSkeleton() {
  return <SettingsTabsSkeleton tabs={1} sections={TAB_SECTIONS.settings} />;
}

