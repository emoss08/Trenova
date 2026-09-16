import { assistantMessageSchema } from "@/types/assistant";
import { describe, expect, it } from "vitest";

/**
 * A stored tool call's `Arguments` is a `map[string]any` without `omitempty`
 * (internal/core/domain/conversation/message.go), so a call the model made
 * with no arguments is serialized as `"arguments": null`. A thread that holds
 * one such call must still open.
 */
const assistantTurn = {
  id: "msg_01JMESSAGE0000000000000000",
  threadId: "cthr_01JTHREAD00000000000000",
  sequence: 2,
  role: "Assistant",
  content: "",
  toolCalls: [{ id: "call_1", name: "list_open_holds", arguments: null }],
  createdAt: 1_758_000_000,
};

describe("assistantMessageSchema", () => {
  it("reads a tool call whose arguments the server wrote as null", () => {
    const parsed = assistantMessageSchema.parse(assistantTurn);

    expect(parsed.toolCalls?.[0].name).toBe("list_open_holds");
    expect(parsed.toolCalls?.[0].arguments).toBeFalsy();
  });

  it("keeps the arguments of a call that has some", () => {
    const parsed = assistantMessageSchema.parse({
      ...assistantTurn,
      toolCalls: [{ id: "call_1", name: "get_shipment", arguments: { proNumber: "S1" } }],
    });

    expect(parsed.toolCalls?.[0].arguments).toEqual({ proNumber: "S1" });
  });
});
