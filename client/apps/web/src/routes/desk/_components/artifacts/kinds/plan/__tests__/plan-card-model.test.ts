import {
  field,
  message,
  money,
  planPreview,
  preview,
  record,
  warning,
} from "@/components/assistant/__tests__/preview-fixtures";
import type { AssistantPlan, AssistantProposal } from "@/types/assistant";
import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import type { PlanArtifact } from "../../../artifact-payloads";
import {
  countdown,
  finishMessage,
  foldedByDefault,
  foldedLine,
  phaseLabel,
  planLog,
  planPhase,
  planSteps,
  recordLine,
  recordsTouched,
  shortDigest,
  stepTone,
  supersededBy,
  unfinished,
} from "../plan-card-model";

const t = translate as TranslateFn;
const NOW = 1_790_000_000;

/* services.AssistantPlan as the thread's plans route serves it. */
function plan(overrides: Partial<AssistantPlan> = {}): AssistantPlan {
  return {
    id: "apl_1",
    runId: "arun_1",
    title: "Recover S-1001",
    summary: "",
    status: "Pending",
    stepCount: 2,
    completedSteps: 0,
    failedStep: null,
    failureError: "",
    decidedAt: null,
    decidedByUserId: "",
    decidedByName: "",
    decisionNote: "",
    previewDigest: "",
    commitsAt: null,
    undoneAt: null,
    runMode: "InOrder",
    expiresAt: NOW + 3600,
    hold: null,
    createdAt: NOW - 600,
    agentId: "agdef_1",
    agentName: "Dispatch",
    ...overrides,
  };
}

/* services.AssistantProposal as the thread's proposals route serves it. */
function step(overrides: Partial<AssistantProposal> = {}): AssistantProposal {
  return {
    id: "aprop_1",
    runId: "arun_1",
    toolName: "shipment_update",
    arguments: {},
    rationale: "Move the delivery",
    autonomyTier: "Propose",
    status: "Pending",
    sourceMessageId: "amsg_1",
    executedAt: null,
    executionError: "",
    expiresAt: NOW + 3600,
    planId: "apl_1",
    planStep: 1,
    fields: [],
    createdAt: NOW - 600,
    decidedAt: null,
    decidedByUserId: "",
    decisionNote: "",
    reviewRequired: true,
    ...overrides,
  };
}

const payload: PlanArtifact = {
  title: "Recover S-1001",
  summary: "",
  stepCount: 2,
  steps: [
    { step: 1, proposalId: "aprop_1", toolName: "shipment_update", rationale: "Move the delivery" },
    { step: 2, proposalId: "aprop_2", toolName: "email_send", rationale: "" },
  ],
};

describe("planPhase", () => {
  it.each([
    [plan({ status: "Pending" }), "awaiting"],
    [plan({ status: "Pending", hold: { reason: "AgentShadow", agentName: "Dispatch" } }), "held"],
    [plan({ status: "Pending", expiresAt: NOW - 1 }), "expired"],
    [plan({ status: "Approving", commitsAt: NOW + 8 }), "undoable"],
    [plan({ status: "Approving", commitsAt: NOW - 1 }), "running"],
    [plan({ status: "Approved" }), "running"],
    [plan({ status: "Completed" }), "done"],
    [plan({ status: "Failed" }), "failed"],
    [plan({ status: "Rejected" }), "declined"],
    [plan({ status: "Expired" }), "expired"],
  ] as const)("reads %o as %s", (given, want) => {
    expect(planPhase(given, NOW)).toBe(want);
  });
});

describe("stepTone", () => {
  it("draws a waiting step as waiting until the plan is decided, then as next while it runs", () => {
    expect(stepTone(step(), "awaiting")).toBe("wait");
    expect(stepTone(step({ status: "Approving" }), "undoable")).toBe("wait");
    expect(stepTone(step(), "running")).toBe("next");
    expect(stepTone(step(), "expired")).toBe("skip");
  });

  it("draws each settled step by its own status", () => {
    expect(stepTone(step({ status: "Accepted" }), "running")).toBe("run");
    expect(stepTone(step({ status: "Executed", executedAt: NOW }), "running")).toBe("done");
    expect(stepTone(step({ status: "Simulated", simulatedAt: NOW }), "done")).toBe("done");
    expect(stepTone(step({ status: "ExecutionFailed", executionError: "refused" }), "failed")).toBe(
      "fail",
    );
    expect(stepTone(step({ status: "Skipped" }), "failed")).toBe("skip");
    expect(stepTone(step({ status: "Rejected" }), "declined")).toBe("skip");
  });

  it("draws a step only the artifact still names as waiting or not run", () => {
    expect(stepTone(undefined, "awaiting")).toBe("wait");
    expect(stepTone(undefined, "done")).toBe("skip");
  });
});

describe("planSteps", () => {
  it("orders the listed steps, reads each one's preview and result, and keeps one the thread lost", () => {
    const views = planSteps(
      payload,
      [
        step({
          status: "Executed",
          executedAt: NOW - 30,
          executionResult: {
            action: "updated",
            kind: "shipment",
            name: "S-1001",
            ids: {},
            state: "Appointment confirmed",
          },
        }),
      ],
      planPreview(),
      "running",
    );

    expect(views.map((view) => [view.number, view.proposalId, view.title, view.tone])).toEqual([
      [1, "aprop_1", "Move the delivery", "done"],
      [2, "aprop_2", "Email send", "skip"],
    ]);
    expect(views[0].result).toBe("Appointment confirmed");
    expect(views[0].ranAt).toBe(NOW - 30);
    expect(views[0].preview?.proposalId).toBe("aprop_1");
    expect(views[1].preview).toBeNull();
  });

  it("names the earlier steps a step starts from, and how much of a write over many went through", () => {
    const views = planSteps(
      payload,
      [
        step(),
        step({
          id: "aprop_2",
          planStep: 2,
          status: "Executed",
          executedAt: NOW,
          executionResult: {
            action: "updated",
            kind: "",
            name: "",
            ids: {},
            total: 5,
            failed: [{ id: "inv_1", label: "INV-1", reason: "posted" }],
          },
        }),
      ],
      planPreview(),
      "running",
    );

    expect(views[1].dependsOn).toEqual([1]);
    expect(views[1].partial).toEqual({ done: 4, total: 5 });
  });
});

describe("foldedByDefault", () => {
  const ran = planSteps(
    payload,
    [step({ status: "Executed", executedAt: NOW })],
    planPreview(),
    "done",
  )[0];

  it("keeps every change open before the plan runs", () => {
    expect(foldedByDefault(ran, "awaiting")).toBe(false);
    expect(foldedByDefault(ran, "undoable")).toBe(false);
  });

  it("folds a step that ran, but not one that failed or carries a warning", () => {
    expect(foldedByDefault(ran, "done")).toBe(true);
    expect(foldedByDefault({ ...ran, tone: "fail" }, "failed")).toBe(false);
    const warned = {
      ...ran,
      preview: preview({ warnings: [warning({ code: "already_told_customer" })] }),
    };
    expect(foldedByDefault(warned, "done")).toBe(false);
  });

  it("does not hold a step open for the dependency it already names", () => {
    const dependent = {
      ...ran,
      preview: preview({ warnings: [warning({ code: "depends_on_step", args: ["1"] })] }),
    };
    expect(foldedByDefault(dependent, "done")).toBe(true);
  });
});

describe("foldedLine", () => {
  it("says what each record's change came to, with the money it moved", () => {
    expect(
      foldedLine(
        preview({
          changes: [
            record({ fields: [field(), field({ path: "eta", label: "ETA" })], money: money() }),
            record({ operation: "Send", label: "Tyson", fields: [], message: message() }),
          ],
        }),
        t,
      ),
    ).toBe("Changed 2 fields on S-1001 · +$150.00 · Message to 1 recipient");
  });

  it("never names a record the reader may not see", () => {
    expect(foldedLine(preview({ changes: [record({ withheld: true })] }), t)).toBe(
      "A record you cannot see",
    );
  });
});

describe("recordsTouched", () => {
  it("lists each record once with every step that changes it, and counts hidden records, not hidden fields", () => {
    const views = planSteps(
      payload,
      [step(), step({ id: "aprop_2", planStep: 2 })],
      planPreview({
        steps: [
          {
            proposalId: "aprop_1",
            step: 1,
            preview: preview({ changes: [record({ fields: [field({ label: "Appointment" })] })] }),
          },
          {
            proposalId: "aprop_2",
            step: 2,
            preview: preview({
              withheldCount: 3,
              changes: [
                record({ fields: [field({ label: "Detention" })] }),
                record({ withheld: true }),
              ],
            }),
          },
        ],
      }),
      "awaiting",
    );

    const { records, withheld } = recordsTouched(views);
    expect(records).toHaveLength(1);
    expect(records[0].label).toBe("S-1001");
    expect(records[0].steps).toEqual([1, 2]);
    expect(records[0].fields).toEqual(["Appointment", "Detention"]);
    expect(withheld).toBe(1);
    expect(recordLine(records[0], t)).toBe("Appointment, Detention");
  });
});

describe("phaseLabel", () => {
  it("says who the plan waits on and where it stopped", () => {
    expect(phaseLabel("awaiting", [], "InOrder", true, t)).toBe("Awaiting your approval");
    expect(phaseLabel("awaiting", [], "InOrder", false, t)).toBe("Awaiting approval");
    const stopped = planSteps(
      payload,
      [
        step({ status: "Executed", executedAt: NOW }),
        step({ id: "aprop_2", planStep: 2, status: "ExecutionFailed", executionError: "no" }),
      ],
      undefined,
      "failed",
    );
    expect(phaseLabel("failed", stopped, "InOrder", false, t)).toBe("Stopped at step 2");
    expect(phaseLabel("failed", stopped, "EachStep", false, t)).toBe("1 step did not go through");
  });
});

describe("finishing a stopped plan", () => {
  const stopped = planSteps(
    payload,
    [
      step({ status: "Executed", executedAt: NOW }),
      step({
        id: "aprop_2",
        planStep: 2,
        status: "ExecutionFailed",
        executionError: "A posted invoice cannot take new charges.",
      }),
    ],
    undefined,
    "failed",
  );

  it("offers to finish a plan that failed or expired with work undone, and not one that finished", () => {
    expect(unfinished("failed", stopped)).toBe(true);
    expect(unfinished("expired", stopped)).toBe(true);
    expect(unfinished("done", stopped)).toBe(false);
    expect(unfinished("declined", stopped)).toBe(false);
  });

  it("names the plan, the step it stopped at and the server's words, and asks for the rest", () => {
    expect(finishMessage(plan({ status: "Failed", failedStep: 2 }), stopped, "failed", t)).toBe(
      'The plan "Recover S-1001" stopped at step 2: A posted invoice cannot take new charges. ' +
        "Step 1 went through. Step 2 did not run. " +
        "Propose the rest of the plan, fixed so it goes through, for me to approve.",
    );
  });

  it("asks again for an expired plan none of whose steps ran", () => {
    const expired = planSteps(payload, [step({ status: "Expired" })], undefined, "expired");
    expect(finishMessage(plan({ status: "Expired" }), expired, "expired", t)).toBe(
      'The plan "Recover S-1001" expired before anyone decided it, so none of its steps ran. ' +
        "Propose what is still needed again, with the records as they are now, for me to approve.",
    );
  });

  it("takes a later plan in the conversation as the one that picks up", () => {
    const first = plan();
    const later = plan({ id: "apl_2", createdAt: first.createdAt + 60 });
    const earlier = plan({ id: "apl_0", createdAt: first.createdAt - 60 });
    expect(supersededBy(first, [earlier, first, later])?.id).toBe("apl_2");
    expect(supersededBy(later, [earlier, first, later])).toBeNull();
  });
});

describe("planLog", () => {
  it("tells an approval from the person, its note, the commit and each step", () => {
    const decided = plan({
      status: "Failed",
      decidedAt: NOW - 500,
      decidedByUserId: "usr_me",
      decidedByName: "Maya Chen",
      decisionNote: "Go ahead.",
      previewDigest: "sha256:abcdef0123456789",
      commitsAt: NOW - 490,
    });
    const views = planSteps(
      payload,
      [
        step({ status: "Executed", executedAt: NOW - 480 }),
        step({ id: "aprop_2", planStep: 2, status: "ExecutionFailed", executionError: "no" }),
      ],
      undefined,
      "failed",
    );

    expect(
      planLog(decided, views, "failed", "usr_me", t).map((entry) => [
        entry.text,
        entry.at,
        entry.note,
        entry.mine,
      ]),
    ).toEqual([
      ["Dispatch proposed 2 changes", NOW - 600, "", false],
      ["You approved the plan as previewed", NOW - 500, "Go ahead.", true],
      ["Went through after the undo window", NOW - 490, "", false],
      ["Step 1 ran", NOW - 480, "", false],
      ["Step 2 did not go through", null, "", false],
    ]);
  });

  it("names another decider, and ends a declined plan at the decline", () => {
    const declined = plan({
      status: "Rejected",
      decidedAt: NOW - 100,
      decidedByUserId: "usr_other",
      decidedByName: "Ana Ruiz",
      decisionNote: "Not today.",
    });
    expect(
      planLog(declined, [], "declined", "usr_me", t).map((entry) => [entry.text, entry.note]),
    ).toEqual([
      ["Dispatch proposed 2 changes", ""],
      ["Ana Ruiz declined the plan", "Not today."],
    ]);
  });

  it("says an approval was taken back and that the plan expired undecided", () => {
    const expired = plan({ status: "Expired", undoneAt: NOW - 300, expiresAt: NOW - 10 });
    expect(planLog(expired, [], "expired", "usr_me", t).map((entry) => entry.text)).toEqual([
      "Dispatch proposed 2 changes",
      "An approval was taken back in its undo window",
      "Expired without a decision",
    ]);
  });
});

describe("formatting", () => {
  it("shortens a digest to the characters a person compares", () => {
    expect(shortDigest("sha256:7f3a91c04be2d8e5a6f0c21e")).toBe("7f3a…c21e");
    expect(shortDigest("abc")).toBe("abc");
  });

  it("writes an undo window's seconds as a clock", () => {
    expect(countdown(594)).toBe("9:54");
    expect(countdown(7)).toBe("0:07");
    expect(countdown(-3)).toBe("0:00");
  });
});
