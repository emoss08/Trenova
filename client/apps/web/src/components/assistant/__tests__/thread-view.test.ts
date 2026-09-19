import type { AssistantMessage } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { groupThread } from "../thread-view";

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
