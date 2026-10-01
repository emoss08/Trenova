import type { AssistantPlan, AssistantProposal } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  MAX_BATCH_PROPOSALS,
  approvalQueue,
  currentEntry,
  focusKeys,
  holdsFocus,
  isDeferred,
  planKey,
  proposalKey,
} from "../approval-queue";

/*
Fixtures follow services.AssistantProposal and services.AssistantPlan as the
thread's proposals and plans routes serve them: createdAt is Unix seconds, a
proposal outside any plan has an empty planId, and a decided one carries who
decided it and when.
*/

function proposal(overrides: Partial<AssistantProposal> = {}): AssistantProposal {
  return {
    id: "aprop_1",
    runId: "arun_1",
    toolName: "post_invoice",
    arguments: { invoiceId: "inv_1" },
    rationale: "",
    autonomyTier: "Propose",
    status: "Pending",
    sourceMessageId: "amsg_1",
    executedAt: null,
    executionError: "",
    expiresAt: 0,
    planId: "",
    planStep: 0,
    fields: [],
    createdAt: 100,
    decidedAt: null,
    decidedByUserId: "",
    decisionNote: "",
    ...overrides,
  };
}

function plan(overrides: Partial<AssistantPlan> = {}): AssistantPlan {
  return {
    id: "apl_1",
    runId: "arun_9",
    title: "Recover load 4471",
    summary: "",
    status: "Pending",
    stepCount: 2,
    completedSteps: 0,
    failedStep: null,
    failureError: "",
    decidedAt: null,
    decidedByUserId: "",
    expiresAt: 0,
    hold: null,
    createdAt: 100,
    ...overrides,
  };
}

const NOW = 1_000;

describe("approvalQueue", () => {
  it("asks the oldest first, plans and proposals alike", () => {
    const queue = approvalQueue(
      [
        proposal({ id: "late", runId: "arun_3", createdAt: 300 }),
        proposal({ id: "early", runId: "arun_1", createdAt: 100, toolName: "send_invoice" }),
        proposal({ id: "step_1", runId: "arun_2", planId: "apl_1", planStep: 1, createdAt: 200 }),
      ],
      [plan({ id: "apl_1", runId: "arun_2", createdAt: 200 })],
      null,
      NOW,
    );

    expect(queue.map((entry) => entry.key)).toEqual([
      proposalKey("early"),
      planKey("apl_1"),
      proposalKey("late"),
    ]);
  });

  it("keeps a plan's steps inside the plan, in the order they run", () => {
    const [entry] = approvalQueue(
      [
        proposal({ id: "s2", planId: "apl_1", planStep: 2 }),
        proposal({ id: "s1", planId: "apl_1", planStep: 1 }),
      ],
      [plan()],
      null,
      NOW,
    );

    expect(entry.kind).toBe("plan");
    expect(entry.kind === "plan" && entry.steps.map((step) => step.id)).toEqual(["s1", "s2"]);
  });

  it("takes writes of one tool from one turn as one answer", () => {
    const queue = approvalQueue(
      [
        proposal({ id: "a", createdAt: 110 }),
        proposal({ id: "b", createdAt: 105 }),
        proposal({ id: "c", toolName: "send_invoice", createdAt: 120 }),
      ],
      [],
      null,
      NOW,
    );

    expect(queue).toHaveLength(2);
    expect(queue[0]).toMatchObject({
      kind: "batch",
      toolName: "post_invoice",
      createdAt: 105,
      members: [proposalKey("a"), proposalKey("b")],
    });
    expect(queue[1]).toMatchObject({ kind: "proposal", key: proposalKey("c") });
  });

  // Each was proposed on its own and is decided against the preview it was
  // shown with; a later turn's write of the same tool is a later question.
  it("never merges writes of one tool raised by different turns", () => {
    const queue = approvalQueue(
      [
        proposal({ id: "first", runId: "arun_1", createdAt: 100 }),
        proposal({ id: "second", runId: "arun_2", createdAt: 200 }),
      ],
      [],
      null,
      NOW,
    );

    expect(queue.map((entry) => entry.kind)).toEqual(["proposal", "proposal"]);
  });

  it("takes proposals the agent asked to be decided together as one answer", () => {
    const queue = approvalQueue(
      [
        proposal({ id: "first", runId: "arun_1", createdAt: 100 }),
        proposal({ id: "second", runId: "arun_2", createdAt: 200 }),
        proposal({ id: "other", runId: "arun_2", toolName: "send_invoice", createdAt: 300 }),
      ],
      [],
      { proposalIds: ["first", "second", "other", "gone"], planId: "" },
      NOW,
    );

    expect(queue.map((entry) => entry.key)).toEqual([
      proposalKey("first"),
      proposalKey("second"),
      proposalKey("other"),
    ]);

    const together = approvalQueue(
      [
        proposal({ id: "first", runId: "arun_1", createdAt: 100 }),
        proposal({ id: "second", runId: "arun_2", createdAt: 200 }),
      ],
      [],
      { proposalIds: ["first", "second"], planId: "" },
      NOW,
    );
    expect(together).toHaveLength(1);
    expect(together[0]).toMatchObject({
      kind: "batch",
      members: [proposalKey("first"), proposalKey("second")],
    });
  });

  it("splits a set larger than one decision takes", () => {
    const many = Array.from({ length: MAX_BATCH_PROPOSALS + 2 }, (_, index) =>
      proposal({ id: `p${String(index).padStart(3, "0")}`, createdAt: 100 + index }),
    );

    const queue = approvalQueue(many, [], null, NOW);

    expect(queue.map((entry) => entry.members.length)).toEqual([MAX_BATCH_PROPOSALS, 2]);
  });

  it("lists only what can be decided now", () => {
    const queue = approvalQueue(
      [
        proposal({ id: "held", hold: { reason: "OrganizationPaused", agentName: "" } }),
        proposal({ id: "expired", runId: "arun_2", expiresAt: NOW - 1 }),
        proposal({ id: "done", runId: "arun_3", status: "Executed", executedAt: 50 }),
        proposal({ id: "orphan_step", runId: "arun_4", planId: "apl_gone", planStep: 1 }),
      ],
      [
        plan({ id: "apl_rejected", status: "Rejected" }),
        plan({ id: "apl_held", hold: { reason: "AgentShadow", agentName: "Billing" } }),
      ],
      null,
      NOW,
    );

    expect(queue.map((entry) => entry.key)).toEqual([proposalKey("orphan_step")]);
  });
});

describe("currentEntry", () => {
  const queue = approvalQueue(
    [
      proposal({ id: "one", runId: "arun_1", createdAt: 100 }),
      proposal({ id: "two", runId: "arun_2", createdAt: 200 }),
      proposal({ id: "s1", runId: "arun_3", planId: "apl_1", planStep: 1, createdAt: 300 }),
    ],
    [plan({ id: "apl_1", runId: "arun_3", createdAt: 300 })],
    null,
    NOW,
  );

  it("shows the oldest, and moves on once it is decided", () => {
    expect(currentEntry(queue, new Set())).toMatchObject({ index: 0 });
    expect(currentEntry(queue, new Set())?.entry.key).toBe(proposalKey("one"));

    const afterFirst = queue.slice(1);
    expect(currentEntry(afterFirst, new Set())?.entry.key).toBe(proposalKey("two"));
  });

  it("passes over what was put off, and shows nothing once all of it was", () => {
    expect(currentEntry(queue, new Set([proposalKey("one")]))?.entry.key).toBe(proposalKey("two"));
    expect(currentEntry(queue, new Set(queue.flatMap((entry) => entry.members)))).toBeNull();
  });

  it("opens on what the agent asked about, put off or not", () => {
    const deferred = new Set(queue.flatMap((entry) => entry.members));

    expect(currentEntry(queue, deferred, { proposalIds: ["s1"], planId: "" })).toMatchObject({
      index: 2,
      entry: { key: planKey("apl_1") },
    });
    expect(currentEntry(queue, deferred, { proposalIds: [], planId: "apl_1" })?.index).toBe(2);
    expect(currentEntry(queue, new Set(), { proposalIds: ["gone"], planId: "" })?.index).toBe(0);
  });
});

describe("isDeferred", () => {
  it("holds a batch put off only while every write in it is", () => {
    const [batch] = approvalQueue(
      [proposal({ id: "a" }), proposal({ id: "b", createdAt: 101 })],
      [],
      null,
      NOW,
    );

    expect(isDeferred(batch, new Set([proposalKey("a")]))).toBe(false);
    expect(isDeferred(batch, new Set([proposalKey("a"), proposalKey("b")]))).toBe(true);
  });
});

describe("focusKeys", () => {
  it("names a plan by the plan and proposals by each of them", () => {
    expect(focusKeys({ proposalIds: ["x"], planId: "apl_1" })).toEqual([planKey("apl_1")]);
    expect(focusKeys({ proposalIds: ["x", "y"], planId: "" })).toEqual([
      proposalKey("x"),
      proposalKey("y"),
    ]);
  });
});

describe("holdsFocus", () => {
  // request_decision names a step of a plan by the step's proposal; the plan
  // is what is decided, so the plan holds the request.
  it("finds a plan by one of its steps, and a batch by any of its writes", () => {
    const [planEntry] = approvalQueue(
      [proposal({ id: "s1", planId: "apl_1", planStep: 1 })],
      [plan()],
      null,
      NOW,
    );
    const [batch] = approvalQueue(
      [proposal({ id: "a" }), proposal({ id: "b", createdAt: 101 })],
      [],
      null,
      NOW,
    );

    expect(holdsFocus(planEntry, { proposalIds: ["s1"], planId: "" })).toBe(true);
    expect(holdsFocus(planEntry, { proposalIds: [], planId: "apl_1" })).toBe(true);
    expect(holdsFocus(planEntry, { proposalIds: ["a"], planId: "" })).toBe(false);
    expect(holdsFocus(batch, { proposalIds: ["b"], planId: "" })).toBe(true);
    expect(holdsFocus(batch, { proposalIds: [], planId: "apl_1" })).toBe(false);
  });
});
