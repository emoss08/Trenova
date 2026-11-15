import { appPath } from "@/lib/app-path";

/** Where a person's own Settings live. */
export const SETTINGS_PATH = appPath("/settings");

/** The query parameter that names the section Settings should land on. */
export const SETTINGS_SECTION_PARAM = "section";

/** The sections Settings scrolls between; a link with one lands on it. */
export const SETTINGS_SECTIONS = ["profile", "preferences", "notifications", "security"] as const;
export type SettingsSection = (typeof SETTINGS_SECTIONS)[number];

export function isSettingsSection(value: string | null | undefined): value is SettingsSection {
  return value != null && (SETTINGS_SECTIONS as readonly string[]).includes(value);
}

/** The address of Settings, landing on a section when one is named. */
export function settingsPath(section?: SettingsSection): string {
  if (section === undefined) {
    return SETTINGS_PATH;
  }
  return `${SETTINGS_PATH}?${new URLSearchParams({ [SETTINGS_SECTION_PARAM]: section })}`;
}
