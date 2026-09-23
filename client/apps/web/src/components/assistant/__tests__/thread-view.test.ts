import type { AssistantMessage } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { decisionHeadline, groupThread, turnPlacements } from "../thread-view";

let sequence = 0;
function message(overrides: Partial<AssistantMessage>): AssistantMessage {
  sequence += 1;
  return {
    id: `msg_${sequence}`,
    threadId: "t1",
    sequence,
    role: "User",
    content: "",
    toolCalls: null,
    toolCallId: "",
    toolName: "",
    toolFailed: false,
    scopeStage: "",
    scopeCategory: "",
    scopeReason: "",
    refused: false,
    model: "",
    inputTokens: 0,
    outputTokens: 0,
    createdAt: 1_700_000_000 + sequence,
    kind: "Message",
    ...overrides,
  };
}

/**
 * A saved thread is flat: an assistant message that asked for tools is
 * followed by one tool message per call. The reader should see each lookup
 * under the turn that asked for it, with its result, which means pairing them
 * back up by call id rather than by position — a model can ask for two tools in
 * one turn, and a result can arrive for a call the thread no longer shows.
 */
describe("groupThread", () => {
  it("attaches each tool result to the call that asked for it", () => {
    const entries = groupThread([
      message({ role: "User", content: "Where are S1 and S2?" }),
      message({
        role: "Assistant",
        content: "Checking both.",
        toolCalls: [
          { id: "c1", name: "get_shipment", arguments: { proNumber: "S1" } },
          { id: "c2", name: "get_shipment", arguments: { proNumber: "S2" } },
        ],
      }),
      message({ role: "Tool", toolCallId: "c2", toolName: "get_shipment", content: "two" }),
      message({ role: "Tool", toolCallId: "c1", toolName: "get_shipment", content: "one" }),
      message({ role: "Assistant", content: "S1 is in transit; S2 delivered." }),
    ]);

    expect(entries.map((entry) => entry.kind)).toEqual(["user", "assistant", "assistant"]);
    const turn = entries[1];
    if (turn.kind !== "assistant") throw new Error("expected an assistant entry");
    expect(turn.tools.map((tool) => [tool.call.id, tool.result?.content])).toEqual([
      ["c1", "one"],
      ["c2", "two"],
    ]);
  });

  it("still shows a tool result whose call is missing rather than dropping it", () => {
    const entries = groupThread([
      message({ role: "Tool", toolCallId: "orphan", toolName: "search_worker", content: "x" }),
      message({ role: "Assistant", content: "Done." }),
    ]);

    expect(entries[0].kind).toBe("assistant");
    if (entries[0].kind !== "assistant") throw new Error("expected an assistant entry");
    expect(entries[0].tools[0]?.call.name).toBe("search_worker");
    expect(entries[0].tools[0]?.result?.content).toBe("x");
  });

  it("presents a refused question and its refusal as boundaries, not as a turn", () => {
    const entries = groupThread([
      message({ role: "User", content: "Write a script", refused: true }),
      message({ role: "Assistant", content: "I can't write code.", refused: true }),
    ]);

    expect(entries.map((entry) => entry.kind)).toEqual(["declined", "refusal"]);
  });

  it("keeps an assistant message with no text but with tool calls, so the lookups show", () => {
    const entries = groupThread([
      message({
        role: "Assistant",
        content: "",
        toolCalls: [{ id: "c1", name: "get_worker", arguments: {} }],
      }),
      message({ role: "Tool", toolCallId: "c1", toolName: "get_worker", content: "w" }),
    ]);

    expect(entries).toHaveLength(1);
    if (entries[0].kind !== "assistant") throw new Error("expected an assistant entry");
    expect(entries[0].tools).toHaveLength(1);
  });
});

/**
 * The input of the turn after a decision is the application's note, not the
 * person's words. Drawn as a bubble it read as though they had typed
 * "Approved create_dashboard, and it ran." themselves.
 */
describe("groupThread decision notes", () => {
  it("shows a decision note as a note, not as the person's message", () => {
    const entries = groupThread([
      message({ role: "User", content: "Build me a dashboard" }),
      message({ role: "Assistant", content: "Here is a proposal." }),
      message({
        role: "User",
        kind: "DecisionNote",
        content:
          "Approved create_dashboard, and it ran.\nDecision on proposal ap_1 (create_dashboard).",
      }),
      message({ role: "Assistant", content: "It is under Reports." }),
    ]);

    expect(entries.map((entry) => entry.kind)).toEqual([
      "user",
      "assistant",
      "decision",
      "assistant",
    ]);
  });

  // A refused note used to become a muted "declined" turn, which prints the
  // whole message: the agent's instructions under the person's name.
  it("keeps a refused decision note a decision", () => {
    const entries = groupThread([
      message({
        role: "User",
        kind: "DecisionNote",
        refused: true,
        content: "Rejected create_report.\nTell the person nothing was changed.",
      }),
      message({ role: "Assistant", refused: true, content: "I can't help with that." }),
    ]);

    expect(entries.map((entry) => entry.kind)).toEqual(["decision", "refusal"]);
  });
});

/**
 * A reply of several model steps is saved as one assistant message per step.
 * Every message's createdAt is when that step was produced, so the reply's
 * time is the distance from the question to its last step.
 */
describe("turnPlacements", () => {
  it("heads a multi-step reply once and times it from the question", () => {
    const question = message({ role: "User", content: "Where is S1?", createdAt: 1_700_000_100 });
    const first = message({
      role: "Assistant",
      toolCalls: [{ id: "c1", name: "get_shipment", arguments: { proNumber: "S1" } }],
      createdAt: 1_700_000_103,
    });
    const result = message({
      role: "Tool",
      toolCallId: "c1",
      toolName: "get_shipment",
      content: "{}",
      createdAt: 1_700_000_105,
    });
    const second = message({ role: "Assistant", content: "In Dallas.", createdAt: 1_700_000_112 });

    const placements = turnPlacements(groupThread([question, first, result, second]));

    expect(placements.get(first.id)).toEqual({ continued: false, workedSeconds: 12 });
    expect(placements.get(second.id)).toEqual({ continued: true, workedSeconds: null });
  });

  it("starts a new header after the person speaks again", () => {
    const entries = groupThread([
      message({ role: "User", content: "One", createdAt: 1_700_000_200 }),
      message({ role: "Assistant", content: "A", createdAt: 1_700_000_202 }),
      message({ role: "User", content: "Two", createdAt: 1_700_000_300 }),
      message({ role: "Assistant", content: "B", createdAt: 1_700_000_301 }),
    ]);
    const placements = turnPlacements(entries);

    expect(
      entries
        .filter((entry) => entry.kind === "assistant")
        .map((entry) => placements.get(entry.message.id)),
    ).toEqual([
      { continued: false, workedSeconds: 2 },
      { continued: false, workedSeconds: 1 },
    ]);
  });

  // A decision note starts the turn that reports the decision, so it is the
  // question that turn is timed from.
  it("times the reply to a decision from the decision note", () => {
    const entries = groupThread([
      message({
        role: "User",
        kind: "DecisionNote",
        content: "Approved create_report.\nSay what was made.",
        createdAt: 1_700_000_400,
      }),
      message({ role: "Assistant", content: "Saved.", createdAt: 1_700_000_404 }),
    ]);
    const reply = entries[1];

    expect(turnPlacements(entries).get(reply.message.id)).toEqual({
      continued: false,
      workedSeconds: 4,
    });
  });

  it("leaves the time out when the question is not in view", () => {
    const entries = groupThread([
      message({ role: "Assistant", content: "Earlier answer", createdAt: 1_700_000_500 }),
    ]);

    expect(turnPlacements(entries).get(entries[0].message.id)).toEqual({
      continued: false,
      workedSeconds: null,
    });
  });
});

describe("decisionHeadline", () => {
  it("keeps the decision and drops the instructions to the agent", () => {
    expect(
      decisionHeadline(
        'Approved create_report, and it ran. It created the report "Late loads".\nDecision on proposal ap_1 (create_report). Tell the person where to find it.',
      ),
    ).toBe('Approved create report, and it ran. It created the report "Late loads".');
  });

  it("is empty for a note with nothing on its first line", () => {
    expect(decisionHeadline("\nDecision on plan pl_1 (2 steps).")).toBe("");
  });
});
