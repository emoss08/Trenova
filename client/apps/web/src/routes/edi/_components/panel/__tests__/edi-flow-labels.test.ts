import { describe, expect, it } from "vitest";
import {
  FLOW_ACTION_LABELS,
  FLOW_EVENT_LABELS,
  FLOW_LOAD_STATE_LABELS,
  flowLabel,
} from "../edi-flow-labels";

const SERVER_EVENT_KINDS = [
  "received",
  "trenova",
  "planner",
  "sent",
  "ack",
  "timer",
  "load",
  "released",
  "advanced",
];

const SERVER_ACTION_KINDS = [
  "acknowledge",
  "create",
  "update",
  "cancel",
  "apply",
  "hold",
  "ignore",
  "askPlanner",
  "send",
  "resend",
  "escalate",
];

const SERVER_LOAD_STATES = ["open", "dispatched", "delivered", "invoiced", "cancelled"];

describe("flow labels", () => {
  it("names every event kind the server journals in words", () => {
    for (const kind of SERVER_EVENT_KINDS) {
      const label = flowLabel(FLOW_EVENT_LABELS, kind, "fallback");
      expect(label).not.toBe("fallback");
      expect(label).not.toBe(kind);
    }
    expect(flowLabel(FLOW_EVENT_LABELS, "timer", "fallback")).toBe("Deadline reached");
    expect(flowLabel(FLOW_EVENT_LABELS, "advanced", "fallback")).toBe("Clock advanced");
  });

  it("names every action and load state the server writes in words", () => {
    for (const kind of SERVER_ACTION_KINDS) {
      expect(flowLabel(FLOW_ACTION_LABELS, kind, "fallback")).not.toBe("fallback");
    }
    for (const state of SERVER_LOAD_STATES) {
      expect(flowLabel(FLOW_LOAD_STATE_LABELS, state, "fallback")).not.toBe("fallback");
    }
    expect(flowLabel(FLOW_ACTION_LABELS, "askPlanner", "fallback")).toBe("Ask a planner");
  });

  it("shows the fallback, never the code, for a value it does not know", () => {
    expect(flowLabel(FLOW_EVENT_LABELS, "rewound", "Other event")).toBe("Other event");
    expect(flowLabel(FLOW_EVENT_LABELS, "toString", "Other event")).toBe("Other event");
    expect(flowLabel(FLOW_EVENT_LABELS, "", "Other event")).toBe("Other event");
  });
});
