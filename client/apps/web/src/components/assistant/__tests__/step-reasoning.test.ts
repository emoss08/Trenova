import type { AssistantMessage, AssistantStreamEvent } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { stepsFromExchanges, stepsFromSegments } from "../activity";
import { groupThread } from "../thread-view";
import { initialTurnState, reduceTurn } from "../turn-stream";

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

function savedSteps(messages: AssistantMessage[]) {
  return groupThread(messages).flatMap((entry) =>
    entry.kind === "assistant" ? stepsFromExchanges(entry.tools, entry.message.createdAt) : [],
  );
}

/**
 * A step's reasoning is what the model thought in the reply that asked for
 * the call, as the server saved it on that assistant message. Calls asked for
 * together share it; a reply that kept no readable thinking gives its calls
 * none, rather than an earlier reply's.
 */
describe("a saved step's reasoning", () => {
  it("is the thinking of the reply that asked for the call", () => {
    const steps = savedSteps([
      message({ role: "User", content: "Where are S1 and S2?" }),
      message({
        role: "Assistant",
        reasoning: { text: "Both are on the Dallas lane; look them up together." },
        toolCalls: [
          { id: "c1", name: "shipment_get", arguments: { proNumber: "S1" } },
          { id: "c2", name: "shipment_get", arguments: { proNumber: "S2" } },
        ],
      }),
      message({ role: "Tool", toolCallId: "c1", toolName: "shipment_get", content: "one" }),
      message({ role: "Tool", toolCallId: "c2", toolName: "shipment_get", content: "two" }),
      message({
        role: "Assistant",
        reasoning: null,
        toolCalls: [{ id: "c3", name: "list_stops", arguments: {} }],
      }),
      message({ role: "Tool", toolCallId: "c3", toolName: "list_stops", content: "stops" }),
      message({
        role: "Assistant",
        reasoning: { text: "" },
        toolCalls: [{ id: "c4", name: "get_worker", arguments: {} }],
      }),
    ]);

    expect(steps.map((step) => [step.id, step.reasoning])).toEqual([
      ["c1", "Both are on the Dallas lane; look them up together."],
      ["c2", "Both are on the Dallas lane; look them up together."],
      ["c3", ""],
      ["c4", ""],
    ]);
  });

  it("is empty on a result whose call is out of view", () => {
    const steps = savedSteps([
      message({ role: "Tool", toolCallId: "c9", toolName: "shipment_get", content: "one" }),
      message({ role: "Assistant", reasoning: { text: "Answer from what came back." } }),
    ]);

    expect(steps.map((step) => [step.id, step.reasoning])).toEqual([["c9", ""]]);
  });
});

function streamed(events: AssistantStreamEvent[]) {
  return stepsFromSegments(events.reduce(reduceTurn, initialTurnState("Where is S1?")).segments);
}

/**
 * While a turn streams, the thought arrives before the reply that asks for
 * calls, and the reply's message event names those calls. Each call shows the
 * thought of the reply that named it, and a reply that streamed no thought
 * gives its calls none.
 */
describe("a streaming step's reasoning", () => {
  it("is the thought of the reply whose message named the call", () => {
    const steps = streamed([
      { event: "reasoning", data: { text: "Check S1 " } },
      { event: "reasoning", data: { text: "first." } },
      {
        event: "message",
        data: {
          content: "",
          model: "m",
          toolCalls: [
            { id: "c1", name: "shipment_get" },
            { id: "c2", name: "get_stops" },
          ],
        },
      },
      { event: "tool_started", data: { callId: "c1", name: "shipment_get" } },
      { event: "tool_finished", data: { callId: "c1", name: "shipment_get", content: "{}" } },
      { event: "tool_started", data: { callId: "c2", name: "get_stops" } },
      { event: "tool_finished", data: { callId: "c2", name: "get_stops", content: "{}" } },
      {
        event: "message",
        data: { content: "", model: "m", toolCalls: [{ id: "c3", name: "get_worker" }] },
      },
      { event: "tool_started", data: { callId: "c3", name: "get_worker" } },
    ] as AssistantStreamEvent[]);

    expect(steps.map((step) => [step.id, step.reasoning])).toEqual([
      ["c1", "Check S1 first."],
      ["c2", "Check S1 first."],
      ["c3", ""],
    ]);
  });

  it("has none until the reply that thought it has named its calls", () => {
    const steps = streamed([
      {
        event: "message",
        data: { content: "", model: "m", toolCalls: [{ id: "c1", name: "shipment_get" }] },
      },
      { event: "tool_started", data: { callId: "c1", name: "shipment_get" } },
      { event: "tool_finished", data: { callId: "c1", name: "shipment_get", content: "{}" } },
      { event: "reasoning", data: { text: "Now the driver." } },
    ] as AssistantStreamEvent[]);

    expect(steps.map((step) => [step.id, step.reasoning])).toEqual([["c1", ""]]);
  });
});
