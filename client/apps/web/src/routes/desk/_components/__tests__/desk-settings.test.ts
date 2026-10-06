import { DESK_SETTINGS_DEFAULTS, useDeskSettingsStore } from "@/stores/desk-settings-store";
import { describe, expect, it } from "vitest";
import { deskSettingsClasses } from "../desk-settings";

describe("deskSettingsClasses", () => {
  it("names the width and text size and nothing else by default", () => {
    expect(deskSettingsClasses(DESK_SETTINGS_DEFAULTS).split(" ")).toEqual([
      "dk-w-default",
      "dk-t-medium",
    ]);
  });

  it("calms everything down when motion is reduced", () => {
    const classes = deskSettingsClasses({
      ...DESK_SETTINGS_DEFAULTS,
      motion: "reduced",
      celebrate: "off",
      refs: "off",
      drop: "composer",
    }).split(" ");

    expect(classes).toEqual(
      expect.arrayContaining(["dk-calm", "dk-no-ring", "dk-calm-ok", "dk-no-ok-fx", "dk-no-refs", "dk-drop-cmp"]),
    );
  });
});

describe("useDeskSettingsStore", () => {
  it("keeps the last three searches, newest first and without repeats", () => {
    useDeskSettingsStore.setState({ recentSearches: [] });
    const { rememberSearch } = useDeskSettingsStore.getState();
    for (const query of ["storm", "biller", "Storm", "posted", "late"]) {
      rememberSearch(query);
    }

    expect(useDeskSettingsStore.getState().recentSearches).toEqual(["late", "posted", "Storm"]);
  });

  it("puts a setting back to its default on reset and leaves searches alone", () => {
    useDeskSettingsStore.setState({ recentSearches: ["storm"] });
    useDeskSettingsStore.getState().set("width", "wide");
    useDeskSettingsStore.getState().reset();

    expect(useDeskSettingsStore.getState().settings.width).toBe("default");
    expect(useDeskSettingsStore.getState().recentSearches).toEqual(["storm"]);
  });
});
