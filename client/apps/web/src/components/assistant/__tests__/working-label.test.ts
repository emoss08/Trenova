import { translate } from "@trenova/shared/i18n/runtime";
import type { AssistantStreamEvent } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { stepsFromSegments } from "../activity";
import { workingLabel } from "../streaming-turn";
import { initialTurnState, reduceTurn, type TurnState } from "../turn-stream";

function label(events: AssistantStreamEvent[], from: TurnState = initialTurnState("Q")): string {
  const state = events.reduce(reduceTurn, from);
  return workingLabel(state, stepsFromSegments(state.segments), translate);
}

const accepted: AssistantStreamEvent = {
  event: "accepted",
  data: { content: "Q", scopeStage: "", scopeCategory: "" },
};

/**
 * The line under a reply in progress says what is happening this moment,
 * in the words of what is happening: an opened page is never "looking up".
 */
describe("workingLabel", () => {
  it("says the guard is checking before anything else", () => {
    expect(label([])).toBe("Checking the question…");
  });

  it("names the step under way by what it does", () => {
    expect(
      label([
        accepted,
        {
          event: "tool_started",
          data: { callId: "c1", name: "open_page", arguments: {}, effect: "navigate" },
        },
      ]),
    ).toBe("Opening a page…");
  });

  it("says the model is reading a result once a step has finished", () => {
    expect(
      label([
        accepted,
        { event: "tool_started", data: { callId: "c1", name: "get_shipment", arguments: {} } },
        {
          event: "tool_finished",
          data: {
            callId: "c1",
            name: "get_shipment",
            failed: false,
            proposed: false,
            content: "{}",
          },
        },
      ]),
    ).toBe("Reading what came back…");
  });

  it("says it is writing while the answer streams", () => {
    expect(label([accepted, { event: "delta", data: { text: "S1 is" } }])).toBe(
      "Writing the answer…",
    );
  });

  it("says it is thinking while reasoning streams", () => {
    expect(label([accepted, { event: "reasoning", data: { text: "Hmm" } }])).toBe("Thinking…");
  });
});
