import { translate } from "@trenova/shared/i18n/runtime";
import type { AssistantStreamEvent } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { stepsFromSegments } from "../activity";
import { formatWorkDuration } from "@/lib/ai-usage-format";
import { workingLabel, workingTally } from "../streaming-turn";
import { initialTurnState, reduceTurn, type TurnState } from "../turn-stream";

function label(
  events: AssistantStreamEvent[],
  from: TurnState = initialTurnState("Q", null, { startedAt: 0 }),
): string {
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

  it("says it is getting going once accepted and before anything arrives", () => {
    expect(label([accepted])).toBe("Thinking…");
  });
});

/**
 * The plain moments each have a few ways of being said. A turn keeps one of
 * them from start to finish, so its words change only when the work does,
 * while the next turn may say it differently.
 */
describe("workingLabel variations", () => {
  const at = (seconds: number) => initialTurnState("Q", null, { startedAt: seconds * 1000 });

  it("opens each turn with one of its ways of getting going", () => {
    expect(label([accepted], at(0))).toBe("Thinking…");
    expect(label([accepted], at(1))).toBe("Pulling up a chair…");
    expect(label([accepted], at(2))).toBe("Getting started…");
  });

  it("writes in one of its ways", () => {
    const delta: AssistantStreamEvent = { event: "delta", data: { text: "S1 is" } };

    expect(label([accepted, delta], at(0))).toBe("Writing the answer…");
    expect(label([accepted, delta], at(1))).toBe("Writing…");
    expect(label([accepted, delta], at(2))).toBe("Putting it into words…");
  });

  it("thinks in one of its ways", () => {
    const reasoning: AssistantStreamEvent = { event: "reasoning", data: { text: "Hmm" } };

    expect(label([accepted, reasoning], at(0))).toBe("Thinking…");
    expect(label([accepted, reasoning], at(1))).toBe("Thinking it through…");
  });

  it("keeps the same words for a turn however long the moment lasts", () => {
    const turn = at(1);
    const delta: AssistantStreamEvent = { event: "delta", data: { text: "S1 " } };
    const more: AssistantStreamEvent = { event: "delta", data: { text: "is in Dallas" } };

    expect(label([accepted, delta], turn)).toBe(label([accepted, delta, more], turn));
  });

  it("never varies what a step, a check or a retry says", () => {
    expect(label([], at(1))).toBe("Checking the question…");
    expect(
      label(
        [
          accepted,
          {
            event: "tool_started",
            data: { callId: "c1", name: "open_page", arguments: {}, effect: "navigate" },
          },
        ],
        at(2),
      ),
    ).toBe("Opening a page…");
  });
});

/** The quiet count beside the words. */
describe("workingTally", () => {
  function tally(events: AssistantStreamEvent[], elapsed: number): string {
    const state = events.reduce(reduceTurn, initialTurnState("Q", null, { startedAt: 0 }));
    return workingTally(stepsFromSegments(state.segments), elapsed, translate);
  }

  const started = (callId: string): AssistantStreamEvent => ({
    event: "tool_started",
    data: { callId, name: "get_shipment", arguments: {} },
  });
  const finished = (callId: string): AssistantStreamEvent => ({
    event: "tool_finished",
    data: { callId, name: "get_shipment", failed: false, proposed: false, content: "{}" },
  });

  it("is only the time before any step", () => {
    expect(tally([accepted], 5)).toBe(formatWorkDuration(5));
  });

  it("counts the steps taken, not the one under way", () => {
    expect(tally([accepted, started("c1"), finished("c1"), started("c2")], 12)).toBe(
      `1 step · ${formatWorkDuration(12)}`,
    );
  });

  it("says how many run at once when more than one does", () => {
    expect(tally([accepted, started("c1"), finished("c1"), started("c2"), started("c3")], 12)).toBe(
      `1 step · 2 running · ${formatWorkDuration(12)}`,
    );
  });
});
