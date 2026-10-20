import {
  DESK_RAIL_DEFAULT_WIDTH,
  DESK_RAIL_MAX_WIDTH,
  DESK_RAIL_MIN_WIDTH,
  useDeskSettingsStore,
} from "@/stores/desk-settings-store";
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
  useDeskSettingsStore.setState({ railWidth: DESK_RAIL_DEFAULT_WIDTH, railFolded: false });
  localStorage.clear();
});

/**
 * The Desk's rail is as wide as the person dragged it, within the bounds the
 * room can hold, and stays folded once folded. Both are kept in the browser,
 * and a width kept by an older build or edited by hand lands inside the bounds.
 */
describe("the Desk rail's width and fold", () => {
  it("holds a dragged width between its bounds", () => {
    const { setRailWidth } = useDeskSettingsStore.getState();

    setRailWidth(276.4);
    expect(useDeskSettingsStore.getState().railWidth).toBe(276);

    setRailWidth(40);
    expect(useDeskSettingsStore.getState().railWidth).toBe(DESK_RAIL_MIN_WIDTH);

    setRailWidth(2000);
    expect(useDeskSettingsStore.getState().railWidth).toBe(DESK_RAIL_MAX_WIDTH);

    setRailWidth(Number.NaN);
    expect(useDeskSettingsStore.getState().railWidth).toBe(DESK_RAIL_DEFAULT_WIDTH);
  });

  it("keeps the width and the fold", () => {
    useDeskSettingsStore.getState().setRailWidth(300);
    useDeskSettingsStore.getState().setRailFolded(true);

    expect(kept()?.railWidth).toBe(300);
    expect(kept()?.railFolded).toBe(true);
  });

  it("opens on the width and fold that were kept", async () => {
    await rehydrateFrom({ railWidth: 264, railFolded: true });

    expect(useDeskSettingsStore.getState().railWidth).toBe(264);
    expect(useDeskSettingsStore.getState().railFolded).toBe(true);
  });

  it("brings a kept width outside the bounds inside them", async () => {
    await rehydrateFrom({ railWidth: 9000 });
    expect(useDeskSettingsStore.getState().railWidth).toBe(DESK_RAIL_MAX_WIDTH);

    await rehydrateFrom({ railWidth: "wide" });
    expect(useDeskSettingsStore.getState().railWidth).toBe(DESK_RAIL_DEFAULT_WIDTH);
  });

  it("opens unfolded at the default width when nothing was kept for the rail", async () => {
    useDeskSettingsStore.setState({ railWidth: 310, railFolded: true });

    await rehydrateFrom({ collapsedShelves: [] });

    expect(useDeskSettingsStore.getState().railWidth).toBe(DESK_RAIL_DEFAULT_WIDTH);
    expect(useDeskSettingsStore.getState().railFolded).toBe(false);
  });

  it("reads anything but true as unfolded", async () => {
    await rehydrateFrom({ railFolded: "yes" });

    expect(useDeskSettingsStore.getState().railFolded).toBe(false);
  });
});
