import { describe, expect, it } from "vitest";
import {
  STEP_READ_AHEAD,
  TRIAGE_KEYMAP,
  loadAheadOfReader,
  nextAfterLeaving,
  resolveTriageKey,
  stepFrom,
  type KeyContext,
} from "../keys";

const press = (
  key: string,
  extra: Partial<{ metaKey: boolean; ctrlKey: boolean; altKey: boolean; editable: boolean }> = {},
) => ({
  key,
  metaKey: false,
  ctrlKey: false,
  altKey: false,
  editable: false,
  ...extra,
});

const open: KeyContext = {
  inQueue: true,
  hasOpenMessage: true,
  selectionSize: 0,
  panelOpen: true,
  overlayOpen: false,
};
const closed: KeyContext = { ...open, hasOpenMessage: false };

describe("the triage keymap", () => {
  it("is read from the keybinds config, letters in either case, arrows as J and K", () => {
    expect(TRIAGE_KEYMAP.get("b")).toBe("briefing");
    expect(TRIAGE_KEYMAP.get("#")).toBe("ignore");
    expect(TRIAGE_KEYMAP.get("?")).toBe("shortcuts");
    expect(TRIAGE_KEYMAP.get("Enter")).toBe("doNextStep");
    expect(TRIAGE_KEYMAP.get("Escape")).toBe("escape");
    expect(resolveTriageKey(press("ArrowDown"), open)).toBe("next");
    expect(resolveTriageKey(press("K"), open)).toBe("previous");
  });
});

describe("resolveTriageKey", () => {
  it("acts on the open message", () => {
    expect(
      ["e", "#", "r", "s", "m", "x", "l", "a", "i", "Enter"].map((key) =>
        resolveTriageKey(press(key), open),
      ),
    ).toEqual([
      "handle",
      "ignore",
      "reply",
      "snooze",
      "assignToMe",
      "toggleSelect",
      "link",
      "askDesk",
      "togglePanel",
      "doNextStep",
    ]);
  });

  it("acts on the selection with nothing open, but only for the bulk changes", () => {
    const selecting = { ...closed, selectionSize: 3 };
    expect(resolveTriageKey(press("e"), selecting)).toBe("handle");
    expect(resolveTriageKey(press("s"), selecting)).toBe("snooze");
    expect(resolveTriageKey(press("r"), selecting)).toBeNull();
    expect(resolveTriageKey(press("e"), closed)).toBeNull();
  });

  it("backs out one step at a time on Escape: selection, then panel, then message", () => {
    expect(resolveTriageKey(press("Escape"), { ...open, selectionSize: 2 })).toBe("clearSelection");
    expect(resolveTriageKey(press("Escape"), open)).toBe("closePanel");
    expect(resolveTriageKey(press("Escape"), { ...open, panelOpen: false })).toBe("closeMessage");
    expect(resolveTriageKey(press("Escape"), closed)).toBeNull();
  });

  it("leaves typing, modified keys and anything under a dialog alone", () => {
    expect(resolveTriageKey(press("e", { editable: true }), open)).toBeNull();
    expect(resolveTriageKey(press("e", { metaKey: true }), open)).toBeNull();
    expect(resolveTriageKey(press("k", { ctrlKey: true }), open)).toBeNull();
    expect(resolveTriageKey(press("e"), { ...open, overlayOpen: true })).toBeNull();
    expect(resolveTriageKey(press("z"), open)).toBeNull();
  });

  it("keeps the list keys to the queue, and the Briefing and shortcuts keys everywhere", () => {
    const briefing = { ...closed, inQueue: false };
    expect(resolveTriageKey(press("j"), briefing)).toBeNull();
    expect(resolveTriageKey(press("/"), briefing)).toBeNull();
    expect(resolveTriageKey(press("b"), briefing)).toBe("briefing");
    expect(resolveTriageKey(press("?"), briefing)).toBe("shortcuts");
  });
});

describe("moving through a folder", () => {
  const ids = ["m1", "m2", "m3"];

  it("opens the first on J and the last on K when nothing is open, and stops at either end", () => {
    expect(stepFrom(ids, null, 1)).toBe("m1");
    expect(stepFrom(ids, null, -1)).toBe("m3");
    expect(stepFrom(ids, "m3", 1)).toBe("m3");
    expect(stepFrom(ids, "m1", -1)).toBe("m1");
    expect(stepFrom(ids, "gone", 1)).toBe("m1");
    expect(stepFrom([], null, 1)).toBeNull();
  });

  it("fetches the folder's next page before the reader reaches the last loaded row", () => {
    const loaded = Array.from({ length: 40 }, (_, index) => `m${index + 1}`);
    const nearEnd = loaded[loaded.length - STEP_READ_AHEAD];
    const farFromEnd = loaded[loaded.length - STEP_READ_AHEAD - 1];

    expect(loadAheadOfReader(loaded, nearEnd, true)).toBe(true);
    expect(loadAheadOfReader(loaded, "m40", true)).toBe(true);
    expect(loadAheadOfReader(loaded, farFromEnd, true)).toBe(false);
    expect(loadAheadOfReader(loaded, "m40", false)).toBe(false);
    expect(loadAheadOfReader(loaded, null, true)).toBe(false);
    expect(loadAheadOfReader(loaded, "not-loaded", true)).toBe(false);
  });

  it("moves the reader past every message that is leaving with the open one", () => {
    expect(nextAfterLeaving(["a", "b", "c", "d"], new Set(["b", "c"]), "b")).toBe("d");
    expect(nextAfterLeaving(["a", "b", "c"], new Set(["c"]), "c")).toBe("b");
    expect(nextAfterLeaving(["a"], new Set(["a"]), "a")).toBeNull();
    expect(nextAfterLeaving(["a", "b"], new Set(["b"]), "a")).toBe("a");
    expect(nextAfterLeaving(["a", "b"], new Set(["b"]), null)).toBeNull();
  });
});
