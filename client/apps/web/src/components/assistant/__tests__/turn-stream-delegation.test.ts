import { parseAssistantStreamEvent, type AssistantStreamEvent } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  advanceTurn,
  initialTurnState,
  reduceTurn,
  type ToolSegment,
  type TurnState,
} from "../turn-stream";

function run(events: AssistantStreamEvent[], from: TurnState = initialTurnState("Build it")) {
  return events.reduce(reduceTurn, from);
}

/**
 * Events as the server writes them (ports/services/assistant.go): parsed from
 * the wire, so the fixtures are the contract and not the reducer's own types.
 */
function wire(event: string, data: unknown): AssistantStreamEvent {
  const parsed = parseAssistantStreamEvent(event, JSON.stringify(data));
  if (parsed === null) {
    throw new Error(`the client does not know ${event}`);
  }
  return parsed;
}

const CALL = "call_delegate_1";
const AGENT = "agdef_report_builder";

const handOff: AssistantStreamEvent[] = [
  wire("accepted", { content: "Build it" }),
  wire("message", {
    content: "I'll ask the Report Builder.",
    toolCalls: [{ id: CALL, name: "delegate_task", arguments: { agentId: AGENT, task: "T" } }],
  }),
  wire("tool_started", {
    callId: CALL,
    name: "delegate_task",
    arguments: { agentId: AGENT, task: "Create a report of on-time deliveries" },
    effect: "delegate",
  }),
  wire("delegate_started", {
    delegateCallId: CALL,
    agentId: AGENT,
    agentName: "Report Builder",
    icon: "file-text",
    accent: "teal",
    task: "Create a report of on-time deliveries",
  }),
];

function delegateSegment(state: TurnState): ToolSegment {
  const segment = state.segments.find(
    (candidate): candidate is ToolSegment => candidate.kind === "tool" && candidate.callId === CALL,
  );
  if (!segment) {
    throw new Error("expected the delegate_task step");
  }
  return segment;
}

/**
 * Another agent's events carry the delegate_task call they belong under.
 * The reader must nest them there: shown among the turn's own steps, the
 * other agent's lookups read as the conversation's agent's, and its words as
 * the reply being written.
 */
describe("reduceTurn hand-offs", () => {
  it("opens the hand-off on the delegate_task step with who was asked and the task", () => {
    const state = run(handOff);
    const segment = delegateSegment(state);

    expect(state.segments.filter((candidate) => candidate.kind === "tool")).toHaveLength(1);
    expect(segment.delegate).toMatchObject({
      agentId: AGENT,
      agentName: "Report Builder",
      icon: "file-text",
      accent: "teal",
      task: "Create a report of on-time deliveries",
      segments: [],
      report: null,
    });
  });

  it("nests the other agent's calls, messages, words and thinking under the hand-off", () => {
    const state = run([
      ...handOff,
      wire("delegate_reasoning", { agentId: AGENT, delegateCallId: CALL, text: "Which dataset…" }),
      wire("message", {
        content: "",
        toolCalls: [{ id: "call_inner_1", name: "create_report", arguments: {} }],
        agentId: AGENT,
        delegateCallId: CALL,
      }),
      wire("tool_started", {
        callId: "call_inner_1",
        name: "create_report",
        arguments: { name: "On-time this month" },
        effect: "change",
        agentId: AGENT,
        delegateCallId: CALL,
      }),
      wire("tool_finished", {
        callId: "call_inner_1",
        name: "create_report",
        failed: false,
        proposed: false,
        content: "{}",
        effect: "change",
        summary: "On-time this month",
        agentId: AGENT,
        delegateCallId: CALL,
      }),
      wire("delegate_delta", { agentId: AGENT, delegateCallId: CALL, text: "I saved " }),
      wire("delegate_delta", { agentId: AGENT, delegateCallId: CALL, text: "the report." }),
    ]);

    // The turn's own list holds its message and the one hand-off, nothing else.
    expect(state.segments.map((segment) => segment.kind)).toEqual(["text", "tool"]);
    expect(state.segments[0]).toEqual({
      kind: "text",
      text: "I'll ask the Report Builder.",
      closed: true,
    });
    // A delegate's words are not the reply being written.
    expect(state.status).toBe("working");

    const nested = delegateSegment(state).delegate?.segments ?? [];
    expect(nested.map((segment) => segment.kind)).toEqual(["reasoning", "tool", "text"]);
    expect(nested[0]).toMatchObject({ kind: "reasoning", text: "Which dataset…", closed: true });
    expect(nested[1]).toMatchObject({
      kind: "tool",
      callId: "call_inner_1",
      status: "done",
      summary: "On-time this month",
    });
    expect(nested[2]).toEqual({ kind: "text", text: "I saved the report.", closed: false });
  });

  it("keeps the turn's own reply apart from the other agent's streamed words", () => {
    const state = run([
      ...handOff,
      wire("delegate_delta", { agentId: AGENT, delegateCallId: CALL, text: "Their words" }),
      wire("delta", { text: "My words" }),
    ]);

    const text = state.segments.filter((segment) => segment.kind === "text");
    expect(text.map((segment) => segment.text)).toEqual([
      "I'll ask the Report Builder.",
      "My words",
    ]);
    expect(delegateSegment(state).delegate?.segments).toEqual([
      { kind: "text", text: "Their words", closed: false },
    ]);
  });

  it("withdraws only the other agent's words when it starts over", () => {
    const state = run([
      ...handOff,
      wire("delegate_delta", { agentId: AGENT, delegateCallId: CALL, text: "Half an ans" }),
      wire("delegate_retrying", {
        attempt: 1,
        provider: "Backup",
        kind: "restart",
        agentId: AGENT,
        delegateCallId: CALL,
      }),
    ]);

    expect(state.retrying).toBeNull();
    expect(state.segments[0]).toMatchObject({ kind: "text", closed: true });
    const delegate = delegateSegment(state).delegate;
    expect(delegate?.segments).toEqual([]);
    expect(delegate?.retrying).toMatchObject({ attempt: 1, provider: "Backup", kind: "restart" });
  });

  it("keeps the account when the task ends, before the call itself settles", () => {
    const finished = run([
      ...handOff,
      wire("delegate_reasoning", { agentId: AGENT, delegateCallId: CALL, text: "Done." }),
      wire("delegate_finished", {
        delegateCallId: CALL,
        agentId: AGENT,
        agentName: "Report Builder",
        status: "completed",
        reply: "I saved the report.",
        made: [
          {
            toolName: "create_report",
            callId: "call_inner_1",
            tier: "AutoExecute",
            summary: "On-time this month",
            result: {
              action: "created",
              kind: "report",
              name: "On-time this month",
              ids: { definitionId: "rd_1" },
            },
          },
        ],
        awaiting: [],
        published: [],
        toolCallsUsed: 1,
      }),
    ]);

    const segment = delegateSegment(finished);
    expect(segment.status).toBe("running");
    expect(segment.delegate?.report).toMatchObject({
      status: "completed",
      reply: "I saved the report.",
      toolCallsUsed: 1,
    });
    expect(segment.delegate?.report?.made[0].result?.ids).toEqual({ definitionId: "rd_1" });
    expect(segment.delegate?.segments.at(-1)).toMatchObject({ kind: "reasoning", closed: true });

    const settled = reduceTurn(
      finished,
      wire("tool_finished", {
        callId: CALL,
        name: "delegate_task",
        failed: false,
        proposed: false,
        content: "Result from delegate_task:\n<untrusted_data>\n{}\n</untrusted_data>",
        effect: "delegate",
        summary: "Report Builder",
      }),
    );
    expect(delegateSegment(settled)).toMatchObject({
      status: "done",
      summary: "Report Builder",
      delegate: { report: { status: "completed" } },
    });
  });

  // A reader that joins a turn partway can hear a delegate's events for a
  // call it never saw start. They still belong under a hand-off, never among
  // the turn's own steps.
  it("opens a hand-off for a delegate event whose call it has not seen", () => {
    const state = run([
      wire("accepted", { content: "Build it" }),
      wire("tool_started", {
        callId: "call_inner_9",
        name: "list_reports",
        arguments: {},
        agentId: AGENT,
        delegateCallId: "call_unseen",
      }),
    ]);

    expect(state.segments).toHaveLength(1);
    expect(state.segments[0]).toMatchObject({
      kind: "tool",
      callId: "call_unseen",
      name: "delegate_task",
      effect: "delegate",
      status: "running",
      delegate: { agentId: AGENT, segments: [{ kind: "tool", callId: "call_inner_9" }] },
    });

    // When the call is announced after all, it is the same step.
    const announced = reduceTurn(
      state,
      wire("tool_started", {
        callId: "call_unseen",
        name: "delegate_task",
        arguments: { agentId: AGENT, task: "T" },
        effect: "delegate",
      }),
    );
    expect(announced.segments).toHaveLength(1);
    expect(announced.segments[0]).toMatchObject({
      arguments: { agentId: AGENT, task: "T" },
      delegate: { segments: [{ kind: "tool", callId: "call_inner_9" }] },
    });
  });

  it("stamps the other agent's calls by the reader's clock", () => {
    let state = handOff.reduce(
      (current, event) => advanceTurn(current, event, 1_000),
      initialTurnState("Build it", null, { startedAt: 1_000 }),
    );
    state = advanceTurn(
      state,
      wire("tool_started", {
        callId: "call_inner_1",
        name: "list_reports",
        arguments: {},
        agentId: AGENT,
        delegateCallId: CALL,
      }),
      2_000,
    );
    state = advanceTurn(
      state,
      wire("tool_finished", {
        callId: "call_inner_1",
        name: "list_reports",
        content: "{}",
        agentId: AGENT,
        delegateCallId: CALL,
      }),
      3_500,
    );

    expect(delegateSegment(state).delegate?.segments[0]).toMatchObject({
      startedAt: 2_000,
      finishedAt: 3_500,
    });
    expect(delegateSegment(state).startedAt).toBe(1_000);
    expect(delegateSegment(state).finishedAt).toBeUndefined();
  });
});

/** The server leaves lists nil and adds endings; a reader must take both. */
describe("parseAssistantStreamEvent hand-off events", () => {
  it("reads a finished task whose lists are null as empty", () => {
    const parsed = wire("delegate_finished", {
      delegateCallId: CALL,
      agentId: AGENT,
      agentName: "Report Builder",
      status: "declined",
      reason: "Report Builder is disabled.",
      made: null,
      awaiting: null,
      published: null,
      toolCallsUsed: 0,
    });

    expect(parsed).toMatchObject({
      event: "delegate_finished",
      data: { status: "declined", made: [], awaiting: [], published: [], reply: "" },
    });
  });

  it("reads an ending this client does not know as failed, which claims the least", () => {
    const parsed = wire("delegate_finished", {
      delegateCallId: CALL,
      agentId: AGENT,
      agentName: "Report Builder",
      status: "vanished",
      made: [],
      awaiting: [],
      published: [],
      toolCallsUsed: 0,
    });

    expect(parsed).toMatchObject({ data: { status: "failed" } });
  });

  it("keeps the scope on tool and message events and leaves it off the turn's own", () => {
    expect(
      wire("tool_finished", {
        callId: "c1",
        name: "get_shipment",
        content: "{}",
        agentId: AGENT,
        delegateCallId: CALL,
      }),
    ).toMatchObject({ data: { agentId: AGENT, delegateCallId: CALL } });

    const own = wire("message", { content: "Hi" });
    expect(own.event === "message" && own.data.delegateCallId).toBeUndefined();
  });
});
