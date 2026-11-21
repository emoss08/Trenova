import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import { usePermission } from "@/hooks/use-permission";
import { captureSettingFields } from "@/types/document-control";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { Operation, Resource } from "@trenova/shared/types/permission";

export type CaptureTab = "settings" | "profiles" | "computers";

export type CaptureSettingField = (typeof captureSettingFields)[number];

export type CaptureTabSpec = SettingsTabSpec<CaptureSettingField> & { value: CaptureTab };

/** Which parts of the page a person may read; each tab is shown only to whoever may. */
export type CaptureAccess = Record<CaptureTab, boolean>;

const TAB_LABELS = defineLabels({
  settings: "Settings",
  profiles: "Scan profiles",
  computers: "Computers",
});

/**
 * The fields each tab holds. Only the settings tab is a form: scan profiles are records with
 * their own forms, and the computers are a list read from the server, so neither has a field
 * the settings save sends.
 */
const TAB_FIELDS: Record<CaptureTab, readonly CaptureSettingField[]> = {
  settings: captureSettingFields,
  profiles: [],
  computers: [],
};

const TAB_ORDER: readonly CaptureTab[] = ["settings", "profiles", "computers"];

/**
 * The tabs of the page, in order: whether scanning and printing is on and how it files, the
 * scan profiles people pick from, and every computer paired to scan. The page, its settings
 * form and its loading state all read this.
 */
export function captureTabs(access: CaptureAccess): CaptureTabSpec[] {
  return TAB_ORDER.filter((tab) => access[tab]).map((tab) => ({
    value: tab,
    label: TAB_LABELS[tab],
    fields: TAB_FIELDS[tab],
  }));
}

/** The tabs this person may open, for the page and its loading state alike. */
export function useCaptureTabs(): CaptureTabSpec[] {
  const { allowed: settings } = usePermission(Resource.DocumentControl, Operation.Read);
  const { allowed: profiles } = usePermission(Resource.CaptureProfile, Operation.Read);
  const { allowed: computers } = usePermission(Resource.CaptureDevice, Operation.Read);

  return captureTabs({ settings, profiles, computers });
}
