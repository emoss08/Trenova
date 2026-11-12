import { describe, expect, it } from "vitest";
import { heldBack, resolveWatchtowerKey } from "../use-watchtower-actions";

const press = (key: string, extra: Partial<Parameters<typeof resolveWatchtowerKey>[0]> = {}) =>
  resolveWatchtowerKey({
    key,
    metaKey: false,
    ctrlKey: false,
    altKey: false,
    editable: false,
    ...extra,
  });

describe("the Watchtower's keys", () => {
  it("walks the queue with J and K and the arrows", () => {
    expect(press("j")).toBe("next");
    expect(press("J")).toBe("next");
    expect(press("ArrowDown")).toBe("next");
    expect(press("k")).toBe("previous");
    expect(press("ArrowUp")).toBe("previous");
  });

  it("hands off with H, finishes with E and snoozes with S", () => {
    expect(press("h")).toBe("handOff");
    expect(press("e")).toBe("done");
    expect(press("s")).toBe("snooze");
  });

  it("leaves a key typed into a field, or held with a modifier, alone", () => {
    expect(press("e", { editable: true })).toBeNull();
    expect(press("s", { metaKey: true })).toBeNull();
    expect(press("j", { ctrlKey: true })).toBeNull();
    expect(press("h", { altKey: true })).toBeNull();
    expect(press("x")).toBeNull();
  });

  it("leaves an item worked in a case alone, and still walks past it", () => {
    const held = new Set(["wt_1"]);

    expect(heldBack("done", "wt_1", held)).toBe(true);
    expect(heldBack("snooze", "wt_1", held)).toBe(true);
    expect(heldBack("handOff", "wt_1", held)).toBe(true);
    expect(heldBack("next", "wt_1", held)).toBe(false);
    expect(heldBack("previous", "wt_1", held)).toBe(false);
    expect(heldBack("done", "wt_2", held)).toBe(false);
    expect(heldBack("done", "wt_1", undefined)).toBe(false);
    expect(heldBack("done", null, held)).toBe(false);
  });
});
