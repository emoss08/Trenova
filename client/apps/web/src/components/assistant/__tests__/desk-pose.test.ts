import type { AssistantStreamEvent, ToolEffect } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { stepsFromSegments } from "../activity";
import { initialTurnState, reduceTurn, type TurnState } from "../turn-stream";
import { isClosingPose, thinkingPose, toolPose, type DeskPose } from "../voice/desk-pose";

function pose(events: AssistantStreamEvent[], from: TurnState = initialTurnState("Q")) {
  const state = events.reduce(reduceTurn, from);
  return thinkingPose(state, stepsFromSegments(state.segments));
}

const accepted: AssistantStreamEvent = {
  event: "accepted",
  data: { content: "Q", scopeStage: "", scopeCategory: "" },
};

const started = (
  callId: string,
  name = "get_shipment",
  effect?: ToolEffect,
): AssistantStreamEvent => ({
  event: "tool_started",
  data: { callId, name, arguments: {}, ...(effect ? { effect } : {}) },
});

const finished = (
  callId: string,
  name = "get_shipment",
  outcome: { failed?: boolean; proposed?: boolean } = {},
): AssistantStreamEvent => ({
  event: "tool_finished",
  data: {
    callId,
    name,
    failed: outcome.failed ?? false,
    proposed: outcome.proposed ?? false,
    content: "{}",
  },
});

const done: AssistantStreamEvent = { event: "done", data: null };

/**
 * The lamp beside the working line draws the same moment the words say: the
 * light coming on while the question is checked, the light breathing while
 * the model decides, a motion for the kind of tool that is running, lines on
 * the desk while the answer arrives, and a closing beat for how the turn
 * ended.
 */
describe("thinkingPose while the turn runs", () => {
  it("switches on while the question is being checked", () => {
    expect(pose([])).toBe("start");
  });

  it("thinks once accepted and before anything arrives", () => {
    expect(pose([accepted])).toBe("think");
  });

  it("thinks while the model is reasoning", () => {
    expect(pose([accepted, { event: "reasoning", data: { text: "Look it up." } }])).toBe("think");
  });

  it("draws the kind of tool that is running", () => {
    expect(pose([accepted, started("c1", "get_shipment")])).toBe("lookup");
    expect(pose([accepted, started("c1", "update_shipment")])).toBe("change");
    expect(pose([accepted, started("c1", "open_page")])).toBe("navigate");
    expect(pose([accepted, started("c1", "find_tools")])).toBe("discover");
    expect(pose([accepted, started("c1", "run_report")])).toBe("present");
    expect(pose([accepted, started("c1", "ask_user")])).toBe("ask");
    expect(pose([accepted, started("c1", "delegate_task")])).toBe("delegate");
  });

  it("takes the kind the server gives over what the name suggests", () => {
    expect(pose([accepted, started("c1", "get_quote_pdf", "present")])).toBe("present");
  });

  it("draws the web for a web read, though the server calls it a lookup", () => {
    expect(pose([accepted, started("c1", "web_search", "lookup")])).toBe("web");
    expect(pose([accepted, started("c1", "web_read", "lookup")])).toBe("web");
  });

  it("follows the latest call still running, the one the words name", () => {
    expect(pose([accepted, started("c1", "get_shipment"), started("c2", "update_rate")])).toBe(
      "change",
    );
  });

  it("goes back to an earlier call still running when the latest one finishes", () => {
    expect(
      pose([
        accepted,
        started("c1", "get_shipment"),
        started("c2", "update_rate"),
        finished("c2", "update_rate"),
      ]),
    ).toBe("lookup");
  });

  it("goes back to thinking once every tool has finished", () => {
    expect(pose([accepted, started("c1"), finished("c1")])).toBe("think");
  });

  it("lights lines on the desk while the answer streams", () => {
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
    ).toBe("lookup");
  });

  it("cuts out while a reply starts over, even with words on screen or a tool running", () => {
    const retrying: AssistantStreamEvent = {
      event: "retrying",
      data: { attempt: 1, provider: "Backup", reason: "", kind: "busy", waitSeconds: 3 },
    };

    expect(pose([accepted, { event: "delta", data: { text: "S1 is in " } }, retrying])).toBe(
      "retry",
    );
    expect(pose([accepted, started("c1"), retrying])).toBe("retry");
  });
});

describe("thinkingPose when the turn is over", () => {
  it("is done when the turn finishes", () => {
    expect(pose([accepted, { event: "delta", data: { text: "S1 is in Dallas." } }, done])).toBe(
      "done",
    );
  });

  it("waits on the reader when a write was proposed rather than made", () => {
    expect(
      pose([
        accepted,
        started("c1", "update_rate"),
        finished("c1", "update_rate", { proposed: true }),
        done,
      ]),
    ).toBe("await");
  });

  it("is done, not failed, when a single call failed and the turn still finished", () => {
    expect(
      pose([accepted, started("c1"), finished("c1", "get_shipment", { failed: true }), done]),
    ).toBe("done");
  });

  it("fails when the turn fails partway through a tool", () => {
    expect(
      pose([accepted, started("c1"), { event: "error", data: { message: "Provider down" } }]),
    ).toBe("failed");
  });

  it("fails rather than waiting when the turn errors after proposing a write", () => {
    expect(
      pose([
        accepted,
        started("c1", "update_rate"),
        finished("c1", "update_rate", { proposed: true }),
        { event: "error", data: { message: "Provider down" } },
      ]),
    ).toBe("failed");
  });

  it("is done when the question is refused", () => {
    expect(
      pose([
        {
          event: "refused",
          data: { message: "Out of scope", stage: "scope", reason: "scope", category: "other" },
        },
      ]),
    ).toBe("done");
  });
});

describe("toolPose", () => {
  it.each<[ToolEffect, DeskPose]>([
    ["lookup", "lookup"],
    ["discover", "discover"],
    ["navigate", "navigate"],
    ["change", "change"],
    ["present", "present"],
    ["ask", "ask"],
    ["delegate", "delegate"],
  ])("draws a %s call as %s", (effect, expected) => {
    expect(toolPose({ name: "any_tool", effect })).toBe(expected);
  });
});

describe("isClosingPose", () => {
  it.each<[DeskPose, boolean]>([
    ["done", true],
    ["await", true],
    ["failed", true],
    ["start", false],
    ["think", false],
    ["write", false],
    ["retry", false],
    ["change", false],
  ])("%s closes a turn: %s", (candidate, closing) => {
    expect(isClosingPose(candidate)).toBe(closing);
  });
});
