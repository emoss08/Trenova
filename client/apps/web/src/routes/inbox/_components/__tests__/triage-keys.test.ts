import { describe, expect, it } from "vitest";
import { nextAfterLeaving, reduceTriageKey, type TriageState } from "../triage-keys";

const ids = ["m1", "m2", "m3"];

function press(state: TriageState, ...keys: string[]): TriageState {
  return keys.reduce((current, key) => reduceTriageKey(current, key, ids), state);
}

const closed: TriageState = { openId: null, ask: null };

describe("reduceTriageKey", () => {
  it("opens the first message on j and the last on k when nothing is open", () => {
    expect(press(closed, "j").openId).toBe("m1");
    expect(press(closed, "k").openId).toBe("m3");
  });

  it("walks the list with j and k, and stops at either end", () => {
    expect(press(closed, "j", "j").openId).toBe("m2");
    expect(press(closed, "j", "j", "j", "j").openId).toBe("m3");
    expect(press(closed, "j", "k", "k").openId).toBe("m1");
    expect(press(closed, "ArrowDown", "ArrowDown", "ArrowUp").openId).toBe("m1");
  });

  it("closes the open message on Escape", () => {
    expect(press(closed, "j", "Escape")).toEqual(closed);
  });

  it("asks to handle, ignore, link or ask about the open message", () => {
    const open = press(closed, "j", "j");
    expect(press(open, "e").ask).toEqual({ kind: "handle", id: "m2" });
    expect(press(open, "#").ask).toEqual({ kind: "ignore", id: "m2" });
    expect(press(open, "l").ask).toEqual({ kind: "link", id: "m2" });
    expect(press(open, "a").ask).toEqual({ kind: "ask", id: "m2" });
  });

  it("asks nothing when no message is open", () => {
    for (const key of ["e", "#", "l", "a"]) {
      expect(press(closed, key).ask).toBeNull();
    }
  });

  it("forgets an open message that has left the list", () => {
    const stale: TriageState = { openId: "gone", ask: null };
    expect(reduceTriageKey(stale, "e", ids)).toEqual({ openId: null, ask: null });
    expect(reduceTriageKey(stale, "j", ids).openId).toBe("m1");
  });

  it("ignores keys it does not own", () => {
    const open = press(closed, "j");
    expect(press(open, "z", "Enter", "Tab")).toBe(open);
  });
});

describe("nextAfterLeaving", () => {
  // Handling a message in the waiting lane takes it out of the lane. The
  // reader goes on to the next one, as a triage run does, rather than being
  // left looking at a message that is no longer in the list beside it.
  it("moves on to the message after the one that left", () => {
    expect(nextAfterLeaving(ids, "m2")).toBe("m3");
  });

  it("falls back to the one before when the last message left", () => {
    expect(nextAfterLeaving(ids, "m3")).toBe("m2");
  });

  it("closes the pane when the list is now empty or the message was not in it", () => {
    expect(nextAfterLeaving(["m1"], "m1")).toBeNull();
    expect(nextAfterLeaving(ids, "elsewhere")).toBeNull();
  });
});
