/** Where a person's own Settings live. */
export const SETTINGS_PATH = "/settings";

/** The query parameter that names the section Settings should land on. */
export const SETTINGS_SECTION_PARAM = "section";

/**
 * The parameter old links used to open Settings over any page (`/?settings=notifications`).
 * Emails already sent still carry it, so the home route turns it into the page's address.
 */
export const LEGACY_SETTINGS_PARAM = "settings";

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

/**
 * Where an old Settings link should go instead, or null when the search carries none. A
 * section the page does not know still opens Settings, as the old dialog did.
 */
export function legacySettingsRedirect(search: string): string | null {
  const value = new URLSearchParams(search).get(LEGACY_SETTINGS_PARAM);
  if (value === null) {
    return null;
  }
  return settingsPath(isSettingsSection(value) ? value : undefined);
}
