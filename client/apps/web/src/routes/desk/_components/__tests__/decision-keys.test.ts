import { describe, expect, it } from "vitest";
import {
  initialQueueSelection,
  reduceQueueKey,
  type QueueSelection,
} from "../decisions/use-decision-keys";

const ids = ["ap_1", "ap_2", "ap_3", "ap_4"];

function press(keys: string[], from: QueueSelection = initialQueueSelection()): QueueSelection {
  return keys.reduce((state, key) => reduceQueueKey(state, key, ids), from);
}

/**
 * The queue is worked from the keyboard: j and k walk it, x marks a row for
 * the batch, and the decision keys act on the focused row. The reducer is
 * pure so the whole session can be replayed here without a DOM.
 */
describe("reduceQueueKey", () => {
  it("starts on the first row when nothing is focused and j is pressed", () => {
    expect(press(["j"]).focusedId).toBe("ap_1");
  });

  it("walks down and up without leaving the list", () => {
    const down = press(["j", "j", "j", "j", "j", "j"]);
    expect(down.focusedId).toBe("ap_4");
    const up = press(["k", "k", "k", "k", "k"], down);
    expect(up.focusedId).toBe("ap_1");
  });

  it("marks and unmarks the focused row with x, keeping focus where it is", () => {
    const marked = press(["j", "x"]);
    expect(marked.selectedIds).toEqual(["ap_1"]);
    expect(marked.focusedId).toBe("ap_1");
    const unmarked = press(["x"], marked);
    expect(unmarked.selectedIds).toEqual([]);
  });

  it("marks nothing when no row is focused", () => {
    expect(press(["x"]).selectedIds).toEqual([]);
  });

  it("names the decision the focused row is asked for", () => {
    expect(press(["j", "a"]).pending).toEqual({ kind: "accept", ids: ["ap_1"] });
    expect(press(["j", "j", "r"]).pending).toEqual({ kind: "reject", ids: ["ap_2"] });
    expect(press(["j", "m"]).pending).toEqual({ kind: "modify", ids: ["ap_1"] });
  });

  it("asks to accept every marked row with Shift+A", () => {
    const state = press(["j", "x", "j", "x", "j", "A"]);
    expect(state.pending).toEqual({ kind: "accept", ids: ["ap_1", "ap_2"] });
  });

  it("falls back to the focused row when Shift+A finds nothing marked", () => {
    expect(press(["j", "j", "A"]).pending).toEqual({ kind: "accept", ids: ["ap_2"] });
  });

  it("clears the marks and the ask with Escape", () => {
    const state = press(["j", "x", "a", "Escape"]);
    expect(state.selectedIds).toEqual([]);
    expect(state.pending).toBeNull();
    expect(state.focusedId).toBe("ap_1");
  });

  it("ignores keys it does not own", () => {
    const state = press(["j"]);
    expect(reduceQueueKey(state, "z", ids)).toBe(state);
  });

  it("drops marks and focus for rows that left the queue", () => {
    const state = press(["j", "x", "j", "x"]);
    const next = reduceQueueKey(state, "j", ["ap_2", "ap_3"]);
    expect(next.selectedIds).toEqual(["ap_2"]);
    expect(next.focusedId).toBe("ap_3");
  });
});
