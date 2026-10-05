import { describe, expect, it } from "vitest";
import {
  IDLE,
  secondsLeft,
  UNDO_SECONDS,
  undoReducer,
  type UndoState,
  type UndoWindow,
} from "../undo-window";

const CLICK = 1_000_000;

function held(overrides: Partial<UndoWindow> = {}): UndoWindow {
  return {
    key: "proposal:ap_1",
    title: "Assign biller",
    what: "Assign biller on 11 items",
    target: { proposalId: "ap_1" },
    proposalIds: ["ap_1"],
    planId: null,
    commitsAt: null,
    startedAt: CLICK,
    ...overrides,
  };
}

function waiting(window: UndoWindow): UndoState {
  return undoReducer(IDLE, { type: "start", window });
}

describe("secondsLeft", () => {
  it("counts from the click until the server answers", () => {
    expect(secondsLeft(held(), CLICK)).toBe(UNDO_SECONDS);
    expect(secondsLeft(held(), CLICK + 1_200)).toBe(4);
    expect(secondsLeft(held(), CLICK + 5_000)).toBe(0);
  });

  it("follows the server's commit time once it has answered", () => {
    const window = held({ commitsAt: (CLICK + 3_000) / 1000 });

    expect(secondsLeft(window, CLICK)).toBe(3);
    expect(secondsLeft(window, CLICK + 2_500)).toBe(1);
    expect(secondsLeft(window, CLICK + 3_000)).toBe(0);
  });

  it("holds a clock far off the server's to the window", () => {
    expect(secondsLeft(held({ commitsAt: (CLICK + 60_000) / 1000 }), CLICK)).toBe(UNDO_SECONDS);
    expect(secondsLeft(held({ commitsAt: (CLICK - 60_000) / 1000 }), CLICK)).toBe(0);
  });
});

describe("undoReducer", () => {
  it("waits, then commits when the server's window runs out", () => {
    let state = waiting(held());
    state = undoReducer(state, { type: "scheduled", key: "proposal:ap_1", commitsAt: 1_005 });
    expect(state).toMatchObject({ phase: "waiting", window: { commitsAt: 1_005 } });

    state = undoReducer(state, { type: "tick", now: 1_004_200 });
    expect(state.phase).toBe("waiting");

    state = undoReducer(state, { type: "tick", now: 1_005_000 });
    expect(state.phase).toBe("committed");
  });

  it("does not count a commit before the server has taken the approval", () => {
    const state = undoReducer(waiting(held()), { type: "tick", now: CLICK + 10_000 });

    expect(state.phase).toBe("waiting");
  });

  it("commits at once when the server carried the approval out without a window", () => {
    const state = undoReducer(waiting(held()), {
      type: "scheduled",
      key: "proposal:ap_1",
      commitsAt: null,
    });

    expect(state.phase).toBe("committed");
  });

  it("clears on undo or refusal, and only for its own approval", () => {
    const state = waiting(held());

    expect(undoReducer(state, { type: "clear", key: "proposal:ap_2" })).toBe(state);
    expect(undoReducer(state, { type: "clear", key: "proposal:ap_1" })).toBe(IDLE);
  });

  it("commits on 'Do it now'", () => {
    const state = undoReducer(waiting(held()), { type: "commit", key: "proposal:ap_1" });

    expect(state.phase).toBe("committed");
  });

  it("ignores a late answer for an approval it no longer holds", () => {
    expect(undoReducer(IDLE, { type: "scheduled", key: "proposal:ap_1", commitsAt: 1_005 })).toBe(
      IDLE,
    );
  });
});
