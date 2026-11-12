import { DESK_SETTINGS_DEFAULTS, useDeskSettingsStore } from "@/stores/desk-settings-store";
import { afterEach, describe, expect, it } from "vitest";

const STORAGE_KEY = "trenova-desk-settings";

function kept() {
  return (
    JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "{}") as { state?: Record<string, unknown> }
  ).state;
}

/** Puts a kept state in the browser and reads it back as a new visit would. */
async function rehydrateFrom(state: Record<string, unknown>) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify({ state, version: 1 }));
  await useDeskSettingsStore.persist.rehydrate();
}

afterEach(() => {
  useDeskSettingsStore.setState({ railFolded: false, settings: DESK_SETTINGS_DEFAULTS });
  localStorage.clear();
});

/** The Desk's rail stays folded once folded, kept in the browser. */
describe("the Desk rail's fold", () => {
  it("keeps the fold", () => {
    useDeskSettingsStore.getState().setRailFolded(true);

    expect(kept()?.railFolded).toBe(true);
  });

  it("opens folded when that was kept", async () => {
    await rehydrateFrom({ railFolded: true });

    expect(useDeskSettingsStore.getState().railFolded).toBe(true);
  });

  it("opens unfolded when nothing was kept for the rail", async () => {
    useDeskSettingsStore.setState({ railFolded: true });

    await rehydrateFrom({});

    expect(useDeskSettingsStore.getState().railFolded).toBe(false);
  });

  it("reads anything but true as unfolded", async () => {
    await rehydrateFrom({ railFolded: "yes" });

    expect(useDeskSettingsStore.getState().railFolded).toBe(false);
  });
});

/** The rail is pinned beside the page unless the person chose to float it. */
describe("where the Desk's rail sits", () => {
  it("is pinned by default, and goes back to pinned on reset", () => {
    expect(DESK_SETTINGS_DEFAULTS.rail).toBe("pinned");

    useDeskSettingsStore.getState().set("rail", "floating");
    useDeskSettingsStore.getState().reset();

    expect(useDeskSettingsStore.getState().settings.rail).toBe("pinned");
  });

  it("is pinned for a visit kept before the setting existed", async () => {
    await rehydrateFrom({ settings: { width: "wide" } });

    expect(useDeskSettingsStore.getState().settings.rail).toBe("pinned");
    expect(useDeskSettingsStore.getState().settings.width).toBe("wide");
  });

  it("floats when that was kept", async () => {
    await rehydrateFrom({ settings: { rail: "floating" } });

    expect(useDeskSettingsStore.getState().settings.rail).toBe("floating");
  });

  it("reads anything but floating as pinned", async () => {
    await rehydrateFrom({ settings: { rail: "sideways" } });

    expect(useDeskSettingsStore.getState().settings.rail).toBe("pinned");
  });
});
