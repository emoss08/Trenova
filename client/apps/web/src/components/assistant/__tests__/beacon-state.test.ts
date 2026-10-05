import type { AssistantLiveTurn } from "@/types/assistant";
import { translate as t } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { beaconState, launcherLabel } from "../beacon-state";

function turn(overrides: Partial<AssistantLiveTurn> = {}): AssistantLiveTurn {
  return {
    turnId: "atrn_1",
    threadId: "athr_1",
    threadTitle: "Which loads are stuck?",
    origin: "Person",
    startedAt: 1_700_000_000,
    ...overrides,
  };
}

const quiet = { pendingCount: 0, liveTurns: [], repliedAgentName: null, lastAgentName: "Billing" };

/*
The beacon has one thing to say at a time. A change waiting on the person
leads, because only it needs them; a reply being written comes next; a reply
that arrived while the panel was closed after that; at rest it offers the
agent last asked.
*/
describe("beaconState", () => {
  it("rests on the agent last asked", () => {
    expect(beaconState(quiet)).toEqual({
      mode: "idle",
      pendingCount: 0,
      writingCount: 0,
      writing: null,
      agentName: "Billing",
    });
  });

  it("says what one reply is being written for, and since when", () => {
    const state = beaconState({ ...quiet, liveTurns: [turn()] });

    expect(state.mode).toBe("writing");
    expect(state.writingCount).toBe(1);
    expect(state.writing).toEqual({ title: "Which loads are stuck?", startedAt: 1_700_000_000 });
  });

  it("counts conversations, not turns: a closing turn and its successor are one reply", () => {
    const state = beaconState({
      ...quiet,
      liveTurns: [
        turn({ turnId: "atrn_1", startedAt: 1_700_000_000 }),
        turn({ turnId: "atrn_2", startedAt: 1_700_000_030 }),
      ],
    });

    expect(state.writingCount).toBe(1);
    expect(state.writing).toEqual({ title: "Which loads are stuck?", startedAt: 1_700_000_030 });
  });

  it("names no single reply when several conversations are writing", () => {
    const state = beaconState({
      ...quiet,
      liveTurns: [turn(), turn({ turnId: "atrn_9", threadId: "athr_2", threadTitle: "" })],
    });

    expect(state.mode).toBe("writing");
    expect(state.writingCount).toBe(2);
    expect(state.writing).toBeNull();
  });

  it("puts a waiting change ahead of a reply being written and keeps both counts", () => {
    const state = beaconState({ ...quiet, pendingCount: 2, liveTurns: [turn()] });

    expect(state.mode).toBe("pending");
    expect(state.pendingCount).toBe(2);
    expect(state.writingCount).toBe(1);
  });

  it("says who replied while the panel was closed, once nothing is under way", () => {
    expect(beaconState({ ...quiet, repliedAgentName: "Dispatch" })).toMatchObject({
      mode: "replied",
      agentName: "Dispatch",
    });
    expect(beaconState({ ...quiet, repliedAgentName: "Dispatch", liveTurns: [turn()] }).mode).toBe(
      "writing",
    );
    expect(beaconState({ ...quiet, repliedAgentName: "Dispatch", pendingCount: 1 }).mode).toBe(
      "pending",
    );
  });

  it("treats a negative or missing count as nothing waiting", () => {
    expect(beaconState({ ...quiet, pendingCount: -1 }).mode).toBe("idle");
  });
});

describe("launcherLabel", () => {
  it("says everything the beacon shows, with the real counts", () => {
    expect(launcherLabel(t, beaconState(quiet))).toBe("Open the assistant");
    expect(launcherLabel(t, beaconState({ ...quiet, pendingCount: 150 }))).toBe(
      "Open the assistant, 150 changes await your decision",
    );
    expect(launcherLabel(t, beaconState({ ...quiet, pendingCount: 2, liveTurns: [turn()] }))).toBe(
      "Open the assistant, 2 changes await your decision, 1 reply is being written",
    );
    expect(launcherLabel(t, beaconState({ ...quiet, repliedAgentName: "Dispatch" }))).toBe(
      "Open the assistant, Dispatch replied",
    );
  });
});
