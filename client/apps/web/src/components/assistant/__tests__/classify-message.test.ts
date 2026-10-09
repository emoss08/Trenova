import { describe, expect, it } from "vitest";
import { classifyMessage } from "@/components/assistant/classify-message";
import type { AssistantMessage } from "@/types/assistant";

/**
 * Fixtures follow the Go contract in
 * services/tms/internal/core/domain/conversation/message.go: role is one of
 * User, Assistant or Tool, and `refused` is an independent flag the guard sets
 * on a turn it declined.
 */
function message(overrides: Partial<AssistantMessage> = {}): AssistantMessage {
  return {
    id: "amsg_1",
    threadId: "athr_1",
    sequence: 0,
    kind: "Message",
    role: "Assistant",
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
    createdAt: 0,
    ...overrides,
  };
}

describe("classifyMessage", () => {
  it("renders an ordinary user turn as a user message", () => {
    expect(classifyMessage(message({ role: "User", content: "Where is S1?" }))).toBe("user");
  });

  it("renders an ordinary assistant turn as an assistant message", () => {
    expect(classifyMessage(message({ role: "Assistant", content: "In Memphis." }))).toBe(
      "assistant",
    );
  });

  // Rendering a refused assistant turn as a normal reply would tell someone the
  // assistant answered when it declined.
  it("renders a refused assistant turn as a refusal, not an answer", () => {
    const refusal = message({
      role: "Assistant",
      refused: true,
      content: "This assistant handles transportation operations, not software development.",
      scopeStage: "Deterministic",
      scopeReason: "CodeGeneration",
    });

    expect(classifyMessage(refusal)).toBe("refusal");
  });

  it("renders the refused prompt muted rather than as a normal question", () => {
    const declined = message({
      role: "User",
      refused: true,
      content: "Write me a Python script",
      scopeReason: "CodeGeneration",
    });

    expect(classifyMessage(declined)).toBe("declined-prompt");
  });

  it("renders a tool result as a lookup", () => {
    expect(
      classifyMessage(message({ role: "Tool", toolName: "get_shipment", content: "{}" })),
    ).toBe("tool");
  });

  // The guard runs on a turn, never on an individual lookup, so a tool message
  // is classified by role alone even if the flag were somehow set.
  it("treats a tool message as a lookup regardless of the refused flag", () => {
    expect(classifyMessage(message({ role: "Tool", refused: true }))).toBe("tool");
  });

  it("renders a failed tool result as a lookup, not a refusal", () => {
    const failed = message({ role: "Tool", toolName: "get_shipment", toolFailed: true });

    expect(classifyMessage(failed)).toBe("tool");
  });

  // An assistant turn that only asked for tools carries no prose. It is still an
  // assistant turn, and the bubble decides separately whether to show text.
  it("renders a tool-only assistant turn as an assistant message", () => {
    const toolCall = message({
      role: "Assistant",
      content: "",
      toolCalls: [{ id: "call_1", name: "get_shipment", arguments: { shipmentId: "shp_1" } }],
    });

    expect(classifyMessage(toolCall)).toBe("assistant");
  });
});

/**
 * The turn after a decision starts from a note the application wrote: the
 * decision on its first line, then instructions to the agent. It is shown as
 * the decision, whatever else the message carries, because every other
 * presentation prints the instructions under the person's name.
 */
describe("classifyMessage with a decision note", () => {
  const note =
    "Approved: create report Late loads\nTell the person what was made and where to find it.";

  it("renders a decision note as a decision", () => {
    expect(classifyMessage(message({ role: "User", kind: "DecisionNote", content: note }))).toBe(
      "decision",
    );
  });

  it("renders a refused decision note as a decision, not a declined prompt", () => {
    expect(
      classifyMessage(
        message({ role: "User", kind: "DecisionNote", refused: true, content: note }),
      ),
    ).toBe("decision");
  });
});

/**
 * Two kinds the runtime saves in the User role (conversation.MessageKindSteer
 * and MessageKindWorldChange). Drawn by role, the notice would read as the
 * person speaking; a steer would read as a fresh question opening a turn.
 */
describe("classifyMessage with what reached a turn while it worked", () => {
  it("draws a steer as the person's, marked as read by the reply under way", () => {
    expect(classifyMessage(message({ role: "User", kind: "Steer", content: "Use Werner" }))).toBe(
      "steer",
    );
  });

  it("draws a change notice as neither side's words", () => {
    expect(
      classifyMessage(message({ role: "User", kind: "WorldChange", content: "[Notice…]" })),
    ).toBe("world");
  });

  it("draws a change notice as a notice even if the guard marked it refused", () => {
    expect(classifyMessage(message({ role: "User", kind: "WorldChange", refused: true }))).toBe(
      "world",
    );
  });
});

describe("classifyMessage with a wait's note", () => {
  it("draws the note that picked a wait up as the wait, never as the person's words", () => {
    expect(
      classifyMessage(
        message({ role: "User", kind: "WaitNote", content: "[Notice…] Wait id: awt_1" }),
      ),
    ).toBe("wait");
  });
});
