import type { AssistantStreamEvent } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { stepsFromSegments } from "../activity";
import { initialTurnState, reduceTurn, type TurnState } from "../turn-stream";
import { thinkingPose } from "../voice/desk-pose";

function pose(events: AssistantStreamEvent[], from: TurnState = initialTurnState("Q")) {
  const state = events.reduce(reduceTurn, from);
  return thinkingPose(state, stepsFromSegments(state.segments));
}

const accepted: AssistantStreamEvent = {
  event: "accepted",
  data: { content: "Q", scopeStage: "", scopeCategory: "" },
};

const started = (callId: string, name = "get_shipment"): AssistantStreamEvent => ({
  event: "tool_started",
  data: { callId, name, arguments: {} },
});

const finished = (callId: string, name = "get_shipment"): AssistantStreamEvent => ({
  event: "tool_finished",
  data: { callId, name, failed: false, proposed: false, content: "{}" },
});

/**
 * The desk beside the working line draws the same moment the words say:
 * thinking dots on the screen while the model decides, hands on the keys
 * while a tool runs, lines written onto the screen while the answer
 * arrives, and a settle once the turn is over — however it ended.
 */
describe("thinkingPose", () => {
  it("thinks while the question is being checked", () => {
    expect(pose([])).toBe("arrive");
  });

  it("thinks once accepted and before anything arrives", () => {
    expect(pose([accepted])).toBe("arrive");
  });

  it("thinks while the model is reasoning", () => {
    expect(pose([accepted, { event: "reasoning", data: { text: "Look it up." } }])).toBe("arrive");
  });

  it("types while a tool is running", () => {
    expect(pose([accepted, started("c1")])).toBe("busy");
  });

  it("stays busy while any one of several tools is still running", () => {
    expect(pose([accepted, started("c1"), started("c2"), finished("c1")])).toBe("busy");
  });

  it("goes back to the model once every tool has finished", () => {
    expect(pose([accepted, started("c1"), finished("c1")])).toBe("arrive");
  });

  it("writes onto the screen while the answer streams", () => {
    expect(pose([accepted, { event: "delta", data: { text: "S1 is" } }])).toBe("write");
  });

  it("stops writing once the message is closed and a tool is asked for", () => {
    expect(
      pose([
        accepted,
        { event: "delta", data: { text: "Let me check." } },
        { event: "message", data: { content: "Let me check.", model: "" } },
        started("c1"),
      ]),
    ).toBe("busy");
  });

  it("goes back to thinking while a reply starts over, even with words on screen", () => {
    expect(
      pose([
        accepted,
        { event: "delta", data: { text: "S1 is in " } },
        {
          event: "retrying",
          data: { attempt: 1, provider: "Backup", reason: "", kind: "busy", waitSeconds: 3 },
        },
      ]),
    ).toBe("arrive");
  });

  it("settles when the turn is done", () => {
    expect(
      pose([
        accepted,
        { event: "delta", data: { text: "S1 is in Dallas." } },
        { event: "done", data: null },
      ]),
    ).toBe("settle");
  });

  it("settles when the turn fails partway through a tool", () => {
    expect(
      pose([accepted, started("c1"), { event: "error", data: { message: "Provider down" } }]),
    ).toBe("settle");
  });

  it("settles when the question is refused", () => {
    expect(
      pose([
        {
          event: "refused",
          data: { message: "Out of scope", stage: "scope", reason: "scope", category: "other" },
        },
      ]),
    ).toBe("settle");
  });
});
