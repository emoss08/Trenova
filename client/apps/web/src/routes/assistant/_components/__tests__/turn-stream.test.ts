import type { AssistantStreamEvent } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { initialTurnState, reduceTurn, type TurnState } from "../turn-stream";

function run(events: AssistantStreamEvent[], from: TurnState = initialTurnState("Where is S1?")) {
  return events.reduce(reduceTurn, from);
}

const accepted: AssistantStreamEvent = {
  event: "accepted",
  data: { content: "Where is S1?", scopeStage: "Deterministic", scopeCategory: "TransportationOperations" },
};

/**
 * The reducer is the whole client-side story of a turn: what the reader sees
 * while the model works is exactly this state rendered. The contract it
 * encodes comes from the server's event order — accepted or refused first,
 * then deltas, message boundaries and tool traffic, then done.
 */
describe("reduceTurn", () => {
  it("starts by waiting on the guard with the person's message shown", () => {
    const state = initialTurnState("Where is S1?");

    expect(state.status).toBe("guarding");
    expect(state.userContent).toBe("Where is S1?");
    expect(state.segments).toEqual([]);
  });

  it("appends deltas to one growing text segment", () => {
    const state = run([
      accepted,
      { event: "delta", data: { text: "S1 is " } },
      { event: "delta", data: { text: "in transit." } },
    ]);

    expect(state.status).toBe("streaming");
    expect(state.segments).toEqual([{ kind: "text", text: "S1 is in transit.", closed: false }]);
  });

  // The text the model wrote before asking for a tool is its own message, so
  // the boundary closes it and the next delta starts a fresh one after the tool.
  it("closes the text at a message boundary and starts a new one after tools", () => {
    const state = run([
      accepted,
      { event: "delta", data: { text: "Let me check." } },
      {
        event: "message",
        data: { content: "Let me check.", toolCalls: [{ id: "c1", name: "get_shipment" }], model: "m" },
      },
      { event: "tool_started", data: { callId: "c1", name: "get_shipment", arguments: { proNumber: "S1" } } },
      { event: "tool_finished", data: { callId: "c1", name: "get_shipment", failed: false, proposed: false, content: "{}" } },
      { event: "delta", data: { text: "S1 is in transit." } },
    ]);

    expect(state.segments).toEqual([
      { kind: "text", text: "Let me check.", closed: true },
      {
        kind: "tool",
        callId: "c1",
        name: "get_shipment",
        arguments: { proNumber: "S1" },
        status: "done",
        content: "{}",
      },
      { kind: "text", text: "S1 is in transit.", closed: false },
    ]);
  });

  it("marks a tool that failed, and one that became a proposal, distinctly", () => {
    const state = run([
      accepted,
      { event: "tool_started", data: { callId: "c1", name: "get_worker", arguments: {} } },
      { event: "tool_finished", data: { callId: "c1", name: "get_worker", failed: true, proposed: false, content: "boom" } },
      { event: "tool_started", data: { callId: "c2", name: "flag_for_manual_review", arguments: {} } },
      { event: "tool_finished", data: { callId: "c2", name: "flag_for_manual_review", failed: false, proposed: true, content: "" } },
    ]);

    const tools = state.segments.filter((segment) => segment.kind === "tool");
    expect(tools.map((tool) => tool.status)).toEqual(["failed", "proposed"]);
    expect(state.status).toBe("working");
  });

  // Text that streamed before the output guard declined it must not linger:
  // the refusal replaces it, and the reader is told the answer was withheld.
  it("drops streamed text when a late refusal arrives", () => {
    const state = run([
      accepted,
      { event: "delta", data: { text: "def export():" } },
      { event: "refused", data: { message: "I can't write code.", stage: "Output", category: "CodeGeneration", reason: "CodeGeneration" } },
    ]);

    expect(state.refusal?.message).toBe("I can't write code.");
    expect(state.segments).toEqual([]);
    expect(state.status).toBe("refused");
  });

  it("finishes with the saved result and reports an error when the server sends one", () => {
    const done = run([accepted, { event: "delta", data: { text: "ok" } }, {
      event: "done",
      data: {
        thread: {
          id: "t1",
          businessUnitId: "bu",
          organizationId: "org",
          userId: "u",
          agentDefinitionId: "agdef",
          title: "t",
          status: "Active",
          lastMessageAt: 0,
          version: 0,
          createdAt: 0,
          updatedAt: 0,
        },
        messages: [],
        reply: "ok",
        refused: false,
        proposals: [],
        proposalsUnrecorded: false,
      },
    }]);
    expect(done.status).toBe("done");
    expect(done.result?.reply).toBe("ok");

    const failed = run([accepted, { event: "error", data: { message: "No AI provider is configured" } }]);
    expect(failed.status).toBe("error");
    expect(failed.error).toBe("No AI provider is configured");
  });
});
