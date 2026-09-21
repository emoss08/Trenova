import type { AssistantMessage, AssistantPlan, AssistantProposal } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { classifyPlan, groupPlans, isPlanDecidable, planStepState } from "../plan-state";

function plan(overrides: Partial<AssistantPlan> = {}): AssistantPlan {
  return {
    id: "apl_1",
    runId: "arun_1",
    title: "Dispatch coverage: 2 changes",
    summary: "Cover the two open moves",
    status: "Pending",
    stepCount: 2,
    completedSteps: 0,
    failedStep: null,
    failureError: "",
    decidedAt: null,
    expiresAt: 0,
    hold: null,
    createdAt: 0,
    ...overrides,
  };
}

function proposal(overrides: Partial<AssistantProposal> = {}): AssistantProposal {
  return {
    id: "aprop_1",
    runId: "arun_1",
    toolName: "assign_move",
    arguments: {},
    rationale: "",
    autonomyTier: "ActWithApproval",
    status: "Pending",
    sourceMessageId: "amsg_1",
    executedAt: null,
    executionError: "",
    expiresAt: 0,
    planId: "",
    planStep: 0,
    fields: [],
    ...overrides,
  };
}

function message(id: string): AssistantMessage {
  return {
    id,
    threadId: "athr_1",
    sequence: 1,
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
  };
}

describe("classifyPlan", () => {
  it("offers a decision only while nobody has made one", () => {
    expect(classifyPlan(plan())).toBe("awaiting");
    expect(isPlanDecidable(plan())).toBe(true);
    expect(isPlanDecidable(plan({ status: "Approved" }))).toBe(false);
  });

  it("keeps a held plan a question nobody can answer yet", () => {
    expect(classifyPlan(plan({ hold: { reason: "OrganizationPaused", agentName: "" } }))).toBe(
      "held",
    );
  });

  // The server sweeps expired plans every quarter hour; between sweeps the
  // clock decides, so no card offers buttons the server will refuse.
  it("closes a pending plan past its expiry before the server has", () => {
    expect(classifyPlan(plan({ expiresAt: 100 }), 101)).toBe("closed");
    expect(classifyPlan(plan({ expiresAt: 100 }), 99)).toBe("awaiting");
    expect(classifyPlan(plan({ expiresAt: 0 }), 10_000)).toBe("awaiting");
  });

  it("separates approved from done, and done from failed", () => {
    expect(classifyPlan(plan({ status: "Approved" }))).toBe("running");
    expect(classifyPlan(plan({ status: "Completed", completedSteps: 2 }))).toBe("done");
    expect(
      classifyPlan(plan({ status: "Failed", completedSteps: 1, failedStep: 2, failureError: "x" })),
    ).toBe("failed");
    expect(classifyPlan(plan({ status: "Rejected" }))).toBe("declined");
    expect(classifyPlan(plan({ status: "Expired" }))).toBe("closed");
  });
});

describe("planStepState", () => {
  it("reads each step from its proposal, not from the plan", () => {
    expect(planStepState(proposal({ status: "Executed", executedAt: 5 }))).toBe("done");
    expect(planStepState(proposal({ status: "ExecutionFailed", executionError: "no" }))).toBe(
      "failed",
    );
    expect(planStepState(proposal({ status: "Accepted" }))).toBe("running");
    expect(planStepState(proposal({ status: "Skipped" }))).toBe("skipped");
    expect(planStepState(proposal({ status: "Rejected" }))).toBe("declined");
    expect(planStepState(proposal({ status: "Pending" }))).toBe("waiting");
    expect(planStepState(proposal({ status: "Simulated", simulatedAt: 5 }))).toBe("simulated");
  });
});

describe("groupPlans", () => {
  const steps = [
    proposal({ id: "aprop_2", planId: "apl_1", planStep: 2, sourceMessageId: "amsg_1" }),
    proposal({ id: "aprop_1", planId: "apl_1", planStep: 1, sourceMessageId: "amsg_1" }),
  ];

  it("puts a plan under the message its first step came from, steps in order", () => {
    const loose = proposal({ id: "aprop_9" });
    const grouped = groupPlans([plan()], [loose, ...steps], [message("amsg_1")]);

    const under = grouped.byMessage.get("amsg_1");
    expect(under?.map((group) => group.plan.id)).toEqual(["apl_1"]);
    expect(under?.[0].steps.map((step) => step.id)).toEqual(["aprop_1", "aprop_2"]);
    expect(grouped.orphans).toEqual([]);
    expect(grouped.standalone.map((item) => item.id)).toEqual(["aprop_9"]);
  });

  it("keeps a plan whose message is gone rather than dropping it", () => {
    const grouped = groupPlans([plan()], steps, [message("amsg_other")]);

    expect(grouped.byMessage.size).toBe(0);
    expect(grouped.orphans.map((group) => group.plan.id)).toEqual(["apl_1"]);
  });

  // A proposal that names a plan the thread does not list is still a pending
  // write; it is shown on its own rather than hidden behind a plan nobody
  // can see.
  it("shows a step of an unknown plan on its own", () => {
    const grouped = groupPlans([], steps, [message("amsg_1")]);

    expect(grouped.orphans).toEqual([]);
    expect(grouped.standalone.map((item) => item.id)).toEqual(["aprop_2", "aprop_1"]);
  });

  it("keeps a plan with no listed steps so its outcome is still readable", () => {
    const grouped = groupPlans([plan({ status: "Completed" })], [], [message("amsg_1")]);

    expect(grouped.orphans.map((group) => group.plan.id)).toEqual(["apl_1"]);
    expect(grouped.orphans[0].steps).toEqual([]);
  });
});
