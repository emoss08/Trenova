import { describe, expect, it } from "vitest";
import {
  NOTHING_WRITTEN,
  writeCaseEvent,
  type CallLedger,
  type CaseWriting,
} from "../desk-case-stream";

function play(events: [string, unknown][]): CaseWriting {
  const calls: CallLedger = new Map();

  return events.reduce(
    (state, [event, data]) => writeCaseEvent(state, event, JSON.stringify(data), calls),
    NOTHING_WRITTEN,
  );
}

describe("a plan taking shape", () => {
  it("keeps none of what the agent says to itself while it works", () => {
    expect(
      play([
        ["delta", { text: "I'll start by checking the move's coverage." }],
        ["reasoning", { text: "thinking" }],
      ]),
    ).toEqual(NOTHING_WRITTEN);
  });

  it("names what the agent is doing now, and counts what it has looked at", () => {
    const reading = play([["tool_started", { callId: "c1", name: "shipment_get" }]]);
    expect(reading).toEqual({ steps: [], checked: 0, tool: "shipment_get", ended: false });

    const read = play([
      ["tool_started", { callId: "c1", name: "shipment_get" }],
      ["tool_finished", { callId: "c1", name: "shipment_get" }],
      ["tool_started", { callId: "c2", name: "dispatch_rank_candidates" }],
      ["tool_finished", { callId: "c2", name: "dispatch_rank_candidates" }],
    ]);
    expect(read).toEqual({ steps: [], checked: 2, tool: "", ended: false });
  });

  it("adds a step for each change the agent drafts, with the values it was drafted with", () => {
    const drafted = play([
      ["tool_started", { callId: "c1", name: "dispatch_assign", arguments: { workerId: "w" } }],
      ["tool_finished", { callId: "c1", name: "dispatch_assign", proposed: true }],
      ["tool_started", { callId: "c2", name: "worker_send_message", arguments: { text: "hi" } }],
      ["tool_finished", { callId: "c2", name: "worker_send_message", proposed: true }],
    ]);

    expect(drafted.steps).toEqual([
      { name: "dispatch_assign", args: { workerId: "w" } },
      { name: "worker_send_message", args: { text: "hi" } },
    ]);
    expect(drafted.checked).toBe(0);
  });

  it("replaces a change the agent drafts again, as one it withdrew and redrafted", () => {
    const redrafted = play([
      ["tool_started", { callId: "c1", name: "dispatch_assign", arguments: { tractorId: "a" } }],
      ["tool_finished", { callId: "c1", name: "dispatch_assign", proposed: true }],
      ["tool_started", { callId: "c2", name: "dispatch_assign", arguments: { tractorId: "b" } }],
      ["tool_finished", { callId: "c2", name: "dispatch_assign", proposed: true }],
    ]);

    expect(redrafted.steps).toEqual([{ name: "dispatch_assign", args: { tractorId: "b" } }]);
  });

  it("counts a call that failed as neither a step nor something checked", () => {
    expect(
      play([
        ["tool_started", { callId: "c1", name: "dispatch_assign" }],
        ["tool_finished", { callId: "c1", name: "dispatch_assign", failed: true, proposed: true }],
      ]),
    ).toEqual(NOTHING_WRITTEN);
  });

  it("leaves out what an agent it delegated to did", () => {
    expect(
      play([
        ["tool_started", { callId: "c9", name: "x", delegateCallId: "d1" }],
        ["tool_finished", { callId: "c9", name: "x", proposed: true, delegateCallId: "d1" }],
      ]),
    ).toEqual(NOTHING_WRITTEN);
  });

  it("ends on done and on error, and ignores an event it does not know or cannot read", () => {
    expect(play([["done", {}]]).ended).toBe(true);
    expect(play([["error", { message: "x" }]]).ended).toBe(true);
    expect(play([["context", { used: 3 }]])).toEqual(NOTHING_WRITTEN);
    expect(writeCaseEvent(NOTHING_WRITTEN, "tool_started", "not json")).toEqual(NOTHING_WRITTEN);
  });
});
