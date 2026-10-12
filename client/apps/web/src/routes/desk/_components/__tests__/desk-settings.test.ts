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
      expect.arrayContaining([
        "dk-calm",
        "dk-no-ring",
        "dk-calm-ok",
        "dk-no-ok-fx",
        "dk-no-refs",
        "dk-drop-cmp",
      ]),
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

  it("folds a rail shelf away and opens it again", () => {
    useDeskSettingsStore.setState({ collapsedShelves: [] });
    const { toggleShelf } = useDeskSettingsStore.getState();
    toggleShelf("today");
    toggleShelf("older");

    expect(useDeskSettingsStore.getState().collapsedShelves).toEqual(["today", "older"]);

    toggleShelf("today");

    expect(useDeskSettingsStore.getState().collapsedShelves).toEqual(["older"]);
  });

  it("keeps folded shelves across a reload", async () => {
    useDeskSettingsStore.setState({ collapsedShelves: [] });
    useDeskSettingsStore.getState().toggleShelf("week");
    useDeskSettingsStore.getState().toggleShelf("month");
    const saved = localStorage.getItem("trenova-desk-settings");

    expect(JSON.parse(saved ?? "{}").state.collapsedShelves).toEqual(["week", "month"]);

    useDeskSettingsStore.setState({ collapsedShelves: [] });
    localStorage.setItem("trenova-desk-settings", saved ?? "");
    await useDeskSettingsStore.persist.rehydrate();

    expect(useDeskSettingsStore.getState().collapsedShelves).toEqual(["week", "month"]);
  });

  it("drops what it does not recognise when it reads folded shelves back", async () => {
    localStorage.setItem(
      "trenova-desk-settings",
      JSON.stringify({
        state: { collapsedShelves: ["today", "settled", "archive", 7, null] },
        version: 1,
      }),
    );

    await useDeskSettingsStore.persist.rehydrate();

    expect(useDeskSettingsStore.getState().collapsedShelves).toEqual(["today"]);
  });

  it("opens every shelf when what was saved is not a list", async () => {
    localStorage.setItem(
      "trenova-desk-settings",
      JSON.stringify({ state: { collapsedShelves: "today" }, version: 1 }),
    );

    await useDeskSettingsStore.persist.rehydrate();

    expect(useDeskSettingsStore.getState().collapsedShelves).toEqual([]);
  });
});
