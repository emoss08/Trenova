import {
  assistantProposalListSchema,
  type AssistantPlan,
  type AssistantProposal,
} from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  MAX_BATCH_PROPOSALS,
  approvalQueue,
  currentEntry,
  decisionArtifactFor,
  focusKeys,
  holdSwitch,
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
    toolName: "invoice_post",
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
    reviewRequired: true,
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
        proposal({ id: "early", runId: "arun_1", createdAt: 100, toolName: "invoice_send" }),
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
        proposal({ id: "c", toolName: "invoice_send", createdAt: 120 }),
      ],
      [],
      null,
      NOW,
    );

    expect(queue).toHaveLength(2);
    expect(queue[0]).toMatchObject({
      kind: "batch",
      toolName: "invoice_post",
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
        proposal({ id: "other", runId: "arun_2", toolName: "invoice_send", createdAt: 300 }),
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

  it("lists what waits on a person, held or not, and leaves out what is history", () => {
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

    expect(queue.map((entry) => [entry.key, entry.hold])).toEqual([
      [planKey("apl_held"), { reason: "AgentShadow", agentName: "Billing" }],
      [proposalKey("held"), { reason: "OrganizationPaused", agentName: "" }],
      [proposalKey("orphan_step"), null],
    ]);
  });

  /*
  Every organization starts with its agents paused, and a pause holds every
  proposal it raises. Dropping held proposals from the queue left a turn that
  said "it waits on your approval" with no card anywhere in the Desk.
  */
  it("asks a proposal held by the organization's pause, as the proposals route serves it", () => {
    const served = assistantProposalListSchema.parse({
      results: [
        {
          id: "ap_01M5CRKNHVDKMZFY187BJ7YDSN",
          runId: "ar_01M5CRKPH3WP32T1JXJ2B72E3T",
          toolName: "home_add_widget",
          arguments: { _owner: "usr_1", key: "fleet-status", version: 0 },
          rationale: "Asked to add home widget",
          autonomyTier: "ActWithApproval",
          status: "Pending",
          sourceMessageId: "amsg_01M5CRKPGXGTAGHT2X9PBNEGSS",
          confidence: 0,
          executedAt: null,
          executionError: "",
          expiresAt: NOW + 3600,
          hold: { reason: "OrganizationPaused", agentName: "" },
          planId: null,
          planStep: 0,
          simulatedAt: null,
          simulation: null,
          fields: [
            {
              name: "key",
              label: "Key",
              description: "The widget, by the key home_list_widgets gives.",
              kind: "Text",
              required: true,
              options: [],
              readOnly: false,
            },
          ],
          modifications: null,
          pendingModifications: null,
          agentId: "agdef_1",
          agentName: "Dispatch desk",
          createdAt: NOW - 10,
          decidedAt: null,
          decidedByUserId: null,
          decisionNote: "",
          reviewRequired: false,
        },
      ],
    });

    const queue = approvalQueue(served.results, [], null, NOW);

    expect(queue).toHaveLength(1);
    expect(queue[0]).toMatchObject({
      kind: "proposal",
      key: proposalKey("ap_01M5CRKNHVDKMZFY187BJ7YDSN"),
      hold: { reason: "OrganizationPaused", agentName: "" },
    });
    expect(currentEntry(queue, new Set())?.entry.key).toBe(
      proposalKey("ap_01M5CRKNHVDKMZFY187BJ7YDSN"),
    );
  });

  it("never batches a held write with a live one of the same tool", () => {
    const queue = approvalQueue(
      [
        proposal({ id: "live", createdAt: 100 }),
        proposal({
          id: "held",
          createdAt: 101,
          hold: { reason: "AgentShadow", agentName: "Billing" },
        }),
      ],
      [],
      null,
      NOW,
    );

    expect(queue.map((entry) => [entry.kind, entry.members, entry.hold?.reason ?? null])).toEqual([
      ["proposal", [proposalKey("live")], null],
      ["proposal", [proposalKey("held")], "AgentShadow"],
    ]);
  });

  it("never asks a held write to be decided together with live ones", () => {
    const queue = approvalQueue(
      [
        proposal({ id: "a", runId: "arun_1" }),
        proposal({
          id: "b",
          runId: "arun_2",
          hold: { reason: "OrganizationPaused", agentName: "" },
        }),
      ],
      [],
      { proposalIds: ["a", "b"], planId: "" },
      NOW,
    );

    expect(queue.every((entry) => entry.kind === "proposal")).toBe(true);
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

describe("currentEntry with held writes", () => {
  it("shows what can be decided before what a switch holds, and the held one once nothing else waits", () => {
    const queue = approvalQueue(
      [
        proposal({
          id: "held",
          runId: "arun_1",
          createdAt: 100,
          hold: { reason: "OrganizationPaused", agentName: "" },
        }),
        proposal({ id: "live", runId: "arun_2", createdAt: 200 }),
      ],
      [],
      null,
      NOW,
    );

    expect(currentEntry(queue, new Set())).toMatchObject({
      index: 1,
      entry: { key: proposalKey("live") },
    });
    expect(currentEntry(queue, new Set([proposalKey("live")]))).toMatchObject({
      index: 0,
      entry: { key: proposalKey("held") },
    });
    expect(currentEntry(queue, new Set([proposalKey("live"), proposalKey("held")]))).toBeNull();
  });
});

describe("holdSwitch", () => {
  it("names nothing for a write no switch holds", () => {
    const [entry] = approvalQueue([proposal()], [], null, NOW);

    expect(holdSwitch(entry)).toBeNull();
  });

  it("sends the organization's pause to the AI Control overview", () => {
    const [entry] = approvalQueue(
      [proposal({ hold: { reason: "OrganizationPaused", agentName: "" } })],
      [],
      null,
      NOW,
    );

    expect(holdSwitch(entry)).toEqual({ scope: "organization", to: "/admin/agent-control" });
  });

  it("sends an agent's shadow switch to that agent's sheet", () => {
    const [entry] = approvalQueue(
      [
        proposal({
          agentId: "agdef_billing",
          agentName: "Billing exceptions",
          hold: { reason: "AgentShadow", agentName: "Billing" },
        }),
      ],
      [],
      null,
      NOW,
    );

    expect(holdSwitch(entry)).toEqual({
      scope: "agent",
      to: "/admin/agent-control?tab=agents&openAgent=agdef_billing",
      agentName: "Billing",
    });
  });

  it("names a held plan's agent when the hold does not", () => {
    const [entry] = approvalQueue(
      [],
      [
        plan({
          agentId: "agdef_plan",
          agentName: "Recovery",
          hold: { reason: "AgentShadow", agentName: "" },
        }),
      ],
      null,
      NOW,
    );

    expect(holdSwitch(entry)).toEqual({
      scope: "agent",
      to: "/admin/agent-control?tab=agents&openAgent=agdef_plan",
      agentName: "Recovery",
    });
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

describe("decisionArtifactFor", () => {
  const request = (id: string, proposalId: string, planId = "") => ({
    id,
    kind: "decision_request" as const,
    proposalId,
    planId,
  });

  it("finds the decision request naming any proposal of a batch", () => {
    const [batch] = approvalQueue(
      [proposal({ id: "aprop_1" }), proposal({ id: "aprop_2", arguments: { invoiceId: "inv_2" } })],
      [],
      null,
      1_000,
    );

    expect(batch?.kind).toBe("batch");
    expect(
      decisionArtifactFor(batch!, [request("aart_other", "aprop_9"), request("aart_1", "aprop_2")]),
    ).toBe("aart_1");
  });

  it("finds the one for a single proposal and nothing for another kind of artifact", () => {
    const [single] = approvalQueue([proposal({ id: "aprop_1" })], [], null, 1_000);

    expect(decisionArtifactFor(single!, [request("aart_1", "aprop_1")])).toBe("aart_1");
    expect(
      decisionArtifactFor(single!, [
        { id: "aart_2", kind: "document", proposalId: "aprop_1", planId: "" },
      ]),
    ).toBeNull();
    expect(decisionArtifactFor(single!, [])).toBeNull();
  });
});

describe("approvalQueue and edited proposals", () => {
  it("keeps a proposal the person edited out of its turn's batch", () => {
    const queue = approvalQueue(
      [
        proposal({ id: "aprop_1", pendingModifications: { invoiceId: "inv_9" } }),
        proposal({ id: "aprop_2", arguments: { invoiceId: "inv_2" } }),
        proposal({ id: "aprop_3", arguments: { invoiceId: "inv_3" } }),
      ],
      [],
      null,
      1_000,
    );

    expect(queue.map((entry) => [entry.kind, entry.members])).toEqual(
      expect.arrayContaining([
        ["proposal", [proposalKey("aprop_1")]],
        ["batch", [proposalKey("aprop_2"), proposalKey("aprop_3")]],
      ]),
    );
    expect(queue).toHaveLength(2);
  });

  it("does not decide an edited proposal together with others even when asked to", () => {
    const queue = approvalQueue(
      [
        proposal({ id: "aprop_1", pendingModifications: { invoiceId: "inv_9" } }),
        proposal({ id: "aprop_2", arguments: { invoiceId: "inv_2" } }),
      ],
      [],
      { proposalIds: ["aprop_1", "aprop_2"], planId: "" },
      1_000,
    );

    expect(queue.every((entry) => entry.kind === "proposal")).toBe(true);
  });
});
