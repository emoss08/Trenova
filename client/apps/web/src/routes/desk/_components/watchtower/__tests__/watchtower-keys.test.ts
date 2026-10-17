import { describe, expect, it } from "vitest";
import { resolveWatchtowerKey } from "../use-watchtower-actions";

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
});
