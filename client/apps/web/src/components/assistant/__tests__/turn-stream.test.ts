import { parseAssistantStreamEvent, type AssistantStreamEvent } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { describeTurnFailure, initialTurnState, reduceTurn, type TurnState } from "../turn-stream";

function run(events: AssistantStreamEvent[], from: TurnState = initialTurnState("Where is S1?")) {
  return events.reduce(reduceTurn, from);
}

const accepted: AssistantStreamEvent = {
  event: "accepted",
  data: {
    content: "Where is S1?",
    scopeStage: "Deterministic",
    scopeCategory: "TransportationOperations",
  },
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
        data: {
          content: "Let me check.",
          toolCalls: [{ id: "c1", name: "get_shipment" }],
          model: "m",
        },
      },
      {
        event: "tool_started",
        data: { callId: "c1", name: "get_shipment", arguments: { proNumber: "S1" } },
      },
      {
        event: "tool_finished",
        data: { callId: "c1", name: "get_shipment", failed: false, proposed: false, content: "{}" },
      },
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
      {
        event: "tool_finished",
        data: { callId: "c1", name: "get_worker", failed: true, proposed: false, content: "boom" },
      },
      {
        event: "tool_started",
        data: { callId: "c2", name: "flag_for_manual_review", arguments: {} },
      },
      {
        event: "tool_finished",
        data: {
          callId: "c2",
          name: "flag_for_manual_review",
          failed: false,
          proposed: true,
          content: "",
        },
      },
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
      {
        event: "refused",
        data: {
          message: "I can't write code.",
          stage: "Output",
          category: "CodeGeneration",
          reason: "CodeGeneration",
        },
      },
    ]);

    expect(state.refusal?.message).toBe("I can't write code.");
    expect(state.segments).toEqual([]);
    expect(state.status).toBe("refused");
  });

  it("finishes with the saved result and reports an error when the server sends one", () => {
    const done = run([
      accepted,
      { event: "delta", data: { text: "ok" } },
      {
        event: "done",
        data: {
          thread: {
            id: "t1",
            businessUnitId: "bu",
            organizationId: "org",
            userId: "u",
            agentDefinitionId: "agdef",
            preferredProviderId: "",
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
      },
    ]);
    expect(done.status).toBe("done");
    expect(done.result?.reply).toBe("ok");

    const failed = run([
      accepted,
      { event: "error", data: { message: "No AI provider is configured" } },
    ]);
    expect(failed.status).toBe("error");
    expect(failed.error).toBe("No AI provider is configured");
  });
});

// A heavy model's silence before its first word is where a reader gives up.
// Thinking arrives as its own segment, stays open while it streams, and
// closes the moment the reply or a tool call begins.
describe("reduceTurn reasoning", () => {
  it("collects thinking into one open segment ahead of the reply", () => {
    const state = run([
      accepted,
      { event: "reasoning", data: { text: "The card " } },
      { event: "reasoning", data: { text: "expires soon." } },
    ]);

    expect(state.status).toBe("streaming");
    expect(state.segments).toEqual([
      { kind: "reasoning", text: "The card expires soon.", closed: false },
    ]);
  });

  it("closes the thinking when the reply starts", () => {
    const state = run([
      accepted,
      { event: "reasoning", data: { text: "Hmm." } },
      { event: "delta", data: { text: "Friday." } },
    ]);

    expect(state.segments).toEqual([
      { kind: "reasoning", text: "Hmm.", closed: true },
      { kind: "text", text: "Friday.", closed: false },
    ]);
  });

  it("closes the thinking when a tool call starts", () => {
    const state = run([
      accepted,
      { event: "reasoning", data: { text: "Look it up." } },
      { event: "tool_started", data: { callId: "c1", name: "get_shipment", arguments: {} } },
    ]);

    expect(state.segments[0]).toEqual({ kind: "reasoning", text: "Look it up.", closed: true });
    expect(state.segments[1]?.kind).toBe("tool");
  });

  it("starts a new thought after a tool round", () => {
    const state = run([
      accepted,
      { event: "reasoning", data: { text: "First." } },
      { event: "tool_started", data: { callId: "c1", name: "get_shipment", arguments: {} } },
      {
        event: "tool_finished",
        data: { callId: "c1", name: "get_shipment", failed: false, proposed: false, content: "{}" },
      },
      { event: "reasoning", data: { text: "Second." } },
    ]);

    const thoughts = state.segments.filter((segment) => segment.kind === "reasoning");
    expect(thoughts).toHaveLength(2);
    expect(thoughts[1]).toEqual({ kind: "reasoning", text: "Second.", closed: false });
  });
});

/**
 * Contract: services/tms/internal/core/ports/services/assistant.go. A
 * `retrying` event says the model died partway and the reply is starting
 * over, so the text the reader watched is withdrawn while the tools that
 * already ran stay, and `done` after a refusal does not undo the refusal.
 */
describe("reduceTurn restarts and refusals", () => {
  it("withdraws the partial reply on retrying and keeps the tools that ran", () => {
    const state = run([
      accepted,
      { event: "tool_started", data: { callId: "c1", name: "get_shipment", arguments: {} } },
      {
        event: "tool_finished",
        data: { callId: "c1", name: "get_shipment", failed: false, proposed: false, content: "{}" },
      },
      { event: "reasoning", data: { text: "Let me think." } },
      { event: "delta", data: { text: "S1 is in " } },
      {
        event: "retrying",
        data: {
          attempt: 1,
          provider: "Backup",
          reason: "stream died",
          kind: "restart",
          waitSeconds: 0,
        },
      },
    ]);

    expect(state.status).toBe("working");
    expect(state.segments.map((segment) => segment.kind)).toEqual(["tool"]);
    expect(state.retrying).toEqual({
      attempt: 1,
      provider: "Backup",
      kind: "restart",
      waitSeconds: 0,
    });

    const resumed = reduceTurn(state, { event: "delta", data: { text: "S1 is in Dallas." } });
    expect(resumed.retrying).toBeNull();
    expect(resumed.segments.at(-1)).toMatchObject({ kind: "text", text: "S1 is in Dallas." });
  });

  // A busy provider being asked again is not a reply starting over: nothing
  // arrived, nothing is withdrawn, and the reader is told how long the wait
  // is rather than that the model stopped partway.
  it("keeps a busy retry apart from a restart and carries the wait", () => {
    const state = run([
      accepted,
      {
        event: "retrying",
        data: { attempt: 2, provider: "Gemini", reason: "503", kind: "busy", waitSeconds: 4 },
      },
    ]);

    expect(state.status).toBe("working");
    expect(state.retrying).toEqual({
      attempt: 2,
      provider: "Gemini",
      kind: "busy",
      waitSeconds: 4,
    });
  });

  it("parses the retrying event from the wire", () => {
    expect(parseAssistantStreamEvent("retrying", '{"attempt":2,"provider":"Backup"}')).toEqual({
      event: "retrying",
      data: { attempt: 2, provider: "Backup", reason: "", kind: "restart", waitSeconds: 0 },
    });
  });

  it("keeps a refusal when done follows it", () => {
    const state = run([
      accepted,
      {
        event: "refused",
        data: { message: "Not here.", stage: "Output", category: "x", reason: "r" },
      },
      {
        event: "done",
        data: {
          thread: {
            id: "thr_1",
            agentDefinitionId: "agdef_1",
            title: "t",
            createdAt: 1,
            updatedAt: 1,
          } as never,
          messages: [],
          reply: "",
          refused: true,
          proposals: null,
          proposalsUnrecorded: false,
        },
      },
    ]);

    expect(state.status).toBe("refused");
  });
});

/**
 * The failure copy says what the reader actually lost: nothing had arrived,
 * or a reply was underway. "What was said so far has been kept" under an
 * empty frame read as a reply that had vanished.
 */
describe("describeTurnFailure", () => {
  it("distinguishes nothing-yet from cut-off for each cause", () => {
    const empty = run([accepted]);
    const underway = run([accepted, { event: "delta", data: { text: "S1 is" } }]);

    expect(describeTurnFailure(empty, "stopped")).toBe("stopped-before-start");
    expect(describeTurnFailure(underway, "stopped")).toBe("stopped");
    expect(describeTurnFailure(empty, "failed")).toBe("failed-before-start");
    expect(describeTurnFailure(underway, "failed")).toBe("cut-off");
    expect(describeTurnFailure(empty, "ended")).toBe("failed-before-start");
    expect(describeTurnFailure(underway, "ended")).toBe("cut-off");
  });
});
