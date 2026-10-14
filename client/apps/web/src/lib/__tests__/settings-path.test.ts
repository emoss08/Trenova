import { describe, expect, it } from "vitest";
import {
  isSettingsSection,
  legacySettingsRedirect,
  SETTINGS_SECTIONS,
  settingsPath,
} from "../settings-path";

describe("settingsPath", () => {
  it("is the bare page when no section is named", () => {
    expect(settingsPath()).toBe("/settings");
  });

  it("names the section to land on", () => {
    expect(settingsPath("notifications")).toBe("/settings?section=notifications");
    expect(settingsPath("security")).toBe("/settings?section=security");
  });
});

describe("isSettingsSection", () => {
  it("accepts every section Settings draws and nothing else", () => {
    for (const section of SETTINGS_SECTIONS) {
      expect(isSettingsSection(section)).toBe(true);
    }
    expect(isSettingsSection("billing")).toBe(false);
    expect(isSettingsSection("")).toBe(false);
    expect(isSettingsSection(null)).toBe(false);
    expect(isSettingsSection(undefined)).toBe(false);
  });
});

describe("legacySettingsRedirect", () => {
  it("sends an old email link to the notifications section of the page", () => {
    expect(legacySettingsRedirect("?settings=notifications")).toBe(
      "/settings?section=notifications",
    );
  });

  it("opens Settings at the top for a section it does not know or an empty one", () => {
    expect(legacySettingsRedirect("?settings=billing")).toBe("/settings");
    expect(legacySettingsRedirect("?settings=")).toBe("/settings");
  });

  it("leaves an address without the old parameter alone", () => {
    expect(legacySettingsRedirect("")).toBeNull();
    expect(legacySettingsRedirect("?section=notifications")).toBeNull();
  });
});
